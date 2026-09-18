package parameters

import (
	"fmt"
	"math"

	"cel.dev/cel-go/cel"
)

// ASME B30.10 Hooks: Discard and Inspection Thresholds (Clause 10-1.10.4 & 10-2.10.4)
//
// Statutory/Standard Citation: ASME B30.10-2019 / ASME B30.10-2024
// - Throat opening stretch: Discard if throat opening exceeds 5% of nominal (or 1/4 in / 6 mm).
// - Hook body twist: Discard if bend or twist exceeds 10 degrees from plane of unbent hook.
// - Saddle / bowl wear: Discard if wear exceeds 10% of original sectional dimension.
// - Cracks, gouges, or nicks: Discard immediately (zero tolerance).
// - Latch engagement: Discard / repair if self-locking latch fails to lock or align properly.

const (
	ASMEB30HookThroatStretchDiscardThreshold = 0.05 // 5% max throat stretch
	ASMEB30HookTwistDiscardThresholdDeg      = 10.0 // 10 degrees max twist
	ASMEB30HookWearDiscardThreshold          = 0.10 // 10% max sectional wear
)

// CEL Expressions for ASME B30.10 hooks
const (
	ASMEB30HookThroatStretchExpression = `measured_throat_mm <= 1.05 * nominal_throat_mm`
	ASMEB30HookTwistExpression         = `twist_deg <= 10.0 && twist_deg >= -10.0`
	ASMEB30HookWearExpression          = `measured_saddle_mm >= 0.90 * nominal_saddle_mm`
)

// HookInspectionInput holds field inspection measurements for an ASME B30.10 hook.
type HookInspectionInput struct {
	NominalThroatMM  float64 `json:"nominal_throat_mm"`
	MeasuredThroatMM float64 `json:"measured_throat_mm"`
	NominalSaddleMM  float64 `json:"nominal_saddle_mm"`
	MeasuredSaddleMM float64 `json:"measured_saddle_mm"`
	TwistDeg         float64 `json:"twist_deg"`
	HasCracks        bool    `json:"has_cracks"`
	LatchOperational bool    `json:"latch_operational"`
}

// HookVerdict represents the ASME B30.10 compliance verdict.
type HookVerdict struct {
	ThroatStretchPct float64  `json:"throat_stretch_pct"`
	SaddleWearPct    float64  `json:"saddle_wear_pct"`
	TwistDeg         float64  `json:"twist_deg"`
	Passed           bool     `json:"passed"`
	Violations       []string `json:"violations,omitempty"`
}

func rejectParamNaN(name string, vals ...float64) error {
	for _, v := range vals {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("parameters: NaN/Inf %s rejected", name)
		}
	}
	return nil
}

// EvaluateASMEB30Hook performs deterministic fail-closed inspection evaluation against ASME B30.10.
func EvaluateASMEB30Hook(input HookInspectionInput) (HookVerdict, error) {
	if err := rejectParamNaN("hook dimensions",
		input.NominalThroatMM, input.MeasuredThroatMM,
		input.NominalSaddleMM, input.MeasuredSaddleMM,
		input.TwistDeg,
	); err != nil {
		return HookVerdict{}, err
	}

	if input.NominalThroatMM <= 0 {
		return HookVerdict{}, fmt.Errorf("parameters: nominal throat must be > 0, got %g", input.NominalThroatMM)
	}
	if input.MeasuredThroatMM <= 0 {
		return HookVerdict{}, fmt.Errorf("parameters: measured throat must be > 0, got %g", input.MeasuredThroatMM)
	}
	if input.NominalSaddleMM <= 0 {
		return HookVerdict{}, fmt.Errorf("parameters: nominal saddle must be > 0, got %g", input.NominalSaddleMM)
	}
	if input.MeasuredSaddleMM <= 0 {
		return HookVerdict{}, fmt.Errorf("parameters: measured saddle must be > 0, got %g", input.MeasuredSaddleMM)
	}

	throatStretch := (input.MeasuredThroatMM - input.NominalThroatMM) / input.NominalThroatMM
	throatStretchPct := throatStretch * 100.0

	saddleWear := (input.NominalSaddleMM - input.MeasuredSaddleMM) / input.NominalSaddleMM
	saddleWearPct := saddleWear * 100.0

	twistAbs := math.Abs(input.TwistDeg)

	verdict := HookVerdict{
		ThroatStretchPct: throatStretchPct,
		SaddleWearPct:    saddleWearPct,
		TwistDeg:         twistAbs,
		Passed:           true,
	}

	if throatStretch > ASMEB30HookThroatStretchDiscardThreshold {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			fmt.Sprintf("ASME B30.10 §10-1.10.4(b): throat opening stretch %.2f%% exceeds 5.0%% threshold", throatStretchPct))
	}

	if saddleWear > ASMEB30HookWearDiscardThreshold {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			fmt.Sprintf("ASME B30.10 §10-1.10.4(d): saddle wear %.2f%% exceeds 10.0%% threshold", saddleWearPct))
	}

	if twistAbs > ASMEB30HookTwistDiscardThresholdDeg {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			fmt.Sprintf("ASME B30.10 §10-1.10.4(c): hook twist %.2f° exceeds 10.0° threshold", twistAbs))
	}

	if input.HasCracks {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			"ASME B30.10 §10-1.10.4(a): cracks, gouges or nicks detected (immediate discard)")
	}

	if !input.LatchOperational {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			"ASME B30.10 §10-1.10.4(e): hook safety latch missing, deformed or inoperable")
	}

	return verdict, nil
}

