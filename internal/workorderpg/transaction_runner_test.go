package workorderpg

import (
	"context"
	"errors"
	"testing"

	"integin/internal/domain/workorder"
)

func TestWithinTransactionPassesRepositoryAndPreservesCallbackError(t *testing.T) {
	repository := &Repository{}
	want := errors.New("callback failed")
	called := false
	err := repository.WithinTransaction(context.Background(), workorder.ActorContext{}, func(_ context.Context, got workorder.Repository) error {
		called = true
		if got != repository {
			t.Fatal("transaction callback did not receive the repository-owned adapter")
		}
		return want
	})
	if !called {
		t.Fatal("transaction callback was not called")
	}
	if !errors.Is(err, want) {
		t.Fatalf("transaction callback error = %v, want %v", err, want)
	}
}

func TestWithinTransactionRejectsNilRepository(t *testing.T) {
	var repository *Repository
	err := repository.WithinTransaction(context.Background(), workorder.ActorContext{}, func(context.Context, workorder.Repository) error { return nil })
	if !errors.Is(err, ErrNilDB) {
		t.Fatalf("nil repository error = %v, want %v", err, ErrNilDB)
	}
}
