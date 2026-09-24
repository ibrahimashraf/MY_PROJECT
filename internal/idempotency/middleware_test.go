package idempotency

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeRow struct {
	status      string
	httpStatus  int
	payload     []byte
	requestHash string
}

type fakeKeyState struct {
	mu     sync.Mutex
	final  bool
	leader bool
	row    *fakeRow
}

type fakeStore struct {
	mu   sync.Mutex
	keys map[string]*fakeKeyState
}

func newFakeStore() *fakeStore {
	return &fakeStore{keys: make(map[string]*fakeKeyState)}
}

func fakeKey(scope Scope) string {
	return scope.TenantID + "|" + scope.OrganizationID + "|" + scope.Endpoint + "|" + scope.KeyHash
}

func (s *fakeStore) Acquire(ctx context.Context, scope Scope) (Claim, *Cached, error) {
	s.mu.Lock()
	st, ok := s.keys[fakeKey(scope)]
	if !ok {
		st = &fakeKeyState{}
		s.keys[fakeKey(scope)] = st
	}
	s.mu.Unlock()

	st.mu.Lock()
	if st.final {
		row := *st.row
		st.mu.Unlock()
		if row.requestHash != scope.RequestHash {
			return nil, nil, ErrConflict
		}
		return nil, &Cached{Status: row.status, HTTPStatus: row.httpStatus, Payload: append([]byte(nil), row.payload...)}, nil
	}
	// Serialized leader acquisition: the key state lock is held until the
	// leader commits (or aborts), mirroring the PostgreSQL row lock.
	st.leader = true
	st.row = &fakeRow{requestHash: scope.RequestHash}
	return &fakeClaim{store: s, st: st, scope: scope}, nil, nil
}

type fakeClaim struct {
	store *fakeStore
	st    *fakeKeyState
	scope Scope
}

func (c *fakeClaim) Commit(ctx context.Context, status string, httpStatus int, payload []byte) error {
	c.st.row.status = status
	c.st.row.httpStatus = httpStatus
	c.st.row.payload = append([]byte(nil), payload...)
	if httpStatus > maximalReplayableStatus {
		// Mirror the PostgreSQL claim: 5xx never gets cached, retries re-run.
		c.st.leader = false
		delete(c.store.keys, fakeKey(c.scope))
		c.st.mu.Unlock()
		return nil
	}
	c.st.final = true
	c.st.leader = false
	c.st.mu.Unlock()
	return nil
}

func (c *fakeClaim) Done() {
	if !c.st.leader {
		return
	}
	c.st.leader = false
	delete(c.store.keys, fakeKey(c.scope))
	c.st.mu.Unlock()
}

func TestMiddlewareRequiresIdempotencyKey(t *testing.T) {
	executed := 0
	handler := NewMiddleware(newFakeStore(), EndpointSync, SyncScopeResolver).Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		executed++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"outcome":"APPLIED"}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(`{"tenant_id":"t1","organization_id":"o1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing Idempotency-Key, got %d", rec.Code)
	}
	if executed != 0 {
		t.Fatalf("downstream executed %d times without a key", executed)
	}
	if !strings.Contains(rec.Body.String(), "idempotency_key_required") {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestMiddlewareRejectsInvalidIdempotencyKey(t *testing.T) {
	executed := 0
	handler := NewMiddleware(newFakeStore(), EndpointSync, SyncScopeResolver).Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		executed++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"outcome":"APPLIED"}`))
	}))
	for _, key := range []string{"short", strings.Repeat("x", 129), "with space", "semi;colon"} {
		req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(`{"tenant_id":"t1","organization_id":"o1"}`))
		req.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("key %q: expected 400, got %d", key, rec.Code)
		}
	}
	if executed != 0 {
		t.Fatalf("downstream executed %d times for invalid keys", executed)
	}
	valid := "12345678-valid-key"
	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(`{"tenant_id":"t1","organization_id":"o1"}`))
	req.Header.Set("Idempotency-Key", valid)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid key %q: expected 200, got %d", valid, rec.Code)
	}
	if executed != 1 {
		t.Fatalf("expected the valid key to execute downstream once, got %d", executed)
	}
}

