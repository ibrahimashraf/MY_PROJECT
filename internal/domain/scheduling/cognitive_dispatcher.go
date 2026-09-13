package scheduling

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"integin/pkg/engine/cognitive"
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
	if agent == nil {
		return nil, fmt.Errorf("%w: nil agent", ErrDispatchInvalidOrder)
	}
	if order.ID == "" || order.Skill == "" {
		return nil, fmt.Errorf("%w: order %q missing id or skill", ErrDispatchInvalidOrder, order.ID)
	}

	dispatchMu.Lock()
	defer dispatchMu.Unlock()

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

func agentBelieves(a *cognitive.BDIAgent, proposition string) bool {
	for _, b := range a.Beliefs {
		if b.Proposition == proposition && b.Confidence >= beliefThreshold {
			return true
		}
	}
	return false
}
