package repository

import (
	"context"
	"errors"

	"integin/internal/eventstore"
	"integin/internal/shared/events"
)

type Reducer[T any] func(*T, events.Envelope) error

type ReplayRepository[T any] struct{ Store eventstore.Store }

func New[T any](store eventstore.Store) (*ReplayRepository[T], error) {
	if store == nil {
		return nil, errors.New("event store is required")
	}
	return &ReplayRepository[T]{Store: store}, nil
}

// Load reconstructs an aggregate by replaying its authoritative event stream
// in aggregate-version order. The reducer owns domain interpretation.
func (r *ReplayRepository[T]) Load(ctx context.Context, aggregateType, aggregateID string, initial T, reduce Reducer[T]) (T, error) {
	if reduce == nil {
		return initial, errors.New("aggregate reducer is required")
	}
	stream, err := r.Store.Load(ctx, aggregateType, aggregateID)
	if err != nil {
		return initial, err
	}
	aggregate := initial
	for _, stored := range stream {
		if err := reduce(&aggregate, stored.Event); err != nil {
			return initial, err
		}
	}
	return aggregate, nil
}