func TestMiddlewareReplaysCachedResponseWithoutDownstream(t *testing.T) {
	store := newFakeStore()
	store.mu.Lock()
	st := &fakeKeyState{final: true}
	sc := Scope{
		TenantID:       "t1",
		OrganizationID: "o1",
		Endpoint:       EndpointSync,
		KeyHash:        HashKey("o1", EndpointSync, "12345678-key"),
	}
	st.row = &fakeRow{status: StatusApplied, httpStatus: http.StatusOK, payload: []byte(`{"outcome":"APPLIED"}`), requestHash: HashRequest([]byte(`{"tenant_id":"t1","organization_id":"o1","payload":"same"}`))}
	store.keys[fakeKey(sc)] = st
	store.mu.Unlock()

	executed := 0
	handler := NewMiddleware(store, EndpointSync, SyncScopeResolver).Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		executed++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"outcome":"APPLIED","executed":true}`))
	}))

	started := time.Now()
	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(`{"tenant_id":"t1","organization_id":"o1","payload":"same"}`))
	req.Header.Set("Idempotency-Key", "12345678-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(started)

	if executed != 0 {
		t.Fatalf("replay executed downstream %d times", executed)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 replay, got %d", rec.Code)
	}
	if rec.Header().Get("X-Idempotent-Replay") != "true" {
		t.Fatalf("expected replay marker header")
	}
	if rec.Body.String() != `{"outcome":"APPLIED"}` {
		t.Fatalf("unexpected replay body %q", rec.Body.String())
	}
	// The invariant under test is zero downstream execution, asserted above.
	// The wall-clock bound is a secondary SLA check and is skipped under the
	// race detector, whose instrumentation inflates timings past any fixed
	// budget and made this assertion fail intermittently under full-suite load.
	if !raceEnabled && elapsed > 5*time.Millisecond {
		t.Fatalf("replay took %v, SLA is <5ms (zero downstream execution)", elapsed)
	}
}

func TestMiddlewareConflictsOnReusedKeyWithDifferentRequest(t *testing.T) {
	store := newFakeStore()
	store.mu.Lock()
	st := &fakeKeyState{final: true}
	sc := Scope{
		TenantID:       "t1",
		OrganizationID: "o1",
		Endpoint:       EndpointSync,
		KeyHash:        HashKey("o1", EndpointSync, "12345678-key"),
	}
	st.row = &fakeRow{status: StatusApplied, httpStatus: http.StatusOK, payload: []byte(`{"outcome":"APPLIED"}`), requestHash: "original"}
	store.keys[fakeKey(sc)] = st
	store.mu.Unlock()

	executed := 0
	handler := NewMiddleware(store, EndpointSync, SyncScopeResolver).Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		executed++
	}))

	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(`{"tenant_id":"t1","organization_id":"o1","payload":"different"}`))
	req.Header.Set("Idempotency-Key", "12345678-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d", rec.Code)
	}
	if executed != 0 {
		t.Fatalf("conflict executed downstream %d times", executed)
	}
}

func TestMiddlewareLeaderCommitsStructuredOutcome(t *testing.T) {
	store := newFakeStore()
	handler := NewMiddleware(store, EndpointSync, SyncScopeResolver).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"outcome":"HELD","expected_sequence":7}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(`{"tenant_id":"t1","organization_id":"o1"}`))
	req.Header.Set("Idempotency-Key", "12345678-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	store.mu.Lock()
	st, ok := store.keys[fakeKey(Scope{TenantID: "t1", OrganizationID: "o1", Endpoint: EndpointSync, KeyHash: HashKey("o1", EndpointSync, "12345678-key")})]
	store.mu.Unlock()
	if !ok || !st.final {
		t.Fatalf("expected a committed cache row, got final=%v ok=%v", ok && st.final, ok)
	}
	if st.row.status != StatusHeld {
		t.Fatalf("expected structured outcome HELD, got %q", st.row.status)
	}
	if st.row.httpStatus != http.StatusOK || string(st.row.payload) != `{"outcome":"HELD","expected_sequence":7}` {
		t.Fatalf("unexpected committed row: %d %q", st.row.httpStatus, st.row.payload)
	}
}

func TestMiddlewareDoesNotCacheServerErrors(t *testing.T) {
	store := newFakeStore()
	handler := NewMiddleware(store, EndpointSync, SyncScopeResolver).Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"outcome":"REJECTED"}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(`{"tenant_id":"t1","organization_id":"o1"}`))
	req.Header.Set("Idempotency-Key", "12345678-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 to pass through, got %d", rec.Code)
	}

	// A retry must be able to re-execute (the key must not be final).
	store.mu.Lock()
	st, ok := store.keys[fakeKey(Scope{TenantID: "t1", OrganizationID: "o1", Endpoint: EndpointSync, KeyHash: HashKey("o1", EndpointSync, "12345678-key")})]
	store.mu.Unlock()
	if ok && st.final {
		t.Fatalf("server error must not be cached for replay")
	}
}

func TestConcurrentSameKeyExecutesDownstreamExactlyOnce(t *testing.T) {
	store := newFakeStore()
	body := `{"tenant_id":"t1","organization_id":"o1","payload":"x"}`
	var executed atomic.Int64
	middleware := NewMiddleware(store, EndpointSync, SyncScopeResolver).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executed.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"outcome":"APPLIED","n":%d}`, executed.Load())
	}))

	const shooters = 16
	var wg sync.WaitGroup
	results := make(chan int, shooters)
	for i := 0; i < shooters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(body))
			req.Header.Set("Idempotency-Key", "12345678-concurrent")
			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)
			results <- rec.Code
		}()
	}
	wg.Wait()
	close(results)

	var applied, replays int
	for code := range results {
		switch code {
		case http.StatusOK:
			applied++
		default:
			t.Fatalf("unexpected status %d", code)
		}
	}
	if executed.Load() != 1 {
		t.Fatalf("expected exactly 1 downstream execution, got %d", executed.Load())
	}
	replays = applied - 1
	if replays != shooters-1 {
		t.Fatalf("expected %d replays, got %d", shooters-1, replays)
	}
}

