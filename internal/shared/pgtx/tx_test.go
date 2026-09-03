package pgtx

import (
	"context"
	"testing"
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
