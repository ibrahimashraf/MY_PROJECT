package rulesengine

import "cel.dev/cel-go/cel"

// CraneLMIOperationalGateRuleID identifies the statutory crane LMI
// operational gate used by pkg/telemetry's SCADA ingress bridge.
const CraneLMIOperationalGateRuleID = "crane-lmi.operational-gate"

// CraneLMIOperationalGateExpression is the Google CEL form of the statutory
// limit-state check applied to every live cranL-MI telemetry sample:
//
//	safe = moment_utilization_pct <= 90.0 && !anti_two_block_triggered
//
// The 90.0% moment-utilization ceiling and the anti-two-block interlock are
// rule data evaluated inside the CEL sandbox (invariant 5: zero LLM math on
// safety gates). The CEL AST is the single source of truth; no Go-side
// numerical copy of the threshold exists anywhere else.
const CraneLMIOperationalGateExpression = `moment_utilization_pct <= 90.0 && !anti_two_block_triggered`

// CraneLMIOperationalGateVars declares the typed variables bound to the
// operational gate. moment_utilization_pct is reported by the crane's own
// LMI computer (never computed here); anti_two_block_triggered is the interlock
// latch from the load-block proximity sensor.
func CraneLMIOperationalGateVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"moment_utilization_pct":   cel.DoubleType,
		"anti_two_block_triggered": cel.BoolType,
	}
}
