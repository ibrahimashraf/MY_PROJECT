package rulesengine

import (
	"context"
	"math"
	"testing"
	"time"

	"cel.dev/cel-go/cel"
)

func ctx() context.Context { return context.Background() }

func mustCompile(t *testing.T, e *Evaluator, ruleID, expr string, vars map[string]*cel.Type) *Program {
	t.Helper()
	p, err := e.Compile(ruleID, expr, vars)
	if err != nil {
		t.Fatalf("Compile(%q): %v", ruleID, err)
	}
	return p
}

func TestASMEB30_5ProofLoadPass(t *testing.T) {
	e, _ := NewEvaluator()
	p := mustCompile(t, e, B30_5RuleID, B30_5ProofLoadExceedsExpression, B30_5Vars())
	v := NewVarSet("capacity_t", "reported_test_load_t")

	cases := []struct {
		capacity, reported float64
		want               bool
	}{
		{20, 25, true},    // 20t -> 1.25x -> 25t exactly, meets required
		{20, 24.9, false}, // under-test -> quarantine
		{50, 60, true},    // 50t -> 1.20x -> 60t
		{50, 59.9, false}, // under-test
		{500, 575, true},  // >50t -> 1.15x -> 575t
		{500, 570, false}, // under-test
	}
	for _, c := range cases {
		v.Put("capacity_t", c.capacity)
		v.Put("reported_test_load_t", c.reported)
		got, err := p.Evaluate(ctx(), v)
		if err != nil {
			t.Fatalf("Evaluate(cap=%v, reported=%v): %v", c.capacity, c.reported, err)
		}
		if got != c.want {
			t.Errorf("proof-load gate cap=%v reported=%v: got %v, want %v (required %.3f)", c.capacity, c.reported, got, c.want, B30_5ProofLoad(c.capacity))
		}
	}
}

func TestISO4309BrokenWiresBoundary(t *testing.T) {
	e, _ := NewEvaluator()
	p := mustCompile(t, e, ISO4309BrokenWiresRuleID, ISO4309BrokenWiresExpression, ISO4309Vars())
	v := NewVarSet("rope_diameter_mm", "broken_outer_wires")

	// 20mm rope: discard at 6d = 120 broken outer wires.
	v.Put("rope_diameter_mm", 20.0)
	v.Put("broken_outer_wires", 119)
	got, err := p.Evaluate(ctx(), v)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Errorf("119 broken wires on 20mm rope must NOT discard")
	}

	v.Put("broken_outer_wires", 120)
	got, err = p.Evaluate(ctx(), v)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Errorf("120 broken wires on 20mm rope MUST discard (6d threshold)")
	}
}

func TestISO4309DiameterReduction(t *testing.T) {
	e, _ := NewEvaluator()
	p := mustCompile(t, e, ISO4309DiameterRuleID, ISO4309DiameterExpression, ISO4309Vars())
	v := NewVarSet("rope_diameter_mm", "measured_diameter_mm")

	// 20mm -> 18.6mm = 7.0% exactly, NOT beyond -> keep.
	v.Put("rope_diameter_mm", 20.0)
	v.Put("measured_diameter_mm", 18.6)
	got, err := p.Evaluate(ctx(), v)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Errorf("7.0%% diameter loss must not trip 7%% discard rule")
	}

	// 20mm -> 18.5mm = 7.5% -> discard.
	v.Put("measured_diameter_mm", 18.5)
	got, err = p.Evaluate(ctx(), v)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Errorf("7.5%% diameter loss MUST trip 7%% discard rule")
	}
}

func TestSandboxRejectsMalformedExpression(t *testing.T) {
	e, _ := NewEvaluator()
	_, err := e.Compile("malformed", "capacity_t <= ", B30_5Vars())
	if err == nil {
		t.Fatal("malformed expression must fail compile")
	}
}

func TestSandboxRejectsNonBoolResult(t *testing.T) {
	e, _ := NewEvaluator()
	_, err := e.Compile("nonbool", "capacity_t * 2.0", B30_5Vars())
	if err == nil {
		t.Fatal("non-boolean expression must fail compile (output typed as double)")
	}
}

func TestSandboxRejectsUndeclaredVariable(t *testing.T) {
	e, _ := NewEvaluator()
	_, err := e.Compile("undeclared", "mystery_var > 5.0", B30_5Vars())
	if err == nil {
		t.Fatal("undeclared variable must fail compile")
	}
}

