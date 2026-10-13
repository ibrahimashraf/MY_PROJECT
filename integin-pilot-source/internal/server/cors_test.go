package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSDisabledByDefault(t *testing.T) {
	t.Setenv("INTEGIN_CORS_ALLOWED_ORIGINS", "")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	handler := withCORS(newCORSConfigFromEnv(), next)

	preflight := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/sync", nil)
	req.Header.Set("Origin", "http://127.0.0.1:8085")
	req.Header.Set("Access-Control-Request-Method", "POST")
	handler.ServeHTTP(preflight, req)
	if preflight.Code != http.StatusTeapot {
		t.Fatalf("disabled CORS must pass preflight through, got %d", preflight.Code)
	}
	if preflight.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("disabled CORS must not emit ACAO headers")
	}
}

func TestCORSPreflightAllowsListedOrigin(t *testing.T) {
	t.Setenv("INTEGIN_CORS_ALLOWED_ORIGINS", "http://127.0.0.1:8085, http://localhost:8085")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("preflight must short-circuit before the handler")
	})
	handler := withCORS(newCORSConfigFromEnv(), next)

	preflight := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/sync", nil)
	req.Header.Set("Origin", "http://127.0.0.1:8085")
	req.Header.Set("Access-Control-Request-Method", "POST")
	handler.ServeHTTP(preflight, req)
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", preflight.Code)
	}
	if got := preflight.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:8085" {
		t.Errorf("ACAO must echo the listed origin, got %q", got)
	}
	if preflight.Header().Get("Access-Control-Allow-Methods") == "" ||
		preflight.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("preflight must declare methods and headers")
	}
	if preflight.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Error("bearer-token CORS must not allow credentials")
	}
}

func TestCORSRejectsUnlistedOrigin(t *testing.T) {
	t.Setenv("INTEGIN_CORS_ALLOWED_ORIGINS", "http://127.0.0.1:8085")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := withCORS(newCORSConfigFromEnv(), next)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sync", nil)
	req.Header.Set("Origin", "https://evil.example")
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unlisted origin must not receive ACAO headers")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("unlisted origin passes through to handler, got %d", rec.Code)
	}

	simple := httptest.NewRecorder()
	simpleReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	simpleReq.Header.Set("Origin", "http://127.0.0.1:8085")
	handler.ServeHTTP(simple, simpleReq)
	if got := simple.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:8085" {
		t.Errorf("listed simple request must carry ACAO, got %q", got)
	}
	if vary := simple.Header().Get("Vary"); vary != "Origin" {
		t.Errorf("Vary must be Origin, got %q", vary)
	}
}
