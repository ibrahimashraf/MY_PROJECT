package eventbus

import (
	"sync/atomic"
	"testing"
)

func TestEventBusPubSub(t *testing.T) {
	bus := NewBus()
	defer bus.Close()
	var count int64

	bus.Subscribe(TopicWindUpdate, func(e Event) {
		atomic.AddInt64(&count, 1)
	})
	bus.Subscribe(TopicWindUpdate, func(e Event) {
		atomic.AddInt64(&count, 1)
	})

	bus.Publish(Event{Topic: TopicWindUpdate, Payload: WindEvent{SpeedMps: 10.0}})
	if count != 2 {
		t.Fatalf("Expected 2 handler calls, got %d", count)
	}

	// Async dispatch: should succeed on empty channel
	if err := bus.PublishAsync(Event{Topic: TopicWildfireIgnition, Payload: nil}); err != nil {
		t.Fatalf("PublishAsync failed unexpectedly: %v", err)
	}
}

func TestEventBusPayloadTyping(t *testing.T) {
	bus := NewBus()
	var received *StressAlertEvent

	bus.Subscribe(TopicStructuralAlert, func(e Event) {
		if alert, ok := e.Payload.(StressAlertEvent); ok {
			received = &alert
		}
	})

	bus.Publish(Event{
		Topic: TopicStructuralAlert,
		Payload: StressAlertEvent{
			ComponentID: "boom-01", StressPa: 400e6, AllowablePa: 355e6, Utilization: 1.13,
		},
	})

	if received == nil || received.ComponentID != "boom-01" {
		t.Fatal("Typed event payload not received correctly")
	}
}
