package federation

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrSovereignBoundaryBreach = errors.New("federation: sovereign regional policy prohibits data export for this class")
	ErrReplayDetected          = errors.New("federation: message counter or timestamp stale (replay attack blocked)")
	ErrTamperedSignature       = errors.New("federation: cryptographic signature validation failed")
	ErrUnknownOriginCell       = errors.New("federation: origin cell not in authorized sovereign topology")
	ErrZeroTenantID            = errors.New("federation: tenant ID cannot be empty")
)

type SovereignCellID string

const (
	CellSaudiCentral01 SovereignCellID = "cell-sa-central-01" // SDAIA / PDPL data residency boundary
	CellEUWest01       SovereignCellID = "cell-eu-west-01"    // GDPR Chapter V boundary
	CellUSGovEast01    SovereignCellID = "cell-us-gov-01"    // FedRAMP / ITAR boundary
)

type DataClassification string

const (
	ClassSovereignRestricted DataClassification = "SOVEREIGN_RESTRICTED" // Never leaves originating nation
	ClassAnonymizedTelemetric DataClassification = "ANONYMIZED_TELEMETRIC" // Cross-cell sync allowed
	ClassGlobalPublicTrust    DataClassification = "GLOBAL_PUBLIC_TRUST"   // Free worldwide broadcast
)

// FederationMessage represents a cross-cell conflict-free replication envelope.
type FederationMessage struct {
	MessageID      string             `json:"message_id"`
	OriginCell     SovereignCellID    `json:"origin_cell"`
	TargetCell     SovereignCellID    `json:"target_cell"`
	TenantID       string             `json:"tenant_id"`
	Classification DataClassification `json:"classification"`
	EntityDID      string             `json:"entity_did"`
	VersionEpoch   int64              `json:"version_epoch"` // Logical Lamport / Vector timestamp
	PayloadDigest  string             `json:"payload_digest"`
	PayloadHex     string             `json:"payload_hex"`
	DispatchedAt   time.Time          `json:"dispatched_at"`
	SignatureHex   string             `json:"signature_hex"`
}

// FederationGatekeeper enforces sovereign export policies and verifies signatures.
type FederationGatekeeper struct {
	mu           sync.Mutex
	localCell    SovereignCellID
	knownCells   map[SovereignCellID]ed25519.PublicKey
	seenEpochs   map[string]int64 // entity_did -> max seen epoch
}

// NewFederationGatekeeper initializes sovereign compliance gatekeeper.
func NewFederationGatekeeper(localCell SovereignCellID) *FederationGatekeeper {
	return &FederationGatekeeper{
		localCell:  localCell,
		knownCells: make(map[SovereignCellID]ed25519.PublicKey),
		seenEpochs: make(map[string]int64),
	}
}

// RegisterCell adds an authorized regional cell's signing authority.
func (fg *FederationGatekeeper) RegisterCell(cellID SovereignCellID, pubKey ed25519.PublicKey) {
	fg.mu.Lock()
	defer fg.mu.Unlock()
	fg.knownCells[cellID] = pubKey
}

// DispatchCrossCellEnvelope constructs and signs an outbound replication envelope.
func (fg *FederationGatekeeper) DispatchCrossCellEnvelope(
	targetCell SovereignCellID,
	tenantID string,
	class DataClassification,
	entityDID string,
	epoch int64,
	payload []byte,
	privKey ed25519.PrivateKey,
) (*FederationMessage, error) {
	if tenantID == "" {
		return nil, ErrZeroTenantID
	}

	// 1. Sovereign Boundary Egress Policy: Restricted data CANNOT cross national borders
	if class == ClassSovereignRestricted && targetCell != fg.localCell {
		return nil, ErrSovereignBoundaryBreach
	}

	digest := sha256.Sum256(payload)
	now := time.Now().UTC()
	msgID := fmt.Sprintf("fed-%s-%s-%d", fg.localCell, entityDID, epoch)

	msg := &FederationMessage{
		MessageID:      msgID,
		OriginCell:     fg.localCell,
		TargetCell:     targetCell,
		TenantID:       tenantID,
		Classification: class,
		EntityDID:      entityDID,
		VersionEpoch:   epoch,
		PayloadDigest:  hex.EncodeToString(digest[:]),
		PayloadHex:     hex.EncodeToString(payload),
		DispatchedAt:   now,
	}

	signPayload := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s|%d",
		msg.MessageID, msg.OriginCell, msg.TargetCell, msg.TenantID,
		msg.EntityDID, msg.VersionEpoch, msg.PayloadDigest, msg.DispatchedAt.Unix(),
	)
	signDigest := sha256.Sum256([]byte(signPayload))
	sig := ed25519.Sign(privKey, signDigest[:])
	msg.SignatureHex = hex.EncodeToString(sig)

	return msg, nil
}

// IngestCrossCellEnvelope verifies authenticity, causal monotonicity, and inbound sovereign admissibility.
func (fg *FederationGatekeeper) IngestCrossCellEnvelope(msg *FederationMessage) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("federation: message is nil")
	}

	fg.mu.Lock()
	defer fg.mu.Unlock()

	// 1. Validate Origin Cell Authority
	pubKey, ok := fg.knownCells[msg.OriginCell]
	if !ok {
		return nil, ErrUnknownOriginCell
	}

	// 2. Sovereign Ingress Policy Check: Destination must match this cell
	if msg.TargetCell != fg.localCell {
		return nil, errors.New("federation: message misrouted to incorrect regional cell")
	}

	// 3. Cryptographic Signature Validation
	signPayload := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s|%d",
		msg.MessageID, msg.OriginCell, msg.TargetCell, msg.TenantID,
		msg.EntityDID, msg.VersionEpoch, msg.PayloadDigest, msg.DispatchedAt.Unix(),
	)
	signDigest := sha256.Sum256([]byte(signPayload))
	sigBytes, err := hex.DecodeString(msg.SignatureHex)
	if err != nil || !ed25519.Verify(pubKey, signDigest[:], sigBytes) {
		return nil, ErrTamperedSignature
	}

	// 4. Causal Monotonicity / CRDT Replay Protection: Epoch must strictly advance
	lastEpoch, seen := fg.seenEpochs[msg.EntityDID]
	if seen && msg.VersionEpoch <= lastEpoch {
		return nil, ErrReplayDetected
	}

	// 5. Payload Integrity Verification
	payload, err := hex.DecodeString(msg.PayloadHex)
	if err != nil {
		return nil, fmt.Errorf("federation: invalid hex payload: %w", err)
	}
	actualDigest := sha256.Sum256(payload)
	if hex.EncodeToString(actualDigest[:]) != msg.PayloadDigest {
		return nil, errors.New("federation: payload digest mismatch")
	}

	// Advance tracked causal state
	fg.seenEpochs[msg.EntityDID] = msg.VersionEpoch

	return payload, nil
}
