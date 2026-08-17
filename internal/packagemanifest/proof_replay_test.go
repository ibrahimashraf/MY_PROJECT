package packagemanifest

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInMemoryProofReplayStoreConsumesOnlyOnce(t *testing.T) {
	now := time.Date(2026, time.August, 17, 10, 0, 0, 0, time.UTC)
	store := NewInMemoryProofReplayStore(func() time.Time { return now })

	if err := store.Consume(context.Background(), "device-1", "work_package_manifest.read", "request-1", now.Add(time.Minute)); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if err := store.Consume(context.Background(), "device-1", "work_package_manifest.read", "request-1", now.Add(time.Minute)); !errors.Is(err, ErrReplayAlreadyConsumed) {
		t.Fatalf("expected replay error, got %v", err)
	}
}

func TestInMemoryProofReplayStoreRejectsExpiredProof(t *testing.T) {
	now := time.Date(2026, time.August, 17, 10, 0, 0, 0, time.UTC)
	store := NewInMemoryProofReplayStore(func() time.Time { return now })

	err := store.Consume(context.Background(), "device-1", "work_package_manifest.read", "request-1", now)
	if !errors.Is(err, ErrReplayExpired) {
		t.Fatalf("expected expiry error, got %v", err)
	}
}
