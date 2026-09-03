package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_TenantRateLimiting(t *testing.T) {
	limiter := NewRateLimiter(1, 2) // 1 req/s, burst of 2
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
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("/healthz should never be rate limited, got %d on request %d", w.Code, i+1)
		}
	}
}
