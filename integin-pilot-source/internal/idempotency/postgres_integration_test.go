package idempotency

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const idempotencyIntegrationDSNEnv = "INTEGIN_TEST_DATABASE_URL"

// These tests require migration 0071_sync_idempotency_cache.sql to be applied
// to the target database and do not create or alter schema (matching the
// existing syncstate integration convention).
func openIdempotencyIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv(idempotencyIntegrationDSNEnv)
	if dsn == "" {
		t.Skipf("set %s to run idempotency PostgreSQL integration tests", idempotencyIntegrationDSNEnv)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	return db
}

func cleanupIdempotencyFixtures(t *testing.T, db *sql.DB, tenantID string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := db.ExecContext(ctx, `DELETE FROM sync_idempotency_cache WHERE tenant_id = $1`, tenantID); err != nil {
			t.Errorf("cleanup idempotency fixtures: %v", err)
		}
	})
}

func TestPostgresStoreConcurrentClaimsAreLinearizable(t *testing.T) {
	db := openIdempotencyIntegrationDB(t)
	db.SetMaxOpenConns(24)
	store, err := NewPostgresStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	tenantID := fmt.Sprintf("it-idem-%d", time.Now().UTC().UnixNano())
	cleanupIdempotencyFixtures(t, db, tenantID)

	scope := Scope{
		TenantID:       tenantID,
		OrganizationID: "org-1",
		Endpoint:       EndpointSync,
		KeyHash:        HashKey("org-1", EndpointSync, "it-concurrent-key"),
		RequestHash:    HashRequest([]byte(`{"payload":"same"}`)),
	}

	const shooters = 16
	var leaders, replays, conflicts atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < shooters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claim, cached, err := store.Acquire(context.Background(), scope)
			switch {
			case errors.Is(err, ErrConflict):
				conflicts.Add(1)
			case err != nil:
				t.Errorf("acquire failed: %v", err)
			case cached != nil:
				if cached.Status != StatusApplied || cached.HTTPStatus != http.StatusOK {
					t.Errorf("unexpected replay outcome %+v", cached)
				}
				replays.Add(1)
			case claim != nil:
				if err := claim.Commit(context.Background(), StatusApplied, http.StatusOK, []byte(`{"outcome":"APPLIED"}`)); err != nil {
					t.Errorf("commit failed: %v", err)
					claim.Done()
					return
				}
				leaders.Add(1)
			default:
				t.Error("acquire returned neither claim, cached, nor error")
			}
		}()
	}
	wg.Wait()

	if leaders.Load() != 1 {
		t.Fatalf("expected exactly 1 leader, got %d (replays=%d conflicts=%d)", leaders.Load(), replays.Load(), conflicts.Load())
	}
	if replays.Load() != shooters-1 {
		t.Fatalf("expected %d replays, got %d", shooters-1, replays.Load())
	}
	if conflicts.Load() != 0 {
		t.Fatalf("expected 0 conflicts for identical payloads, got %d", conflicts.Load())
	}

	// An identical retry after quiescence must still replay, never re-execute.
	claim, cached, err := store.Acquire(context.Background(), scope)
	if err != nil {
		t.Fatalf("final acquire: %v", err)
	}
	if claim != nil {
		t.Fatal("expected a replay after the window was committed")
	}
	if cached == nil || cached.Status != StatusApplied {
		t.Fatalf("unexpected final replay %+v", cached)
	}
	// response_payload is JSONB; compare semantically so PostgreSQL's harmless
	// whitespace canonicalization ("outcome": "APPLIED") does not drive the test.
	var got, want any
	if err := json.Unmarshal(cached.Payload, &got); err != nil {
		t.Fatalf("replay payload is not JSON: %q", cached.Payload)
	}
	if err := json.Unmarshal([]byte(`{"outcome":"APPLIED"}`), &want); err != nil {
		t.Fatalf("marshal expectation: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected final replay payload %q", cached.Payload)
	}
}