// ASMEB30HookGateVars returns CEL typed variable mappings for hook validation.
func ASMEB30HookGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"measured_throat_mm": cel.DoubleType,
		"nominal_throat_mm":  cel.DoubleType,
		"measured_saddle_mm": cel.DoubleType,
		"nominal_saddle_mm":  cel.DoubleType,
		"twist_deg":          cel.DoubleType,
	}
}

// ASME B30.5-2025 Mobile Crane & ASME B30.26-2026 Rigging Hardware Parameters

const (
	ASMEB30_5StabilityLimitMobilePct  = 85.0 // Mobile crane tipping margin (<= 85% of tipping load on outriggers)
	ASMEB30_5StabilityLimitCrawlerPct = 75.0 // Crawler crane tipping margin (<= 75% of tipping load on tracks)
	ASMEB30_5MaxOperationalWindMS     = 14.0 // Standard operational wind cutoff (approx 31 mph)
	ASMEB30_26ShacklePinWearDiscardPct  = 10.0 // 10% max reduction in pin diameter (Clause 26-1.9.4)
	ASMEB30_26ShackleBowWearDiscardPct  = 10.0 // 10% max reduction in bow sectional dimension
	ASMEB30_26ShackleSpreadDiscardPct   = 5.0  // 5% or 1/4 in max throat opening stretch
)

// ASMEB30_5CraneStabilityInput evaluates tipping stability and operating wind.
type ASMEB30_5CraneStabilityInput struct {
	IsCrawler       bool    `json:"is_crawler"`
	AppliedLoadT    float64 `json:"applied_load_t"`
	TippingLoadT    float64 `json:"tipping_load_t"`
	OperatingWindMS float64 `json:"operating_wind_ms"`
}

// EvaluateASMEB30_5Crane evaluates crane tipping stability and wind limits per ASME B30.5 §5-1.1.
func EvaluateASMEB30_5Crane(in ASMEB30_5CraneStabilityInput) (bool, float64, error) {
	if err := rejectParamNaN("ASME B30.5 crane inputs", in.AppliedLoadT, in.TippingLoadT, in.OperatingWindMS); err != nil {
		return false, 0, err
	}
	if in.TippingLoadT <= 0 {
		return false, 0, fmt.Errorf("parameters: tipping load must be > 0, got %g", in.TippingLoadT)
	}
	if in.AppliedLoadT < 0 {
		return false, 0, fmt.Errorf("parameters: applied load must be >= 0, got %g", in.AppliedLoadT)
	}
	if in.OperatingWindMS < 0 {
		return false, 0, fmt.Errorf("parameters: wind speed must be >= 0, got %g", in.OperatingWindMS)
	}

	limitPct := ASMEB30_5StabilityLimitMobilePct
	if in.IsCrawler {
		limitPct = ASMEB30_5StabilityLimitCrawlerPct
	}

	maxAllowable := (limitPct / 100.0) * in.TippingLoadT
	utilization := in.AppliedLoadT / maxAllowable

	if in.OperatingWindMS > ASMEB30_5MaxOperationalWindMS {
		return false, utilization, fmt.Errorf("ASME B30.5: wind speed %.1f m/s exceeds operational ceiling %.1f m/s", in.OperatingWindMS, ASMEB30_5MaxOperationalWindMS)
	}

	return utilization <= 1.0, utilization, nil
}

