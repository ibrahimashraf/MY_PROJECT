package security

import (
	"context"
	"testing"
	"time"
)

func TestInMemorySecretProviderAndManager(t *testing.T) {
	provider := NewInMemorySecretProvider()

	// 1. Path format validation
	err := provider.SetSecret("invalid/path", map[string]string{"key": "val"}, 0)
	if err == nil {
		t.Fatal("expected error for path not starting with kv/production/ or kv/test/")
	}

	// 2. Set valid secret
	testPath := "kv/production/integin/runtime/postgres"
	err = provider.SetSecret(testPath, map[string]string{
		"username": "runtime_user",
		"password": "runtime_password",
	}, 60*time.Second)
	if err != nil {
		t.Fatalf("failed to set secret: %v", err)
	}

	mgr, err := NewDynamicSecretManager(provider)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	// 3. Resolve secret through manager
	lease, err := mgr.ResolveSecret(ctx, testPath)
	if err != nil {
		t.Fatalf("failed to resolve secret: %v", err)
	}
	if lease.Data["username"] != "runtime_user" {
		t.Fatalf("expected username runtime_user, got %s", lease.Data["username"])
	}
	if lease.Version != 1 {
		t.Fatalf("expected version 1, got %d", lease.Version)
	}

	// 4. Update secret -> version increment
	_ = provider.SetSecret(testPath, map[string]string{
		"username": "runtime_user_v2",
		"password": "runtime_password_v2",
	}, 60*time.Second)

	// Simulate cache expiration
	mgr.now = func() time.Time {
		return time.Now().UTC().Add(50 * time.Second)
	}

	lease2, err := mgr.ResolveSecret(ctx, testPath)
	if err != nil {
		t.Fatalf("failed to resolve refreshed secret: %v", err)
	}
	if lease2.Data["username"] != "runtime_user_v2" {
		t.Fatalf("expected username runtime_user_v2, got %s", lease2.Data["username"])
	}
	if lease2.Version != 2 {
		t.Fatalf("expected version 2, got %d", lease2.Version)
	}

	// 5. Unknown secret fails closed
	_, err = mgr.ResolveSecret(ctx, "kv/production/unknown")
	if err == nil {
		t.Fatal("expected error on unknown secret path, got nil")
	}
}
