package oidchttp

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

var errFakeSession = errors.New("fake session driver failure")

// fakeSessionRow mirrors oidc_session_store row semantics, with revokedAt nil
// meaning "not revoked".
type fakeSessionRow struct {
	sessionID    string
	subject      string
	issuedAt     time.Time
	lastActiveAt time.Time
	revokedAt    any
	expiresAt    time.Time
}

type fakeSessionRows struct {
	row *fakeSessionRow
}

func (r *fakeSessionRows) Columns() []string {
	return []string{"session_id", "subject", "issued_at", "last_active_at", "revoked_at", "expires_at"}
}
func (r *fakeSessionRows) Close() error { return nil }
func (r *fakeSessionRows) Next(dest []driver.Value) error {
	if r.row == nil {
		return io.EOF
	}
	dest[0] = r.row.sessionID
	dest[1] = r.row.subject
	dest[2] = r.row.issuedAt
	dest[3] = r.row.lastActiveAt
	dest[4] = r.row.revokedAt
	dest[5] = r.row.expiresAt
	r.row = nil
	return nil
}

type fakeSessionStmt struct{}

func (s *fakeSessionStmt) Close() error  { return nil }
func (s *fakeSessionStmt) NumInput() int { return 0 }
func (s *fakeSessionStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, errors.New("fake driver only executes via context methods")
}
func (s *fakeSessionStmt) Query(args []driver.Value) (driver.Rows, error) {
	return nil, errors.New("fake driver only queries via context methods")
}

type fakeSessionTx struct{ c *fakeSessionConn }

func (t *fakeSessionTx) Commit() error   { return nil }
func (t *fakeSessionTx) Rollback() error { return nil }

// fakeSessionConn is a stateful in-memory emulation of oidc_session_store for
// the three statements PostgresSessionStore issues.
type fakeSessionConn struct {
	rows        map[string]*fakeSessionRow
	execQueries []string
	execArgs    [][]driver.Value
}

func (c *fakeSessionConn) Prepare(query string) (driver.Stmt, error) { return &fakeSessionStmt{}, nil }
func (c *fakeSessionConn) Close() error                              { return nil }
func (c *fakeSessionConn) Begin() (driver.Tx, error)                 { return &fakeSessionTx{c: c}, nil }

func (c *fakeSessionConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.execQueries = append(c.execQueries, query)
	values := make([]driver.Value, 0, len(args))
	for _, a := range args {
		values = append(values, a.Value)
	}
	c.execArgs = append(c.execArgs, values)
	switch {
	case strings.Contains(query, "ON CONFLICT"):
		c.upsert(values)
	case strings.Contains(query, "DELETE"):
		c.purge(values)
	case strings.Contains(query, "UPDATE"):
		c.revokeSubject(values)
	}
	return fakeSessionResult{}, nil
}

func (c *fakeSessionConn) revokeSubject(values []driver.Value) {
	revokedAt := values[0].(time.Time)
	subject := values[1].(string)
	for _, row := range c.rows {
		if row.subject == subject && row.revokedAt == nil {
			row.revokedAt = revokedAt
		}
	}
}

func (c *fakeSessionConn) upsert(values []driver.Value) {
	c.rows[values[0].(string)] = &fakeSessionRow{
		sessionID:    values[0].(string),
		subject:      values[1].(string),
		issuedAt:     values[2].(time.Time),
		lastActiveAt: values[3].(time.Time),
		revokedAt:    values[4],
		expiresAt:    values[5].(time.Time),
	}
}

func (c *fakeSessionConn) purge(values []driver.Value) {
	now := values[0].(time.Time)
	for sid, row := range c.rows {
		if row.revokedAt == nil && row.expiresAt.Before(now) {
			delete(c.rows, sid)
		}
	}
}

func (c *fakeSessionConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if len(args) != 1 {
		return nil, errors.New("fake session get requires one arg")
	}
	return &fakeSessionRows{row: c.rows[args[0].Value.(string)]}, nil
}

type fakeSessionResult struct{}

func (fakeSessionResult) LastInsertId() (int64, error) { return 0, errors.New("not supported") }
func (fakeSessionResult) RowsAffected() (int64, error) { return 1, nil }

type fakeSessionConnector struct{ c *fakeSessionConn }

func (f *fakeSessionConnector) Connect(context.Context) (driver.Conn, error) { return f.c, nil }
func (f *fakeSessionConnector) Driver() driver.Driver                        { return stubSessionDriver{} }

type stubSessionDriver struct{}

func (stubSessionDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("fake session driver opens via connector only")
}

func newFakeSessionStore(c *fakeSessionConn) *PostgresSessionStore {
	db := sql.OpenDB(&fakeSessionConnector{c: c})
	store, err := NewPostgresSessionStore(db)
	if err != nil {
		panic(err)
	}
	return store
}