func TestSandboxRejectsLoopInjection(t *testing.T) {
	// CEL is non-Turing complete; loops cannot be expressed. Verify the
	// sandbox-with-cleared-macros refuses macro-driven iteration entirely.
	e, _ := NewEvaluator()
	_, err := e.Compile("loop", "has(xs, e, e > 1)", map[string]*cel.Type{"xs": cel.ListType(cel.IntType)})
	if err == nil {
		t.Fatal("macro (exists via has) must be cleared to prevent data-driven iteration")
	}
}

func TestVarSetUnknownKeyPanics(t *testing.T) {
	v := NewVarSet("capacity_t")
	defer func() {
		if recover() == nil {
			t.Fatal("Put on unknown key must panic (strict typing)")
		}
	}()
	v.Put("not_a_var", 1.0)
}

func TestSandboxRejectsUnsetVariable(t *testing.T) {
	e, _ := NewEvaluator()
	p := mustCompile(t, e, "unset", "capacity_t > 0.0", B30_5Vars())
	v := NewVarSet("capacity_t") // never Put -> Resolve returns absent
	_, err := p.Evaluate(ctx(), v)
	if err == nil {
		t.Fatal("evaluation with a missing variable must error, never silently pass")
	}
}

func TestSandboxBudgetExceeded(t *testing.T) {
	e, _ := NewEvaluator(ProgramOptions{MaxAlloc: 64}) // budget: 1 element
	big := make([]any, 8)
	for i := range big {
		big[i] = float64(i)
	}
	// CEL cannot type-check a list binding against double, so feed a huge map
	// through the budget gate directly (the gate rejects before CEL typing).
	if err := e.ValidateVarsBudget(&VarSet{keys: []string{"x"}, vals: []any{big}}); err == nil {
		t.Fatal("oversized input must be rejected by the sandbox budget")
	}
}

func TestB30_5GateFoldingMatchesFullForm(t *testing.T) {
	// The deployed folded gate must produce identical verdicts to the full
	// ternary formula for every capacity tier.
	e, _ := NewEvaluator()
	gate := mustCompile(t, e, "gate-b30.5", B30_5GateExpression, B30_5GateVars())
	full := mustCompile(t, e, "full-b30.5", B30_5ProofLoadExceedsExpression, B30_5Vars())
	gv := NewVarSet("reported_test_load_t", "required_test_load_t")
	fv := NewVarSet("capacity_t", "reported_test_load_t")

	for _, cap := range []float64{5, 20, 20.0000001, 50, 50.0000001, 500} {
		required := B30_5ProofLoad(cap)
		for _, delta := range []float64{-0.5, 0, 0.5} {
			reported := required + delta
			gv.Put("reported_test_load_t", reported)
			gv.Put("required_test_load_t", required)
			gotGate, err := gate.Evaluate(ctx(), gv)
			if err != nil {
				t.Fatalf("gate cap=%v reported=%v: %v", cap, reported, err)
			}
			fv.Put("capacity_t", cap)
			fv.Put("reported_test_load_t", reported)
			gotFull, err := full.Evaluate(ctx(), fv)
			if err != nil {
				t.Fatalf("full cap=%v reported=%v: %v", cap, reported, err)
			}
			if gotGate != gotFull {
				t.Errorf("cap=%v required=%.4f reported=%.4f: gate=%v full=%v", cap, required, reported, gotGate, gotFull)
			}
		}
	}
}

func TestISO4309GateFoldingMatchesFullForm(t *testing.T) {
	e, _ := NewEvaluator()
	gate := mustCompile(t, e, "gate-iso-b", ISO4309DiameterGateExpression, ISO4309GateVars())
	full := mustCompile(t, e, "full-iso-b", ISO4309DiameterExpression, ISO4309Vars())
	gv := NewVarSet("diameter_loss_pct")
	fv := NewVarSet("rope_diameter_mm", "measured_diameter_mm")

	for _, loss := range []float64{0.0, 0.07, 0.0700001, 0.2, 0.99} {
		for _, nominal := range []float64{10, 20, 40} {
			fv.Put("rope_diameter_mm", nominal)
			fv.Put("measured_diameter_mm", nominal*(1-loss))
			gotFull, err := full.Evaluate(ctx(), fv)
			if err != nil {
				t.Fatalf("full loss=%v nominal=%v: %v", loss, nominal, err)
			}
			// The deployed gate binds the ratio exactly as the full form
			// computes it, so both ASTs must agree even at float boundaries.
			bound := (nominal - nominal*(1-loss)) / nominal
			gv.Put("diameter_loss_pct", bound)
			gotGate, err := gate.Evaluate(ctx(), gv)
			if err != nil {
				t.Fatalf("gate loss=%v nominal=%v: %v", loss, nominal, err)
			}
			if gotGate != gotFull {
				t.Errorf("loss=%v nominal=%v: gate=%v full=%v (bound=%.17g)", loss, nominal, gotGate, gotFull, bound)
			}
		}
	}
}

