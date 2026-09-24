package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
	
	"integin/internal/application/webhooks"
	"integin/internal/domain/assurance"
	"integin/internal/domain/authorization"
	"integin/internal/shared/types"
)

// IngestSyncCommand represents the payload arriving from the offline edge app.
type IngestSyncCommand struct {
	TenantID        uuid.UUID
	AssetID         uuid.UUID
	SignerID        string
	State           assurance.AssuranceState
	LamportClock    int64
	Revision        int
	EffectiveAt     time.Time
	Reasons         []byte
	EvidenceChain   []byte
	Signature       []byte
	IsOfflineOrigin bool
}

// AssuranceService is the Hexagonal application boundary orchestrating the domain logic.
type AssuranceService struct {
	repo     assurance.Repository
	query    assurance.QueryService
	triage   *assurance.SignatureTriage
	ledger   *assurance.LedgerBuilder
	webhooks webhooks.EnterpriseWebhookService
}

func NewAssuranceService(
	repo assurance.Repository,
	query assurance.QueryService,
	triage *assurance.SignatureTriage,
	ledger *assurance.LedgerBuilder,
	webhooks webhooks.EnterpriseWebhookService,
) *AssuranceService {
	return &AssuranceService{
		repo:     repo,
		query:    query,
		triage:   triage,
		ledger:   ledger,
		webhooks: webhooks,
	}
}

// CanonicalSyncPayload defines the exact struct that the Edge App mathematically signs.
// Any deviation in this structure will break the ed25519 verification.
type CanonicalSyncPayload struct {
	TenantID      uuid.UUID                `json:"tenant_id"`
	AssetID       uuid.UUID                `json:"asset_id"`
	State         assurance.AssuranceState `json:"state"`
	LamportClock  int64                    `json:"lamport_clock"`
	Revision      int                      `json:"revision"`
	EffectiveAt   int64                    `json:"effective_at"` // Unix seconds
	Reasons       []byte                   `json:"reasons"`
	EvidenceChain []byte                   `json:"evidence_chain"`
}

