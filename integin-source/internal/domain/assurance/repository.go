package assurance

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"
)

// Repository handles the atomic persistence of Assurance records.
type Repository interface {
	IngestSyncRecord(ctx context.Context, rec Record) error
	GetLatestLedgerCID(ctx context.Context, tenantID, assetID uuid.UUID) (string, error)
	InsertRuleBundle(ctx context.Context, tenantID *uuid.UUID, bundleID, version string, rules []byte, ruleHash string) error
}

type sqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &sqlRepository{db: db}
}

// GetLatestLedgerCID retrieves the most recent cryptographic ledger CID for an asset to bind the next link.
func (r *sqlRepository) GetLatestLedgerCID(ctx context.Context, tenantID, assetID uuid.UUID) (string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin cid lookup tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SET LOCAL app.current_tenant = $1", tenantID.String()); err != nil {
		return "", fmt.Errorf("failed to set RLS tenant context for cid lookup: %w", err)
	}

	var cid string
	query := `
		SELECT audit_ledger_cid 
		FROM assurance_records 
		WHERE tenant_id = $1 AND asset_id = $2 
		ORDER BY created_at DESC 
		LIMIT 1
	`
	if err := tx.GetContext(ctx, &cid, query, tenantID, assetID); err != nil {
		// If no records exist, return an empty string to signal genesis block
		return "", nil
	}
	return cid, nil
}

