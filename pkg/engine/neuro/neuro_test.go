package neuro

import (
	"math"
	"testing"
)

func TestHodgkinHuxleySpike(t *testing.T) {
	neuron := DefaultHodgkinHuxley()
	maxV := neuron.V
	// Apply 10 uA/cm^2 current for 50ms
	for i := 0; i < 500; i++ {
		neuron = neuron.Step(0.1, 10.0)
		if neuron.V > maxV {
			maxV = neuron.V
		}
	}
	// A spike should produce peak potential > 0mV (resting is -65mV)
	if maxV <= 0 {
		t.Fatalf("No action potential generated: max V = %v mV", maxV)
	}
}

func TestSTDPWeightUpdate(t *testing.T) {
	w := 0.5
	// Pre fires before post -> LTP (weight increase)
	wLTP := STDPWeightUpdate(w, 10.0, 0.01, 0.01, 20.0, 20.0)
	if wLTP <= w {
		t.Fatalf("LTP should increase weight: got %v from %v", wLTP, w)
	}

	// Post fires before pre -> LTD (weight decrease)
	wLTD := STDPWeightUpdate(w, -10.0, 0.01, 0.01, 20.0, 20.0)
	if wLTD >= w {
		t.Fatalf("LTD should decrease weight: got %v from %v", wLTD, w)
	}

	_ = math.NaN
}
