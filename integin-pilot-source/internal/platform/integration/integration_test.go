package integration

import (
	"context"
	"errors"
	"testing"
)

type testAdapter struct{ calls int }

func (a *testAdapter) Name() string                 { return "test" }
func (a *testAdapter) Health(context.Context) error { return nil }
func (a *testAdapter) Execute(context.Context, []byte) ([]byte, error) {
	a.calls++
	return []byte("ok"), nil
}

type failingAdapter struct{}

func (failingAdapter) Name() string                 { return "failing" }
func (failingAdapter) Health(context.Context) error { return errors.New("down") }
func (failingAdapter) Execute(context.Context, []byte) ([]byte, error) {
	return nil, errors.New("failed")
}

func TestIntegrationRegistryAndIdempotentExecutor(t *testing.T) {
	registry := NewRegistry()
	adapter := &testAdapter{}
	if err := registry.Register(adapter); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(failingAdapter{}); err != nil {
		t.Fatal(err)
	}
	statuses := registry.Health(context.Background())
	if len(statuses) != 2 {
		t.Fatalf("expected two statuses: %#v", statuses)
	}
	executor := NewExecutor()
	first, err := executor.Execute(context.Background(), "op-1", adapter, []byte("payload"))
	if err != nil || string(first) != "ok" {
		t.Fatal(err)
	}
	second, err := executor.Execute(context.Background(), "op-1", adapter, []byte("changed"))
	if err != nil || string(second) != "ok" {
		t.Fatal(err)
	}
	if adapter.calls != 1 {
		t.Fatalf("expected idempotent execution, got %d calls", adapter.calls)
	}
}
