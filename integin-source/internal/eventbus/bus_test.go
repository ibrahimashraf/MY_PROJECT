package eventbus

import (
	"context"
	"testing"
	"time"

	"integin/internal/eventstore"
	"integin/internal/shared/events"
)

func busEvent(t *testing.T, id string) events.Envelope {
	t.Helper()
	event, err := events.NewEnvelope(id, "InspectionStarted", "tenant-1", "org-1", "LIVE", "inspection", "inspection-1", map[string]string{"id": id}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func TestBusPublishesToSubscribers(t *testing.T) {
	bus := NewInProcessBus()
	received := make(chan int, 1)
	if err := bus.Subscribe("InspectionStarted", func(ctx context.Context, stored eventstore.StoredEvent) error {
		received <- stored.AggregateVersion
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish(context.Background(), eventstore.StoredEvent{AggregateType: "inspection", AggregateID: "inspection-1", AggregateVersion: 1, Event: busEvent(t, "event-1")}); err != nil {
		t.Fatal(err)
	}
	if version := <-received; version != 1 {
		t.Fatalf("expected version 1, got %d", version)
	}
}

func TestCommitterStoresBeforePublishingAndRejectsStaleWrites(t *testing.T) {
	store := eventstore.NewInMemoryStore()
	bus := NewInProcessBus()
	published := 0
	if err := bus.Subscribe("InspectionStarted", func(ctx context.Context, stored eventstore.StoredEvent) error { published++; return nil }); err != nil {
		t.Fatal(err)
	}
	committer, err := NewCommitter(store, bus)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := committer.AppendAndPublish(ctx, "inspection", "inspection-1", 0, busEvent(t, "event-1")); err != nil {
		t.Fatal(err)
	}
	if published != 1 {
		t.Fatalf("expected one published event, got %d", published)
	}
	if _, err := committer.AppendAndPublish(ctx, "inspection", "inspection-1", 0, busEvent(t, "event-2")); err != eventstore.ErrOptimisticConcurrency {
		t.Fatalf("expected stale write rejection, got %v", err)
	}
	if published != 1 {
		t.Fatal("stale write must not publish")
	}
	replay, err := store.Load(ctx, "inspection", "inspection-1")
	if err != nil || len(replay) != 1 {
		t.Fatalf("expected one authoritative stored event: %v %#v", err, replay)
	}
}
