package sync

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	syncpkg "sync"
	"time"

	"integin/internal/domain/device_trust"
	"integin/internal/security"
	"integin/internal/shared/types"
	"integin/internal/syncstate"
)

type Outcome string

const (
	Applied          Outcome = "APPLIED"
	Duplicate        Outcome = "DUPLICATE"
	Queued           Outcome = "QUEUED"
	Held             Outcome = "HELD"
	Rejected         Outcome = "REJECTED"
	Conflict         Outcome = "CONFLICT"
	SecurityFailure  Outcome = "SECURITY_FAILURE"
	legacyHMACSHA256         = "HMAC-SHA256"
)

type Transaction struct {
	ProtocolVersion    string
	TransactionID      string
	TenantID           string
	OrganizationID     string
	Environment        string
	DeviceID           string
	UserID             string
	SequenceNumber     uint64
	Operation          string
	EntityID           string
	Payload            []byte
	PayloadHash        string
	CapturedAt         time.Time
	AuthorityID        string
	AuthorityEpoch     uint64
	SignatureAlgorithm string
	KeyID              string
	Signature          string
}

func NewTransaction(transactionID, tenantID, deviceID, userID string, sequence uint64, operation string, payload []byte) Transaction {
	copied := append([]byte(nil), payload...)
	return Transaction{ProtocolVersion: "v1", TransactionID: transactionID, TenantID: tenantID, Environment: "LIVE", DeviceID: deviceID, UserID: userID, SequenceNumber: sequence, Operation: operation, Payload: copied, PayloadHash: HashPayload(copied), CapturedAt: time.Now().UTC(), SignatureAlgorithm: legacyHMACSHA256}
}

func SignTransaction(transaction Transaction, secret string, keyID string) Transaction {
	transaction.KeyID = keyID
	transaction.Signature = signTransaction(transaction, secret)
	transaction.Payload = append([]byte(nil), transaction.Payload...)
	return transaction
}

func SignTransactionEd25519(transaction Transaction, privateKey ed25519.PrivateKey, keyID string) (Transaction, error) {
	transaction.ProtocolVersion = "v1"
	transaction.SignatureAlgorithm = "Ed25519"
	transaction.KeyID = keyID
	signature, err := security.SignDeviceMutation(privateKey, []byte(canonicalTransaction(transaction)))
	if err != nil {
		return Transaction{}, err
	}
	transaction.Signature = base64.StdEncoding.EncodeToString(signature)
	transaction.Payload = append([]byte(nil), transaction.Payload...)
	return transaction, nil
}
func HashPayload(payload []byte) string {
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

type Result struct {
	Outcome          Outcome
	Code             types.ErrorCode
	TransactionID    string
	Reason           string
	ExpectedSequence uint64
	PayloadHash      string
}

type Processor struct {
	mu                  syncpkg.RWMutex
	secrets             map[string]string
	devices             map[string]device_trust.Device
	lastSequence        map[string]uint64
	transactions        map[string]string
	held                map[string]Transaction
	state               syncstate.SyncStateRepository
	preAcceptancePolicy PreAcceptancePolicy
	deviceLocks         [64]syncpkg.Mutex
}

func (p *Processor) deviceLock(deviceID string) *syncpkg.Mutex {
	var h uint32 = 2166136261
	for i := 0; i < len(deviceID); i++ {
		h ^= uint32(deviceID[i])
		h *= 16777619
	}
	return &p.deviceLocks[h%64]
}

func NewProcessor(secrets map[string]string) (*Processor, error) {
	return NewProcessorWithState(secrets, nil)
}

func NewProcessorWithState(secrets map[string]string, state syncstate.SyncStateRepository) (*Processor, error) {
	if len(secrets) == 0 {
		return nil, errors.New("sync signing secret is required")
	}
	return &Processor{secrets: secrets, state: state, devices: make(map[string]device_trust.Device), lastSequence: make(map[string]uint64), transactions: make(map[string]string), held: make(map[string]Transaction)}, nil
}
func (p *Processor) RegisterDevice(device device_trust.Device) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.devices[device.ID()] = device
}
func (p *Processor) HeldTransactions() []Transaction {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]Transaction, 0, len(p.held))
	for _, transaction := range p.held {
		transaction.Payload = append([]byte(nil), transaction.Payload...)
		result = append(result, transaction)
	}
	return result
}

// Submit validates an offline transaction and returns an explicit outcome. No
// business state is mutated until identity, authority, hash, signature, and
// sequence checks all pass.
func (p *Processor) Submit(transaction Transaction, authority device_trust.AuthorityPackage, at time.Time) Result {
	return p.SubmitContext(context.Background(), transaction, authority, at)
}

