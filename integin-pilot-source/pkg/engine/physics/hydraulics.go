package physics

import (
	"errors"
	"fmt"
	"math"
)

// HydraulicRamParams defines mechanical and geometric configuration of a hydraulic cylinder.
type HydraulicRamParams struct {
	BoreDiameterM        float64 `json:"bore_diameter_m"`
	RodDiameterM         float64 `json:"rod_diameter_m"`
	StrokeM              float64 `json:"stroke_m"`
	MechanicalEfficiency float64 `json:"mechanical_efficiency"` // e.g. 0.95
}

// HydraulicRamOutput holds force capabilities and pressurized areas.
type HydraulicRamOutput struct {
	ExtendAreaM2        float64 `json:"extend_area_m2"`
	RetractAreaM2       float64 `json:"retract_area_m2"`
	ExtendThrustN       float64 `json:"extend_thrust_n"`
	RetractPullN        float64 `json:"retract_pull_n"`
	OperatingPressurePa float64 `json:"operating_pressure_pa"`
}

// EvaluateHydraulicRam calculates extension thrust and retraction pull forces.
func EvaluateHydraulicRam(params HydraulicRamParams, pressurePa float64) (HydraulicRamOutput, error) {
	if params.BoreDiameterM <= 0 {
		return HydraulicRamOutput{}, errors.New("bore diameter must be positive")
	}
	if params.RodDiameterM < 0 || params.RodDiameterM >= params.BoreDiameterM {
		return HydraulicRamOutput{}, errors.New("rod diameter must be non-negative and less than bore diameter")
	}
	if pressurePa < 0 {
		return HydraulicRamOutput{}, errors.New("pressure must be non-negative")
	}
	eff := params.MechanicalEfficiency
	if eff <= 0 || eff > 1.0 {
		eff = 0.95 // Default standard efficiency
	}

	extendArea := (math.Pi * math.Pow(params.BoreDiameterM, 2)) / 4.0
	retractArea := (math.Pi * (math.Pow(params.BoreDiameterM, 2) - math.Pow(params.RodDiameterM, 2))) / 4.0

	extendThrust := pressurePa * extendArea * eff
	retractPull := pressurePa * retractArea * eff

	return HydraulicRamOutput{
		ExtendAreaM2:        extendArea,
		RetractAreaM2:       retractArea,
		ExtendThrustN:       extendThrust,
		RetractPullN:        retractPull,
		OperatingPressurePa: pressurePa,
	}, nil
}

// ComputeRequiredRamPressure determines operating pressure needed for target thrust force.
func ComputeRequiredRamPressure(params HydraulicRamParams, requiredThrustN float64) (float64, error) {
	if params.BoreDiameterM <= 0 {
		return 0, errors.New("bore diameter must be positive")
	}
	if requiredThrustN <= 0 {
		return 0, errors.New("required thrust must be positive")
	}
	eff := params.MechanicalEfficiency
	if eff <= 0 || eff > 1.0 {
		eff = 0.95
	}
	extendArea := (math.Pi * math.Pow(params.BoreDiameterM, 2)) / 4.0
	pressurePa := requiredThrustN / (extendArea * eff)
	return pressurePa, nil
}

// SPMTSupportType defines the suspension hydraulic grouping mode.
type SPMTSupportType string

const (
	SPMTSupport3Point SPMTSupportType = "3-POINT"
	SPMTSupport4Point SPMTSupportType = "4-POINT"
)

// SPMTConfiguration configures multi-axle modular transporter geometry and CoG.
type SPMTConfiguration struct {
	SupportType SPMTSupportType `json:"support_type"` // 3-POINT or 4-POINT
	TotalLoadKg float64         `json:"total_load_kg"`
	CoGOffsetXM float64         `json:"cog_offset_x_m"` // longitudinal offset from centroid
	CoGOffsetYM float64         `json:"cog_offset_y_m"` // transverse offset from centroid
	CoGHeightM  float64         `json:"cog_height_m"`   // vertical CoG above suspension pivot
	WheelbaseM  float64         `json:"wheelbase_m"`    // total length along X
	TrackWidthM float64         `json:"track_width_m"`  // total width along Y
}

// SPMTStabilityOutput holds reaction distribution, tipping margins, and safety state.
type SPMTStabilityOutput struct {
	PointLoadsKg       []float64 `json:"point_loads_kg"`
	MinTippingAngleDeg float64   `json:"min_tipping_angle_deg"`
	CriticalAxis       string    `json:"critical_axis"`
	IsStable           bool      `json:"is_stable"`
	SafetyMarginRatio  float64   `json:"safety_margin_ratio"`
}

