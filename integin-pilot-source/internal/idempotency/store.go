// Package idempotency provides a Stripe-grade idempotency guard for the sync
// and evidence ingress endpoints. An Idempotency-Key is enforced as mandatory,
// serialized through a PostgreSQL advisory lock plus a durable row lock, and
// completed responses are replayed from the sync_idempotency_cache table
// without re-executing the downstream handler.
package idempotency

import (
	"context"
	"errors"
)

// Scope identifies the idempotency domain for a single request. The endpoint
// and key_hash together scope a key inside a tenant+organization, mirroring
// Stripe's account-scoped idempotency model.
type Scope struct {
	TenantID       string
	OrganizationID string
	Endpoint       string
	KeyHash        string
	RequestHash    string
}

// Cached is a previously committed response that can be replayed verbatim.
type Cached struct {
	Status     string
	HTTPStatus int
	Payload    []byte
}

// Claim is a held lock on an idempotency key. Exactly one request owns the
// claim for a given key at a time; the winner executes the downstream handler
// and the losers replay (or conflict against) the winner's committed outcome.
type Claim interface {
	// Commit finalizes the response for the claimed key. A status with a
	// persisted HTTP status of 5xx is intentionally not cached, matching
	// Stripe's contract of never caching server-side failures.
	Commit(ctx context.Context, status string, httpStatus int, payload []byte) error
	// Done releases the claim without recording a response.
	Done()
}

// ErrConflict is returned when an idempotency key is reused with a different
// request body within the caching window.
var ErrConflict = errors.New("idempotency key reused with a different request")

// Store acquires idempotency claims. Implementations must be linearizable for
// identical (tenant, organization, endpoint, key_hash) tuples: concurrent
// Acquisitions for the same key must result in exactly one leader.
type Store interface {
	// Acquire returns (claim, nil, nil) for the leader, (nil, cached, nil) for
	// a replayable committed response, (nil, nil, ErrConflict) when the key was
	// reused with a different request, and (nil, nil, err) on failure.
	Acquire(ctx context.Context, scope Scope) (Claim, *Cached, error)
}
