// Package packagemanifest contains replay controls for the future package-manifest route.
package packagemanifest

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	// ErrReplayAlreadyConsumed means an accepted proof request ID cannot be reused.
	ErrReplayAlreadyConsumed = errors.New("device proof replay detected")
	// ErrReplayExpired means a proof must not enter replay storage after its validity window.
	ErrReplayExpired = errors.New("device proof replay window is expired")
)

// ProofReplayStore atomically consumes one verified proof request.
// A mounted environment must use a tenant-safe, durable PostgreSQL implementation.
type ProofReplayStore interface {
	Consume(ctx context.Context, deviceID, purpose, requestID string, expiresAt time.Time) error
}

// InMemoryProofReplayStore exists only for deterministic unit tests. It is not durable,
// is lost on process restart, and must never be composed into a mounted runtime route.
type InMemoryProofReplayStore struct {
	mu   sync.Mutex
	now  func() time.Time
	used map[string]time.Time
}

// NewInMemoryProofReplayStore creates a test-only replay store. A nil clock uses UTC now.
func NewInMemoryProofReplayStore(now func() time.Time) *InMemoryProofReplayStore {
	if now == nil {
		now = time.Now
	}
	return &InMemoryProofReplayStore{now: now, used: make(map[string]time.Time)}
}

// Consume records a proof once through its expiry and rejects all duplicates.
func (s *InMemoryProofReplayStore) Consume(ctx context.Context, deviceID, purpose, requestID string, expiresAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.now == nil || s.used == nil {
		return errors.New("proof replay store is not configured")
	}
	if strings.TrimSpace(deviceID) == "" || strings.TrimSpace(purpose) == "" || strings.TrimSpace(requestID) == "" {
		return errors.New("proof replay key is incomplete")
	}
	now := s.now().UTC()
	if expiresAt.IsZero() || !expiresAt.After(now) {
		return ErrReplayExpired
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for key, expiry := range s.used {
		if !expiry.After(now) {
			delete(s.used, key)
		}
	}
	key := replayKey(deviceID, purpose, requestID)
	if _, exists := s.used[key]; exists {
		return ErrReplayAlreadyConsumed
	}
	s.used[key] = expiresAt.UTC()
	return nil
}

func replayKey(deviceID, purpose, requestID string) string {
	return deviceID + "\x00" + purpose + "\x00" + requestID
}
