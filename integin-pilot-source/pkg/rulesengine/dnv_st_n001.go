package rulesengine

import "cel.dev/cel-go/cel"

// DNVSTN001RuleID identifies the DNV-ST-N001 offshore skew load factor rule
// applied to multi-sling (multi-point) lifts.
const DNVSTN001RuleID = "dnv-st-n001.5.3"

// DNVSKLMinimum is the statutory skew load factor floor. Kept in one place
// for Go-side diagnostics; enforcement happens via the CEL AST.
const DNVSKLMinimum = 1.25

// DNVSKLExpression is the Google CEL form of the DNV-ST-N001 §5.3 skew load
// factor rule. Any lift with two or more sling legs must apply a skew load
// factor SKL >= 1.25 per leg to compensate for load-share imbalance:
//
//	(sling_count < 2) || (skl_factor >= 1.25)
//
// Single-sling lifts are not skew-sensitive and short-circuit to PASS; the
// rule constrains multi-sling lifts only. The expression is data, evaluated
// inside the sandbox; the CEL AST is the single source of truth for the
// threshold decision.
const DNVSKLExpression = `double(sling_count) < 2.0 || skl_factor >= 1.25`

// DNVSKLGateExpression is the deployed zero-allocation gate. The sling-count
// check is folded server-side: the gate is only reached for multi-sling
// lifts, leaving a single comparison on the device hot path:
//
//	skl_factor >= 1.25
const DNVSKLGateExpression = `skl_factor >= 1.25`

// DNVSKLVars declares the typed variables for the DNV-ST-N001 SKL rule.
func DNVSKLVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"skl_factor":  cel.DoubleType,
		"sling_count": cel.IntType,
	}
}

// DNVSKLGateVars declares the variables for the deployed folded gate. The
// severity threshold 1.25 is the immutable standard value, so only the
// measured factor is variable at evaluate time.
func DNVSKLGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"skl_factor": cel.DoubleType,
	}
}

// DNVSTN001Definition is the standard citation card for DNV-ST-N001.
func DNVSTN001Definition() DynamicStandardDefinition {
	return DynamicStandardDefinition{
		StandardDID:      "did:integin:standard:dnv-st-n001-2024",
		StandardBody:     "DNV",
		Code:             "ST-N001",
		RevisionYear:     2024,
		Title:            "Marine operations and marine warranty",
		ScopeAbstract:    "Marine operations planning and criteria, including multi-sling offshore lifts where a skew load factor (SKL) of at least 1.25 is applied per sling leg to compensate for load share imbalance.",
		LifecycleState:   LifecycleActive,
		OfficialStoreURL: "https://www.dnv.com/standards/downloads/",
	}
}
