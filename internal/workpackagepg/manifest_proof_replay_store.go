// Package workpackagepg persists approved INTEGIN work packages under tenant RLS.
package workpackagepg

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/workpackage"
)

var (
	// Error identity is domain-owned so manifests do not depend on PostgreSQL.
	ErrManifestProofReplayAlreadyConsumed = workpackage.ErrManifestProofReplayAlreadyConsumed
	ErrManifestProofReplayExpired         = workpackage.ErrManifestProofReplayExpired
)

// ManifestProofReplayStore durably consumes verified manifest proofs through a
// Repository’s tenant-RLS transaction helper. It is safe to compose only after
// the additive replay migration is applied.
type ManifestProofReplayStore struct {
	repository *Repository
	now        func() time.Time
}

// NewManifestProofReplayStore creates the narrowly scoped adapter required by
// a future pilot-only manifest composition. It does not apply schema, mount
// routes, or expose the repository database handle.
func NewManifestProofReplayStore(repository *Repository) *ManifestProofReplayStore {
	return &ManifestProofReplayStore{repository: repository, now: time.Now}
}

// Consume atomically removes an expired matching record and inserts the current
// verified proof. The full tenant/organization/device/purpose/request identity
// is unique, so concurrent use of a request yields exactly one success.
func (s *ManifestProofReplayStore) Consume(
	ctx context.Context,
	tenantID, organizationID, deviceID, purpose, requestID string,
	expiresAt time.Time,
) error {
	if s == nil || s.repository == nil {
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
		return ErrManifestProofReplayExpired
	}
	if s.repository.db == nil {
		return errors.New("proof replay store is not configured")
	}

	tx, err := s.repository.scopedTx(ctx, tenantID, organizationID)
	if err != nil {
		return fmt.Errorf("begin proof replay transaction: %w", err)
	}
	defer tx.Rollback()
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
		return ErrManifestProofReplayAlreadyConsumed
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit proof replay transaction: %w", err)
	}
	return nil
}
