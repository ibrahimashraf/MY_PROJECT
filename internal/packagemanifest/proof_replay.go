// Package packagemanifest contains replay controls for the future package-manifest route.
package packagemanifest

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"integin/internal/domain/workpackage"
)

var (
	// Error identity is owned by the PostgreSQL layer because the production
	// manifest package already depends on that layer for assignment types.
	ErrReplayAlreadyConsumed = workpackage.ErrManifestProofReplayAlreadyConsumed
	ErrReplayExpired         = workpackage.ErrManifestProofReplayExpired
)

// ProofReplayStore atomically consumes one verified, tenant-scoped proof request.
type ProofReplayStore interface {
	Consume(ctx context.Context, tenantID, organizationID, deviceID, purpose, requestID string, expiresAt time.Time) error
}

// ProofReplayCounter exposes only a tenant-scoped aggregate of active replay
// entries. It never returns request IDs, proof bodies, device keys, or other
// replay identities.
type ProofReplayCounter interface {
	Count(ctx context.Context, tenantID, organizationID, deviceID, purpose string, at time.Time) (int64, error)
}

// InMemoryProofReplayStore exists only for deterministic unit tests. It is not
// durable and must never authorize a mounted runtime endpoint.
type InMemoryProofReplayStore struct {
	now  func() time.Time
	mu   sync.Mutex
	used map[string]time.Time
}

func NewInMemoryProofReplayStore(now func() time.Time) *InMemoryProofReplayStore {
	if now == nil {
		now = time.Now
	}
	return &InMemoryProofReplayStore{now: now, used: map[string]time.Time{}}
}

func (s *InMemoryProofReplayStore) Consume(_ context.Context, tenantID, organizationID, deviceID, purpose, requestID string, expiresAt time.Time) error {
	if s == nil {
		return errors.New("proof replay store is not configured")
	}
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(organizationID) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(purpose) == "" || strings.TrimSpace(requestID) == "" {
		return errors.New("proof replay key is incomplete")
	}
	now := s.now().UTC()
	if !expiresAt.After(now) {
		return ErrReplayExpired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, expiry := range s.used {
		if !expiry.After(now) {
			delete(s.used, key)
		}
	}
	key := replayKey(tenantID, organizationID, deviceID, purpose, requestID)
	if _, exists := s.used[key]; exists {
		return ErrReplayAlreadyConsumed
	}
	s.used[key] = expiresAt.UTC()
	return nil
}

// Count returns the active replay-entry total for one already verified scope.
// It exists for deterministic tests and source-owned non-secret receipt deltas;
// it must not be used as an identity enumeration API.
func (s *InMemoryProofReplayStore) Count(_ context.Context, tenantID, organizationID, deviceID, purpose string, at time.Time) (int64, error) {
	if s == nil {
		return 0, errors.New("proof replay store is not configured")
	}
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(organizationID) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(purpose) == "" || at.IsZero() {
		return 0, errors.New("proof replay count scope is incomplete")
	}
	now := at.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, expiry := range s.used {
		if !expiry.After(now) {
			delete(s.used, key)
		}
	}
	prefix := tenantID + "\x00" + organizationID + "\x00" + deviceID + "\x00" + purpose + "\x00"
	var count int64
	for key := range s.used {
		if strings.HasPrefix(key, prefix) {
			count++
		}
	}
	return count, nil
}

func replayKey(tenantID, organizationID, deviceID, purpose, requestID string) string {
	return tenantID + "\x00" + organizationID + "\x00" + deviceID + "\x00" + purpose + "\x00" + requestID
}