// IngestSyncRecord executes a Go-side atomic transaction to safely ingest an edge sync
// record. It relies on the Lamport causal clock and the Severity Monotonicity rules
// evaluated within Postgres ON CONFLICT DO UPDATE guards.
func (r *sqlRepository) IngestSyncRecord(ctx context.Context, rec Record) error {
	// The transaction isolates the insert into the append-only ledger and the upsert
	// into the materialized dashboard lookup table.
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin ingest transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SET LOCAL app.current_tenant = $1", rec.TenantID.String()); err != nil {
		return fmt.Errorf("failed to set RLS tenant context for ingest: %w", err)
	}

	// 1. Insert into immutable history
	insertSQL := `
		INSERT INTO assurance_records (
			id, tenant_id, asset_id, state, previous_state, transition_guard,
			lamport_clock, revision, effective_at, expires_at,
			reasons, evidence_chain, evaluator_version, rule_hash, config_hash,
			evaluated_by, signature, signed_by, audit_ledger_cid, is_offline_origin,
			created_at
		) VALUES (
			:id, :tenant_id, :asset_id, :state, :previous_state, :transition_guard,
			:lamport_clock, :revision, :effective_at, :expires_at,
			:reasons, :evidence_chain, :evaluator_version, :rule_hash, :config_hash,
			:evaluated_by, :signature, :signed_by, :audit_ledger_cid, :is_offline_origin,
			:created_at
		);
	`
	_, err = tx.NamedExecContext(ctx, insertSQL, rec)
	if err != nil {
		return fmt.Errorf("failed to append assurance_records ledger: %w", err)
	}

	// 2. Atomic UPSERT into current state with Lamport ordering and Severity Precedence
	// Postgres will map the severity levels dynamically using a CASE statement.
	// This ensures that older critical hazards from offline edges are never overwritten
	// by newer routine planned syncs, while identical severities fallback to Lamport clocks.
	upsertSQL := `
		INSERT INTO current_assurance_states (
			tenant_id, asset_id, record_id, state, lamport_clock, revision,
			effective_at, expires_at, reasons, evidence_chain, is_signed,
			audit_ledger_cid, updated_at
		) VALUES (
			:tenant_id, :asset_id, :id, :state, :lamport_clock, :revision,
			:effective_at, :expires_at, :reasons, :evidence_chain, :is_signed,
			:audit_ledger_cid, now()
		)
		ON CONFLICT (tenant_id, asset_id) DO UPDATE SET
			record_id        = EXCLUDED.record_id,
			state            = EXCLUDED.state,
			lamport_clock    = EXCLUDED.lamport_clock,
			revision         = EXCLUDED.revision,
			effective_at     = EXCLUDED.effective_at,
			expires_at       = EXCLUDED.expires_at,
			reasons          = EXCLUDED.reasons,
			evidence_chain   = EXCLUDED.evidence_chain,
			is_signed        = EXCLUDED.is_signed,
			audit_ledger_cid = EXCLUDED.audit_ledger_cid,
			updated_at       = now()
		WHERE (
			CASE EXCLUDED.state
				WHEN 'CONDEMNED' THEN 130 WHEN 'REVOKED' THEN 120 WHEN 'EXPIRED' THEN 110
				WHEN 'NON_COMPLIANT' THEN 100 WHEN 'PROVISIONALLY_NON_COMPLIANT' THEN 90
				WHEN 'SUSPENDED' THEN 80 WHEN 'RESTRICTED' THEN 70 WHEN 'CONDITIONAL' THEN 60
				WHEN 'ASSURED' THEN 50 WHEN 'UNDER_REVIEW' THEN 40 WHEN 'EVIDENCE_PENDING' THEN 30
				WHEN 'IN_PROGRESS' THEN 20 WHEN 'PLANNED' THEN 10 ELSE 0
			END
			>
			CASE current_assurance_states.state
				WHEN 'CONDEMNED' THEN 130 WHEN 'REVOKED' THEN 120 WHEN 'EXPIRED' THEN 110
				WHEN 'NON_COMPLIANT' THEN 100 WHEN 'PROVISIONALLY_NON_COMPLIANT' THEN 90
				WHEN 'SUSPENDED' THEN 80 WHEN 'RESTRICTED' THEN 70 WHEN 'CONDITIONAL' THEN 60
				WHEN 'ASSURED' THEN 50 WHEN 'UNDER_REVIEW' THEN 40 WHEN 'EVIDENCE_PENDING' THEN 30
				WHEN 'IN_PROGRESS' THEN 20 WHEN 'PLANNED' THEN 10 ELSE 0
			END
		)
		OR (
			(CASE EXCLUDED.state WHEN 'CONDEMNED' THEN 130 WHEN 'REVOKED' THEN 120 WHEN 'EXPIRED' THEN 110 WHEN 'NON_COMPLIANT' THEN 100 WHEN 'PROVISIONALLY_NON_COMPLIANT' THEN 90 WHEN 'SUSPENDED' THEN 80 WHEN 'RESTRICTED' THEN 70 WHEN 'CONDITIONAL' THEN 60 WHEN 'ASSURED' THEN 50 WHEN 'UNDER_REVIEW' THEN 40 WHEN 'EVIDENCE_PENDING' THEN 30 WHEN 'IN_PROGRESS' THEN 20 WHEN 'PLANNED' THEN 10 ELSE 0 END)
			= 
			(CASE current_assurance_states.state WHEN 'CONDEMNED' THEN 130 WHEN 'REVOKED' THEN 120 WHEN 'EXPIRED' THEN 110 WHEN 'NON_COMPLIANT' THEN 100 WHEN 'PROVISIONALLY_NON_COMPLIANT' THEN 90 WHEN 'SUSPENDED' THEN 80 WHEN 'RESTRICTED' THEN 70 WHEN 'CONDITIONAL' THEN 60 WHEN 'ASSURED' THEN 50 WHEN 'UNDER_REVIEW' THEN 40 WHEN 'EVIDENCE_PENDING' THEN 30 WHEN 'IN_PROGRESS' THEN 20 WHEN 'PLANNED' THEN 10 ELSE 0 END)
			AND EXCLUDED.lamport_clock >= current_assurance_states.lamport_clock
		);
	`
	_, err = tx.NamedExecContext(ctx, upsertSQL, rec)
	if err != nil {
		return fmt.Errorf("failed to upsert current_assurance_states: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit sync ingest: %w", err)
	}

	return nil
}

func (r *sqlRepository) InsertRuleBundle(ctx context.Context, tenantID *uuid.UUID, bundleID, version string, rules []byte, ruleHash string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	if tenantID != nil {
		if _, err := tx.ExecContext(ctx, "SET LOCAL app.current_tenant = $1", tenantID.String()); err != nil {
			return fmt.Errorf("failed to set RLS tenant context: %w", err)
		}
	} else {
		// Global rule insert, bypass RLS isolation to allow writing NULL tenant
	}

	query := `
		INSERT INTO assurance_rule_bundles (tenant_id, bundle_id, version, rules, rule_hash)
		VALUES ($1, $2, $3, $4, $5)
	`
	var tID interface{}
	if tenantID != nil {
		tID = *tenantID
	}

	if _, err := tx.ExecContext(ctx, query, tID, bundleID, version, rules, ruleHash); err != nil {
		return fmt.Errorf("failed to insert rule bundle: %w", err)
	}

	return tx.Commit()
}
