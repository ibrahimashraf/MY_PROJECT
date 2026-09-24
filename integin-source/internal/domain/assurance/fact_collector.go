package assurance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"

	"integin/internal/domain/assurance/rules"
)

// SnapshotFacts bundles the cross-domain records necessary for a stateless evaluation.
type SnapshotFacts struct {
	Asset             rules.AssetFacts    `json:"asset"`
	LatestInspection  *InspectionFact     `json:"latest_inspection"`
	OpenFindings      []rules.FindingFact `json:"open_findings"`
	CurrentCertificate *CertificateFact   `json:"current_certificate"`
}

type InspectionFact struct {
	ID                  uuid.UUID `json:"id"`
	Status              string    `json:"status"`
	Verdict             string    `json:"verdict"`
	AssignedInspectorID uuid.UUID `json:"assigned_inspector_id"`
}

type CertificateFact struct {
	ID           uuid.UUID `json:"id"`
	Status       string    `json:"status"`
	Restrictions []string  `json:"restrictions"`
}

// FactCollector is responsible for assembling cross-domain facts without torn reads.
type FactCollector interface {
	CollectSnapshot(ctx context.Context, tenantID, assetID uuid.UUID) (*SnapshotFacts, error)
}

type postgresFactCollector struct {
	db *sqlx.DB
}

func NewPostgresFactCollector(db *sqlx.DB) FactCollector {
	return &postgresFactCollector{db: db}
}

// CollectSnapshot executes a single REPEATABLE READ transaction to fetch all cross-domain
// facts as a coherent snapshot, eliminating network round-trips and torn read races.
func (c *postgresFactCollector) CollectSnapshot(ctx context.Context, tenantID, assetID uuid.UUID) (*SnapshotFacts, error) {
	// 3. Defensive Timeout: Bound the REPEATABLE READ snapshot to prevent Postgres bloat
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 1. Begin Snapshot-Isolated Transaction
	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin fact collection tx: %w", err)
	}
	defer tx.Rollback()

	// Enforce Repeatable Read for cross-table consistency during collection
	_, err = tx.ExecContext(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ")
	if err != nil {
		return nil, fmt.Errorf("failed to set isolation level: %w", err)
	}

	// 2. Execute Composite CTE Query
	// Note: We use json_build_object to push the projection graph to Postgres
	query := `
		WITH asset_ctx AS (
			SELECT asset_class, nominal_diameter, rope_construction
			FROM asset_registry WHERE id = $1 AND tenant_id = $2
		),
		latest_insp AS (
			SELECT id, status, verdict, assigned_inspector_id
			FROM inspection_record
			WHERE asset_id = $1 AND tenant_id = $2
			ORDER BY created_at DESC LIMIT 1
		),
		open_findings AS (
			SELECT code, measured_value
			FROM inspection_finding
			WHERE asset_id = $1 AND tenant_id = $2 AND status IN ('OPEN', 'ACKNOWLEDGED')
		),
		current_cert AS (
			SELECT id, status, restrictions
			FROM certificate_record
			WHERE asset_id = $1 AND tenant_id = $2
			ORDER BY created_at DESC LIMIT 1
		)
		SELECT json_build_object(
			'asset', (SELECT row_to_json(asset_ctx) FROM asset_ctx),
			'latest_inspection', (SELECT row_to_json(latest_insp) FROM latest_insp),
			'open_findings', COALESCE((SELECT json_agg(row_to_json(open_findings)) FROM open_findings), '[]'::json),
			'current_certificate', (SELECT row_to_json(current_cert) FROM current_cert)
		) AS snapshot_json;
	`

	var rawJSON []byte
	err = tx.QueryRowContext(ctx, query, assetID, tenantID).Scan(&rawJSON)
	if err != nil {
		return nil, fmt.Errorf("fact snapshot query failed: %w", err)
	}

	// 3. Deserialize directly into strictly-typed structures for CEL compiler
	var facts SnapshotFacts
	if err := json.Unmarshal(rawJSON, &facts); err != nil {
		return nil, fmt.Errorf("failed to decode snapshot payload: %w", err)
	}

	return &facts, nil
}
