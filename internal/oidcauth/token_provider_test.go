package oidcauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientCredentialsTokenProvider(t *testing.T) {
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse form: %v", err)
		}
		if r.FormValue("grant_type") != "client_credentials" {
			t.Errorf("expected grant_type=client_credentials, got %s", r.FormValue("grant_type"))
		}
		if r.FormValue("client_id") != "test-client" {
			t.Errorf("expected client_id=test-client, got %s", r.FormValue("client_id"))
		}
		if r.FormValue("client_secret") != "test-secret" {
			t.Errorf("expected client_secret=test-secret, got %s", r.FormValue("client_secret"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "token-xyz-123",
			"token_type":   "Bearer",
			"expires_in":   300,
		})
	}))
	defer server.Close()

	cfg := ClientCredentialsConfig{
		TokenEndpoint: server.URL,
		ClientID:      "test-client",
		ClientSecret:  "test-secret",
		RefreshBuffer: 30 * time.Second,
	}

	provider, err := NewClientCredentialsTokenProvider(cfg, server.Client())
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()

	// 1. Initial token fetch
	token, err := provider.GetToken(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting token: %v", err)
	}
	if token != "token-xyz-123" {
		t.Fatalf("expected token-xyz-123, got %s", token)
	}
	if count := atomic.LoadInt64(&requestCount); count != 1 {
		t.Fatalf("expected 1 request, got %d", count)
	}

	// 2. Cached token fetch (no network call)
	token2, err := provider.GetToken(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting cached token: %v", err)
	}
	if token2 != "token-xyz-123" {
		t.Fatalf("expected token-xyz-123, got %s", token2)
	}
	if count := atomic.LoadInt64(&requestCount); count != 1 {
		t.Fatalf("expected still 1 request due to cache, got %d", count)
	}

	// 3. Fast-forward time past expiry buffer -> should trigger refresh
	provider.now = func() time.Time {
		return time.Now().UTC().Add(350 * time.Second)
	}

	token3, err := provider.GetToken(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting refreshed token: %v", err)
	}
	if token3 != "token-xyz-123" {
		t.Fatalf("expected token-xyz-123, got %s", token3)
	}
	if count := atomic.LoadInt64(&requestCount); count != 2 {
		t.Fatalf("expected 2 requests after expiry, got %d", count)
	}
}

func TestClientCredentialsTokenProviderErrors(t *testing.T) {
	// Rejection when server returns 401 unauthorized
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized_client"}`))
	}))
	defer server.Close()

	cfg := ClientCredentialsConfig{
		TokenEndpoint: server.URL,
		ClientID:      "bad-client",
		ClientSecret:  "bad-secret",
	}

	provider, err := NewClientCredentialsTokenProvider(cfg, server.Client())
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	_, err = provider.GetToken(context.Background())
	if err == nil {
		t.Fatal("expected error on 401 response, got nil")
	}
}
