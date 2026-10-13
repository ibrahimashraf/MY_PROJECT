package engine

import (
	"fmt"
)

type MassSpec struct {
	MassTonnes float64 `json:"mass_tonnes"`
	LengthM    float64 `json:"length_m"`
	RadiusM    float64 `json:"radius_m"`
}

type LiftStage struct {
	TimeS        float64 `json:"time_s"`
	Crane1Angle  float64 `json:"crane1_angle"`
	Crane2Angle  float64 `json:"crane2_angle"`
	Crane1Slew   float64 `json:"crane1_slew"`
	Crane2Slew   float64 `json:"crane2_slew"`
	WindSpeedMPS float64 `json:"wind_speed_mps,omitempty"`
}

type Trajectory4DResult struct {
	Passed     bool    `json:"passed"`
	FailReason string  `json:"fail_reason,omitempty"`
	FailTimeS  float64 `json:"fail_time_s,omitempty"`
}

const (
	// MaxTandemWindSpeedMPS defines the operational wind limit for tandem lifts (ASME B30.5 / BS 7121: 9.8 m/s ~ 20 knots).
	MaxTandemWindSpeedMPS = 9.8
)

func Solve4DLiftSequence(c1, c2 CraneKinematics, load MassSpec, stages []LiftStage) (*Trajectory4DResult, error) {
	for _, stage := range stages {
		if stage.WindSpeedMPS > MaxTandemWindSpeedMPS {
			return &Trajectory4DResult{
				Passed:     false,
				FailReason: fmt.Sprintf("weather limit exceeded: wind speed %.1f m/s exceeds max permissible %.1f m/s (ASME B30.5 / BS 7121 limit)", stage.WindSpeedMPS, MaxTandemWindSpeedMPS),
				FailTimeS:  stage.TimeS,
			}, nil
		}

		stepC1 := c1
		stepC1.BoomAngleDeg = stage.Crane1Angle
		stepC1.SlewAngleDeg = stage.Crane1Slew

		stepC2 := c2
		stepC2.BoomAngleDeg = stage.Crane2Angle
		stepC2.SlewAngleDeg = stage.Crane2Slew

		res, err := SolveTandemLift(stepC1, stepC2, load.MassTonnes, 0)
		if err != nil {
			return &Trajectory4DResult{Passed: false, FailReason: fmt.Sprintf("solve error at %v: %v", stage.TimeS, err), FailTimeS: stage.TimeS}, nil
		}

		capacity1 := 100.0
		capacity2 := 100.0

		if res.Crane1LoadTonnes > capacity1*0.75 {
			return &Trajectory4DResult{Passed: false, FailReason: "capacity limit exceeded on crane 1 (ASME B30.5 75% limit)", FailTimeS: stage.TimeS}, nil
		}
		if res.Crane2LoadTonnes > capacity2*0.75 {
			return &Trajectory4DResult{Passed: false, FailReason: "capacity limit exceeded on crane 2 (ASME B30.5 75% limit)", FailTimeS: stage.TimeS}, nil
		}

		if res.MinBoomClearanceM < 1.0 {
			return &Trajectory4DResult{Passed: false, FailReason: "spatial hook clearance < 1.0m", FailTimeS: stage.TimeS}, nil
		}
	}

	return &Trajectory4DResult{Passed: true}, nil
}
