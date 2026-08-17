// Package packagemanifest contains replay controls for the future package-manifest route.
package packagemanifest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PostgresProofReplayStore durably consumes verified manifest proofs under the
// tenant and organization RLS scope supplied by the already-verified device.
// It is safe to compose only after its additive migration is applied.
type PostgresProofReplayStore struct {
	db  *sql.DB
	now func() time.Time
}

// NewPostgresProofReplayStore creates a durable replay store. Passing nil is
// permitted for construction tests but Consume will refuse to operate.
func NewPostgresProofReplayStore(db *sql.DB) *PostgresProofReplayStore {
	return &PostgresProofReplayStore{db: db, now: time.Now}
}

// Consume atomically removes an expired matching record and inserts the current
// verified proof. The full tenant/organization/device/purpose/request identity
// is unique, so concurrent use of a request yields exactly one success.
func (s *PostgresProofReplayStore) Consume(
	ctx context.Context,
	tenantID, organizationID, deviceID, purpose, requestID string,
	expiresAt time.Time,
) error {
	if s == nil {
		return errors.New("proof replay store is not configured")
	}
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(organizationID) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(purpose) == "" || strings.TrimSpace(requestID) == "" {
		return errors.New("proof replay key is incomplete")
	}
	clock := s.now
	if clock == nil {
		clock = time.Now
	}
	now := clock().UTC()
	if !expiresAt.After(now) {
		return ErrReplayExpired
	}
	if s.db == nil {
		return errors.New("proof replay store is not configured")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin proof replay transaction: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		SELECT set_config('integin.tenant_id', $1, true),
		       set_config('integin.organization_id', $2, true)
	`, tenantID, organizationID); err != nil {
		return fmt.Errorf("scope proof replay transaction: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM manifest_proof_replay
		WHERE tenant_id = $1
		  AND organization_id = $2
		  AND device_id = $3
		  AND purpose = $4
		  AND request_id = $5
		  AND expires_at <= $6
	`, tenantID, organizationID, deviceID, purpose, requestID, now); err != nil {
		return fmt.Errorf("purge expired proof replay: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO manifest_proof_replay (
			tenant_id, organization_id, device_id, purpose, request_id, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, organization_id, device_id, purpose, request_id) DO NOTHING
	`, tenantID, organizationID, deviceID, purpose, requestID, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("consume proof replay: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read proof replay insert result: %w", err)
	}
	if inserted != 1 {
		return ErrReplayAlreadyConsumed
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit proof replay transaction: %w", err)
	}
	return nil
}
