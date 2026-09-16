package scheduling

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"integin/pkg/engine/cognitive"
	"integin/pkg/engine/eventbus"
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

func TestHandleTelemetryAlertDeliberatesRemediation(t *testing.T) {
	agent := &cognitive.BDIAgent{ID: "tech-1"}

	if _, err := DispatchWorkOrder(agent, order("wo-routine", 0.8)); err != nil {
		t.Fatalf("routine dispatch failed: %v", err)
	}
	if agent.Intention == nil || agent.Intention.Action != "wo-routine" {
		t.Fatalf("expected routine order to hold the intention first, got %v", agent.Intention)
	}

	intent, err := HandleTelemetryAlert(agent, eventbus.TelemetryAlertEvent{
		AlertID:              "al-1",
		WorkOrderID:          "wo-emergency",
		CraneID:              "crane-7",
		Kind:                 "MOMENT_UTILIZATION_EXCEEDED",
		MomentUtilizationPct: 93.4,
		Message:              "moment utilization above statutory ceiling",
		Timestamp:            time.Now(),
	}, order("wo-emergency", 0.1))
	if err != nil {
		t.Fatalf("emergency remediation failed: %v", err)
	}
	if intent == nil || intent.Action != "wo-emergency" {
		t.Fatalf("expected emergency remediation intention to win over the routine desire, got %v", intent)
	}
	if agent.Intention != intent {
		t.Fatal("agent intention must follow the committed emergency remediation")
	}
	if intent.Urgency < 0.9 {
		t.Fatalf("emergency remediation must carry emergency-level urgency, got %v", intent.Urgency)
	}
	if !agentBelieves(agent, "alert:al-1") || !agentBelieves(agent, "hazard:MOMENT_UTILIZATION_EXCEEDED") {
		t.Fatal("alert provenance and hazard kind must be recorded as high-confidence beliefs")
	}
}

func TestHandleTelemetryAlertEnforcesCompetencyAndSoD(t *testing.T) {
	alert := eventbus.TelemetryAlertEvent{
		AlertID:     "al-1",
		WorkOrderID: "wo-1",
		CraneID:     "crane-7",
		Kind:        "ANTI_TWO_BLOCK_TRIGGERED",
		Timestamp:   time.Now(),
	}

	noCompetency := &cognitive.BDIAgent{ID: "tech-1"}
	incompetent := order("wo-1", 0.5)
	incompetent.Competency = expired()
	if _, err := HandleTelemetryAlert(noCompetency, alert, incompetent); !errors.Is(err, ErrDispatchIncompetent) {
		t.Fatalf("expired competency must fail closed, got %v", err)
	}
	if noCompetency.Intention != nil {
		t.Fatalf("no intention may form when competency is denied, got %v", noCompetency.Intention)
	}

	conflict := &cognitive.BDIAgent{ID: "tech-1"}
	taken := order("wo-1", 0.5)
	taken.AssignedTo = "tech-2"
	if _, err := HandleTelemetryAlert(conflict, alert, taken); !errors.Is(err, ErrDispatchSeparationOfDuties) {
		t.Fatalf("order held by another technician must fail closed, got %v", err)
	}
	if conflict.Intention != nil {
		t.Fatalf("no intention may form on an SoD conflict, got %v", conflict.Intention)
	}

	if _, err := HandleTelemetryAlert(&cognitive.BDIAgent{ID: "tech-1"}, eventbus.TelemetryAlertEvent{AlertID: "", WorkOrderID: "wo-1"}, order("wo-1", 0.5)); !errors.Is(err, ErrDispatchInvalidOrder) {
		t.Fatalf("incomplete alert must be rejected, got %v", err)
	}
	if _, err := HandleTelemetryAlert(nil, alert, order("wo-1", 0.5)); !errors.Is(err, ErrDispatchInvalidOrder) {
		t.Fatalf("nil agent must be rejected, got %v", err)
	}
}

func TestSubscribeAlertDispatcherEndToEnd(t *testing.T) {
	bus := eventbus.NewBus()
	defer bus.Close()
	agent := &cognitive.BDIAgent{ID: "tech-1"}

	resolver := func(ev eventbus.TelemetryAlertEvent) (WorkOrderCandidate, error) {
		c := order(ev.WorkOrderID, 0.4)
		c.Target = [3]float64{5, 6, 7}
		return c, nil
	}
	unsubscribe := SubscribeAlertDispatcher(bus, agent, resolver)

	bus.Publish(eventbus.Event{
		Topic: eventbus.TopicTelemetryAlert,
		Payload: eventbus.TelemetryAlertEvent{
			AlertID:     "al-e2e",
			WorkOrderID: "wo-remedy",
			CraneID:     "crane-9",
			Kind:        "ANTI_TWO_BLOCK_TRIGGERED",
			Message:     "two-block hazard",
			Timestamp:   time.Now(),
		},
	})

	if agent.Intention == nil || agent.Intention.Action != "wo-remedy" {
		t.Fatalf("published alert must form the remediation intention, got %v", agent.Intention)
	}
	if agent.Intention.Urgency != 1.0 {
		t.Fatalf("two-block hazard must carry top emergency priority, got %v", agent.Intention.Urgency)
	}
	if !agentBelieves(agent, "alert:al-e2e") || !agentBelieves(agent, "hazard:ANTI_TWO_BLOCK_TRIGGERED") {
		t.Fatal("published alert must be recorded as high-confidence beliefs")
	}

	unsubscribe()
	bus.Publish(eventbus.Event{
		Topic: eventbus.TopicTelemetryAlert,
		Payload: eventbus.TelemetryAlertEvent{
			AlertID: "al-post", WorkOrderID: "wo-late", Kind: "MOMENT_UTILIZATION_EXCEEDED", Timestamp: time.Now(),
		},
	})
	if agent.Intention.Action != "wo-remedy" {
		t.Fatalf("unsubscribed dispatcher must not alter the intention, got %v", agent.Intention.Action)
	}
}

func TestSubscribeEnvironmentalTelemetry(t *testing.T) {
	bus := eventbus.NewBus()
	defer bus.Close()
	agent := &cognitive.BDIAgent{ID: "tech-1"}

	var jobArgs river.JobArgs
	inserter := func(ctx context.Context, args river.JobArgs) error {
		jobArgs = args
		return nil
	}

	unsubscribe := SubscribeEnvironmentalTelemetry(bus, agent, inserter)
	defer unsubscribe()

	// 1. Wind Update
	bus.Publish(eventbus.Event{
		Topic: eventbus.TopicWindUpdate,
		Payload: eventbus.WindEvent{
			SpeedMps: 25.5,
		},
	})

	if !agentBelieves(agent, "wind_speed_mps:25.50") {
		t.Fatal("agent should believe high wind speed")
	}

	// 2. Structural Alert (Utilization > 90%)
	bus.Publish(eventbus.Event{
		Topic: eventbus.TopicStructuralAlert,
		Payload: eventbus.StressAlertEvent{
			ComponentID: "crane-leg-A",
			Utilization: 0.95, // exceeds 0.90
		},
	})

	if !agentBelieves(agent, "structural_alert:crane-leg-A") {
		t.Fatal("agent should believe structural alert")
	}

	if jobArgs == nil {
		t.Fatal("expected corrective work order job to be inserted via river queue")
	}
	correctiveArgs, ok := jobArgs.(CorrectiveWorkOrderArgs)
	if !ok || correctiveArgs.ComponentID != "crane-leg-A" {
		t.Fatalf("expected CorrectiveWorkOrderArgs for crane-leg-A, got %v", jobArgs)
	}
}
