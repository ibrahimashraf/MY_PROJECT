package pgtx

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrNilDB        = errors.New("database handle is required")
	ErrInvalidScope = errors.New("tenant and organization scope are required")
	ErrCircuitOpen  = errors.New("database circuit breaker open: requests failing fast")
)

type CircuitBreaker struct {
	maxFailures uint32
	cooldown    time.Duration
	state       atomic.Uint32
	failCount   atomic.Uint32
	lastFailure atomic.Int64
	mu          sync.Mutex
}

var defaultBreaker = NewCircuitBreaker(5, 3*time.Second)

func DefaultBreaker() *CircuitBreaker {
	return defaultBreaker
}

func NewCircuitBreaker(maxFailures uint32, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		cooldown:    cooldown,
	}
}

func (cb *CircuitBreaker) Allow() error {
	if cb.state.Load() == 0 { // Closed
		return nil
	}
	last := time.Unix(0, cb.lastFailure.Load())
	if time.Since(last) > cb.cooldown {
		cb.mu.Lock()
		defer cb.mu.Unlock()
		if cb.state.Load() == 2 { // Open -> HalfOpen
			cb.state.Store(1)
			return nil
		}
	}
	if cb.state.Load() == 1 { // HalfOpen
		return nil
	}
	return ErrCircuitOpen
}

func (cb *CircuitBreaker) RecordSuccess() {
	if cb.state.Load() != 0 {
		cb.mu.Lock()
		cb.failCount.Store(0)
		cb.state.Store(0)
		cb.mu.Unlock()
	} else {
		cb.failCount.Store(0)
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.lastFailure.Store(time.Now().UnixNano())
	if cb.failCount.Add(1) >= cb.maxFailures {
		cb.mu.Lock()
		cb.state.Store(2) // Open
		cb.mu.Unlock()
	}
}

// BeginScope begins a new transaction on db and configures the PostgreSQL session
// variables 'integin.tenant_id' and 'integin.organization_id' with local transaction scope (is_local = true).
// If any step fails, the transaction is safely rolled back and the error is returned.
func BeginScope(ctx context.Context, db *sql.DB, tenantID, organizationID string) (*sql.Tx, error) {
	if err := defaultBreaker.Allow(); err != nil {
		return nil, err
	}
	if db == nil {
		return nil, ErrNilDB
	}
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(organizationID) == "" {
		return nil, ErrInvalidScope
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		defaultBreaker.RecordFailure()
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`,
		tenantID, organizationID,
	); err != nil {
		_ = tx.Rollback()
		defaultBreaker.RecordFailure()
		return nil, err
	}
	defaultBreaker.RecordSuccess()
	return tx, nil
}
