package oidcauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"integin/internal/security"
)

func TestDynamicSecretManagerWithTokenProviderIntegration(t *testing.T) {
	// 1. Setup mock Keycloak M2M token endpoint
	tokenEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.FormValue("client_id") == "integin-service" && r.FormValue("client_secret") == "dynamic-vault-secret" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"ey-jwt-dynamic-token","token_type":"Bearer","expires_in":1800}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer tokenEndpoint.Close()

	// 2. Setup OpenBao / Vault KV v2 secret provider
	secretProvider := security.NewInMemorySecretProvider()
	_ = secretProvider.SetSecret("kv/production/keycloak/admin/bootstrap", map[string]string{
		"client_id":     "integin-service",
		"client_secret": "dynamic-vault-secret",
	}, 1*time.Hour)

	secretMgr, err := security.NewDynamicSecretManager(secretProvider)
	if err != nil {
		t.Fatalf("failed to create secret manager: %v", err)
	}

	ctx := context.Background()

	// 3. Resolve client secret dynamically through SecretManager
	lease, err := secretMgr.ResolveSecret(ctx, "kv/production/keycloak/admin/bootstrap")
	if err != nil {
		t.Fatalf("failed to resolve keycloak secret: %v", err)
	}

	// 4. Construct TokenProvider using dynamic credentials
	tokenProvider, err := NewClientCredentialsTokenProvider(ClientCredentialsConfig{
		TokenEndpoint: tokenEndpoint.URL,
		ClientID:      lease.Data["client_id"],
		ClientSecret:  lease.Data["client_secret"],
	}, tokenEndpoint.Client())
	if err != nil {
		t.Fatalf("failed to create token provider: %v", err)
	}

	// 5. Fetch dynamic token
	token, err := tokenProvider.GetToken(ctx)
	if err != nil {
		t.Fatalf("failed to get dynamic token: %v", err)
	}
	if token != "ey-jwt-dynamic-token" {
		t.Fatalf("expected ey-jwt-dynamic-token, got %s", token)
	}

	// 6. Test fail-closed on corrupted secret
	_ = secretProvider.SetSecret("kv/production/keycloak/admin/bootstrap", map[string]string{
		"client_id":     "integin-service",
		"client_secret": "revoked-secret",
	}, 1*time.Hour)

	badProvider, _ := NewClientCredentialsTokenProvider(ClientCredentialsConfig{
		TokenEndpoint: tokenEndpoint.URL,
		ClientID:      "integin-service",
		ClientSecret:  "revoked-secret",
	}, tokenEndpoint.Client())

	_, err = badProvider.GetToken(ctx)
	if err == nil {
		t.Fatal("expected authentication failure with revoked secret, got nil")
	}
}
