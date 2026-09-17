package idempotency

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	leimenv "integin/internal/shared/env"
	"time"
)

// CacheTTL bounds how long a committed idempotent response may be replayed.
const DefaultCacheTTL = 24 * time.Hour

// CacheTTL is kept for backward compatibility; prefer CacheTTLHours.
const CacheTTL = DefaultCacheTTL

// CacheTTLHours returns the replay window in whole hours.
func CacheTTLHours() int {
	return leimenv.Int("INTEGIN_IDEMPOTENCY_CACHE_TTL_HOURS", 24, 1, 168)
}

// maximalReplayableStatus is the highest HTTP status that is persisted for
// replay. Server-side failures (5xx) are never cached so a retry can succeed.
const maximalReplayableStatus = 499

const (
	tableColumns = `status, request_hash, http_status, response_payload`
)

// PostgresStore synchronizes idempotency claims through PostgreSQL advisory
// locks and the durable sync_idempotency_cache row lock.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore requires a live database handle.
func NewPostgresStore(db *sql.DB) (*PostgresStore, error) {
	if db == nil {
		return nil, errors.New("database handle is required")
	}
	return &PostgresStore{db: db}, nil
}

// advisoryKey is the shared-memory serialization domain for a single
// idempotency key. It intentionally reuses the request scope fields so two
// requests that differ only in body still contend on the same advisory lock.
// The scope is hashed in Go first: PostgreSQL text values cannot contain the
// NUL byte, and a raw concatenation would be ambiguous across field edges.
func advisoryKey(scope Scope) string {
	sum := sha256.Sum256([]byte(scope.TenantID + "\x00" + scope.OrganizationID + "\x00" + scope.Endpoint + "\x00" + scope.KeyHash))
	return hex.EncodeToString(sum[:])
}

func (s *PostgresStore) Acquire(ctx context.Context, scope Scope) (Claim, *Cached, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Transaction-scoped tenant GUCs gate every RLS policy in this session.
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, scope.TenantID, scope.OrganizationID); err != nil {
		return nil, nil, err
	}
	// Serialize identical keys in shared memory first so the row lock below is
	// contended by one waiter at a time per key.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, advisoryKey(scope)); err != nil {
		return nil, nil, err
	}

	status, requestHash, httpStatus, payload, found, err := readCached(ctx, tx, scope)
	if err != nil {
		return nil, nil, err
	}
	if found {
		if requestHash != scope.RequestHash {
			return nil, nil, ErrConflict
		}
		return nil, &Cached{Status: status, HTTPStatus: httpStatus, Payload: payload}, nil
	}

	// Reclaim any expired duplicate row before creating a fresh window so a
	// stale response is never served beyond its TTL.
	if _, err := tx.ExecContext(ctx, `DELETE FROM sync_idempotency_cache WHERE tenant_id = $1 AND organization_id = $2 AND key_hash = $3 AND expires_at <= now()`, scope.TenantID, scope.OrganizationID, scope.KeyHash); err != nil {
		return nil, nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sync_idempotency_cache (key_hash, tenant_id, organization_id, endpoint, request_hash, status, expires_at) VALUES ($1,$2,$3,$4,$5,$6, now() + make_interval(hours => $7)) ON CONFLICT (tenant_id, organization_id, key_hash) DO NOTHING`, scope.KeyHash, scope.TenantID, scope.OrganizationID, scope.Endpoint, scope.RequestHash, string(StatusInProgress), CacheTTLHours()); err != nil {
		return nil, nil, err
	}
	// Lock the just-inserted row for the duration of downstream execution.
	// Concurrent same-key requests block either on the unique index insert or
	// on this FOR UPDATE until the leader commits and the completed outcome is
	// visible for replay.
	if _, _, _, _, found, err = readCached(ctx, tx, scope); err != nil {
		return nil, nil, err
	} else if !found {
		return nil, nil, errors.New("idempotency row is not visible after claim")
	}
	committed = true
	return &postgresClaim{store: s, tx: tx, scope: scope}, nil, nil
}

func readCached(ctx context.Context, tx *sql.Tx, scope Scope) (status, requestHash string, httpStatus int, payload []byte, found bool, err error) {
	err = tx.QueryRowContext(ctx, `SELECT `+tableColumns+` FROM sync_idempotency_cache WHERE tenant_id = $1 AND organization_id = $2 AND key_hash = $3 AND expires_at > now()`, scope.TenantID, scope.OrganizationID, scope.KeyHash).Scan(&status, &requestHash, &httpStatus, &payload)
	switch {
	case err == nil:
		return status, requestHash, httpStatus, payload, true, nil
	case errors.Is(err, sql.ErrNoRows):
		return "", "", 0, nil, false, nil
	default:
		return "", "", 0, nil, false, err
	}
}

type postgresClaim struct {
	store *PostgresStore
	tx    *sql.Tx
	scope Scope
	used  bool
}

func (c *postgresClaim) Commit(ctx context.Context, status string, httpStatus int, payload []byte) error {
	if c.used {
		return errors.New("idempotency claim already finalized")
	}
	c.used = true
	if httpStatus > maximalReplayableStatus {
		_ = c.tx.Rollback()
		return nil
	}
	if _, err := c.tx.ExecContext(ctx, `UPDATE sync_idempotency_cache SET status = $1, http_status = $2, response_payload = $3, request_hash = $4, expires_at = now() + make_interval(hours => $8) WHERE tenant_id = $5 AND organization_id = $6 AND key_hash = $7`, status, httpStatus, string(payload), c.scope.RequestHash, c.scope.TenantID, c.scope.OrganizationID, c.scope.KeyHash, CacheTTLHours()); err != nil {
		_ = c.tx.Rollback()
		return err
	}
	return c.tx.Commit()
}

func (c *postgresClaim) Done() {
	if c.used {
		return
	}
	c.used = true
	_ = c.tx.Rollback()
}

// HashKey derives the opaque cache row key. The tenant is the RLS partition
// and is not mixed into the digest; organization + endpoint + client key form
// the scoped identity, matching account-scoped idempotency semantics.
func HashKey(organizationID, endpoint, idempotencyKey string) string {
	sum := sha256.Sum256([]byte(organizationID + "\x00" + endpoint + "\x00" + idempotencyKey))
	return hex.EncodeToString(sum[:])
}

// HashRequest derives a fingerprint of the exact request body. Two requests
// with different bodies but the same idempotency key must conflict.
func HashRequest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
