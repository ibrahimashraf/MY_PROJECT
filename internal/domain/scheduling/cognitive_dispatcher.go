package scheduling

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"integin/pkg/engine/cognitive"
	"integin/pkg/engine/eventbus"
)

const (
	beliefAssignedPrefix  = "assigned:"
	beliefCompetentPrefix = "competent:"
	beliefThreshold       = 0.5
)

var (
	// ErrDispatchInvalidOrder rejects malformed work order candidates.
	ErrDispatchInvalidOrder = errors.New("work order candidate is invalid")
	// ErrDispatchSeparationOfDuties denies dispatch when the agent is already
	// committed to the order or another technician holds the assignment.
	ErrDispatchSeparationOfDuties = errors.New("separation of duties blocks work order dispatch")
	// ErrDispatchIncompetent denies dispatch when the technician lacks a
	// CURRENT, unexpired competency for the order's skill.
	ErrDispatchIncompetent = errors.New("technician lacks current competency for work order")

	// dispatchMu serializes agent belief/desire mutation and intention
	// formation because DispatchWorkOrder mutates a shared BDIAgent.
	// ponytail: global lock, fine at scheduling volumes; per-agent locks if throughput matters.
	dispatchMu sync.Mutex
)

// WorkOrderCandidate is an open work order viewed as a desire whose priority
// weight drives BDI intention formation for a technician agent.
type WorkOrderCandidate struct {
	ID         string
	Skill      string
	Priority   float64
	Target     [3]float64 // world position target of the work order
	Competency TechnicianCompetency
	AssignedTo string // technician ID already committed to the order (SoD gate)
}

// DispatchWorkOrder forms a cognitive intention for the agent iff the order
// does not violate separation of duties and the agent holds a current
// competency for the order's skill. The formed desire is added to the agent's
// desire set so Deliberate always commits to the highest-priority open order.
func DispatchWorkOrder(agent *cognitive.BDIAgent, order WorkOrderCandidate) (*cognitive.Intention, error) {
	dispatchMu.Lock()
	defer dispatchMu.Unlock()
	return dispatchLocked(agent, order)
}

// dispatchLocked is the gate-enforcing core of DispatchWorkOrder. The caller
// must hold dispatchMu so belief/desire mutation stays serialized across the
// shared BDIAgent (Hazard 26).
func dispatchLocked(agent *cognitive.BDIAgent, order WorkOrderCandidate) (*cognitive.Intention, error) {
	if agent == nil {
		return nil, fmt.Errorf("%w: nil agent", ErrDispatchInvalidOrder)
	}
	if order.ID == "" || order.Skill == "" {
		return nil, fmt.Errorf("%w: order %q missing id or skill", ErrDispatchInvalidOrder, order.ID)
	}

	if agentBelieves(agent, beliefAssignedPrefix+order.ID) {
		return nil, fmt.Errorf("%w: technician %s is already assigned to %s", ErrDispatchSeparationOfDuties, agent.ID, order.ID)
	}
	if order.AssignedTo != "" && order.AssignedTo != agent.ID {
		return nil, fmt.Errorf("%w: order %s already assigned to technician %s", ErrDispatchSeparationOfDuties, order.ID, order.AssignedTo)
	}
	if !order.Competency.IsCurrent(time.Now()) {
		return nil, fmt.Errorf("%w: technician %s lacks current competency for skill %q", ErrDispatchIncompetent, agent.ID, order.Skill)
	}

	agent.UpdateBelief(beliefCompetentPrefix+order.Skill, 1.0)
	agent.UpdateBelief(beliefAssignedPrefix+order.ID, 1.0)
	agent.Desires = append(agent.Desires, cognitive.Desire{Goal: order.ID, Priority: order.Priority})
	agent.Deliberate()
	if agent.Intention != nil && agent.Intention.Action == order.ID {
		agent.Intention.Target = order.Target
	}
	return agent.Intention, nil
}

// emergencyPriority maps a sealed rule-gate verdict kind to the remediation
// desire weight. Static constants only — no runtime math (Hazard 7). The two
// statutory breach kinds outrank every routine work order desire.
func emergencyPriority(kind string) float64 {
	switch kind {
	case "ANTI_TWO_BLOCK_TRIGGERED":
		return 1.0
	case "MOMENT_UTILIZATION_EXCEEDED":
		return 0.95
	default:
		return 0.9
	}
}

// HandleTelemetryAlert folds a sealed SCADA/LMI critical alert into the agent
// as high-confidence beliefs, then commits it to the emergency remediation
// intention — but only after the same competency and separation-of-duties
// gates DispatchWorkOrder enforces. The alert provenance is recorded before
// the gates so the hazard belief holds even when this technician fails closed.
func HandleTelemetryAlert(agent *cognitive.BDIAgent, event eventbus.TelemetryAlertEvent, remedyOrder WorkOrderCandidate) (*cognitive.Intention, error) {
	if agent == nil {
		return nil, fmt.Errorf("%w: nil agent", ErrDispatchInvalidOrder)
	}
	if event.AlertID == "" || event.WorkOrderID == "" {
		return nil, fmt.Errorf("%w: alert %q incomplete (alert id or work order id missing)", ErrDispatchInvalidOrder, event.AlertID)
	}

	dispatchMu.Lock()
	defer dispatchMu.Unlock()

	agent.UpdateBelief("alert:"+event.AlertID, 1.0)
	agent.UpdateBelief("hazard:"+event.Kind, 1.0)

	// The remediation desire outranks routine work regardless of its nominal
	// priority: the statutory breach is the highest-priority open fact.
	remedyOrder.Priority = emergencyPriority(event.Kind)
	return dispatchLocked(agent, remedyOrder)
}

// SubscribeAlertDispatcher wires telemetry.alert events into the agent so a
// critical alert becomes a remediation intention. The candidateResolver maps a
// sealed alert to a concrete work order candidate; a failed resolve or a
// denied gate fails closed (no intention formed, no remediation dispatched).
// The returned closure unsubscribes the handler.
func SubscribeAlertDispatcher(bus *eventbus.Bus, agent *cognitive.BDIAgent, candidateResolver func(event eventbus.TelemetryAlertEvent) (WorkOrderCandidate, error)) func() {
	handler := func(e eventbus.Event) {
		alert, ok := e.Payload.(eventbus.TelemetryAlertEvent)
		if !ok {
			// Not a telemetry alert; nothing to deliberate on.
			return
		}
		remedy, err := candidateResolver(alert)
		if err != nil {
			// No resolvable remediation candidate — fail closed.
			return
		}
		if _, err := HandleTelemetryAlert(agent, alert, remedy); err != nil {
			// Gate denied (competency/SoD) — fail closed.
			return
		}
	}
	bus.Subscribe(eventbus.TopicTelemetryAlert, handler)
	return func() {
		bus.Unsubscribe(eventbus.TopicTelemetryAlert, handler)
	}
}

func agentBelieves(a *cognitive.BDIAgent, proposition string) bool {
	for _, b := range a.Beliefs {
		if b.Proposition == proposition && b.Confidence >= beliefThreshold {
			return true
		}
	}
	return false
}