// ShackleInspectionInput holds field inspection dimensions for an ASME B30.26 shackle.
type ShackleInspectionInput struct {
	NominalPinDiameterMM  float64 `json:"nominal_pin_dia_mm"`
	MeasuredPinDiameterMM float64 `json:"measured_pin_dia_mm"`
	NominalBowDiameterMM  float64 `json:"nominal_bow_dia_mm"`
	MeasuredBowDiameterMM float64 `json:"measured_bow_dia_mm"`
	NominalThroatGapMM    float64 `json:"nominal_throat_gap_mm"`
	MeasuredThroatGapMM   float64 `json:"measured_throat_gap_mm"`
	HasCracks             bool    `json:"has_cracks"`
	BodyTwistDeg          float64 `json:"body_twist_deg"`
}

// ShackleVerdict represents the ASME B30.26 inspection verdict.
type ShackleVerdict struct {
	PinWearPct       float64  `json:"pin_wear_pct"`
	BowWearPct       float64  `json:"bow_wear_pct"`
	ThroatStretchPct float64  `json:"throat_stretch_pct"`
	Passed           bool     `json:"passed"`
	Violations       []string `json:"violations,omitempty"`
}

// EvaluateASMEB30_26Shackle evaluates shackle wear against ASME B30.26 Clause 26-1.9.4.
func EvaluateASMEB30_26Shackle(in ShackleInspectionInput) (ShackleVerdict, error) {
	if err := rejectParamNaN("shackle dimensions",
		in.NominalPinDiameterMM, in.MeasuredPinDiameterMM,
		in.NominalBowDiameterMM, in.MeasuredBowDiameterMM,
		in.NominalThroatGapMM, in.MeasuredThroatGapMM,
		in.BodyTwistDeg,
	); err != nil {
		return ShackleVerdict{}, err
	}

	if in.NominalPinDiameterMM <= 0 || in.MeasuredPinDiameterMM <= 0 {
		return ShackleVerdict{}, fmt.Errorf("parameters: shackle pin diameter must be > 0")
	}
	if in.NominalBowDiameterMM <= 0 || in.MeasuredBowDiameterMM <= 0 {
		return ShackleVerdict{}, fmt.Errorf("parameters: shackle bow diameter must be > 0")
	}
	if in.NominalThroatGapMM <= 0 || in.MeasuredThroatGapMM <= 0 {
		return ShackleVerdict{}, fmt.Errorf("parameters: shackle throat gap must be > 0")
	}

	pinWearPct := ((in.NominalPinDiameterMM - in.MeasuredPinDiameterMM) / in.NominalPinDiameterMM) * 100.0
	bowWearPct := ((in.NominalBowDiameterMM - in.MeasuredBowDiameterMM) / in.NominalBowDiameterMM) * 100.0
	throatStretchPct := ((in.MeasuredThroatGapMM - in.NominalThroatGapMM) / in.NominalThroatGapMM) * 100.0

	verdict := ShackleVerdict{
		PinWearPct:       math.Max(0, pinWearPct),
		BowWearPct:       math.Max(0, bowWearPct),
		ThroatStretchPct: math.Max(0, throatStretchPct),
		Passed:           true,
	}

	if pinWearPct > ASMEB30_26ShacklePinWearDiscardPct {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			fmt.Sprintf("ASME B30.26 §26-1.9.4: pin wear %.1f%% exceeds 10.0%% discard limit", pinWearPct))
	}
	if bowWearPct > ASMEB30_26ShackleBowWearDiscardPct {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			fmt.Sprintf("ASME B30.26 §26-1.9.4: bow wear %.1f%% exceeds 10.0%% discard limit", bowWearPct))
	}
	if throatStretchPct > ASMEB30_26ShackleSpreadDiscardPct {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			fmt.Sprintf("ASME B30.26 §26-1.9.4: throat spread %.1f%% exceeds 5.0%% discard limit", throatStretchPct))
	}
	if in.HasCracks {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			"ASME B30.26 §26-1.9.4: cracks, nicks, or gouges detected (immediate discard)")
	}
	if math.Abs(in.BodyTwistDeg) > 0.1 {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			fmt.Sprintf("ASME B30.26 §26-1.9.4: shackle body twisted %.1f° (must be in-plane)", math.Abs(in.BodyTwistDeg)))
	}

	return verdict, nil
}

// ASMEB30GateVars declares typed CEL variables for ASME B30 checks.
func ASMEB30GateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"applied_load_t":    cel.DoubleType,
		"tipping_load_t":    cel.DoubleType,
		"crane_utilization": cel.DoubleType,
		"pin_wear_pct":      cel.DoubleType,
		"bow_wear_pct":      cel.DoubleType,
		"throat_spread_pct": cel.DoubleType,
		"operating_wind_ms": cel.DoubleType,
	}
}

