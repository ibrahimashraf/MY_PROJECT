package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
)

func TestNewMuxMountsHealthAndSyncRoutes(t *testing.T) {
	processor, err := domainsync.NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	device, err := device_trust.NewDevice("device-1", "tenant-1", "org-1", "user-1", "public-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	authority, err := device_trust.IssueAuthorityPackage(device, "auth-1", "secret", []string{"inspection.perform"}, time.Now().UTC(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	mux := NewMux(Dependencies{SyncProcessor: processor, Devices: []device_trust.Device{device}, Authorities: []device_trust.AuthorityPackage{authority}})
	health := httptest.NewRecorder()
	mux.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d", health.Code)
	}

	method := httptest.NewRecorder()
	mux.ServeHTTP(method, httptest.NewRequest(http.MethodGet, "/sync", nil))
	if method.Code != http.StatusMethodNotAllowed {
		t.Fatalf("sync method status = %d", method.Code)
	}
}

func TestSecurityHeadersMiddlewareEnforcesOWASPBaseline(t *testing.T) {
	mux := NewMux(Dependencies{})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if ct := rec.Header().Get("X-Content-Type-Options"); ct != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %q", ct)
	}
	if fo := rec.Header().Get("X-Frame-Options"); fo != "DENY" {
		t.Errorf("expected X-Frame-Options: DENY, got %q", fo)
	}
	if rp := rec.Header().Get("Referrer-Policy"); rp != "strict-origin-when-cross-origin" {
		t.Errorf("expected Referrer-Policy: strict-origin-when-cross-origin, got %q", rp)
	}
	if hsts := rec.Header().Get("Strict-Transport-Security"); !strings.Contains(hsts, "max-age=") {
		t.Errorf("expected Strict-Transport-Security header, got %q", hsts)
	}
}

func TestNewMuxDoesNotMountOIDCSessionRouteWithoutDependency(t *testing.T) {
	mux := NewMux(Dependencies{})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/identity/session", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("OIDC-disabled session route status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestOperationalHealthAndReadinessSemantics(t *testing.T) {
	failing := NewMux(Dependencies{Readiness: func(context.Context) error { return errors.New("dependency unavailable") }})
	health := httptest.NewRecorder()
	failing.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK || health.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("liveness response = status %d headers %#v", health.Code, health.Header())
	}
	ready := httptest.NewRecorder()
	failing.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusServiceUnavailable || strings.Contains(ready.Body.String(), "dependency") {
		t.Fatalf("failing readiness response = %d %s", ready.Code, ready.Body.String())
	}
	passing := NewMux(Dependencies{Readiness: func(context.Context) error { return nil }})
	ready = httptest.NewRecorder()
	passing.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusOK {
		t.Fatalf("ready status = %d", ready.Code)
	}
	timedOut := NewMux(Dependencies{ReadinessTimeout: 10 * time.Millisecond, Readiness: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}})
	ready = httptest.NewRecorder()
	timedOut.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusServiceUnavailable {
		t.Fatalf("deadline readiness status = %d", ready.Code)
	}
}

func TestCorrelationLimitAndStructuredLogRedaction(t *testing.T) {
	var called int
	endpoint := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		called++
		writer.WriteHeader(http.StatusNoContent)
	})
	mux := NewMux(Dependencies{OIDCSessionHandler: endpoint})
	safe := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/identity/session", nil)
	request.Header.Set("X-Correlation-ID", "trace_1.safe-id")
	mux.ServeHTTP(safe, request)
	if safe.Header().Get("X-Correlation-ID") != "trace_1.safe-id" {
		t.Fatalf("safe correlation id = %q", safe.Header().Get("X-Correlation-ID"))
	}
	unsafe := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/identity/session", nil)
	request.Header.Set("X-Correlation-ID", "unsafe value")
	mux.ServeHTTP(unsafe, request)
	if unsafe.Header().Get("X-Correlation-ID") == "unsafe value" || unsafe.Header().Get("X-Correlation-ID") == "" {
		t.Fatalf("unsafe correlation handling = %q", unsafe.Header().Get("X-Correlation-ID"))
	}
	oversized := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/identity/session", bytes.NewReader(make([]byte, 10<<20+1)))
	mux.ServeHTTP(oversized, request)
	if oversized.Code != http.StatusRequestEntityTooLarge || called != 2 || oversized.Body.String() != `{"error":"request_too_large"}` {
		t.Fatalf("oversized response=%d body=%s called=%d", oversized.Code, oversized.Body.String(), called)
	}

	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz?token=do-not-log", nil))
	if strings.Contains(logs.String(), "do-not-log") || !strings.Contains(logs.String(), `"path":"/healthz"`) || !strings.Contains(logs.String(), `"correlation_id"`) {
		t.Fatalf("redacted structured log = %s", logs.String())
	}
	logs.Reset()
	mux = NewMux(Dependencies{CertificatePublicHandler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})})
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/verify/certificates/token-that-must-not-appear", nil))
	if strings.Contains(logs.String(), "token-that-must-not-appear") || !strings.Contains(logs.String(), `"path":"/verify/certificates/:token"`) {
		t.Fatalf("public token path log redaction = %s", logs.String())
	}
}

