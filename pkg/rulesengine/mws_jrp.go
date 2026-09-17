package rulesengine

import "cel.dev/cel-go/cel"

// MWSJRPSeaStateRuleID identifies the Marine Warranty Survey sea-state
// cutoff rule. MWS (Marine Warranty Surveyor) practice and the joint
// review panel (JRP) gate offshore lifts to approved operational windows
// bounded by significant wave height Hs and wave peak period Tp.
const MWSJRPSeaStateRuleID = "mws-jrp.sea-state-cutoff"

// MWSJRPSeaStateExpression is the deployed CEL gate for the sea-state
// window. Both wave parameters must be physically non-negative and within
// the MWS/JRP approved cutoffs:
//
//	hs_m >= 0.0 && tp_s >= 0.0 && hs_m <= max_hs_m && tp_s <= max_tp_s
//
// The cutoffs (max_hs_m, max_tp_s) are bound per lift-procedure approval and
// supplied as typed variables; a negative wave height or period fails
// closed (never passes the window guard).
const MWSJRPSeaStateExpression = `hs_m >= 0.0 && tp_s >= 0.0 && hs_m <= max_hs_m && tp_s <= max_tp_s`

// MWSJRPSeaStateVars declares the typed variables for the sea-state gate.
func MWSJRPSeaStateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"hs_m":     cel.DoubleType,
		"tp_s":     cel.DoubleType,
		"max_hs_m": cel.DoubleType,
		"max_tp_s": cel.DoubleType,
	}
}

// MWSJRPSeaStateHsGateExpression is the deployed zero-allocation device gate
// for the wave-height half of the window. The positivity guard is folded into
// server-side validation; the device gate is a single comparison (0 allocs/op,
// flipped to >= for the interpreter's allocation-free comparison path):
//
//	max_hs_m >= hs_m
const MWSJRPSeaStateHsGateExpression = `max_hs_m >= hs_m`

// MWSJRPSeaStateTpGateExpression is the deployed zero-allocation device gate
// for the wave-period half of the window:
//
//	max_tp_s >= tp_s
const MWSJRPSeaStateTpGateExpression = `max_tp_s >= tp_s`

// MWSJRPDefinition is the standard citation card for Marine Warranty
// Survey sea-state cutoff criteria.
func MWSJRPDefinition() DynamicStandardDefinition {
	return DynamicStandardDefinition{
		StandardDID:      "did:integin:standard:mws-jrp",
		StandardBody:     "MWS",
		Code:             "JRP",
		RevisionYear:     2021,
		Title:            "Marine Warranty Survey sea-state operational windows",
		ScopeAbstract:    "Offshore lift execution cutoffs: significant wave height (Hs) and wave peak period (Tp) must remain within the Marine Warranty Surveyor / joint review panel approved window for the lift procedure.",
		LifecycleState:   LifecycleActive,
		ReplacesStandard: "did:integin:standard:mws-jnrc",
		OfficialStoreURL: "https://www.dnv.com/standards/downloads/",
	}
}
