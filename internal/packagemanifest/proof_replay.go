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

func replayKey(tenantID, organizationID, deviceID, purpose, requestID string) string {
	return tenantID + "\x00" + organizationID + "\x00" + deviceID + "\x00" + purpose + "\x00" + requestID
}
