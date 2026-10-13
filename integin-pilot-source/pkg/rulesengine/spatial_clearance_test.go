package rulesengine_test

import (
	"math"
	"testing"

	"integin/pkg/rulesengine"
)

func TestSpatialClearanceGate(t *testing.T) {
	// Crane tip at (10, 10, 25), energized line at (10, 10, 29) -> distance = 4.0 m
	craneTip := rulesengine.Point3D{X: 10, Y: 10, Z: 25}
	powerline := rulesengine.Point3D{X: 10, Y: 10, Z: 29}

	res, err := rulesengine.CheckPointClearance(craneTip, powerline, rulesengine.OSHAMinPowerlineClearance50kVM)
	if err != nil {
		t.Fatalf("CheckPointClearance failed: %v", err)
	}
	if math.Abs(res.ActualDistanceM-4.0) > 1e-6 {
		t.Errorf("expected 4.0m distance, got %.4f", res.ActualDistanceM)
	}
	if !res.Passed {
		t.Errorf("4.0m distance >= 3.048m must pass OSHA Table A gate")
	}

	// Move crane tip closer: (10, 10, 27) -> distance = 2.0 m < 3.048 m -> MUST FAIL
	craneTipClose := rulesengine.Point3D{X: 10, Y: 10, Z: 27}
	resClose, err := rulesengine.CheckPointClearance(craneTipClose, powerline, rulesengine.OSHAMinPowerlineClearance50kVM)
	if err != nil {
		t.Fatalf("CheckPointClearance failed: %v", err)
	}
	if resClose.Passed {
		t.Errorf("2.0m distance must fail OSHA 10ft requirement")
	}
}
