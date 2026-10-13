package cognitive

import (
	"testing"
)

func TestBDIAgentDeliberation(t *testing.T) {
	agent := BDIAgent{ID: "inspector-01"}
	agent.UpdateBelief("structure_safe", 0.9)
	agent.Desires = []Desire{
		{Goal: "InspectBeam", Priority: 0.6},
		{Goal: "EmergencyEvacuate", Priority: 0.95},
	}
	agent.Deliberate()
	if agent.Intention == nil {
		t.Fatal("BDI agent failed to form intention")
	}
	if agent.Intention.Action != "EmergencyEvacuate" {
		t.Fatalf("Expected highest priority goal, got %s", agent.Intention.Action)
	}
}

func TestQLearningConvergence(t *testing.T) {
	agent := NewQLearningAgent(4, 0.1, 0.9, 0.0) // greedy (epsilon=0)
	goal := [2]int{5, 5}

	// Train 100 episodes from (0,0)
	for ep := 0; ep < 100; ep++ {
		state := [2]int{0, 0}
		for step := 0; step < 20; step++ {
			action := agent.SelectAction(state, 0.01) // low random
			next := [2]int{state[0] + action/2, state[1] + action%2}
			reward := RewardGradient(next, goal)
			agent.Update(state, action, reward, next)
			state = next
		}
	}
	// After training, Q-values at (0,0) should be non-zero
	qs := agent.getQ([2]int{0, 0})
	nonZero := false
	for _, v := range qs {
		if v != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Fatal("Q-learning failed to update Q-values after training")
	}
}
