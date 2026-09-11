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
}

// NewBus constructs an initialized event bus.
func NewBus() *Bus {
	return &Bus{handlers: make(map[Topic][]Handler)}
}

// Subscribe registers a handler for a given topic.
func (b *Bus) Subscribe(topic Topic, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], h)
}

// Publish dispatches an event to all registered topic subscribers (synchronous).
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	handlers := b.handlers[e.Topic]
	b.mu.RUnlock()
	for _, h := range handlers {
		h(e)
	}
}

// PublishAsync dispatches event concurrently to all subscribers via goroutines.
func (b *Bus) PublishAsync(e Event) {
	b.mu.RLock()
	handlers := b.handlers[e.Topic]
	b.mu.RUnlock()
	var wg sync.WaitGroup
	for _, h := range handlers {
		wg.Add(1)
		go func(fn Handler) {
			defer wg.Done()
			fn(e)
		}(h)
	}
	wg.Wait()
}
