package workpackagepg

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestManifestProofReplayStoreRejectsExpiredProofBeforeDatabaseAccess(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	store := NewManifestProofReplayStore(&Repository{})
	store.now = func() time.Time { return now }
	err := store.Consume(context.Background(), "tenant", "organization", "device", "purpose", "request", now)
	if !errors.Is(err, ErrManifestProofReplayExpired) {
		t.Fatalf("expected expired proof error, got %v", err)
	}
}

func TestManifestProofReplayStoreRejectsIncompleteScopeBeforeDatabaseAccess(t *testing.T) {
	store := NewManifestProofReplayStore(&Repository{})
	err := store.Consume(context.Background(), "", "organization", "device", "purpose", "request", time.Now().Add(time.Minute))
	if err == nil || err.Error() != "proof replay key is incomplete" {
		t.Fatalf("expected incomplete-key error before database access, got %v", err)
	}
}
