package rulesengine

import "cel.dev/cel-go/cel"

// DesignBasis selects the deterministic design-verification philosophy for a
// structural check. ASD (Allowable Stress Design) folds the factor of safety
// into an allowable stress and verifies utilization <= 1.0. LRFD (Load and
// Resistance Factor Design) verifies factored resistance against factored
// demand using partial factors.
type DesignBasis string

// Supported design-basis philosophies.
const (
	DesignBasisASD  DesignBasis = "ASD"
	DesignBasisLRFD DesignBasis = "LRFD"
)

// DesignASDCapacityUtilizationRuleID identifies the ASD utilization gate.
const DesignASDCapacityUtilizationRuleID = "design.asd-capacity-utilization"

// CapacityUtilizationGateExpression is the shared canonical CEL gate for
// every capacity-utilization decision: the ratio of applied demand to
// capacity must lie in [0, 1]. A negative utilization is physically
// impossible and fails closed. The upper bound mirrors the "utilization
// <= 1.0" ASD verdict used by pkg/cad/structural, where the allowable stress
// already carries the factor of safety (utilization = sigma * SF /
// sigma_yield <= 1 <=> FoS realized >= SF). The CEL AST is the single source
// of truth for the gate.
const CapacityUtilizationGateExpression = `utilization >= 0.0 && utilization <= 1.0`

// CapacityUtilizationFoldedExpression is the deployed zero-allocation ASD
// gate. Non-negative utilization is guaranteed by input validation before
// the value reaches the device, so the single comparison mirrors the
// structural verdict exactly (0 allocs/op on the hot path). Flipped to >= per
// the interpreter's allocation-free comparison specialization:
//
//	1.0 >= utilization
const CapacityUtilizationFoldedExpression = `1.0 >= utilization`

// DesignLRFDPartialFactorsRuleID identifies the LRFD partial-factor gate.
const DesignLRFDPartialFactorsRuleID = "design.lrfd-partial-factors"

// DesignLRFDGateExpression is the limit-state partial-factor inequality:
// the factored resistance must meet or exceed the factored demand:
//
//	phi * R_n >= gamma_D * D + gamma_L * L
//
// phi is the resistance factor, R_n the nominal resistance, gamma_D and
// gamma_L the dead and live load factors, D and L the coincident dead and
// live loads (Eurocode/ASCE partial-factor model, expressed parameter-free
// so the CEL AST is the sole arithmetic authority).
const DesignLRFDGateExpression = `phi * design_resistance >= gamma_d * dead_load + gamma_l * live_load`

// DesignLRFDGateFoldedExpression is the deployed zero-allocation device
// gate. The partial-factor products are folded server-side into
// factored_resistance (phi * R_n) and factored_demand (gamma_D*D + gamma_L*L)
// per the B30.5 folding doctrine, leaving a single comparison on the hot
// path:
//
//	factored_resistance >= factored_demand
const DesignLRFDGateFoldedExpression = `factored_resistance >= factored_demand`

// CapacityUtilizationVars declares the typed variable for capacity gates.
func CapacityUtilizationVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"utilization": cel.DoubleType,
	}
}

// DesignLRFDVars declares the typed variables for the LRFD partial-factor
// gate.
func DesignLRFDVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"phi":               cel.DoubleType,
		"design_resistance": cel.DoubleType,
		"gamma_d":           cel.DoubleType,
		"dead_load":         cel.DoubleType,
		"gamma_l":           cel.DoubleType,
		"live_load":         cel.DoubleType,
	}
}

// DesignLRFDFoldedVars declares the variables for the deployed folded LRFD
// gate.
func DesignLRFDFoldedVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"factored_resistance": cel.DoubleType,
		"factored_demand":     cel.DoubleType,
	}
}

// DesignGate returns the CEL rule card for the chosen design basis so
// callers evaluate the switch declaratively without branching on thresholds
// themselves.
func DesignGate(b DesignBasis) (ruleID string, expression string, vars map[string]*cel.Type) {
	switch b {
	case DesignBasisLRFD:
		return DesignLRFDPartialFactorsRuleID, DesignLRFDGateExpression, DesignLRFDVars()
	case DesignBasisASD:
		return DesignASDCapacityUtilizationRuleID, CapacityUtilizationGateExpression, CapacityUtilizationVars()
	default:
		panic("rulesengine: unknown design basis " + string(b))
	}
}
