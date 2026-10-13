package rulesengine

import "cel.dev/cel-go/cel"

// ISO4309RuleIDs identify the wire-rope discard rules used in rulesets.
const (
	ISO4309BrokenWiresRuleID = "iso-4309.6.2.a"
	ISO4309DiameterRuleID    = "iso-4309.6.2.b"
)

// ISO4309BrokenWiresExpression flags discard when the count of visible broken
// outer wires in one rope lay reaches 6 x the rope diameter (mm), per the
// ISO 4309 discard model codified in the tracker:
//
//	n_broken >= 6d
const ISO4309BrokenWiresExpression = `double(broken_outer_wires) >= 6.0 * rope_diameter_mm`

// ISO4309DiameterExpression flags discard when diameter loss exceeds 7% of
// the nominal rope diameter:
//
//	(d_nominal - d_measured) / d_nominal > 0.07
const ISO4309DiameterExpression = `(rope_diameter_mm - measured_diameter_mm) / rope_diameter_mm > 0.07`

// ISO4309DiameterGateExpression is the deployed zero-allocation density gate.
// The reduction is folded server-side into diameter_loss_pct, leaving the CEL
// AST as a single comparison (0 allocs/op on the device hot path):
//
//	diameter_loss_pct > 0.07
const ISO4309DiameterGateExpression = `diameter_loss_pct > 0.07`

// ISO4309Vars declares the typed variables for ISO 4309 discard rules.
func ISO4309Vars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"rope_diameter_mm":     cel.DoubleType,
		"broken_outer_wires":   cel.IntType,
		"measured_diameter_mm": cel.DoubleType,
	}
}

// ISO4309GateVars declares the variables for the deployed folded gate.
func ISO4309GateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"diameter_loss_pct": cel.DoubleType,
	}
}

// ISO4309Definition is the standard citation card for ISO 4309.
func ISO4309Definition() DynamicStandardDefinition {
	return DynamicStandardDefinition{
		StandardDID:      "did:integin:standard:iso-4309-2017",
		StandardBody:     "ISO",
		Code:             "4309",
		RevisionYear:     2017,
		Title:            "Cranes — Wire ropes — Care and maintenance, inspection and discard",
		ScopeAbstract:    "Discard criteria for steel wire ropes used on cranes, including outer-wire break counts and diameter loss thresholds.",
		LifecycleState:   LifecycleActive,
		ReplacesStandard: "did:integin:standard:iso-4309-2010",
		OfficialStoreURL: "https://www.iso.org/standard/75886.html",
	}
}
