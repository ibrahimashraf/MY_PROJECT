package pgtx

import (
	"context"
	"testing"
	"time"
)

func TestBeginScope_Validations(t *testing.T) {
	ctx := context.Background()

	// Nil DB
	if _, err := BeginScope(ctx, nil, "tenant-1", "org-1"); err != ErrNilDB {
		t.Errorf("expected ErrNilDB, got %v", err)
	}

	// Empty tenant
	if _, err := BeginScope(ctx, nil, "", "org-1"); err != ErrNilDB {
		t.Errorf("expected ErrNilDB check before scope, got %v", err)
	}
}

func TestCircuitBreaker_TrippingAndRecovery(t *testing.T) {
	cb := NewCircuitBreaker(3, 50*time.Millisecond)

	// Initially closed
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected closed, got %v", err)
	}

	// Record 3 failures
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Now open
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Half-open allows 1 trial
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected trial allowed, got %v", err)
	}

	// Success restores closed
	cb.RecordSuccess()
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected closed after success, got %v", err)
	}
}