// EvaluateSPMTStability evaluates hydraulic group equalization and tipping angle margins.
func EvaluateSPMTStability(cfg SPMTConfiguration) (SPMTStabilityOutput, error) {
	if cfg.TotalLoadKg <= 0 {
		return SPMTStabilityOutput{}, errors.New("total load must be positive")
	}
	if cfg.WheelbaseM <= 0 || cfg.TrackWidthM <= 0 {
		return SPMTStabilityOutput{}, errors.New("wheelbase and track width must be positive")
	}
	if cfg.CoGHeightM <= 0 {
		return SPMTStabilityOutput{}, errors.New("CoG height must be positive")
	}

	halfL := cfg.WheelbaseM / 2.0
	halfW := cfg.TrackWidthM / 2.0

	// Check if CoG is outside physical boundary
	if math.Abs(cfg.CoGOffsetXM) >= halfL || math.Abs(cfg.CoGOffsetYM) >= halfW {
		return SPMTStabilityOutput{
			IsStable:          false,
			CriticalAxis:      "BOUNDARY_EXCEEDED",
			SafetyMarginRatio: 0,
		}, errors.New("center of gravity falls outside support envelope")
	}

	var loads []float64
	var minTippingAngle float64
	var critAxis string

	distTransverse := halfW - math.Abs(cfg.CoGOffsetYM)
	distLongitudinal := halfL - math.Abs(cfg.CoGOffsetXM)

	angleTransverseRad := math.Atan2(distTransverse, cfg.CoGHeightM)
	angleLongitudinalRad := math.Atan2(distLongitudinal, cfg.CoGHeightM)

	if angleTransverseRad <= angleLongitudinalRad {
		minTippingAngle = angleTransverseRad * 180.0 / math.Pi
		critAxis = "TRANSVERSE"
	} else {
		minTippingAngle = angleLongitudinalRad * 180.0 / math.Pi
		critAxis = "LONGITUDINAL"
	}

	switch cfg.SupportType {
	case SPMTSupport3Point:
		// 3-point: Group 1 (front pivot), Group 2 (rear left), Group 3 (rear right)
		// Front carries 1/2 nominal, rear left & right carry 1/4 nominal with offset balancing
		w := cfg.TotalLoadKg
		fx := cfg.CoGOffsetXM / halfL
		fy := cfg.CoGOffsetYM / halfW

		// Group 1 front: P1 = W * (0.5 + 0.5*fx)
		p1 := w * (0.5 + 0.5*fx)
		// Rear split:
		pRear := w * (0.5 - 0.5*fx)
		p2 := pRear * (0.5 - 0.5*fy) // Rear left
		p3 := pRear * (0.5 + 0.5*fy) // Rear right

		loads = []float64{p1, p2, p3}

	case SPMTSupport4Point:
		// 4-point: 4 quadrants [FL, FR, RL, RR]
		w := cfg.TotalLoadKg
		baseP := w / 4.0
		deltaX := (cfg.CoGOffsetXM / halfL) * (w / 4.0)
		deltaY := (cfg.CoGOffsetYM / halfW) * (w / 4.0)

		fl := baseP + deltaX - deltaY
		fr := baseP + deltaX + deltaY
		rl := baseP - deltaX - deltaY
		rr := baseP - deltaX + deltaY

		loads = []float64{fl, fr, rl, rr}

	default:
		return SPMTStabilityOutput{}, fmt.Errorf("unsupported SPMT support type: %s", cfg.SupportType)
	}

	isStable := true
	for _, l := range loads {
		if l <= 0 {
			isStable = false
			critAxis = "LIFT_OFF_DETECTED"
			break
		}
	}

	// Safety margin ratio: distance to tipping line / CoG height
	minDist := math.Min(distTransverse, distLongitudinal)
	safetyMarginRatio := minDist / cfg.CoGHeightM

	return SPMTStabilityOutput{
		PointLoadsKg:       loads,
		MinTippingAngleDeg: minTippingAngle,
		CriticalAxis:       critAxis,
		IsStable:           isStable,
		SafetyMarginRatio:  safetyMarginRatio,
	}, nil
}

// DarcyWeisbachPressureDrop calculates pipe hydraulic pressure drop via Darcy-Weisbach equation:
// DeltaP = f * (L / D) * (rho * v^2 / 2)
// For laminar flow (Re < 2300): f = 64 / Re
// For turbulent flow (Re >= 2300): Haaland approximation for Colebrook-White equation.
func DarcyWeisbachPressureDrop(lengthM, diameterM, velocityMs, densityKgM3, dynamicViscosityPaS, pipeRoughnessM float64) (deltaPPa, reynolds, frictionFactor float64, err error) {
	if lengthM <= 0 || diameterM <= 0 || velocityMs <= 0 || densityKgM3 <= 0 || dynamicViscosityPaS <= 0 {
		return 0, 0, 0, errors.New("pipe length, diameter, velocity, density, and viscosity must be positive")
	}

	// Reynolds number: Re = (rho * v * D) / mu
	reynolds = (densityKgM3 * velocityMs * diameterM) / dynamicViscosityPaS

	if reynolds < 2300.0 {
		// Laminar flow
		frictionFactor = 64.0 / reynolds
	} else {
		// Haaland explicit approximation: 1 / sqrt(f) = -1.8 * log10((roughness / (3.7 * D))^1.11 + 6.9 / Re)
		ed := pipeRoughnessM / (3.7 * diameterM)
		term := math.Pow(ed, 1.11) + (6.9 / reynolds)
		invSqrtF := -1.8 * math.Log10(term)
		frictionFactor = 1.0 / (invSqrtF * invSqrtF)
	}

	// Dynamic pressure: 0.5 * rho * v^2
	dynamicP := 0.5 * densityKgM3 * math.Pow(velocityMs, 2)
	deltaPPa = frictionFactor * (lengthM / diameterM) * dynamicP

	return deltaPPa, reynolds, frictionFactor, nil
}
