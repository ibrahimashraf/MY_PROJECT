package parameters

import (
	"fmt"
	"math"

	"cel.dev/cel-go/cel"
)

// ISO 5817:2023 - Welding: Quality levels for imperfections in steel, nickel, titanium
//
// Quality Levels:
// - Level B: Stringent (high fatigue, offshore, nuclear, dynamic crane booms)
// - Level C: Intermediate (standard structural and industrial lifting equipment)
// - Level D: Moderate (non-critical, low-stress, static secondary structures)

type WeldQualityLevel string

const (
	WeldQualityLevelB WeldQualityLevel = "B" // Stringent
	WeldQualityLevelC WeldQualityLevel = "C" // Intermediate
	WeldQualityLevelD WeldQualityLevel = "D" // Moderate
)

// WeldInspectionInput holds visual and dimensional inspection data for an ISO 5817 weld seam.
type WeldInspectionInput struct {
	Level               WeldQualityLevel `json:"level"`
	NominalThicknessMM  float64          `json:"nominal_thickness_mm"`  // t (parent metal)
	WeldWidthMM         float64          `json:"weld_width_mm"`         // b
	MeasuredUndercutMM  float64          `json:"measured_undercut_mm"`  // h (imperfection 5011/5012)
	ExcessWeldHeightMM  float64          `json:"excess_weld_height_mm"` // h (imperfection 502)
	PorosityAreaPct     float64          `json:"porosity_area_pct"`     // projected pore area %
	HasCracks           bool             `json:"has_cracks"`            // imperfection 100 (crack)
	HasLackOfFusion     bool             `json:"has_lack_of_fusion"`    // imperfection 401 (lack of fusion)
	HasIncompletePenetr bool             `json:"has_incomplete_penetr"` // imperfection 402 (incomplete root penetration)
}

// WeldVerdict represents the ISO 5817 compliance verdict.
type WeldVerdict struct {
	Level            WeldQualityLevel `json:"level"`
	MaxAllowUndercut float64          `json:"max_allow_undercut_mm"`
	MaxAllowExcess   float64          `json:"max_allow_excess_mm"`
	MaxAllowPorosity float64          `json:"max_allow_porosity_pct"`
	Passed           bool             `json:"passed"`
	Violations       []string         `json:"violations,omitempty"`
}

// UndercutLimit calculates the ISO 5817 maximum allowable undercut depth h.
func UndercutLimit(level WeldQualityLevel, t float64) (float64, error) {
	if t <= 0 {
		return 0, fmt.Errorf("parameters: nominal thickness t must be > 0, got %g", t)
	}
	switch level {
	case WeldQualityLevelB:
		// h <= 0.05 t, max 0.5 mm
		return math.Min(0.05*t, 0.5), nil
	case WeldQualityLevelC:
		// h <= 0.1 t, max 1.0 mm
		return math.Min(0.10*t, 1.0), nil
	case WeldQualityLevelD:
		// h <= 0.2 t, max 2.0 mm
		return math.Min(0.20*t, 2.0), nil
	default:
		return 0, fmt.Errorf("parameters: invalid ISO 5817 quality level %q (must be B, C, or D)", level)
	}
}

// ExcessWeldMetalLimit calculates the ISO 5817 maximum allowable reinforcement height h.
func ExcessWeldMetalLimit(level WeldQualityLevel, b float64) (float64, error) {
	if b <= 0 {
		return 0, fmt.Errorf("parameters: weld width b must be > 0, got %g", b)
	}
	switch level {
	case WeldQualityLevelB:
		// h <= 1 mm + 0.10 b, max 5 mm
		return math.Min(1.0+0.10*b, 5.0), nil
	case WeldQualityLevelC:
		// h <= 1 mm + 0.15 b, max 7 mm
		return math.Min(1.0+0.15*b, 7.0), nil
	case WeldQualityLevelD:
		// h <= 1 mm + 0.25 b, max 10 mm
		return math.Min(1.0+0.25*b, 10.0), nil
	default:
		return 0, fmt.Errorf("parameters: invalid ISO 5817 quality level %q (must be B, C, or D)", level)
	}
}

// MaxAllowablePorosityPct returns the maximum projected porosity percentage for the level.
func MaxAllowablePorosityPct(level WeldQualityLevel) (float64, error) {
	switch level {
	case WeldQualityLevelB:
		return 1.0, nil // <= 1.0%
	case WeldQualityLevelC:
		return 2.5, nil // <= 2.5%
	case WeldQualityLevelD:
		return 4.0, nil // <= 4.0%
	default:
		return 0, fmt.Errorf("parameters: invalid ISO 5817 quality level %q", level)
	}
}

// EvaluateISO5817Weld performs fail-closed evaluation of weld imperfections.
func EvaluateISO5817Weld(input WeldInspectionInput) (WeldVerdict, error) {
	if err := rejectParamNaN("weld dimensions",
		input.NominalThicknessMM, input.WeldWidthMM,
		input.MeasuredUndercutMM, input.ExcessWeldHeightMM,
		input.PorosityAreaPct,
	); err != nil {
		return WeldVerdict{}, err
	}

	maxUndercut, err := UndercutLimit(input.Level, input.NominalThicknessMM)
	if err != nil {
		return WeldVerdict{}, err
	}

	maxExcess, err := ExcessWeldMetalLimit(input.Level, input.WeldWidthMM)
	if err != nil {
		return WeldVerdict{}, err
	}

	maxPorosity, err := MaxAllowablePorosityPct(input.Level)
	if err != nil {
		return WeldVerdict{}, err
	}

	verdict := WeldVerdict{
		Level:            input.Level,
		MaxAllowUndercut: maxUndercut,
		MaxAllowExcess:   maxExcess,
		MaxAllowPorosity: maxPorosity,
		Passed:           true,
	}

	// Zero-tolerance planar defects
	if input.HasCracks {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			"ISO 5817 No. 100: crack detected (strictly prohibited across all Quality Levels)")
	}
	if input.HasLackOfFusion {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			"ISO 5817 No. 401: lack of fusion detected (strictly prohibited across all Quality Levels)")
	}
	if input.HasIncompletePenetr {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			"ISO 5817 No. 402: incomplete penetration detected (strictly prohibited across Level B and C)")
	}

	// Dimensional threshold checks
	if input.MeasuredUndercutMM > maxUndercut {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			fmt.Sprintf("ISO 5817 No. 5011: undercut depth %.2f mm exceeds Level %s limit %.2f mm",
				input.MeasuredUndercutMM, input.Level, maxUndercut))
	}

	if input.ExcessWeldHeightMM > maxExcess {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			fmt.Sprintf("ISO 5817 No. 502: excess weld metal %.2f mm exceeds Level %s limit %.2f mm",
				input.ExcessWeldHeightMM, input.Level, maxExcess))
	}

	if input.PorosityAreaPct > maxPorosity {
		verdict.Passed = false
		verdict.Violations = append(verdict.Violations,
			fmt.Sprintf("ISO 5817 No. 2017: porosity area %.2f%% exceeds Level %s limit %.2f%%",
				input.PorosityAreaPct, input.Level, maxPorosity))
	}

	return verdict, nil
}

// ISO5817WeldGateVars returns CEL typed variable mappings for weld evaluation.
func ISO5817WeldGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"measured_undercut_mm": cel.DoubleType,
		"max_allow_undercut":   cel.DoubleType,
		"excess_weld_mm":       cel.DoubleType,
		"max_allow_excess":     cel.DoubleType,
		"porosity_pct":         cel.DoubleType,
		"max_allow_porosity":   cel.DoubleType,
	}
}
