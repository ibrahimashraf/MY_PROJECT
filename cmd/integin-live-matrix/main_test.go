package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExplicitPilotServerURL(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		wantOK bool
	}{
		{"accepts pilot", "http://127.0.0.1:18080/", true},
		{"rejects omitted", "", false},
		{"rejects acceptance", "http://127.0.0.1:8080", false},
		{"rejects alternate local port", "http://127.0.0.1:19080", false},
		{"rejects https", "https://127.0.0.1:18080", false},
		{"rejects path", "http://127.0.0.1:18080/sync", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := explicitPilotServerURL(test.raw)
			if test.wantOK {
				if err != nil || got != "http://127.0.0.1:18080" {
					t.Fatalf("explicitPilotServerURL(%q) = %q, %v", test.raw, got, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("explicitPilotServerURL(%q) unexpectedly succeeded", test.raw)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// postJSON
// ---------------------------------------------------------------------------

func TestPostJSON_InvalidURL(t *testing.T) {
	client := &http.Client{}
	_, err := postJSON(client, "ftp://bad-scheme.example/sync", struct{}{}, "test-key-000")
	if err == nil || !strings.Contains(err.Error(), "invalid endpoint URL") {
		t.Fatalf("expected invalid endpoint URL error, got %v", err)
	}
}

func TestPostJSON_EmptyHost(t *testing.T) {
	client := &http.Client{}
	_, err := postJSON(client, "http:///no-host", struct{}{}, "test-key-000")
	if err == nil || !strings.Contains(err.Error(), "invalid endpoint URL") {
		t.Fatalf("expected invalid endpoint URL error, got %v", err)
	}
}

func TestPostJSON_ServerReturnsOutcome(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("unexpected Content-Type: %s", ct)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "test-key-000" {
			t.Errorf("expected Idempotency-Key header, got %q", key)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(outcomeResponse{Outcome: "APPLIED"})
	}))
	defer srv.Close()

	got, err := postJSON(srv.Client(), srv.URL+"/sync", struct{ X int }{X: 1}, "test-key-000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Outcome != "APPLIED" {
		t.Fatalf("expected APPLIED, got %q", got.Outcome)
	}
}

func TestPostJSON_Non200StillDecodes(t *testing.T) {
	// The server contract returns structured outcomes even on 4xx (e.g. CONFLICT).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(outcomeResponse{Outcome: "CONFLICT", Reason: "duplicate"})
	}))
	defer srv.Close()

	got, err := postJSON(srv.Client(), srv.URL+"/sync", struct{}{}, "test-key-000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Outcome != "CONFLICT" {
		t.Fatalf("expected CONFLICT, got %q", got.Outcome)
	}
}

func TestPostJSON_MalformedResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := postJSON(srv.Client(), srv.URL+"/sync", struct{}{}, "test-key-000")
	if err == nil {
		t.Fatal("expected JSON decode error, got nil")
	}
}

// ---------------------------------------------------------------------------
// localConfig
// ---------------------------------------------------------------------------

func TestLocalConfig_MissingFixturePath(t *testing.T) {
	t.Setenv("INTEGIN_LIVE_FIXTURE_FILE", "")
	_, _, _, _, err := localConfig()
	if err == nil || !strings.Contains(err.Error(), "INTEGIN_LIVE_FIXTURE_FILE") {
		t.Fatalf("expected INTEGIN_LIVE_FIXTURE_FILE error, got %v", err)
	}
}

func TestLocalConfig_MissingDBURL(t *testing.T) {
	t.Setenv("INTEGIN_LIVE_FIXTURE_FILE", "/tmp/fixture.json")
	t.Setenv("INTEGIN_DB_URL", "")
	t.Setenv("INTEGIN_TENANT_ID", "tenant-1")
	t.Setenv("INTEGIN_SYNC_SECRET", "secret-1")
	_, _, _, _, err := localConfig()
	if err == nil || !strings.Contains(err.Error(), "INTEGIN_DB_URL") {
		t.Fatalf("expected INTEGIN_DB_URL error, got %v", err)
	}
}

func TestLocalConfig_MissingTenantID(t *testing.T) {
	t.Setenv("INTEGIN_LIVE_FIXTURE_FILE", "/tmp/fixture.json")
	t.Setenv("INTEGIN_DB_URL", "postgres://localhost/integin")
	t.Setenv("INTEGIN_TENANT_ID", "")
	t.Setenv("INTEGIN_SYNC_SECRET", "secret-1")
	_, _, _, _, err := localConfig()
	if err == nil || !strings.Contains(err.Error(), "INTEGIN_TENANT_ID") {
		t.Fatalf("expected INTEGIN_TENANT_ID error, got %v", err)
	}
}

func TestLocalConfig_MissingSecret(t *testing.T) {
	t.Setenv("INTEGIN_LIVE_FIXTURE_FILE", "/tmp/fixture.json")
	t.Setenv("INTEGIN_DB_URL", "postgres://localhost/integin")
	t.Setenv("INTEGIN_TENANT_ID", "tenant-1")
	t.Setenv("INTEGIN_SYNC_SECRET", "")
	_, _, _, _, err := localConfig()
	if err == nil || !strings.Contains(err.Error(), "INTEGIN_SYNC_SECRET") {
		t.Fatalf("expected INTEGIN_SYNC_SECRET error, got %v", err)
	}
}

func TestLocalConfig_AllPresent(t *testing.T) {
	t.Setenv("INTEGIN_LIVE_FIXTURE_FILE", "/tmp/fixture.json")
	t.Setenv("INTEGIN_DB_URL", "postgres://localhost/integin")
	t.Setenv("INTEGIN_TENANT_ID", "tenant-1")
	t.Setenv("INTEGIN_SYNC_SECRET", "secret-1")
	dbURL, tenantID, secret, fixturePath, err := localConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dbURL != "postgres://localhost/integin" || tenantID != "tenant-1" || secret != "secret-1" || fixturePath != "/tmp/fixture.json" {
		t.Fatalf("localConfig returned unexpected values: %q %q %q %q", dbURL, tenantID, secret, fixturePath)
	}
}
