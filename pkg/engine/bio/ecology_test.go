package bio

import (
	"math"
	"testing"
)

func TestLotkaVolterraOscillation(t *testing.T) {
	params := LotkaVolterraParams{Alpha: 0.1, Beta: 0.02, Delta: 0.01, Gamma: 0.1}
	state := LotkaVolterraState{Prey: 40.0, Predator: 9.0}

	initialPrey := state.Prey
	for i := 0; i < 1000; i++ {
		state = state.Step(params, 0.1)
	}

	if state.Prey <= 0 || state.Predator <= 0 {
		t.Fatal("Population went extinct unexpectedly")
	}
	if math.IsNaN(state.Prey) || math.IsNaN(state.Predator) {
		t.Fatal("NaN in Lotka-Volterra system")
	}
	_ = initialPrey
}

func TestWildfireSpread(t *testing.T) {
	g := NewWildfireGrid(20, 20, 0.4, 0.0, 10, 10)
	burnedBefore := g.BurnedCount()

	for i := 0; i < 5; i++ {
		g.Step(0.8)
	}

	burnedAfter := g.BurnedCount()
	if burnedAfter <= burnedBefore {
		t.Fatal("Wildfire should spread after 5 steps")
	}
}