func TestConcurrentSameKeyDifferentPayloadConflicts(t *testing.T) {
	store := newFakeStore()
	middleware := NewMiddleware(store, EndpointSync, SyncScopeResolver).Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"outcome":"APPLIED"}`))
	}))

	var wg sync.WaitGroup
	codes := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"tenant_id":"t1","organization_id":"o1","payload":"%d"}`, i)
			req := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(body))
			req.Header.Set("Idempotency-Key", "12345678-conflict")
			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)
			codes <- rec.Code
		}(i)
	}
	wg.Wait()
	close(codes)

	seen := map[int]bool{}
	for code := range codes {
		seen[code] = true
	}
	if !seen[http.StatusOK] {
		t.Fatalf("expected at least one applied outcome, got %v", seen)
	}
	if !seen[http.StatusConflict] {
		t.Fatalf("expected at least one 409 conflict, got %v", seen)
	}
}

func TestDeriveStatus(t *testing.T) {
	body := []byte(`{"outcome":"SECURITY_FAILURE","reason":"digest mismatch"}`)
	if got := deriveStatus(body, http.StatusBadRequest); got != StatusSecurityFailure {
		t.Fatalf("expected SECURITY_FAILURE from body, got %q", got)
	}
	if got := deriveStatus(nil, http.StatusConflict); got != StatusConflict {
		t.Fatalf("expected CONFLICT from 409, got %q", got)
	}
	if got := deriveStatus(nil, http.StatusBadRequest); got != StatusRejected {
		t.Fatalf("expected REJECTED from 400, got %q", got)
	}
	if got := deriveStatus(nil, http.StatusOK); got != StatusApplied {
		t.Fatalf("expected APPLIED from 200, got %q", got)
	}
}

func TestHashesAreDeterministicAndScoped(t *testing.T) {
	a := HashKey("o1", EndpointSync, "key-1")
	if a == "" || len(a) != 64 {
		t.Fatalf("unexpected key hash %q", a)
	}
	if HashKey("o1", EndpointSync, "key-1") != a {
		t.Fatal("key hash must be deterministic")
	}
	if HashKey("o2", EndpointSync, "key-1") == a {
		t.Fatal("key hash must be scoped to organization")
	}
	if HashKey("o1", EndpointEvidence, "key-1") == a {
		t.Fatal("key hash must be scoped to endpoint")
	}
	if HashRequest([]byte("body")) != HashRequest([]byte("body")) {
		t.Fatal("request hash must be deterministic")
	}
	if HashRequest([]byte("body")) == "" {
		t.Fatal("request hash must be non-empty")
	}
}

func TestScopeResolversRejectMissingTenant(t *testing.T) {
	if _, _, err := SyncScopeResolver(nil, []byte(`{"tenant_id":"t1"}`)); err == nil {
		t.Fatal("expected resolution failure when organization is missing")
	}
	if _, _, err := EvidenceScopeResolver(nil, []byte(`{}`)); err == nil {
		t.Fatal("expected resolution failure for empty envelope")
	}
}
