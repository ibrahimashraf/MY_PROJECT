package eventbus

import (
	"context"
	"errors"
	"sync"

	"integin/internal/eventstore"
	"integin/internal/shared/events"
)

type Handler func(context.Context, eventstore.StoredEvent) error

type InProcessBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewInProcessBus() *InProcessBus { return &InProcessBus{handlers: make(map[string][]Handler)} }

func (b *InProcessBus) Subscribe(eventType string, handler Handler) error {
	if eventType == "" || handler == nil {
		return errors.New("event type and handler are required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	return nil
}

func (b *InProcessBus) Publish(ctx context.Context, stored eventstore.StoredEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[stored.Event.EventType()]...)
	b.mu.RUnlock()
	for _, handler := range handlers {
		if err := handler(ctx, stored); err != nil {
			return err
		}
	}
	return nil
}

type Committer struct {
	store eventstore.Store
	bus   *InProcessBus
}

func NewCommitter(store eventstore.Store, bus *InProcessBus) (*Committer, error) {
	if store == nil || bus == nil {
		return nil, errors.New("store and bus are required")
	}
	return &Committer{store: store, bus: bus}, nil
}

// AppendAndPublish writes to the event store before publishing. A publish
// failure is returned to the caller, but the stored event remains authoritative
// and can be retried or replayed by an operational worker.
func (c *Committer) AppendAndPublish(ctx context.Context, aggregateType, aggregateID string, expectedVersion int, event events.Envelope) (int, error) {
	version, err := c.store.Append(ctx, aggregateType, aggregateID, expectedVersion, event)
	if err != nil {
		return 0, err
	}
	if err := c.bus.Publish(ctx, eventstore.StoredEvent{AggregateType: aggregateType, AggregateID: aggregateID, AggregateVersion: version, Event: event}); err != nil {
		return version, err
	}
	return version, nil
}