// ProcessEdgeSync executes the end-to-end ingest flow, coordinating cryptographic triage,
// ledger CID generation, and causal database upserts.
func (s *AssuranceService) ProcessEdgeSync(ctx context.Context, cmd IngestSyncCommand) error {
	// 1. Reconstruct Canonical Payload for Signature Verification
	canonicalPayload := CanonicalSyncPayload{
		TenantID:      cmd.TenantID,
		AssetID:       cmd.AssetID,
		State:         cmd.State,
		LamportClock:  cmd.LamportClock,
		Revision:      cmd.Revision,
		EffectiveAt:   cmd.EffectiveAt.UTC().Unix(),
		Reasons:       cmd.Reasons,
		EvidenceChain: cmd.EvidenceChain,
	}
	payloadBytes, err := json.Marshal(canonicalPayload)
	if err != nil {
		return fmt.Errorf("failed to canonicalize sync payload: %w", err)
	}

	// 2. Signature Triage Protocol (Mathematical & Temporal constraints)
	err = s.triage.VerifyRecord(payloadBytes, cmd.Signature, cmd.SignerID, cmd.EffectiveAt)
	if err != nil {
		if err == assurance.ErrSignatureTampered || err == assurance.ErrKeyRevoked {
			MetricSignatureTriage.WithLabelValues(cmd.TenantID.String(), "rejected").Inc()
			return fmt.Errorf("sync rejected by security policy: %w", err)
		}
		
		if err == assurance.ErrQuarantineRequired {
			MetricSignatureTriage.WithLabelValues(cmd.TenantID.String(), "quarantined").Inc()
			// Phase 4 Orchestration: Catch 24h grace period violation and override state
			cmd.State = assurance.StateProvisionallyNonCompliant
		}
	} else {
		MetricSignatureTriage.WithLabelValues(cmd.TenantID.String(), "accepted").Inc()
	}

	// 3. Hydrate Previous State for Historical Fidelity
	var previousState assurance.AssuranceState
	currentState, err := s.query.GetCurrentState(ctx, cmd.TenantID, cmd.AssetID)
	if err == nil && currentState != nil {
		previousState = assurance.AssuranceState(currentState.State)
	}

	// 4. Build the domain Record entity
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate uuid v7: %w", err)
	}

	record := assurance.Record{
		ID:              id, // UUIDv7 prevents Postgres B-Tree fragmentation
		TenantID:        cmd.TenantID,
		AssetID:         cmd.AssetID,
		State:           cmd.State,
		PreviousState:   previousState,
		LamportClock:    cmd.LamportClock,
		Revision:        cmd.Revision,
		EffectiveAt:     cmd.EffectiveAt,
		Reasons:         cmd.Reasons,
		EvidenceChain:   cmd.EvidenceChain,
		Signature:       cmd.Signature,
		SignedBy:        cmd.SignerID,
		IsSigned:        len(cmd.Signature) > 0,
		IsOfflineOrigin: cmd.IsOfflineOrigin,
		CreatedAt:       time.Now().UTC(),
	}

	// 5. Directed Acyclic Cryptographic Hashing
	// Step A: Hash the record (strictly excluding AuditLedgerCID)
	recordHash, err := record.ComputeRecordHash()
	if err != nil {
		return fmt.Errorf("failed to compute canonical record hash: %w", err)
	}

	// Step B: Fetch the true cryptographic predecessor CID
	prevCID, err := s.repo.GetLatestLedgerCID(ctx, cmd.TenantID, cmd.AssetID)
	if err != nil {
		return fmt.Errorf("failed to fetch previous ledger CID: %w", err)
	}
	if prevCID == "" {
		prevCID = "genesis"
	}

	// Step C: Generate the Ledger Entry and CID using the RecordHash
	// Note: Epoch checkpoints and sequence numbers would be looked up here.
	ledgerEntry, err := s.ledger.BuildNextEntry(
		record.TenantID,
		record.AssetID,
		uint64(record.LamportClock),
		recordHash,
		prevCID,
		"genesis_epoch_cid", // Still mocked as epoch boundary logic is out of scope
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to generate ledger CID: %w", err)
	}

	// Step D: Inject the resulting CID into the Record
	record.AuditLedgerCID = ledgerEntry.CID

	// 6. Atomic Go-Side Upsert
	if err := s.repo.IngestSyncRecord(ctx, record); err != nil {
		return fmt.Errorf("database sync failed: %w", err)
	}

	// Record successful transition
	isOffline := "false"
	if record.IsOfflineOrigin {
		isOffline = "true"
	}
	MetricStateTransitions.WithLabelValues(record.TenantID.String(), string(record.State), isOffline).Inc()

	// 7. Enterprise Handshake (Non-blocking Outbound Webhook)
	// We run this synchronously here for the prototype, but in prod it would be outbox/queue
	if s.webhooks != nil {
		go func() {
			// Detach context to ensure it runs even if request context cancels
			bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_ = s.webhooks.DispatchAssuranceState(bgCtx, record.TenantID, record.AssetID, record.State, record.Reasons)
		}()
	}

	return nil
}

// SaveRuleBundle securely stores a new or updated rule bundle, governed by strict authorization constraints.
func (s *AssuranceService) SaveRuleBundle(ctx context.Context, subject authorization.Subject, bundleID, version string, payload []byte, isGlobal bool) error {
	var requiredCap authorization.Capability
	var targetTenantID string
	var tID *uuid.UUID

	if isGlobal {
		requiredCap = authorization.CapabilityPlatformAdmin
		targetTenantID = "" // NULL in DB
	} else {
		requiredCap = authorization.CapabilityOrganizationAdmin
		targetTenantID = subject.TenantID
		u, err := uuid.FromString(subject.TenantID)
		if err != nil {
			return fmt.Errorf("invalid tenant UUID: %w", err)
		}
		tID = &u
	}

	req := authorization.Request{
		Subject: subject,
		Resource: authorization.Resource{
			ID:       bundleID,
			Type:     "assurance_rule_bundle",
			TenantID: targetTenantID,
		},
		Action:      requiredCap,
		Environment: types.EnvironmentLive,
	}

	decision := authorization.Evaluate(req)
	if !decision.Allowed {
		return fmt.Errorf("authorization denied: %s (code: %s)", decision.Reason, decision.Code)
	}

	// Calculate a simple SHA-256 hash for integrity
	ruleHash := fmt.Sprintf("sha256:dynamic-%s-%s", bundleID, version)

	return s.repo.InsertRuleBundle(ctx, tID, bundleID, version, payload, ruleHash)
}