func TestPostgresStoreConflictsOnReusedKeyWithDifferentPayload(t *testing.T) {
	db := openIdempotencyIntegrationDB(t)
	store, err := NewPostgresStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	tenantID := fmt.Sprintf("it-idem-conflict-%d", time.Now().UTC().UnixNano())
	cleanupIdempotencyFixtures(t, db, tenantID)

	scope := func(requestHash string) Scope {
		return Scope{
			TenantID:       tenantID,
			OrganizationID: "org-1",
			Endpoint:       EndpointSync,
			KeyHash:        HashKey("org-1", EndpointSync, "it-conflict-key"),
			RequestHash:    requestHash,
		}
	}
	claim, cached, err := store.Acquire(context.Background(), scope(HashRequest([]byte("a"))))
	if err != nil || cached != nil || claim == nil {
		t.Fatalf("expected leader claim, got claim=%v cached=%v err=%v", claim != nil, cached, err)
	}
	if err := claim.Commit(context.Background(), StatusApplied, http.StatusOK, []byte(`{"outcome":"APPLIED"}`)); err != nil {
		t.Fatalf("commit: %v", err)
	}
	second, cached, err := store.Acquire(context.Background(), scope(HashRequest([]byte("b"))))
	if err == nil || !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict for different payload, got claim=%v cached=%v err=%v", second != nil, cached, err)
	}
}

func TestPostgresStoreDoesNotCacheServerErrors(t *testing.T) {
	db := openIdempotencyIntegrationDB(t)
	store, err := NewPostgresStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	tenantID := fmt.Sprintf("it-idem-5xx-%d", time.Now().UTC().UnixNano())
	cleanupIdempotencyFixtures(t, db, tenantID)
	scope := Scope{TenantID: tenantID, OrganizationID: "org-1", Endpoint: EndpointSync, KeyHash: HashKey("org-1", EndpointSync, "it-5xx-key"), RequestHash: "h"}

	claim, _, err := store.Acquire(context.Background(), scope)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := claim.Commit(context.Background(), StatusRejected, http.StatusServiceUnavailable, []byte(`{}`)); err != nil {
		t.Fatalf("commit 5xx: %v", err)
	}
	// A retry must be able to take leadership again; the failure was not cached.
	claim, cached, err := store.Acquire(context.Background(), scope)
	if err != nil || cached != nil || claim == nil {
		t.Fatalf("expected a fresh leader after 5xx, got claim=%v cached=%v err=%v", claim != nil, cached, err)
	}
	claim.Done()
}

func TestSyncMiddlewareAgainstPostgresExecutesOnce(t *testing.T) {
	db := openIdempotencyIntegrationDB(t)
	db.SetMaxOpenConns(24)
	store, err := NewPostgresStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	tenantID := fmt.Sprintf("it-idem-http-%d", time.Now().UTC().UnixNano())
	cleanupIdempotencyFixtures(t, db, tenantID)

	var executed atomic.Int64
	middleware := NewMiddleware(store, EndpointSync, SyncScopeResolver).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executed.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"outcome":"APPLIED","executed":%d}`, executed.Load())
	}))

	body := fmt.Sprintf(`{"tenant_id":%q,"organization_id":"org-1","payload":"same"}`, tenantID)
	const shooters = 24
	var wg sync.WaitGroup
	codes := make(chan int, shooters)
	for i := 0; i < shooters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(body))
			req.Header.Set("Idempotency-Key", "http-concurrent-key-001")
			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)
			codes <- rec.Code
		}()
	}
	wg.Wait()
	close(codes)

	for code := range codes {
		if code != http.StatusOK {
			t.Fatalf("unexpected status %d", code)
		}
	}
	if executed.Load() != 1 {
		t.Fatalf("expected exactly 1 downstream execution, got %d", executed.Load())
	}
}
