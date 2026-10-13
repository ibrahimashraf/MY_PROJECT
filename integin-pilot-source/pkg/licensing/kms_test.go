package licensing_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/pkg/licensing"
)

func TestOpenBaoKMSClient_Init(t *testing.T) {
	client, err := licensing.NewOpenBaoKMSClient("http://127.0.0.1:8200", "test-token")
	if err != nil {
		t.Fatalf("NewOpenBaoKMSClient failed: %v", err)
	}
	if client.Client() == nil {
		t.Fatal("expected non-nil vault client")
	}

	_, err = licensing.NewOpenBaoKMSClient("", "token")
	if err == nil {
		t.Fatal("expected error on empty address")
	}
}

func TestOpenBaoKMSClient_LiveHealthCheckRPC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"initialized":  true,
				"sealed":       false,
				"standby":      false,
				"version":      "2.0.0",
				"cluster_name": "integin-pilot-openbao",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := licensing.NewOpenBaoKMSClient(server.URL, "s.integin-pilot-token")
	if err != nil {
		t.Fatalf("failed to init OpenBao client: %v", err)
	}

	healthy, err := client.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck RPC failed: %v", err)
	}
	if !healthy {
		t.Fatal("expected OpenBao cluster to report healthy and unsealed")
	}
}

func TestOpenBaoKMSClient_SealedClusterDetection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable) // Vault / OpenBao returns 503 when sealed
			_ = json.NewEncoder(w).Encode(map[string]any{
				"initialized": true,
				"sealed":      true,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := licensing.NewOpenBaoKMSClient(server.URL, "s.integin-pilot-token")
	if err != nil {
		t.Fatalf("failed to init client: %v", err)
	}

	healthy, err := client.HealthCheck(context.Background())
	if healthy {
		t.Fatal("sealed cluster must not report healthy")
	}
}
