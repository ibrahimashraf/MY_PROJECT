package scheduling

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"integin/pkg/engine/cognitive"
)

func competent() TechnicianCompetency {
	return TechnicianCompetency{Status: CompetencyStatusCurrent}
}

func expired() TechnicianCompetency {
	return TechnicianCompetency{Status: CompetencyStatusCurrent, ExpiresAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func order(id string, priority float64) WorkOrderCandidate {
	return WorkOrderCandidate{
		ID:         id,
		Skill:      "lifting.inspection",
		Priority:   priority,
		Target:     [3]float64{10, 20, 30},
		Competency: competent(),
	}
}

func TestDispatchWorkOrderDeliberatesHighestPriority(t *testing.T) {
	agent := &cognitive.BDIAgent{ID: "tech-1", Position: [3]float64{1, 2, 3}}

	low, err := DispatchWorkOrder(agent, order("wo-low", 0.4))
	if err != nil {
		t.Fatalf("low-priority dispatch failed: %v", err)
	}
	if low == nil || low.Action != "wo-low" {
		t.Fatalf("expected sole desire to win, got intention %v", low)
	}

	high, err := DispatchWorkOrder(agent, order("wo-high", 0.9))
	if err != nil {
		t.Fatalf("high-priority dispatch failed: %v", err)
	}
	if high == nil || high.Action != "wo-high" {
		t.Fatalf("expected highest-priority order to win, got %v", high)
	}
	if high.Target != ([3]float64{10, 20, 30}) {
		t.Fatalf("winning intention must carry the order target, got %v", high.Target)
	}
	if agent.Intention != high {
		t.Fatal("agent intention must reflect the dispatched desire")
	}
}

func TestDispatchWorkOrderExcludesIncompetentTechnician(t *testing.T) {
	agent := &cognitive.BDIAgent{ID: "tech-1"}

	cases := []struct {
		name string
		comp TechnicianCompetency
	}{
		{"expired competency", expired()},
		{"missing competency", TechnicianCompetency{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := order("wo-1", 0.8)
			candidate.Competency = tc.comp
			_, err := DispatchWorkOrder(agent, candidate)
			if !errors.Is(err, ErrDispatchIncompetent) {
				t.Fatalf("expected ErrDispatchIncompetent, got %v", err)
			}
			if agent.Intention != nil {
				t.Fatalf("no intention may form for an incompetent technician, got %v", agent.Intention)
			}
		})
	}
}

func TestDispatchWorkOrderExcludesConflict(t *testing.T) {
	conflicted := &cognitive.BDIAgent{ID: "tech-1"}
	conflicted.UpdateBelief(beliefAssignedPrefix+"wo-1", 1.0)
	_, err := DispatchWorkOrder(conflicted, order("wo-1", 0.8))
	if !errors.Is(err, ErrDispatchSeparationOfDuties) {
		t.Fatalf("already-assigned belief must block dispatch, got %v", err)
	}

	agent := &cognitive.BDIAgent{ID: "tech-1"}
	taken := order("wo-2", 0.8)
	taken.AssignedTo = "tech-2"
	_, err = DispatchWorkOrder(agent, taken)
	if !errors.Is(err, ErrDispatchSeparationOfDuties) {
		t.Fatalf("order held by another technician must block dispatch, got %v", err)
	}
}

func TestDispatchWorkOrderRejectsInvalidCandidate(t *testing.T) {
	agent := &cognitive.BDIAgent{ID: "tech-1"}

	if _, err := DispatchWorkOrder(agent, WorkOrderCandidate{Skill: "lifting.inspection"}); !errors.Is(err, ErrDispatchInvalidOrder) {
		t.Fatalf("missing order id must be rejected, got %v", err)
	}
	if _, err := DispatchWorkOrder(agent, WorkOrderCandidate{ID: "wo-1"}); !errors.Is(err, ErrDispatchInvalidOrder) {
		t.Fatalf("missing skill must be rejected, got %v", err)
	}
	if _, err := DispatchWorkOrder(nil, order("wo-1", 0.5)); !errors.Is(err, ErrDispatchInvalidOrder) {
		t.Fatalf("nil agent must be rejected, got %v", err)
	}
}

func TestDispatchWorkOrderConcurrentDispatchIsRaceFree(t *testing.T) {
	agent := &cognitive.BDIAgent{ID: "tech-conc"}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("wo-%d", i)
			if _, err := DispatchWorkOrder(agent, order(id, float64(i)/32)); err != nil {
				t.Errorf("dispatch of %s failed: %v", id, err)
			}
		}(i)
	}
	wg.Wait()

	if len(agent.Desires) != 32 {
		t.Fatalf("expected 32 desires recorded, got %d", len(agent.Desires))
	}
	if got := agent.Intention; got == nil || got.Action != "wo-31" {
		t.Fatalf("highest-priority concurrent order must win, got %v", got)
	}
}
