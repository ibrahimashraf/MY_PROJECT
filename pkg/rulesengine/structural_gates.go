package rulesengine

import "cel.dev/cel-go/cel"

// Structural gate IDs mirror the pass/fail thresholds and safety factor
// decisions hardcoded in pkg/cad/structural, so callers can evaluate the
// identical safety criteria declaratively via CEL instead of re-implementing
// threshold branches in business handlers.

const (
	// StructuralBearingPressureRuleID gates contact pressure against the
	// allowable bearing pressure (mirrors structural.GroundBearingPressure).
	StructuralBearingPressureRuleID = "structural.bearing-pressure"
	// StructuralSlingSpreadRuleID gates the sling spread geometry lockout
	// (mirrors structural.MultiLegSlingTension refusal when
	// cos(theta) <= 0.1).
	StructuralSlingSpreadRuleID = "structural.sling-spread-lockout"
	// StructuralBeamUtilizationRuleID gates beam bending utilization
	// (mirrors structural.BeamStressAnalysis safe = utilization <= 1.0).
	StructuralBeamUtilizationRuleID = "structural.beam-utilization"
	// StructuralTandemCraneUtilizationRuleID gates each crane's share against
	// rated capacity in a tandem lift (mirrors structural.TandemLiftLoadDistribution).
	StructuralTandemCraneUtilizationRuleID = "structural.tandem-crane-utilization"
)

// BearingPressureGateExpression mirrors the structural ground bearing check:
// contact pressure must be non-negative and within the allowable bearing
// pressure.
const BearingPressureGateExpression = `actual_pressure_kpa >= 0.0 && actual_pressure_kpa <= allowable_pressure_kpa`

// SlingSpreadGateExpression mirrors the MultiLegSlingTension lockout
// (cos(theta) <= 0.1 -> refused). The cosine of the angle from vertical is
// folded server-side into the single variable, keeping the deployed AST a
// single comparison:
//
//	cos_leg_factor > 0.1
const SlingSpreadGateExpression = `cos_leg_factor > 0.1`

// BearingPressureGateVars declares the typed variables for the bearing
// pressure gate.
func BearingPressureGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"actual_pressure_kpa":    cel.DoubleType,
		"allowable_pressure_kpa": cel.DoubleType,
	}
}

// SlingSpreadGateVars declares the typed variable for the sling spread gate.
func SlingSpreadGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"cos_leg_factor": cel.DoubleType,
	}
}
