package engine

import (
	"strings"
	"testing"
)

func TestSolve4DLiftSequence(t *testing.T) {
	c1 := CraneKinematics{
		BoomLengthMeters: 40.0,
		BoomAngleDeg:     65.0,
		SlewAngleDeg:     0.0,
	}
	c2 := CraneKinematics{
		BoomLengthMeters: 40.0,
		BoomAngleDeg:     60.0,
		SlewAngleDeg:     0.0,
	}
	load := MassSpec{
		MassTonnes: 40.0,
		LengthM:    10.0,
		RadiusM:    1.2,
	}

	t.Run("valid trajectory passes", func(t *testing.T) {
		stages := []LiftStage{
			{TimeS: 0, Crane1Angle: 65, Crane2Angle: 60, WindSpeedMPS: 5.0},
			{TimeS: 10, Crane1Angle: 63, Crane2Angle: 58, WindSpeedMPS: 6.2},
			{TimeS: 20, Crane1Angle: 60, Crane2Angle: 55, WindSpeedMPS: 5.8},
		}
		res, err := Solve4DLiftSequence(c1, c2, load, stages)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Passed {
			t.Fatalf("expected pass, got fail: %s at t=%v", res.FailReason, res.FailTimeS)
		}
	})

	t.Run("excessive wind speed fails closed", func(t *testing.T) {
		stages := []LiftStage{
			{TimeS: 0, Crane1Angle: 65, Crane2Angle: 60, WindSpeedMPS: 5.0},
			{TimeS: 10, Crane1Angle: 63, Crane2Angle: 58, WindSpeedMPS: 12.5}, // Exceeds 9.8 m/s
		}
		res, err := Solve4DLiftSequence(c1, c2, load, stages)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Passed {
			t.Fatal("expected failure due to weather wind speed limit")
		}
		if res.FailTimeS != 10 {
			t.Fatalf("expected failure at t=10, got t=%v", res.FailTimeS)
		}
		if !strings.Contains(res.FailReason, "weather limit exceeded") {
			t.Fatalf("expected weather limit error, got: %s", res.FailReason)
		}
	})

	t.Run("overload at stage fails closed with timestamp", func(t *testing.T) {
		heavyLoad := MassSpec{
			MassTonnes: 160.0, // Exceeds 75% capacity gate
			LengthM:    10.0,
			RadiusM:    1.2,
		}
		stages := []LiftStage{
			{TimeS: 0, Crane1Angle: 65, Crane2Angle: 60, WindSpeedMPS: 3.0},
		}
		res, err := Solve4DLiftSequence(c1, c2, heavyLoad, stages)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Passed {
			t.Fatal("expected failure due to overload")
		}
		if !strings.Contains(res.FailReason, "capacity limit exceeded") {
			t.Fatalf("expected capacity limit fail, got: %s", res.FailReason)
		}
	})
}
