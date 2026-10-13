package aiintegration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"integin/internal/advisory"
)

func TestAIClientNormalizesResponseThroughAdvisoryContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/advisory" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("unexpected request: %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		var request Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Version != ContractVersion || request.TenantID != "tenant-1" {
			t.Fatalf("unexpected contract request: %#v", request)
		}
		_ = json.NewEncoder(w).Encode(Response{Version: ContractVersion, TenantID: "tenant-1", Title: "advisory", Summary: "summary", Severity: "ADVISORY", Confidence: .8, Rationale: "evidence", Provider: "python", Model: "model", PromptVersion: "prompt", Blocking: true})
	}))
	defer server.Close()
	client, err := New(Config{Endpoint: server.URL, APIKey: "secret"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	insight, err := advisory.Generate(context.Background(), client, advisory.Request{TenantID: "tenant-1", Zone: advisory.ZoneMonitoring, Lens: "TREND_ANOMALY", EvidenceRefs: []string{"event-1"}}, "insight-1", time.Now())
	if err != nil || insight.Blocking || insight.Provider != "python" {
		t.Fatalf("unexpected normalized insight: %#v %v", insight, err)
	}
}

func TestAIClientRejectsFreeZoneAndTenantMismatch(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_ = json.NewEncoder(w).Encode(Response{Version: ContractVersion, TenantID: "tenant-2", Confidence: .5, Rationale: "bad"})
	}))
	defer server.Close()
	client, err := New(Config{Endpoint: server.URL}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Generate(context.Background(), advisory.Request{TenantID: "tenant-1", Zone: advisory.ZoneCertificate}); err == nil {
		t.Fatal("free zone should reject")
	}
	if called {
		t.Fatal("free zone reached provider")
	}
	if _, err := client.Generate(context.Background(), advisory.Request{TenantID: "tenant-1", Zone: advisory.ZoneMonitoring, Lens: "SAFETY_RISK"}); err == nil {
		t.Fatal("tenant mismatch should reject")
	}
}

func TestAIClientHealthAndMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Write([]byte("not-json"))
	}))
	defer server.Close()
	client, err := New(Config{Endpoint: server.URL, Timeout: time.Second}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Generate(context.Background(), advisory.Request{TenantID: "tenant-1", Zone: advisory.ZoneMonitoring, Lens: "TREND_ANOMALY"}); err == nil {
		t.Fatal("malformed response should fail")
	}
}
