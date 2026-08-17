package packagemanifest

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPostgresProofReplayStoreRejectsExpiredProofBeforeDatabaseAccess(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	store := NewPostgresProofReplayStore(nil)
	store.now = func() time.Time { return now }
	err := store.Consume(context.Background(), "tenant", "organization", "device", "purpose", "request", now)
	if !errors.Is(err, ErrReplayExpired) {
		t.Fatalf("expected expired proof error, got %v", err)
	}
}

func TestPostgresProofReplayStoreRejectsIncompleteScopeBeforeDatabaseAccess(t *testing.T) {
	store := NewPostgresProofReplayStore(nil)
	err := store.Consume(context.Background(), "", "organization", "device", "purpose", "request", time.Now().Add(time.Minute))
	if err == nil || err.Error() != "proof replay key is incomplete" {
		t.Fatalf("expected incomplete-key error before database access, got %v", err)
	}
}
