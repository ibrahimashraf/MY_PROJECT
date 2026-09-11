package eventbus

import "sync"

// Topic is a string channel identifier for cross-domain events.
type Topic string

const (
	TopicWindUpdate         Topic = "physics.wind_update"
	TopicWildfireIgnition   Topic = "bio.wildfire_ignition"
	TopicStructuralAlert    Topic = "structural.stress_alert"
	TopicECSEntitySpawned   Topic = "ecs.entity_spawned"
	TopicAtmosphereChanged  Topic = "physics.atmosphere_changed"
	TopicGeologyErosion     Topic = "geology.erosion_step"
	TopicAgentGoalReached   Topic = "sim.agent_goal_reached"
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
	SpeedMps  float64
	DirectionX float64
	DirectionY float64
	AltitudeM float64
}

// StressAlertEvent carries structural overload notification.
type StressAlertEvent struct {
	ComponentID  string
	StressPa     float64
	AllowablePa  float64
	Utilization  float64
}

// Bus is the cross-domain publish/subscribe event dispatcher.
type Bus struct {
	mu       sync.RWMutex
	handlers map[Topic][]Handler
	eventCh  chan Event
	done     chan struct{}
}

// NewBus constructs an initialized event bus with a buffered async channel.
func NewBus() *Bus {
	b := &Bus{
		handlers: make(map[Topic][]Handler),
		eventCh:  make(chan Event, 256), // Buffered: publishers never block
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
			return
		}
	}
}

// Close shuts down the async dispatch goroutine.
func (b *Bus) Close() {
	close(b.done)
}

// Subscribe registers a handler for a given topic.
func (b *Bus) Subscribe(topic Topic, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], h)
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
// Handlers execute in the bus goroutine via channel select, not WaitGroup/goroutine fan-out.
func (b *Bus) PublishAsync(e Event) {
	select {
	case b.eventCh <- e:
	default:
		// Channel full (back-pressure): drop or handle in production via metrics
	}
}
