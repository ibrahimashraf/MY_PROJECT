package vault

import (
	"context"
	"testing"
)

func TestVault_TenantIsolationAndBYOK(t *testing.T) {
	masterKey := []byte("01234567890123456789012345678901") // 32 bytes

	v, err := NewVault(masterKey)
	if err != nil {
		t.Fatalf("failed initializing vault: %v", err)
	}

	ctx := context.Background()
	tenantA := "tenant-aramco"
	tenantB := "tenant-sabic"
	orgA := "org-east"
	orgB := "org-west"

	rawOpenAIKey := "sk-live-992819283921829102"
	rawAnthropicKey := "sk-ant-481920192830192831"

	// 1. Store Tenant A keys
	secretA, err := v.StoreSecret(ctx, tenantA, orgA, "openai", rawOpenAIKey)
	if err != nil {
		t.Fatalf("failed storing secret: %v", err)
	}
	if secretA.KeyFingerprint != "sk-l...9102" {
		t.Fatalf("unexpected fingerprint: %s", secretA.KeyFingerprint)
	}

	// 2. Resolve Tenant A key successfully
	resolvedA, err := v.ResolveSecret(ctx, tenantA, orgA, "openai")
	if err != nil {
		t.Fatalf("failed resolving secret: %v", err)
	}
	if resolvedA != rawOpenAIKey {
		t.Fatalf("expected %s, got %s", rawOpenAIKey, resolvedA)
	}

	// 3. Cross-Tenant Breach attempt: Tenant B cannot resolve Tenant A secret
	_, err = v.ResolveSecret(ctx, tenantB, orgB, "openai")
	if err == nil || err != ErrSecretNotFound {
		t.Fatalf("expected ErrSecretNotFound for cross-tenant query, got: %v", err)
	}

	// 4. Storing key for Tenant B with same provider
	_, err = v.StoreSecret(ctx, tenantB, orgB, "openai", rawAnthropicKey)
	if err != nil {
		t.Fatalf("failed storing secret for tenant B: %v", err)
	}

	resolvedB, err := v.ResolveSecret(ctx, tenantB, orgB, "openai")
	if err != nil {
		t.Fatalf("failed resolving tenant B secret: %v", err)
	}
	if resolvedB != rawAnthropicKey {
		t.Fatalf("expected %s, got %s", rawAnthropicKey, resolvedB)
	}

	// 5. Tampered organization context must fail decryption (AAD mismatch)
	_, err = v.ResolveSecret(ctx, tenantA, "tampered-org", "openai")
	if err == nil || err != ErrDecryptionFailed {
		t.Fatalf("expected ErrDecryptionFailed on tampered context, got: %v", err)
	}
}