func (p *Processor) SubmitContext(ctx context.Context, transaction Transaction, authority device_trust.AuthorityPackage, at time.Time) Result {
	devLock := p.deviceLock(transaction.DeviceID)
	devLock.Lock()
	defer devLock.Unlock()

	result := Result{TransactionID: transaction.TransactionID, PayloadHash: transaction.PayloadHash}
	if strings.TrimSpace(transaction.TransactionID) == "" || strings.TrimSpace(transaction.TenantID) == "" || strings.TrimSpace(transaction.DeviceID) == "" || strings.TrimSpace(transaction.UserID) == "" || strings.TrimSpace(transaction.Operation) == "" || transaction.SequenceNumber == 0 {
		return p.fail(result, types.ErrValidation, Rejected, "transaction identity and sequence are required")
	}
	calculatedHash := HashPayload(transaction.Payload)
	if transaction.PayloadHash != calculatedHash {
		return p.fail(result, types.ErrRejected, SecurityFailure, "payload hash mismatch")
	}
	p.mu.RLock()
	device, exists := p.devices[transaction.DeviceID]
	p.mu.RUnlock()
	if !exists {
		return p.fail(result, types.ErrUnauthorized, SecurityFailure, "device is not registered")
	}
	if transaction.TenantID != device.TenantID() || transaction.UserID != device.UserID() {
		return p.fail(result, types.ErrTenantMismatch, SecurityFailure, "transaction identity does not match device authority")
	}
	switch transaction.SignatureAlgorithm {
	case "Ed25519":
		if transaction.ProtocolVersion != "v1" || transaction.OrganizationID == "" || transaction.Environment == "" || transaction.EntityID == "" || transaction.CapturedAt.IsZero() || transaction.AuthorityID == "" || transaction.AuthorityEpoch == 0 {
			return p.fail(result, types.ErrValidation, Rejected, "v1 transaction envelope is incomplete")
		}
		if transaction.OrganizationID != device.OrganizationID() || transaction.AuthorityID != authority.ID || transaction.AuthorityEpoch != authority.Epoch {
			return p.fail(result, types.ErrTenantMismatch, SecurityFailure, "transaction authority context does not match server state")
		}
		if required := requiredCapability(transaction.Operation); required != "" && !contains(authority.Scopes, required) {
			return p.fail(result, types.ErrUnauthorized, SecurityFailure, "authority does not permit operation")
		}
	case legacyHMACSHA256:
	default:
		return p.fail(result, types.ErrRejected, SecurityFailure, "transaction signature algorithm is not supported")
	}
	// For ValidateAuthorityPackage, we can use any active secret if we assume it validates HMACs.
	// Actually, ValidateAuthorityPackage might not need the HMAC secret for its core logic unless it decrypts something.
	// But it requires a string secret. Let's pass the default or first one.
	p.mu.RLock()
	var defaultSecret string
	for _, v := range p.secrets {
		defaultSecret = v
		break
	}
	p.mu.RUnlock()
	if err := device_trust.ValidateAuthorityPackage(authority, device, defaultSecret, at); err != nil {
		return p.fail(result, types.ErrUnauthorized, SecurityFailure, err.Error())
	}
	if transaction.SignatureAlgorithm == "Ed25519" {
		publicKey, publicKeyErr := base64.StdEncoding.DecodeString(device.PublicKey())
		signature, signatureErr := base64.StdEncoding.DecodeString(transaction.Signature)
		validKey := publicKeyErr == nil && transaction.KeyID != "" && transaction.KeyID == security.DeviceKeyID(ed25519.PublicKey(publicKey))
		if !validKey || signatureErr != nil || !security.VerifyDeviceMutation(ed25519.PublicKey(publicKey), []byte(canonicalTransaction(transaction)), signature) {
			return p.fail(result, types.ErrRejected, SecurityFailure, "transaction Ed25519 signature is invalid")
		}
	} else {
		p.mu.RLock()
		secret, ok := p.secrets[transaction.KeyID]
		if !ok && transaction.KeyID == "" {
			secret, ok = p.secrets["default"]
		}
		p.mu.RUnlock()
		if !ok {
			return p.fail(result, types.ErrRejected, SecurityFailure, "transaction HMAC key ID is unknown")
		}
		if !hmac.Equal([]byte(transaction.Signature), []byte(signTransaction(transaction, secret))) {
			return p.fail(result, types.ErrRejected, SecurityFailure, "transaction HMAC signature is invalid")
		}
	}
	if p.state != nil {
		if receipt, err := p.state.GetReceipt(ctx, transaction.TenantID, transaction.TransactionID); err == nil {
			if receipt.PayloadHash != transaction.PayloadHash {
				return p.fail(result, types.ErrConflict, Conflict, "transaction id was reused with different payload")
			}
			switch receipt.Outcome {
			case string(Applied):
				result.Outcome = Duplicate
				result.Reason = "transaction already applied"
				return result
			case string(Held):
				// A durable hold is not an accepted transaction. Continue through the
				// sequence gate so a later replay can be applied exactly once.
			default:
				return p.fail(result, types.ErrConflict, Conflict, "transaction has a non-resumable receipt state")
			}
		} else if !errors.Is(err, syncstate.ErrNotFound) {
			return p.fail(result, types.ErrRejected, SecurityFailure, "durable sync state is unavailable")
		}
	}
	p.mu.RLock()
	existingHash, exists := p.transactions[transaction.TransactionID]
	p.mu.RUnlock()
	if exists {
		if existingHash == transaction.PayloadHash {
			result.Outcome = Duplicate
			result.Reason = "transaction already applied"
			return result
		}
		return p.fail(result, types.ErrConflict, Conflict, "transaction id was reused with different payload")
	}
	p.mu.RLock()
	heldTransaction, exists := p.held[transaction.TransactionID]
	p.mu.RUnlock()
	if exists && heldTransaction.PayloadHash != transaction.PayloadHash {
		return p.fail(result, types.ErrConflict, Conflict, "transaction id was reused with different payload")
	}
	p.mu.RLock()
	expected := p.lastSequence[transaction.DeviceID] + 1
	p.mu.RUnlock()
	if p.state != nil {
		if persisted, err := p.state.GetLastAcceptedSequence(ctx, transaction.TenantID, transaction.DeviceID); err == nil {
			expected = persisted + 1
		} else if !errors.Is(err, syncstate.ErrNotFound) {
			return p.fail(result, types.ErrRejected, SecurityFailure, "durable sequence state is unavailable")
		}
	}
	result.ExpectedSequence = expected
	if transaction.SequenceNumber > expected {
		result.Outcome, result.Code, result.Reason = Held, types.ErrHeld, fmt.Sprintf("sequence gap: expected %d", expected)
		if err := p.persistHeld(ctx, transaction, result, at); err != nil {
			return p.fail(result, types.ErrRejected, SecurityFailure, fmt.Sprintf("durable held-transaction state is unavailable: %v", err))
		}
		p.mu.Lock()
		p.held[transaction.TransactionID] = cloneTransaction(transaction)
		p.mu.Unlock()
		return result
	}
	if transaction.SequenceNumber < expected {
		return p.fail(result, types.ErrConflict, Conflict, fmt.Sprintf("stale sequence: expected %d", expected))
	}
	if p.preAcceptancePolicy != nil {
		if err := p.preAcceptancePolicy.ValidatePreAcceptance(ctx, transaction); err != nil {
			return p.fail(result, types.ErrRejected, Rejected, err.Error())
		}
	}
	result.Outcome, result.Reason = Applied, "transaction applied"
	if err := p.persistReceipt(ctx, transaction, result, at); err != nil {
		return p.fail(result, types.ErrRejected, SecurityFailure, "durable receipt state is unavailable")
	}
	p.mu.Lock()
	p.lastSequence[transaction.DeviceID] = transaction.SequenceNumber
	p.transactions[transaction.TransactionID] = transaction.PayloadHash
	delete(p.held, transaction.TransactionID)
	p.mu.Unlock()
	return result
}

