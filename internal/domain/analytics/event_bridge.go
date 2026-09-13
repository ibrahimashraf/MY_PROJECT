package analytics

import (
	"math"
	"sync"

	"integin/pkg/engine/eventbus"
)

const maxAlerts = 50

// StructuralAlert is the in-memory structural overload notification surfaced by the EventBridge.
type StructuralAlert struct {
	ComponentID string  `json:"component_id"`
	StressPa    float64 `json:"stress_pa"`
	AllowablePa float64 `json:"allowable_pa"`
	Utilization float64 `json:"utilization"`
}

// EventBridge bridges the engine event bus into the analytics dashboard,
// updating in-memory metrics and alerts in near real-time without DB polling.
type EventBridge struct {
	mu     sync.RWMutex
	kpis   map[string]KPI
	alerts []StructuralAlert
	wind   struct {
		speedMps   float64
		directionX float64
		directionY float64
		altitudeM  float64
		set        bool
	}
}

// NewEventBridge subscribes to wind and structural alert topics and returns a thread-safe bridge.
func NewEventBridge(bus *eventbus.Bus) *EventBridge {
	br := &EventBridge{kpis: make(map[string]KPI)}
	bus.Subscribe(eventbus.TopicWindUpdate, func(e eventbus.Event) {
		if we, ok := e.Payload.(eventbus.WindEvent); ok {
			br.applyWind(we)
		}
	})
	bus.Subscribe(eventbus.TopicStructuralAlert, func(e eventbus.Event) {
		if se, ok := e.Payload.(eventbus.StressAlertEvent); ok {
			br.applyStress(se)
		}
	})
	return br
}

func (b *EventBridge) applyWind(we eventbus.WindEvent) {
	if !finite(we.SpeedMps) {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.wind.speedMps = we.SpeedMps
	b.wind.directionX = we.DirectionX
	b.wind.directionY = we.DirectionY
	b.wind.altitudeM = we.AltitudeM
	b.wind.set = true
	b.kpis["wind_speed"] = KPI{Name: "wind_speed", Value: we.SpeedMps, Unit: "m/s", SIUnit: SIUnitMetersPerSec}
}

func (b *EventBridge) applyStress(se eventbus.StressAlertEvent) {
	if !finite(se.StressPa) {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.kpis["component_stress"] = KPI{Name: "component_stress", Value: se.StressPa, Unit: "Pa", SIUnit: SIUnitPascals, Trend: TrendUp}
	if len(b.alerts) == maxAlerts {
		copy(b.alerts, b.alerts[1:])
		b.alerts = b.alerts[:maxAlerts-1]
	}
	b.alerts = append(b.alerts, StructuralAlert{
		ComponentID: se.ComponentID,
		StressPa:    se.StressPa,
		AllowablePa: se.AllowablePa,
		Utilization: se.Utilization,
	})
}

// KPIs returns the current in-memory metrics as a snapshot slice.
func (b *EventBridge) KPIs() []KPI {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]KPI, 0, len(b.kpis))
	for _, k := range b.kpis {
		out = append(out, k)
	}
	return out
}

// Alerts returns the most recent in-memory structural alerts.
func (b *EventBridge) Alerts() []StructuralAlert {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return append([]StructuralAlert(nil), b.alerts...)
}

// LatestWind returns the most recent wind update, if one has been received.
func (b *EventBridge) LatestWind() (eventbus.WindEvent, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if !b.wind.set {
		return eventbus.WindEvent{}, false
	}
	return eventbus.WindEvent{
		SpeedMps:   b.wind.speedMps,
		DirectionX: b.wind.directionX,
		DirectionY: b.wind.directionY,
		AltitudeM:  b.wind.altitudeM,
	}, true
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