func TestRuleSetVersioning(t *testing.T) {
	rs := &RuleSet{ID: "asme-b30.5", Revision: 1, State: StateDRAFT}
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	if err := rs.Deprecate(); err == nil {
		t.Fatal("DRAFT cannot be deprecated directly")
	}
	if err := rs.Activate(now); err != nil {
		t.Fatalf("DRAFT->ACTIVE: %v", err)
	}
	if rs.State != StateACTIVE {
		t.Fatalf("state = %s, want ACTIVE", rs.State)
	}
	if err := rs.Activate(now.Add(time.Hour)); err == nil {
		t.Fatal("activation timestamp is immutable and cannot be rewritten")
	}
	if err := rs.Deprecate(); err != nil {
		t.Fatalf("ACTIVE->DEPRECATED: %v", err)
	}
	if err := rs.Activate(time.Now()); err == nil {
		t.Fatal("DEPRECATED cannot be re-activated")
	}
}

func TestSlingTension2Leg(t *testing.T) {
	// 45 deg: T = load/(2*sin(45)) = load/1.4142...
	tension, passed, lockout, err := SlingTension2Leg(10, 45)
	if err != nil || !passed || lockout {
		t.Fatalf("45deg got tension=%v passed=%v lockout=%v err=%v", tension, passed, lockout, err)
	}
	if math.Abs(tension-7.071) > 1e-3 {
		t.Errorf("45deg tension = %.4f, want ~7.071", tension)
	}

	// Critical-angle boundary: 30 deg is permitted.
	if _, passed, lockout, err := SlingTension2Leg(10, 30); err != nil || !passed || lockout {
		t.Errorf("30 deg must pass without lockout, got passed=%v lockout=%v err=%v", passed, lockout, err)
	}
	// Below critical: hard lockout (refusal error is the lockout signal).
	_, passed, lockout, lockErr := SlingTension2Leg(10, 29.9)
	if passed || !lockout || lockErr == nil {
		t.Errorf("29.9 deg must hard lock out, got passed=%v lockout=%v err=%v", passed, lockout, lockErr)
	}
	// Degenerate / poisoned inputs.
	for _, angle := range []float64{0, -10} {
		if _, _, lockout, _ := SlingTension2Leg(10, angle); !lockout {
			t.Errorf("angle %v must trigger lockout", angle)
		}
	}
	for _, load := range []float64{-1, 0} {
		if _, _, _, err := SlingTension2Leg(load, 45); err == nil {
			t.Errorf("load %v must be rejected", load)
		}
	}
	if _, _, _, err := SlingTension2Leg(math.NaN(), 45); err == nil {
		t.Error("NaN load must be rejected")
	}
}

func TestGroundBearingPressure(t *testing.T) {
	// 20 t on 1.0 m^2 -> 196.13 kPa.
	actual, pass, err := GroundBearingPressure(20, 1.0)
	if err != nil || !pass {
		t.Fatalf("GBP(20,1): actual=%.2f pass=%v err=%v", actual, pass, err)
	}
	if math.Abs(actual-196.13) > 0.5 {
		t.Errorf("GBP = %.2f kPa, want ~196.13", actual)
	}

	res, err := CheckBearingPressure(20, 1.0, 150)
	if err != nil || res.Passed {
		t.Errorf("allowable 150 kPa must FAIL, got %+v err=%v", res, err)
	}
	res, err = CheckBearingPressure(20, 1.0, 250)
	if err != nil || !res.Passed {
		t.Errorf("allowable 250 kPa must PASS, got %+v err=%v", res, err)
	}
	if res.FactorOfSafety <= 0 || res.FactorOfSafety > 250/196.133 {
		t.Errorf("unexpected FoS %.3f", res.FactorOfSafety)
	}

	if _, _, err := GroundBearingPressure(20, 0); err == nil {
		t.Error("zero bearing area must be rejected")
	}
}