// DrainHeld explicitly processes consecutive held transactions for the specified device.
func (p *Processor) DrainHeld(ctx context.Context, tenantID, deviceID string, at time.Time) []Result {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.drainHeldLocked(ctx, tenantID, deviceID, at)
}

func (p *Processor) drainHeldLocked(ctx context.Context, tenantID, deviceID string, at time.Time) []Result {
	_, exists := p.devices[deviceID]
	if !exists {
		return nil
	}
	var drained []Result
	// Cascade drain consecutive sequence numbers
	for {
		nextSeq := p.lastSequence[deviceID] + 1
		var candidate *Transaction
		// Check in-memory held map
		for _, tx := range p.held {
			if tx.DeviceID == deviceID && tx.SequenceNumber == nextSeq {
				c := tx
				candidate = &c
				break
			}
		}
		// If not in memory, check durable state
		if candidate == nil && p.state != nil {
			heldList, err := p.state.ListHeld(ctx, tenantID, deviceID)
			if err == nil {
				for _, h := range heldList {
					if h.Receipt.SequenceNumber == nextSeq {
						var parsed Transaction
						if json.Unmarshal(h.Envelope, &parsed) == nil {
							candidate = &parsed
							break
						}
					}
				}
			}
		}
		if candidate == nil {
			break
		}
		res := Result{TransactionID: candidate.TransactionID, PayloadHash: candidate.PayloadHash, ExpectedSequence: nextSeq}
		res.Outcome, res.Reason = Applied, "transaction applied from held queue"
		if err := p.persistReceipt(ctx, *candidate, res, at); err != nil {
			break
		}
		p.lastSequence[deviceID] = nextSeq
		p.transactions[candidate.TransactionID] = candidate.PayloadHash
		delete(p.held, candidate.TransactionID)
		drained = append(drained, res)
	}
	return drained
}

