package eventbus

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

// Topic is a string channel identifier for cross-domain events.
type Topic string

const (
	TopicWindUpdate           Topic = "physics.wind_update"
	TopicWildfireIgnition     Topic = "bio.wildfire_ignition"
	TopicStructuralAlert      Topic = "structural.stress_alert"
	TopicECSEntitySpawned     Topic = "ecs.entity_spawned"
	TopicAtmosphereChanged    Topic = "physics.atmosphere_changed"
	TopicGeologyErosion       Topic = "geology.erosion_step"
	TopicAgentGoalReached     Topic = "sim.agent_goal_reached"
	TopicTelemetryAlert       Topic = "telemetry.alert"
	TopicWorkOrderRemediation Topic = "scheduling.remediation"
)

// Event is a typed payload published across engine systems.
type Event struct {
	Topic   Topic
	Payload interface{}
}

// Handler is a subscriber callback receiving events.
type Handler func(e Event)

// WindEvent carries atmospheric wind state.
type WindEvent struct {
	SpeedMps   float64
	DirectionX float64
	DirectionY float64
	AltitudeM  float64
}

// StressAlertEvent carries structural overload notification.
type StressAlertEvent struct {
	ComponentID string
	StressPa    float64
	AllowablePa float64
	Utilization float64
}

// TelemetryAlertEvent is a sealed, rule-gated SCADA/LMI safety alert bound to
// the exact work order its telemetry stream was ingested under.
type TelemetryAlertEvent struct {
	AlertID              string
	WorkOrderID          string
	CraneID              string
	Kind                 string
	MomentUtilizationPct float64
	Message              string
	Timestamp            time.Time
}

// Bus is the cross-domain publish/subscribe event dispatcher.
type Bus struct {
	mu           sync.RWMutex
	handlers     map[Topic][]Handler
	eventCh      chan Event
	done         chan struct{}
	DroppedCount uint64 // Atomic counter of back-pressure drops
}

// NewBus constructs an initialized event bus with a buffered async channel.
func NewBus() *Bus {
	b := &Bus{
		handlers: make(map[Topic][]Handler),
		eventCh:  make(chan Event, 1024),
		done:     make(chan struct{}),
	}
	go b.run()
	return b
}

// run is the single goroutine dispatching async events via channel — idiomatic Go.
func (b *Bus) run() {
	for {
		select {
		case e := <-b.eventCh:
			b.mu.RLock()
			handlers := make([]Handler, len(b.handlers[e.Topic]))
			copy(handlers, b.handlers[e.Topic])
			b.mu.RUnlock()
			for _, h := range handlers {
				h(e)
			}
		case <-b.done:
			// Drain remaining events before exit
			for {
				select {
				case e := <-b.eventCh:
					b.mu.RLock()
					handlers := make([]Handler, len(b.handlers[e.Topic]))
					copy(handlers, b.handlers[e.Topic])
					b.mu.RUnlock()
					for _, h := range handlers {
						h(e)
					}
				default:
					return
				}
			}
		}
	}
}

// Close gracefully shuts down the async dispatch goroutine, draining pending events.
func (b *Bus) Close() {
	close(b.done)
}

// Subscribe registers a handler for a given topic.
func (b *Bus) Subscribe(topic Topic, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], h)
}

// Unsubscribe removes a previously subscribed handler. A concurrent delivery
// already in flight on a copied handler slice may still complete once.
// ponytail: handler identity via reflect code pointer; distinct closures from
// one literal share the pointer, upgrade to opaque tokens if that ever matters.
func (b *Bus) Unsubscribe(topic Topic, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	hs := b.handlers[topic]
	for i, cur := range hs {
		if reflect.ValueOf(cur).Pointer() == reflect.ValueOf(h).Pointer() {
			b.handlers[topic] = append(hs[:i], hs[i+1:]...)
			return
		}
	}
}

// Publish dispatches an event synchronously to all registered topic subscribers.
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[e.Topic]))
	copy(handlers, b.handlers[e.Topic])
	b.mu.RUnlock()
	for _, h := range handlers {
		h(e)
	}
}

// PublishAsync sends event to the channel pipeline — non-blocking, idiomatic Go.
// Returns error and increments DroppedCount if channel is full (back-pressure signal).
func (b *Bus) PublishAsync(e Event) error {
	select {
	case b.eventCh <- e:
		return nil
	default:
		b.DroppedCount++
		return fmt.Errorf("eventbus: back-pressure on topic %q — event dropped (total dropped: %d)", e.Topic, b.DroppedCount)
	}
}
