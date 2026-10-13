package analytics

import (
	"math"
	"sync"
	"testing"

	"integin/pkg/engine/eventbus"
)

func TestEventBridgeWindUpdate(t *testing.T) {
	bus := eventbus.NewBus()
	defer bus.Close()
	br := NewEventBridge(bus)

	bus.Publish(eventbus.Event{
		Topic:   eventbus.TopicWindUpdate,
		Payload: eventbus.WindEvent{SpeedMps: 10.5, DirectionX: 1, DirectionY: 0, AltitudeM: 30},
	})

	kpis := br.KPIs()
	if len(kpis) != 1 {
		t.Fatalf("expected 1 KPI, got %d", len(kpis))
	}
	k := kpis[0]
	if k.Name != "wind_speed" || k.SIUnit != SIUnitMetersPerSec || k.Unit != "m/s" {
		t.Fatalf("unexpected wind KPI: %+v", k)
	}

	wind, ok := br.LatestWind()
	if !ok {
		t.Fatal("expected wind update to be present")
	}
	if wind.SpeedMps != 10.5 || wind.AltitudeM != 30 {
		t.Fatalf("unexpected wind: %+v", wind)
	}
}

func TestEventBridgeStructuralAlert(t *testing.T) {
	bus := eventbus.NewBus()
	defer bus.Close()
	br := NewEventBridge(bus)

	bus.Publish(eventbus.Event{
		Topic: eventbus.TopicStructuralAlert,
		Payload: eventbus.StressAlertEvent{
			ComponentID: "mast-01", StressPa: 3.1e8, AllowablePa: 2.8e8, Utilization: 1.1,
		},
	})

	alerts := br.Alerts()
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].ComponentID != "mast-01" || alerts[0].StressPa != 3.1e8 {
		t.Fatalf("unexpected alert: %+v", alerts[0])
	}

	kpis := br.KPIs()
	found := false
	for _, k := range kpis {
		if k.Name == "component_stress" && k.SIUnit == SIUnitPascals {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected component_stress KPI in Pascals, got %+v", kpis)
	}
}

func TestEventBridgeIgnoresNonFiniteWind(t *testing.T) {
	bus := eventbus.NewBus()
	defer bus.Close()
	br := NewEventBridge(bus)

	bus.Publish(eventbus.Event{
		Topic:   eventbus.TopicWindUpdate,
		Payload: eventbus.WindEvent{SpeedMps: math.Inf(1)},
	})
	bus.Publish(eventbus.Event{
		Topic:   eventbus.TopicWindUpdate,
		Payload: eventbus.WindEvent{SpeedMps: math.NaN()},
	})

	if _, ok := br.LatestWind(); ok {
		t.Fatal("expected non-finite wind to be skipped")
	}
}

func TestEventBridgeBoundedAlerts(t *testing.T) {
	bus := eventbus.NewBus()
	defer bus.Close()
	br := NewEventBridge(bus)

	for i := 0; i < maxAlerts+10; i++ {
		bus.Publish(eventbus.Event{
			Topic: eventbus.TopicStructuralAlert,
			Payload: eventbus.StressAlertEvent{
				ComponentID: string(rune('A' + i%26)),
			},
		})
	}

	alerts := br.Alerts()
	if len(alerts) != maxAlerts {
		t.Fatalf("expected %d alerts, got %d", maxAlerts, len(alerts))
	}
}

func TestEventBridgeConcurrentAccess(t *testing.T) {
	bus := eventbus.NewBus()
	defer bus.Close()
	br := NewEventBridge(bus)

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(eventbus.Event{
				Topic:   eventbus.TopicWindUpdate,
				Payload: eventbus.WindEvent{SpeedMps: 5, DirectionX: 1, DirectionY: 0, AltitudeM: 10},
			})
			bus.Publish(eventbus.Event{
				Topic: eventbus.TopicStructuralAlert,
				Payload: eventbus.StressAlertEvent{
					ComponentID: "c", StressPa: 1e8, AllowablePa: 9e7, Utilization: 1.11,
				},
			})
			_ = br.KPIs()
			_ = br.Alerts()
			_, _ = br.LatestWind()
		}()
	}
	wg.Wait()
}
