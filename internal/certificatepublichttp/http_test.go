package certificatepublichttp

import (
	"context"
	"integin/internal/certificatepg"
	"integin/internal/server"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type verifierStub struct {
	calls int
	found bool
}

func (s *verifierStub) VerifyPublic(context.Context, string) (certificatepg.PublicProjection, bool, error) {
	s.calls++
	return certificatepg.PublicProjection{CertificateNumber: "CERT-00000001", Status: "ISSUED", AssetID: "asset-a", IssuedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), ExpiresAt: time.Date(2027, 8, 1, 0, 0, 0, 0, time.UTC)}, s.found, nil
}

type richVerifierStub struct{}

func (richVerifierStub) VerifyPublic(context.Context, string) (certificatepg.PublicProjection, bool, error) {
	return certificatepg.PublicProjection{CertificateNumber: "CERT-00000002", Status: "ISSUED", AssetID: "asset-a", AssetSerialNumber: "SERIAL-A", AssetDescription: "Controlled lifting asset", AssetType: "lifting", InspectionType: "periodic_lifting", TestScope: []byte(`{"result_state":"PASS","items":[{"code":"visual","outcome":"PASS"}]}`), IssuedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), ExpiresAt: time.Date(2027, 8, 1, 0, 0, 0, 0, time.UTC)}, true, nil
}

func TestPublicProjectionIsBounded(t *testing.T) {
	v := &verifierStub{found: true}
	h := &Handler{Verifier: v, Limit: 2}
	r := httptest.NewRequest(http.MethodGet, "/verify/certificates/"+strings.Repeat("a", 43), nil)
	r.RemoteAddr = "127.0.0.1:1"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Body.String(), "asset-a") {
		t.Fatalf("status=%d cache=%q body=%s", w.Code, w.Header().Get("Cache-Control"), w.Body.String())
	}
}
func TestInvalidAndUnknownTokensAreIndistinguishable(t *testing.T) {
	v := &verifierStub{}
	h := &Handler{Verifier: v}
	a := httptest.NewRequest(http.MethodGet, "/verify/certificates/short", nil)
	a.RemoteAddr = "127.0.0.1:1"
	aw := httptest.NewRecorder()
	h.ServeHTTP(aw, a)
	b := httptest.NewRequest(http.MethodGet, "/verify/certificates/"+strings.Repeat("b", 43), nil)
	b.RemoteAddr = "127.0.0.2:1"
	bw := httptest.NewRecorder()
	h.ServeHTTP(bw, b)
	if aw.Code != http.StatusNotFound || bw.Code != http.StatusNotFound || v.calls != 1 {
		t.Fatalf("invalid=%d unknown=%d calls=%d", aw.Code, bw.Code, v.calls)
	}
}
func TestRateLimitPrecedesLookup(t *testing.T) {
	v := &verifierStub{found: true}
	h := &Handler{Verifier: v, Limit: 1}
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest(http.MethodGet, "/verify/certificates/"+strings.Repeat("c", 43), nil)
		r.RemoteAddr = "127.0.0.1:1"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if i == 1 && w.Code != http.StatusTooManyRequests {
			t.Fatalf("status=%d", w.Code)
		}
	}
	if v.calls != 1 {
		t.Fatalf("calls=%d", v.calls)
	}
}

func TestPublicProjectionAllowsOnlySnapshotBackedFields(t *testing.T) {
	h := &Handler{Verifier: richVerifierStub{}, Limit: 2}
	r := httptest.NewRequest(http.MethodGet, "/verify/certificates/"+strings.Repeat("d", 43), nil)
	r.RemoteAddr = "127.0.0.1:1"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	body := w.Body.String()
	for _, expected := range []string{"SERIAL-A", "Controlled lifting asset", "periodic_lifting", "test_scope"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing %q from body=%s", expected, body)
		}
	}
	for _, forbidden := range []string{"tenant", "organization", "actor", "token", "audit"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("forbidden %q leaked in body=%s", forbidden, body)
		}
	}
}

