package assurance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/gofrs/uuid/v5"
)

// AssuranceState represents the deterministic lifecycle phase of an asset's assurance.
type AssuranceState string

const (
	StateUnassured                 AssuranceState = "UNASSURED"
	StatePlanned                   AssuranceState = "PLANNED"
	StateInProgress                AssuranceState = "IN_PROGRESS"
	StateEvidencePending           AssuranceState = "EVIDENCE_PENDING"
	StateUnderReview               AssuranceState = "UNDER_REVIEW"
	StateAssured                   AssuranceState = "ASSURED"
	StateConditional               AssuranceState = "CONDITIONAL"
	StateRestricted                AssuranceState = "RESTRICTED"
	StateSuspended                 AssuranceState = "SUSPENDED"
	StateProvisionallyNonCompliant AssuranceState = "PROVISIONALLY_NON_COMPLIANT" // Quarantine
	StateNonCompliant              AssuranceState = "NON_COMPLIANT"               // Dual-signed
	StateExpired                   AssuranceState = "EXPIRED"
	StateRevoked                   AssuranceState = "REVOKED"
	StateCondemned                 AssuranceState = "CONDEMNED"                   // Terminal absorbing
)

var (
	ErrTerminalState    = errors.New("cannot transition from terminal CONDEMNED state")
	ErrInvalidState     = errors.New("invalid assurance state transition")
	ErrDemotionRejected = errors.New("transition rejected: cannot demote to a lower severity state")
)

var StateSeverity = map[AssuranceState]int{
	StateUnassured:                 0,
	StatePlanned:                   10,
	StateInProgress:                20,
	StateEvidencePending:           30,
	StateUnderReview:               40,
	StateAssured:                   50,
	StateConditional:               60,
	StateRestricted:                70,
	StateSuspended:                 80,
	StateProvisionallyNonCompliant: 90,
	StateNonCompliant:              100,
	StateExpired:                   110,
	StateRevoked:                   120,
	StateCondemned:                 130, // Absorbing
}

// Record represents the immutable ledger entry for an assurance evaluation.
type Record struct {
	ID               uuid.UUID      `json:"id" db:"id"`
	TenantID         uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	AssetID          uuid.UUID      `json:"asset_id" db:"asset_id"`
	State            AssuranceState `json:"state" db:"state"`
	PreviousState    AssuranceState `json:"previous_state" db:"previous_state"`
	TransitionGuard  string         `json:"transition_guard" db:"transition_guard"`
	LamportClock     int64          `json:"lamport_clock" db:"lamport_clock"`
	Revision         int            `json:"revision" db:"revision"`
	EffectiveAt      time.Time       `json:"effective_at" db:"effective_at"`
	ExpiresAt        *time.Time      `json:"expires_at" db:"expires_at"`
	Reasons          json.RawMessage `json:"reasons" db:"reasons"`               // JSONB
	EvidenceChain    json.RawMessage `json:"evidence_chain" db:"evidence_chain"` // JSONB
	EvaluatorVersion string          `json:"evaluator_version" db:"evaluator_version"`
	RuleHash         string          `json:"rule_hash" db:"rule_hash"`
	ConfigHash       string          `json:"config_hash" db:"config_hash"`
	EvaluatedBy      json.RawMessage `json:"evaluated_by" db:"evaluated_by"`     // JSONB
	Signature        []byte          `json:"signature" db:"signature"`
	SignedBy         string          `json:"signed_by" db:"signed_by"`
	IsSigned         bool            `json:"is_signed" db:"is_signed"` // Explicitly extracted for DB mapping
	AuditLedgerCID   string          `json:"audit_ledger_cid" db:"audit_ledger_cid"`
	IsOfflineOrigin  bool            `json:"is_offline_origin" db:"is_offline_origin"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

// ComputeRecordHash generates a deterministic SHA-256 hash of the Record payload,
// strictly excluding the AuditLedgerCID to prevent cyclic cryptographic dependencies.
func (r *Record) ComputeRecordHash() (string, error) {
	type canonicalRecord struct {
		ID               uuid.UUID       `json:"id"`
		TenantID         uuid.UUID       `json:"tenant_id"`
		AssetID          uuid.UUID       `json:"asset_id"`
		State            AssuranceState  `json:"state"`
		LamportClock     int64           `json:"lamport_clock"`
		Revision         int             `json:"revision"`
		Reasons          json.RawMessage `json:"reasons"`
		EvidenceChain    json.RawMessage `json:"evidence_chain"`
		EvaluatorVersion string          `json:"evaluator_version"`
		RuleHash         string          `json:"rule_hash"`
		ConfigHash       string          `json:"config_hash"`
		EvaluatedBy      json.RawMessage `json:"evaluated_by"`
		Signature        []byte          `json:"signature"`
		SignedBy         string          `json:"signed_by"`
	}

	canonical := canonicalRecord{
		ID:               r.ID,
		TenantID:         r.TenantID,
		AssetID:          r.AssetID,
		State:            r.State,
		LamportClock:     r.LamportClock,
		Revision:         r.Revision,
		Reasons:          r.Reasons,
		EvidenceChain:    r.EvidenceChain,
		EvaluatorVersion: r.EvaluatorVersion,
		RuleHash:         r.RuleHash,
		ConfigHash:       r.ConfigHash,
		EvaluatedBy:      r.EvaluatedBy,
		Signature:        r.Signature,
		SignedBy:         r.SignedBy,
	}

	hashBytes, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	hasher.Write(hashBytes)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

// EvaluateTransition applies precedence logic. Higher severity states override,
// and Lamport clocks resolve offline sync conflicts.
func (r *Record) EvaluateTransition(nextState AssuranceState, edgeLamportClock int64) error {
	if r.State == StateCondemned {
		return ErrTerminalState
	}

	currentRank := StateSeverity[r.State]
	nextRank := StateSeverity[nextState]

	// Enforce Severity Monotonicity
	if nextRank < currentRank {
		return ErrDemotionRejected
	}

	// Update lamport causality
	if edgeLamportClock > r.LamportClock {
		r.LamportClock = edgeLamportClock
	} else {
		r.LamportClock++
	}

	r.PreviousState = r.State
	r.State = nextState
	r.Revision++
	return nil
}
