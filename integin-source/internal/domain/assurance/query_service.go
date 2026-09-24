package assurance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"
	assurance_rules "integin/internal/domain/assurance/rules"
)

// CurrentState represents the materialized O(1) read model for the dashboard.
type CurrentState struct {
	TenantID       uuid.UUID  `db:"tenant_id"`
	AssetID        uuid.UUID  `db:"asset_id"`
	RecordID       uuid.UUID  `db:"record_id"`
	State          string     `db:"state"`
	LamportClock   int64      `db:"lamport_clock"`
	Revision       int        `db:"revision"`
	EffectiveAt    time.Time       `db:"effective_at"`
	ExpiresAt      *time.Time      `db:"expires_at"`
	Reasons        json.RawMessage `db:"reasons"` // JSONB
	EvidenceChain  json.RawMessage `db:"evidence_chain"` // JSONB
	IsSigned       bool            `db:"is_signed"`
	AuditLedgerCID *string         `db:"audit_ledger_cid"`
	UpdatedAt      time.Time       `db:"updated_at"`
}

// QueryService defines the Hexagonal read ports for the Assurance domain.
type QueryService interface {
	GetCurrentState(ctx context.Context, tenantID, assetID uuid.UUID) (*CurrentState, error)
	GetLedgerHistory(ctx context.Context, tenantID, assetID uuid.UUID, limit int) ([]Record, error)
	GetRuleBundle(ctx context.Context, tenantID uuid.UUID, bundleID string) (*assurance_rules.RuleBundleJSON, error)
}

type sqlQueryService struct {
	db *sqlx.DB
}

func NewQueryService(db *sqlx.DB) QueryService {
	return &sqlQueryService{db: db}
}

func (s *sqlQueryService) GetCurrentState(ctx context.Context, tenantID, assetID uuid.UUID) (*CurrentState, error) {
	var state CurrentState
	
	// Open RLS-bound transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin query tx: %w", err)
	}
	defer tx.Rollback()

	// Enforce RLS boundary for this connection lease
	if _, err := tx.ExecContext(ctx, "SET LOCAL app.current_tenant = $1", tenantID.String()); err != nil {
		return nil, fmt.Errorf("failed to set RLS tenant context: %w", err)
	}

	query := `
		SELECT 
			tenant_id, asset_id, record_id, state, lamport_clock, revision,
			effective_at, expires_at, reasons, evidence_chain, is_signed,
			audit_ledger_cid, updated_at
		FROM current_assurance_states
		WHERE tenant_id = $1 AND asset_id = $2
	`
	if err := tx.GetContext(ctx, &state, query, tenantID, assetID); err != nil {
		return nil, fmt.Errorf("failed to get current assurance state: %w", err)
	}
	return &state, nil
}

func (s *sqlQueryService) GetLedgerHistory(ctx context.Context, tenantID, assetID uuid.UUID, limit int) ([]Record, error) {
	var records []Record

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin query tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SET LOCAL app.current_tenant = $1", tenantID.String()); err != nil {
		return nil, fmt.Errorf("failed to set RLS tenant context: %w", err)
	}

	query := `
		SELECT 
			id, tenant_id, asset_id, state, previous_state, transition_guard,
			lamport_clock, revision, effective_at, expires_at,
			reasons, evidence_chain, evaluator_version, rule_hash, config_hash,
			evaluated_by, signature, signed_by, audit_ledger_cid, is_offline_origin,
			created_at
		FROM assurance_records
		WHERE tenant_id = $1 AND asset_id = $2
		ORDER BY lamport_clock DESC
		LIMIT $3
	`
	if err := tx.SelectContext(ctx, &records, query, tenantID, assetID, limit); err != nil {
		return nil, fmt.Errorf("failed to fetch ledger history: %w", err)
	}
	return records, nil
}

func (s *sqlQueryService) GetRuleBundle(ctx context.Context, tenantID uuid.UUID, bundleID string) (*assurance_rules.RuleBundleJSON, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SET LOCAL app.current_tenant = $1", tenantID.String()); err != nil {
		return nil, fmt.Errorf("failed to set RLS tenant context: %w", err)
	}

	var row struct {
		BundleID string `db:"bundle_id"`
		Version  string `db:"version"`
		Rules    []byte `db:"rules"`
		RuleHash string `db:"rule_hash"`
	}

	query := `
		SELECT bundle_id, version, rules, rule_hash 
		FROM assurance_rule_bundles 
		WHERE bundle_id = $1 
		ORDER BY 
			(tenant_id IS NOT NULL) DESC, -- Prioritize Tenant Overrides
			created_at DESC 
		LIMIT 1
	`
	if err := tx.GetContext(ctx, &row, query, bundleID); err != nil {
		return nil, fmt.Errorf("failed to fetch rule bundle %s: %w", bundleID, err)
	}

	var rules []assurance_rules.DecisionRule
	if err := json.Unmarshal(row.Rules, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse rules array JSON: %w", err)
	}

	bundle := &assurance_rules.RuleBundleJSON{
		BundleID: row.BundleID,
		Version:  row.Version,
		Rules:    rules,
		RuleHash: row.RuleHash,
	}

	return bundle, nil
}
