package packagemanifest

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInMemoryProofReplayStoreIsTenantScoped(t *testing.T) {
	now := time.Date(2026, time.August, 17, 10, 0, 0, 0, time.UTC)
	store := NewInMemoryProofReplayStore(func() time.Time { return now })
	expiresAt := now.Add(time.Minute)
	if err := store.Consume(context.Background(), "tenant-1", "organization-1", "device-1", "purpose", "request-1", expiresAt); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if err := store.Consume(context.Background(), "tenant-2", "organization-1", "device-1", "purpose", "request-1", expiresAt); err != nil {
		t.Fatalf("different tenant must be independent: %v", err)
	}
	if err := store.Consume(context.Background(), "tenant-1", "organization-1", "device-1", "purpose", "request-1", expiresAt); !errors.Is(err, ErrReplayAlreadyConsumed) {
		t.Fatalf("expected replay rejection, got %v", err)
	}
}
