package rulesengine

import "cel.dev/cel-go/cel"

// B30_5RuleID identifies the ASME B30.5 proof-load rule in rulesets.
const B30_5RuleID = "asme-b30.5.2.2.3.1a"

// B30_5ProofLoadExpression is the Google CEL form of the ASME B30.5 mobile
// crane structural proof-load model:
//
//	P_test = C <= 20 t ? C * 1.25 : (C <= 50 t ? C * 1.20 : C * 1.15)
//
// where C is the rated capacity and P_test is the applied proof load. The
// expression is data, evaluated inside the sandbox; inline comment math is
// never trusted, the CEL AST is the single source of truth.
const B30_5ProofLoadExpression = `capacity_t <= 20.0 ? capacity_t * 1.25 : (capacity_t <= 50.0 ? capacity_t * 1.20 : capacity_t * 1.15)`

// B30_5ProofLoadExceedsExpression detects a reported test load P that fails
// to reach the required proof load (under-test), critical for quarantine:
//
//	under_test = P_report < P_test_required
const B30_5ProofLoadExceedsExpression = `reported_test_load_t >= (capacity_t <= 20.0 ? capacity_t * 1.25 : (capacity_t <= 50.0 ? capacity_t * 1.20 : capacity_t * 1.15))`

// B30_5GateExpression is the deployed zero-allocation gate. The tier
// coefficient is folded server-side into required_test_load_t (via
// B30_5ProofLoad), so the CEL AST contains a single comparison and the device
// hot path evaluates with 0 allocs/op:
//
//	reported_test_load_t >= required_test_load_t
const B30_5GateExpression = `reported_test_load_t >= required_test_load_t`

// B30_5Vars declares the typed variables for B30.5 proof-load rules.
func B30_5Vars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"capacity_t":           cel.DoubleType,
		"reported_test_load_t": cel.DoubleType,
	}
}

// B30_5GateVars declares the variables for the deployed folded gate.
func B30_5GateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"reported_test_load_t": cel.DoubleType,
		"required_test_load_t": cel.DoubleType,
	}
}

// ASME standard definition card used by standardsync importers.
func B30_5Definition() DynamicStandardDefinition {
	return DynamicStandardDefinition{
		StandardDID:      "did:integin:standard:asme-b30.5-2024",
		StandardBody:     "ASME",
		Code:             "B30.5",
		RevisionYear:     2024,
		Title:            "Mobile and Locomotive Cranes",
		ScopeAbstract:    "Safety requirements for construction, operation, and maintenance of mobile and locomotive cranes, including proof-load testing.",
		LifecycleState:   "ACTIVE",
		ReplacesStandard: "did:integin:standard:asme-b30.5-2018",
		OfficialStoreURL: "https://www.asme.org/codes-standards/b30-5",
	}
}

// B30_5ProofLoad derives the required proof load (tonnes) for a rated
// capacity using the deterministic, hard-coded ASME B30.5 coefficient table.
// Only used for Go-side diagnostics; enforcement happens via the CEL rule.
func B30_5ProofLoad(capacityT float64) float64 {
	switch {
	case capacityT <= 20:
		return capacityT * 1.25
	case capacityT <= 50:
		return capacityT * 1.20
	default:
		return capacityT * 1.15
	}
}
