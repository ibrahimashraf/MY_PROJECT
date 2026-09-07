package eventstore

import (
	"context"
	"testing"
	"time"

	"integin/internal/shared/events"
)

func testEvent(t *testing.T, id, eventType, aggregateType, aggregateID string) events.Envelope {
	t.Helper()
	event, err := events.NewEnvelope(id, eventType, "tenant-1", "org-1", "LIVE", aggregateType, aggregateID, map[string]string{"value": id}, time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func TestInMemoryStoreAppendsAndReplaysInOrder(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()
	first := testEvent(t, "event-1", "InspectionStarted", "inspection", "inspection-1")
	second := testEvent(t, "event-2", "FindingRecorded", "inspection", "inspection-1")
	if version, err := store.Append(ctx, "inspection", "inspection-1", 0, first); err != nil || version != 1 {
		t.Fatalf("first append: %d %v", version, err)
	}
	if version, err := store.Append(ctx, "inspection", "inspection-1", 1, second); err != nil || version != 2 {
		t.Fatalf("second append: %d %v", version, err)
	}
	replay, err := store.Load(ctx, "inspection", "inspection-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(replay) != 2 || replay[0].AggregateVersion != 1 || replay[1].Event.EventType() != "FindingRecorded" {
		t.Fatalf("unexpected replay: %#v", replay)
	}
}

func TestInMemoryStoreRejectsStaleVersionAndWrongAggregate(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()
	event := testEvent(t, "event-1", "InspectionStarted", "inspection", "inspection-1")
	if _, err := store.Append(ctx, "inspection", "inspection-1", 1, event); err != ErrOptimisticConcurrency {
		t.Fatalf("expected concurrency error, got %v", err)
	}
	wrong := testEvent(t, "event-2", "InspectionStarted", "inspection", "inspection-2")
	if _, err := store.Append(ctx, "inspection", "inspection-1", 0, wrong); err == nil {
		t.Fatal("expected aggregate identity error")
	}
}
