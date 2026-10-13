// Database-backed session persistence for SessionLifecycleManager. A session
// is keyed by its opaque bearer token and bound to the OIDC subject that owns
// it; the token itself is the isolation boundary (subject-bound table, no
// tenant RLS), see migration 0076.
package oidchttp

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// StoredSession is the durable record persisted for one active session.
type StoredSession struct {
	SessionID    string
	Subject      string
	IssuedAt     time.Time
	LastActiveAt time.Time
	RevokedAt    time.Time // zero while the session is active
	ExpiresAt    time.Time
}

// SessionStore persists session lifecycle state so multi-instance restarts
// retain active sessions. Implementations return ErrSessionMissing for tokens
// the store has never seen so the manager can distinguish unknown from
// revoked. Not safe for concurrent Save of the same session to two different
// instances; a single lifecycle owner per session is assumed.
type SessionStore interface {
	Save(ctx context.Context, session StoredSession) error
	Get(ctx context.Context, sessionID string) (StoredSession, error)
	PurgeExpired(ctx context.Context, now time.Time) error
	RevokeSubject(ctx context.Context, subject string, revokedAt time.Time) error
}

var ErrSessionStoreNilDB = errors.New("OIDC session store requires a database")

var (
	sqlUpsertSession = `INSERT INTO oidc_session_store
		(session_id, subject, issued_at, last_active_at, revoked_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (session_id) DO UPDATE SET
			subject = EXCLUDED.subject,
			issued_at = EXCLUDED.issued_at,
			last_active_at = EXCLUDED.last_active_at,
			revoked_at = EXCLUDED.revoked_at,
			expires_at = EXCLUDED.expires_at`

	sqlGetSession = `SELECT session_id, subject, issued_at, last_active_at, revoked_at, expires_at
		FROM oidc_session_store
		WHERE session_id = $1`

	sqlPurgeExpiredSessions = `DELETE FROM oidc_session_store
		WHERE revoked_at IS NULL AND expires_at < $1`

	sqlRevokeSubject = `UPDATE oidc_session_store
		SET revoked_at = $1
		WHERE subject = $2 AND revoked_at IS NULL`
)

// PostgresSessionStore persists session lifecycle state in oidc_session_store.
// Queries run against the shared pool without tenant scoping: the session
// token, not session configuration, is the identity boundary of this table.
type PostgresSessionStore struct {
	db *sql.DB
}

// NewPostgresSessionStore validates its database handle and returns a ready store.
func NewPostgresSessionStore(db *sql.DB) (*PostgresSessionStore, error) {
	if db == nil {
		return nil, ErrSessionStoreNilDB
	}
	return &PostgresSessionStore{db: db}, nil
}

// Save upserts one session record, covering Open, Touch, and Revoke writes.
func (s *PostgresSessionStore) Save(ctx context.Context, session StoredSession) error {
	if session.SessionID == "" {
		return ErrSessionMissing
	}
	_, err := s.db.ExecContext(ctx, sqlUpsertSession,
		session.SessionID, session.Subject,
		session.IssuedAt.UTC(), session.LastActiveAt.UTC(),
		nullableTime(session.RevokedAt), session.ExpiresAt.UTC())
	return err
}

// Get loads one session record, returning ErrSessionMissing when unknown.
func (s *PostgresSessionStore) Get(ctx context.Context, sessionID string) (StoredSession, error) {
	if sessionID == "" {
		return StoredSession{}, ErrSessionMissing
	}
	var (
		session   StoredSession
		revokedAt sql.NullTime
	)
	err := s.db.QueryRowContext(ctx, sqlGetSession, sessionID).Scan(
		&session.SessionID, &session.Subject, &session.IssuedAt, &session.LastActiveAt,
		&revokedAt, &session.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return StoredSession{}, ErrSessionMissing
	}
	if err != nil {
		return StoredSession{}, err
	}
	if revokedAt.Valid {
		session.RevokedAt = revokedAt.Time
	}
	return session, nil
}

// PurgeExpired removes expired, never-revoked sessions. Revoked tombstones are
// retained so their tokens keep failing closed.
func (s *PostgresSessionStore) PurgeExpired(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, sqlPurgeExpiredSessions, now.UTC())
	return err
}

// RevokeSubject sets the revoked_at tombstone on all active sessions for a subject.
func (s *PostgresSessionStore) RevokeSubject(ctx context.Context, subject string, revokedAt time.Time) error {
	if subject == "" {
		return errors.New("subject is required for revocation")
	}
	if revokedAt.IsZero() {
		revokedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, sqlRevokeSubject, revokedAt.UTC(), subject)
	return err
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC()
}
