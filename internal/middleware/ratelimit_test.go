package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"integin/internal/identity"
	"integin/internal/oidchttp"
)

func TestRateLimiter_TenantRateLimiting(t *testing.T) {
	limiter := NewRateLimiter(1, 2) // 1 req/s, burst of 2
	defer limiter.Close()
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Tenant 1 makes 2 requests (within burst)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		req.Header.Set("X-Tenant-ID", "tenant-1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d expected 200, got %d", i+1, w.Code)
		}
	}

	// Tenant 1 makes 3rd request (exceeds burst)
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for tenant-1, got %d", w.Code)
	}

	// Tenant 2 makes request (isolated, should succeed)
	req2 := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req2.Header.Set("X-Tenant-ID", "tenant-2")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for tenant-2, got %d", w2.Code)
	}
}

func TestRateLimiter_IPFallbackRateLimiting(t *testing.T) {
	limiter := NewRateLimiter(1, 2) // 1 req/s, burst of 2
	defer limiter.Close()
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Client IP 192.168.1.100 without X-Tenant-ID
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/public/resource", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d expected 200, got %d", i+1, w.Code)
		}
	}

	// 3rd request from same IP exceeds burst and is rate limited
	req := httptest.NewRequest(http.MethodGet, "/public/resource", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for IP 192.168.1.100, got %d", w.Code)
	}

	// Different IP should not be blocked
	reqOther := httptest.NewRequest(http.MethodGet, "/public/resource", nil)
	reqOther.RemoteAddr = "192.168.1.200:12345"
	wOther := httptest.NewRecorder()
	handler.ServeHTTP(wOther, reqOther)
	if wOther.Code != http.StatusOK {
		t.Fatalf("expected 200 for different IP, got %d", wOther.Code)
	}
}

func TestRateLimiter_SkipPaths(t *testing.T) {
	limiter := NewRateLimiter(1, 1)
	defer limiter.Close()
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.RemoteAddr = "10.0.0.1:1234"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("%s should never be rate limited, got %d on request %d", path, w.Code, i+1)
			}
		}
	}
}

func TestRateLimiter_StatsCounters(t *testing.T) {
	limiter := NewRateLimiter(1, 2)
	defer limiter.Close()
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 2 allowed requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		req.Header.Set("X-Tenant-ID", "stats-tenant")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// 1 denied request (exceeds burst)
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("X-Tenant-ID", "stats-tenant")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}

	allowed, denied := limiter.Stats()
	if allowed != 2 {
		t.Fatalf("expected 2 allowed, got %d", allowed)
	}
	if denied != 1 {
		t.Fatalf("expected 1 denied, got %d", denied)
	}
}

func TestRateLimiter_EvictIdle(t *testing.T) {
	limiter := NewRateLimiter(10, 10)
	defer limiter.Close()

	_ = limiter.getLimiter("tenant:idle-1")
	_ = limiter.getLimiter("ip:1.2.3.4")

	// Artificially age items
	for _, shard := range limiter.shards {
		shard.mu.Lock()
		for _, item := range shard.limiters {
			item.lastSeen = time.Now().Add(-2 * time.Hour)
		}
		shard.mu.Unlock()
	}

	limiter.EvictIdle(1 * time.Hour)

	// Verify all shards are now empty
	total := 0
	for _, shard := range limiter.shards {
		shard.mu.RLock()
		total += len(shard.limiters)
		shard.mu.RUnlock()
	}
	if total != 0 {
		t.Fatalf("expected 0 limiters after EvictIdle, got %d", total)
	}
}

func TestRateLimiter_HighConcurrency(t *testing.T) {
	limiter := NewRateLimiter(1000, 100)
	defer limiter.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				l := limiter.getLimiter("tenant:worker")
				_ = l.Allow()
			}
		}(i)
	}
	wg.Wait()
}

