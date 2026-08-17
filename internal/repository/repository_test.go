package repository

import (
	"context"
	"testing"
	"time"

	"integin/internal/eventstore"
	"integin/internal/shared/events"
)

type aggregate struct {
	Started  bool
	Findings int
}

func TestReplayRepositoryLoadsAggregateFromEvents(t *testing.T) {
	store := eventstore.NewInMemoryStore()
	ctx := context.Background()
	first, err := events.NewEnvelope("event-1", "InspectionStarted", "tenant-1", "org-1", "LIVE", "inspection", "inspection-1", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	second, err := events.NewEnvelope("event-2", "FindingRecorded", "tenant-1", "org-1", "LIVE", "inspection", "inspection-1", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Append(ctx, "inspection", "inspection-1", 0, first); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Append(ctx, "inspection", "inspection-1", 1, second); err != nil {
		t.Fatal(err)
	}
	repository, err := New[aggregate](store)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := repository.Load(ctx, "inspection", "inspection-1", aggregate{}, func(value *aggregate, event events.Envelope) error {
		if event.EventType() == "InspectionStarted" {
			value.Started = true
		}
		if event.EventType() == "FindingRecorded" {
			value.Findings++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Started || loaded.Findings != 1 {
		t.Fatalf("unexpected replayed aggregate: %#v", loaded)
	}
}