func TestRequestLimitProtectsUnknownLengthStreamingBody(t *testing.T) {
	limited := withRequestLimit(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, err := io.ReadAll(request.Body)
		if err == nil {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		writeOperationalJSON(writer, http.StatusRequestEntityTooLarge, `{"error":"request_too_large"}`)
	}), 4)
	request := httptest.NewRequest(http.MethodPost, "/stream", io.NopCloser(strings.NewReader("12345")))
	request.ContentLength = -1
	response := httptest.NewRecorder()
	limited.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || response.Body.String() != `{"error":"request_too_large"}` {
		t.Fatalf("streaming limit response = %d %s", response.Code, response.Body.String())
	}
}

func TestNewMuxMountsCertificateRoutesOnlyWhenHandlersProvided(t *testing.T) {
	absent := NewMux(Dependencies{})
	for _, path := range []string{"/certificates/id/issue", "/verify/certificates/token"} {
		response := httptest.NewRecorder()
		absent.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("absent route %s status=%d", path, response.Code)
		}
	}
	authenticatedCalls := 0
	publicCalls := 0
	mux := NewMux(Dependencies{
		CertificateHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authenticatedCalls++
			if r.URL.Path != "/certificates/id/issue" {
				t.Fatalf("authenticated path=%q", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		}),
		CertificatePublicHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			publicCalls++
			if r.URL.Path != "/verify/certificates/token" {
				t.Fatalf("public path=%q", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}),
	})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/certificates/id/issue", nil))
	if response.Code != http.StatusNoContent || authenticatedCalls != 1 {
		t.Fatalf("authenticated status=%d calls=%d", response.Code, authenticatedCalls)
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/verify/certificates/token", nil))
	if response.Code != http.StatusOK || publicCalls != 1 {
		t.Fatalf("public status=%d calls=%d", response.Code, publicCalls)
	}
}

func TestMetricsEndpointReturnsPrometheusFormat(t *testing.T) {
	processor, _ := domainsync.NewProcessor(map[string]string{"test-tenant": "test-secret-32-bytes-minimum---!"})
	mux := NewMux(Dependencies{SyncProcessor: processor})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/metrics status = %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Fatalf("/metrics Content-Type = %q, want text/plain", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"integin_ratelimit_allowed_total", "integin_ratelimit_denied_total", "integin_go_goroutines", "integin_sync_offline_hmac_fallback_total"} {
		if !strings.Contains(body, want) {
			t.Fatalf("/metrics body missing %s", want)
		}
	}
}

func TestNewMuxMountsTUSRoutesOnlyWhenHandlerProvided(t *testing.T) {
	absent := NewMux(Dependencies{})
	for _, path := range []string{"/uploads", "/uploads/some-id/chunks"} {
		response := httptest.NewRecorder()
		absent.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("absent route %s status=%d", path, response.Code)
		}
	}
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/uploads" {
			t.Fatalf("TUS path=%q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux := NewMux(Dependencies{TUSHandler: handler, Validator: tusValidatorStub{}})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/uploads", nil)
	request.Header.Set("Authorization", "Bearer valid")
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || calls != 1 {
		t.Fatalf("mounted TUS status=%d calls=%d", response.Code, calls)
	}
}