func TestPublicVerifierComposesThroughServerMux(t *testing.T) {
	verifier := &verifierStub{found: true}
	runtime := httptest.NewServer(server.NewMux(server.Dependencies{CertificatePublicHandler: &Handler{Verifier: verifier, Limit: 1}}))
	t.Cleanup(runtime.Close)
	token := strings.Repeat("z", 43)
	response, err := http.Get(runtime.URL + "/verify/certificates/" + token)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("status=%d cache=%q", response.StatusCode, response.Header.Get("Cache-Control"))
	}
	limited, err := http.Get(runtime.URL + "/verify/certificates/" + token)
	if err != nil {
		t.Fatal(err)
	}
	defer limited.Body.Close()
	if limited.StatusCode != http.StatusTooManyRequests || limited.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("rate status=%d cache=%q", limited.StatusCode, limited.Header.Get("Cache-Control"))
	}
	if verifier.calls != 1 {
		t.Fatalf("lookup calls=%d", verifier.calls)
	}
}

func TestTrustedProxyRemoteKeyExtraction(t *testing.T) {
	v := &verifierStub{found: true}
	h := &Handler{
		Verifier:       v,
		Limit:          1,
		TrustedProxies: []string{"10.0.0.1", "127.0.0.1"},
	}

	token := strings.Repeat("x", 43)

	// First request from client 198.51.100.25 via trusted proxy 10.0.0.1
	r1 := httptest.NewRequest(http.MethodGet, "/verify/certificates/"+token, nil)
	r1.RemoteAddr = "10.0.0.1:54321"
	r1.Header.Set("X-Forwarded-For", "198.51.100.25, 10.0.0.1")
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}

	// Second request from client 198.51.100.25 via trusted proxy should hit rate limit (429)
	r2 := httptest.NewRequest(http.MethodGet, "/verify/certificates/"+token, nil)
	r2.RemoteAddr = "10.0.0.1:54322"
	r2.Header.Set("X-Forwarded-For", "198.51.100.25, 10.0.0.1")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for same client IP, got %d", w2.Code)
	}

	// Request from a DIFFERENT client 198.51.100.26 via the same proxy should be ALLOWED (200)
	r3 := httptest.NewRequest(http.MethodGet, "/verify/certificates/"+token, nil)
	r3.RemoteAddr = "10.0.0.1:54323"
	r3.Header.Set("X-Forwarded-For", "198.51.100.26, 10.0.0.1")
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, r3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 for different client IP, got %d", w3.Code)
	}
}

func TestUntrustedProxyIgnoresForwardedHeaders(t *testing.T) {
	v := &verifierStub{found: true}
	// Handler without trusted proxies
	h := &Handler{
		Verifier: v,
		Limit:    1,
	}

	token := strings.Repeat("y", 43)

	// Caller attempts spoofing X-Forwarded-For from untrusted remote 192.0.2.1
	r1 := httptest.NewRequest(http.MethodGet, "/verify/certificates/"+token, nil)
	r1.RemoteAddr = "192.0.2.1:12345"
	r1.Header.Set("X-Forwarded-For", " spoofed.ip.1")
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}

	// Next request sends different spoofed header but same RemoteAddr -> must be blocked
	r2 := httptest.NewRequest(http.MethodGet, "/verify/certificates/"+token, nil)
	r2.RemoteAddr = "192.0.2.1:12346"
	r2.Header.Set("X-Forwarded-For", "spoofed.ip.2")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 because untrusted proxy header must be ignored, got %d", w2.Code)
	}
}


func BenchmarkPublicVerifierHandler(b *testing.B) {
	handler := &Handler{Verifier: richVerifierStub{}, Limit: b.N + 1}
	path := "/verify/certificates/" + strings.Repeat("p", 43)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.RemoteAddr = "127.0.0.1:1"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			b.Fatalf("status=%d", response.Code)
		}
	}
}