func TestRateLimiter_AuthenticatedOrganizationContext(t *testing.T) {
	limiter := NewRateLimiter(1, 2) // 1 req/s, burst of 2
	defer limiter.Close()
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ctxOrg1ActorA := oidchttp.WithOrganizationContext(
		t.Context(),
		identity.OrganizationContext{TenantID: "tenant-alpha", ActorID: "actor-1"},
	)
	ctxOrg1ActorB := oidchttp.WithOrganizationContext(
		t.Context(),
		identity.OrganizationContext{TenantID: "tenant-alpha", ActorID: "actor-2"},
	)

	// Actor 1 exhausts burst of 2
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/secure", nil).WithContext(ctxOrg1ActorA)
		// Client sends spoofed X-Tenant-ID; must be IGNORED in favor of authenticated context
		req.Header.Set("X-Tenant-ID", "spoofed-tenant")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("actor-1 request %d expected 200, got %d", i+1, w.Code)
		}
	}

	// Actor 1 request 3 -> rate limited
	req3 := httptest.NewRequest(http.MethodGet, "/secure", nil).WithContext(ctxOrg1ActorA)
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Fatalf("actor-1 request 3 expected 429, got %d", w3.Code)
	}

	// Actor 2 in same tenant is isolated on separate actor partition
	reqActorB := httptest.NewRequest(http.MethodGet, "/secure", nil).WithContext(ctxOrg1ActorB)
	wActorB := httptest.NewRecorder()
	handler.ServeHTTP(wActorB, reqActorB)
	if wActorB.Code != http.StatusOK {
		t.Fatalf("actor-2 in tenant-alpha expected 200, got %d", wActorB.Code)
	}
}

func TestRateLimiter_BearerTokenAndUnauthTenant(t *testing.T) {
	limiter := NewRateLimiter(1, 2) // 1 req/s, burst of 2
	defer limiter.Close()
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Unauthenticated with Bearer token without OrgContext
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.Header.Set("Authorization", "Bearer token-12345")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("bearer token request %d expected 200, got %d", i+1, w.Code)
		}
	}

	// 3rd request with same token fails
	reqTokenDenied := httptest.NewRequest(http.MethodGet, "/api", nil)
	reqTokenDenied.Header.Set("Authorization", "Bearer token-12345")
	wTokenDenied := httptest.NewRecorder()
	handler.ServeHTTP(wTokenDenied, reqTokenDenied)
	if wTokenDenied.Code != http.StatusTooManyRequests {
		t.Fatalf("bearer token request 3 expected 429, got %d", wTokenDenied.Code)
	}

	// Different token is isolated
	reqOtherToken := httptest.NewRequest(http.MethodGet, "/api", nil)
	reqOtherToken.Header.Set("Authorization", "Bearer token-67890")
	wOtherToken := httptest.NewRecorder()
	handler.ServeHTTP(wOtherToken, reqOtherToken)
	if wOtherToken.Code != http.StatusOK {
		t.Fatalf("different bearer token expected 200, got %d", wOtherToken.Code)
	}
}

func TestEarlyDataMiddleware_RejectsUnsafeMethods(t *testing.T) {
	handler := EarlyDataMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	unsafeMethods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, method := range unsafeMethods {
		req := httptest.NewRequest(method, "/api/resource", nil)
		req.Header.Set("Early-Data", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooEarly {
			t.Fatalf("method %s with Early-Data: 1 expected 425 Too Early, got %d", method, w.Code)
		}
		if retryAfter := w.Header().Get("Retry-After"); retryAfter != "0" {
			t.Fatalf("method %s expected Retry-After: 0, got %q", method, retryAfter)
		}
	}
}

func TestEarlyDataMiddleware_AllowsSafeMethodsAndWithoutHeader(t *testing.T) {
	handler := EarlyDataMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Safe methods with Early-Data: 1 are allowed
	safeMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	for _, method := range safeMethods {
		req := httptest.NewRequest(method, "/api/resource", nil)
		req.Header.Set("Early-Data", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("safe method %s with Early-Data: 1 expected 200 OK, got %d", method, w.Code)
		}
	}

	// Unsafe methods without Early-Data header are allowed
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/resource", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("unsafe method %s without Early-Data expected 200 OK, got %d", method, w.Code)
		}
	}

	// Early-Data with value "0" or invalid value does not trigger rejection
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/resource", nil)
	reqInvalid.Header.Set("Early-Data", "0")
	wInvalid := httptest.NewRecorder()
	handler.ServeHTTP(wInvalid, reqInvalid)
	if wInvalid.Code != http.StatusOK {
		t.Fatalf("POST with Early-Data: 0 expected 200 OK, got %d", wInvalid.Code)
	}
}