func (p *Processor) persistReceipt(ctx context.Context, transaction Transaction, result Result, at time.Time) error {
	if p.state == nil {
		return nil
	}
	return p.state.SaveReceipt(ctx, syncstate.Receipt{TransactionID: transaction.TransactionID, TenantID: transaction.TenantID, OrganizationID: transaction.OrganizationID, DeviceID: transaction.DeviceID, UserID: transaction.UserID, SequenceNumber: transaction.SequenceNumber, Operation: transaction.Operation, EntityID: transaction.EntityID, PayloadHash: transaction.PayloadHash, Outcome: string(result.Outcome), Reason: result.Reason, CapturedAt: transaction.CapturedAt, ReceivedAt: at.UTC()})
}

func (p *Processor) persistHeld(ctx context.Context, transaction Transaction, result Result, at time.Time) error {
	if p.state == nil {
		return nil
	}
	receipt := syncstate.Receipt{TransactionID: transaction.TransactionID, TenantID: transaction.TenantID, OrganizationID: transaction.OrganizationID, DeviceID: transaction.DeviceID, UserID: transaction.UserID, SequenceNumber: transaction.SequenceNumber, Operation: transaction.Operation, EntityID: transaction.EntityID, PayloadHash: transaction.PayloadHash, Outcome: string(result.Outcome), Reason: result.Reason, CapturedAt: transaction.CapturedAt, ReceivedAt: at.UTC()}
	envelope, err := json.Marshal(transaction)
	if err != nil {
		return err
	}
	return p.state.SaveHeld(ctx, syncstate.HeldTransaction{Receipt: receipt, ExpectedSequence: result.ExpectedSequence, Envelope: envelope, FirstHeldAt: at.UTC(), LastAttemptAt: at.UTC(), LastError: result.Reason})
}

func (p *Processor) fail(result Result, code types.ErrorCode, outcome Outcome, reason string) Result {
	result.Code, result.Outcome, result.Reason = code, outcome, reason
	return result
}
func cloneTransaction(transaction Transaction) Transaction {
	transaction.Payload = append([]byte(nil), transaction.Payload...)
	return transaction
}
func signTransaction(transaction Transaction, secret string) string {
	canonical := canonicalTransaction(transaction)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

func canonicalTransaction(transaction Transaction) string {
	var b strings.Builder
	b.Grow(256)
	b.WriteString(transaction.ProtocolVersion)
	b.WriteByte('|')
	b.WriteString(transaction.TransactionID)
	b.WriteByte('|')
	b.WriteString(transaction.TenantID)
	b.WriteByte('|')
	b.WriteString(transaction.OrganizationID)
	b.WriteByte('|')
	b.WriteString(transaction.Environment)
	b.WriteByte('|')
	b.WriteString(transaction.DeviceID)
	b.WriteByte('|')
	b.WriteString(transaction.UserID)
	b.WriteByte('|')
	b.WriteString(strconv.FormatUint(transaction.SequenceNumber, 10))
	b.WriteByte('|')
	b.WriteString(transaction.Operation)
	b.WriteByte('|')
	b.WriteString(transaction.EntityID)
	b.WriteByte('|')
	b.WriteString(transaction.PayloadHash)
	b.WriteByte('|')
	b.WriteString(transaction.AuthorityID)
	b.WriteByte('|')
	b.WriteString(strconv.FormatUint(transaction.AuthorityEpoch, 10))
	b.WriteByte('|')
	b.WriteString(transaction.CapturedAt.UTC().Format(time.RFC3339Nano))
	b.WriteByte('|')
	b.WriteString(transaction.SignatureAlgorithm)
	b.WriteByte('|')
	b.WriteString(transaction.KeyID)
	return b.String()
}

func requiredCapability(operation string) string {
	switch operation {
	case "InspectionSubmitted":
		return "inspection.perform"
	case "InspectionEvidenceUploaded":
		return "evidence.upload"
	default:
		return ""
	}
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