func TestNewPostgresSessionStoreNilGuard(t *testing.T) {
	if _, err := NewPostgresSessionStore(nil); err != ErrSessionStoreNilDB {
		t.Fatalf("expected ErrSessionStoreNilDB, got %v", err)
	}
}

func TestNewSessionLifecycleManagerWithStoreRejectsNil(t *testing.T) {
	if _, err := NewSessionLifecycleManagerWithStore(time.Hour, nil); err == nil {
		t.Fatal("nil store must be rejected")
	}
}

func TestPostgresSessionStoreSaveGetRoundTrip(t *testing.T) {
	c := &fakeSessionConn{rows: make(map[string]*fakeSessionRow)}
	store := newFakeSessionStore(c)
	ctx := context.Background()
	now := time.Date(2026, 9, 14, 8, 30, 0, 0, time.UTC)
	if err := store.Save(ctx, StoredSession{SessionID: "tok-1", Subject: "human-001", IssuedAt: now, LastActiveAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, "tok-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != "tok-1" || got.Subject != "human-001" || !got.IssuedAt.Equal(now) || !got.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if !got.RevokedAt.IsZero() {
		t.Fatalf("fresh session must not carry a tombstone: %+v", got)
	}
	if len(c.execQueries) != 1 || c.execQueries[0] != sqlUpsertSession {
		t.Fatalf("expected single upsert, got %v", c.execQueries)
	}
	if c.execArgs[0][0] != "tok-1" || c.execArgs[0][1] != "human-001" {
		t.Fatalf("upsert identity args wrong: %#v", c.execArgs[0][:2])
	}
}

func TestPostgresSessionStoreGetUnknownReturnsMissing(t *testing.T) {
	c := &fakeSessionConn{rows: make(map[string]*fakeSessionRow)}
	store := newFakeSessionStore(c)
	if _, err := store.Get(context.Background(), "unknown"); err != ErrSessionMissing {
		t.Fatalf("expected ErrSessionMissing, got %v", err)
	}
}

func TestPostgresSessionStoreRevokePersistsTombstone(t *testing.T) {
	c := &fakeSessionConn{rows: make(map[string]*fakeSessionRow)}
	store := newFakeSessionStore(c)
	ctx := context.Background()
	now := time.Now()
	if err := store.Save(ctx, StoredSession{SessionID: "tok-1", Subject: "human-001", IssuedAt: now, LastActiveAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	revokedAt := now.Add(5 * time.Minute)
	if err := store.Save(ctx, StoredSession{SessionID: "tok-1", Subject: "human-001", IssuedAt: now, LastActiveAt: now, RevokedAt: revokedAt, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, "tok-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.RevokedAt.Equal(revokedAt) {
		t.Fatalf("tombstone not persisted: %+v", got)
	}
}

func TestPostgresSessionStoreRevokeSubject(t *testing.T) {
	c := &fakeSessionConn{rows: make(map[string]*fakeSessionRow)}
	store := newFakeSessionStore(c)
	ctx := context.Background()
	now := time.Now()
	_ = store.Save(ctx, StoredSession{SessionID: "tok-1", Subject: "user-alpha", IssuedAt: now, LastActiveAt: now, ExpiresAt: now.Add(time.Hour)})
	_ = store.Save(ctx, StoredSession{SessionID: "tok-2", Subject: "user-alpha", IssuedAt: now, LastActiveAt: now, ExpiresAt: now.Add(time.Hour)})
	_ = store.Save(ctx, StoredSession{SessionID: "tok-3", Subject: "user-beta", IssuedAt: now, LastActiveAt: now, ExpiresAt: now.Add(time.Hour)})

	revokedAt := now.Add(time.Minute)
	if err := store.RevokeSubject(ctx, "user-alpha", revokedAt); err != nil {
		t.Fatalf("revoke subject: %v", err)
	}

	s1, _ := store.Get(ctx, "tok-1")
	if !s1.RevokedAt.Equal(revokedAt) {
		t.Fatalf("expected s1 revoked, got: %+v", s1)
	}
	s2, _ := store.Get(ctx, "tok-2")
	if !s2.RevokedAt.Equal(revokedAt) {
		t.Fatalf("expected s2 revoked, got: %+v", s2)
	}
	s3, _ := store.Get(ctx, "tok-3")
	if !s3.RevokedAt.IsZero() {
		t.Fatalf("user-beta session should not be revoked: %+v", s3)
	}
}

func TestPostgresSessionStorePurgeExpiredKeepsActiveAndTombstones(t *testing.T) {
	c := &fakeSessionConn{rows: make(map[string]*fakeSessionRow)}
	store := newFakeSessionStore(c)
	ctx := context.Background()
	now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
	sessions := []StoredSession{
		{SessionID: "expired-active", Subject: "s1", IssuedAt: now.Add(-2 * time.Hour), LastActiveAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour)},
		{SessionID: "future-active", Subject: "s2", IssuedAt: now, LastActiveAt: now, ExpiresAt: now.Add(time.Hour)},
		{SessionID: "expired-revoked", Subject: "s3", IssuedAt: now.Add(-2 * time.Hour), LastActiveAt: now.Add(-2 * time.Hour), RevokedAt: now.Add(-90 * time.Minute), ExpiresAt: now.Add(-time.Hour)},
	}
	for _, session := range sessions {
		if err := store.Save(ctx, session); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.PurgeExpired(ctx, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "expired-active"); err != ErrSessionMissing {
		t.Fatalf("expired active session must be purged, got err=%v", err)
	}
	if _, err := store.Get(ctx, "future-active"); err != nil {
		t.Fatalf("active session must survive purge: %v", err)
	}
	if _, err := store.Get(ctx, "expired-revoked"); err != nil {
		t.Fatalf("revoked tombstone must survive purge so its token keeps failing closed: %v", err)
	}
	if len(c.execQueries) != 4 || c.execQueries[3] != sqlPurgeExpiredSessions {
		t.Fatalf("expected upserts then purge, got %v", c.execQueries)
	}
}

func TestPostgresSessionStoreRestartRetentionAcrossManagers(t *testing.T) {
	clock := &fakeClock{current: time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)}
	db := sql.OpenDB(&fakeSessionConnector{c: &fakeSessionConn{rows: make(map[string]*fakeSessionRow)}})
	store, err := NewPostgresSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	managerA, err := NewSessionLifecycleManagerWithStore(time.Hour, store)
	if err != nil {
		t.Fatal(err)
	}
	managerA.now = clock.now
	if err := managerA.Open("tok-1", "human-001", clock.now()); err != nil {
		t.Fatal(err)
	}
	if err := managerA.Touch("tok-1"); err != nil {
		t.Fatal(err)
	}

	managerB, err := NewSessionLifecycleManagerWithStore(time.Hour, store)
	if err != nil {
		t.Fatal(err)
	}
	managerB.now = clock.now
	if !managerB.IsActive("tok-1") {
		t.Fatal("restarted instance must retain the active session opened by manager A")
	}
	if !managerB.Known("tok-1") {
		t.Fatal("restarted instance must know the session opened by manager A")
	}
	if err := managerB.Touch("tok-1"); err != nil {
		t.Fatalf("touching the retained session: %v", err)
	}
	if err := managerA.RevokeSession("tok-1"); err != nil {
		t.Fatal(err)
	}
	if !managerB.IsRevoked("tok-1") || managerB.IsActive("tok-1") {
		t.Fatal("revocation from one instance must be visible to the other")
	}
	if err := managerB.Open("tok-1", "human-001", clock.now()); err != ErrSessionRevoked {
		t.Fatalf("reopening a revoked token across restart must fail closed, got %v", err)
	}
}

func TestManagerStoreModeExpiryHonorsClock(t *testing.T) {
	clock := &fakeClock{current: time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)}
	db := sql.OpenDB(&fakeSessionConnector{c: &fakeSessionConn{rows: make(map[string]*fakeSessionRow)}})
	store, err := NewPostgresSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewSessionLifecycleManagerWithStore(30*time.Minute, store)
	if err != nil {
		t.Fatal(err)
	}
	manager.now = clock.now
	if err := manager.Open("tok-1", "human-001", clock.now()); err != nil {
		t.Fatal(err)
	}
	clock.advance(2 * time.Hour)
	if manager.IsActive("tok-1") {
		t.Fatal("store-mode session must expire past the maximum TTL")
	}
	if !manager.Known("tok-1") {
		t.Fatal("expired store-mode session must remain known until purge")
	}
	if err := manager.Touch("tok-1"); err != ErrSessionExpired {
		t.Fatalf("touching an expired store-mode session: got %v", err)
	}
	manager.PurgeExpired()
	if manager.Known("tok-1") {
		t.Fatal("purged store-mode session must no longer be known")
	}
}

func TestSessionStoreSQLParameterizedPlaceholders(t *testing.T) {
	for i := 1; i <= 6; i++ {
		if !strings.Contains(sqlUpsertSession, fmt.Sprintf("$%d", i)) {
			t.Fatalf("upsert SQL missing placeholder $%d", i)
		}
	}
	for _, fragment := range []string{"ON CONFLICT (session_id)", "WHERE session_id = $1", "revoked_at IS NULL AND expires_at < $1", "oidc_session_store"} {
		if !strings.Contains(sqlUpsertSession, fragment) && !strings.Contains(sqlGetSession, fragment) && !strings.Contains(sqlPurgeExpiredSessions, fragment) {
			t.Fatalf("SQL missing %q", fragment)
		}
	}
}
