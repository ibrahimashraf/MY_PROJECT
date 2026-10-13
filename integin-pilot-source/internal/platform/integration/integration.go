package integration

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Adapter interface {
	Name() string
	Health(context.Context) error
	Execute(context.Context, []byte) ([]byte, error)
}
type Status struct {
	Name    string
	Healthy bool
	Error   string
}
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

func NewRegistry() *Registry { return &Registry{adapters: make(map[string]Adapter)} }
func (r *Registry) Register(adapter Adapter) error {
	if adapter == nil || adapter.Name() == "" {
		return errors.New("adapter and name are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.Name()] = adapter
	return nil
}
func (r *Registry) Health(ctx context.Context) []Status {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Status, 0, len(r.adapters))
	for name, adapter := range r.adapters {
		status := Status{Name: name, Healthy: true}
		if err := adapter.Health(ctx); err != nil {
			status.Healthy = false
			status.Error = err.Error()
		}
		result = append(result, status)
	}
	return result
}

type Executor struct {
	mu      sync.Mutex
	results map[string][]byte
}

func NewExecutor() *Executor { return &Executor{results: make(map[string][]byte)} }
func (e *Executor) Execute(ctx context.Context, operationID string, adapter Adapter, payload []byte) ([]byte, error) {
	if operationID == "" || adapter == nil {
		return nil, errors.New("operation id and adapter are required")
	}
	e.mu.Lock()
	if result, ok := e.results[operationID]; ok {
		copied := append([]byte(nil), result...)
		e.mu.Unlock()
		return copied, nil
	}
	e.mu.Unlock()
	result, err := adapter.Execute(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("integration operation %s failed: %w", operationID, err)
	}
	e.mu.Lock()
	e.results[operationID] = append([]byte(nil), result...)
	e.mu.Unlock()
	return append([]byte(nil), result...), nil
}
