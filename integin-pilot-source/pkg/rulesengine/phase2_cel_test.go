package rulesengine

import (
	"testing"
)

// TestDNVSTN001SKLBoundary checks the full DNV-ST-N001 §5.3 skew load factor
// rule: 1.25 is the inclusive floor for multi-sling lifts, and single-sling
// lifts are not skew-sensitive.
func TestDNVSTN001SKLBoundary(t *testing.T) {
	e, _ := NewEvaluator()
	p := mustCompile(t, e, DNVSTN001RuleID, DNVSKLExpression, DNVSKLVars())
	v := NewVarSet("skl_factor", "sling_count")

	cases := []struct {
		name   string
		skl    float64
		slings int
		want   bool
	}{
		{"multi at floor 1.25", 1.25, 2, true},
		{"multi just below floor", 1.2499, 2, false},
		{"multi generous factor", 1.50, 4, true},
		{"multi gross under-factor", 1.0, 2, false},
		{"single sling no skew gate", 0.5, 1, true},
		{"single sling at zero factor", 0.0, 1, true},
		{"multi negative factor fails", -0.5, 3, false},
	}
	for _, c := range cases {
		v.Put("skl_factor", c.skl)
		v.Put("sling_count", c.slings)
		got, err := p.Evaluate(ctx(), v)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: skl=%v sling_count=%v got %v, want %v", c.name, c.skl, c.slings, got, c.want)
		}
	}
}

// TestDNVSKLGateFoldingMatchesFullForm proves the deployed folded gate is
// verdict-identical to the full form for every multi-sling lift and always
// passes for single-sling lifts.
func TestDNVSKLGateFoldingMatchesFullForm(t *testing.T) {
	e, _ := NewEvaluator()
	gate := mustCompile(t, e, "gate-dnv", DNVSKLGateExpression, DNVSKLGateVars())
	full := mustCompile(t, e, "full-dnv", DNVSKLExpression, DNVSKLVars())
	gv := NewVarSet("skl_factor")
	fv := NewVarSet("skl_factor", "sling_count")

	for _, skl := range []float64{1.0, 1.2499, 1.25, 1.2500001, 2.0} {
		for _, slings := range []int{1, 2, 3, 8} {
			gv.Put("skl_factor", skl)
			gotGate, err := gate.Evaluate(ctx(), gv)
			if err != nil {
				t.Fatalf("gate skl=%v slings=%v: %v", skl, slings, err)
			}
			fv.Put("skl_factor", skl)
			fv.Put("sling_count", slings)
			gotFull, err := full.Evaluate(ctx(), fv)
			if err != nil {
				t.Fatalf("full skl=%v slings=%v: %v", skl, slings, err)
			}
			if slings >= 2 && gotGate != gotFull {
				t.Errorf("multi-sling skl=%v slings=%v: gate=%v full=%v must agree", skl, slings, gotGate, gotFull)
			}
			if slings == 1 && !gotFull {
				t.Errorf("single-sling skl=%v: full form must always pass (no skew sensitivity)", skl)
			}
		}
	}
}

// TestMWSJRPSeaStateCutoff checks the sea-state window boundary: Hs and Tp
// must stay within the approved maxima and may not be negative.
func TestMWSJRPSeaStateCutoff(t *testing.T) {
	e, _ := NewEvaluator()
	p := mustCompile(t, e, MWSJRPSeaStateRuleID, MWSJRPSeaStateExpression, MWSJRPSeaStateVars())
	v := NewVarSet("hs_m", "tp_s", "max_hs_m", "max_tp_s")

	cases := []struct {
		name          string
		hs, tp, maxHs float64
		maxTp         float64
		want          bool
	}{
		{"calm inside window", 0.5, 5.0, 1.5, 8.0, true},
		{"Hs at cutoff inclusive", 1.5, 5.0, 1.5, 8.0, true},
		{"Hs just above cutoff", 1.5001, 5.0, 1.5, 8.0, false},
		{"Tp at cutoff inclusive", 0.5, 8.0, 1.5, 8.0, true},
		{"Tp just above cutoff", 0.5, 8.0001, 1.5, 8.0, false},
		{"both inside", 1.0, 6.0, 1.5, 8.0, true},
		{"both over", 2.0, 10.0, 1.5, 8.0, false},
		{"negative Hs fails closed", -0.2, 5.0, 1.5, 8.0, false},
		{"negative Tp fails closed", 0.5, -1.0, 1.5, 8.0, false},
		{"zero cutoff locks out", 0.5, 5.0, 0.0, 8.0, false},
	}
	for _, c := range cases {
		v.Put("hs_m", c.hs)
		v.Put("tp_s", c.tp)
		v.Put("max_hs_m", c.maxHs)
		v.Put("max_tp_s", c.maxTp)
		got, err := p.Evaluate(ctx(), v)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: hs=%v tp=%v maxHs=%v maxTp=%v got %v, want %v", c.name, c.hs, c.tp, c.maxHs, c.maxTp, got, c.want)
		}
	}
}

// TestDesignBasisASDUtilization checks the canonical ASD capacity
// utilization gate: utilization must lie in [0, 1], matching
// structural.BeamStressAnalysis. The folded device gate must agree for every
// physically valid (non-negative) utilization.
func TestDesignBasisASDUtilization(t *testing.T) {
	e, _ := NewEvaluator()
	ruleID, expr, vars := DesignGate(DesignBasisASD)
	p := mustCompile(t, e, ruleID, expr, vars)
	folded := mustCompile(t, e, "asd-folded", CapacityUtilizationFoldedExpression, CapacityUtilizationVars())
	v := NewVarSet("utilization")
	fv := NewVarSet("utilization")

	cases := []struct {
		util float64
		want bool
	}{
		{0.0, true},
		{0.85, true},
		{1.0, true},     // exactly at capacity: allowed
		{1.0001, false}, // over-utilized
		{-0.1, false},   // physically impossible, fails closed
	}
	for _, c := range cases {
		v.Put("utilization", c.util)
		got, err := p.Evaluate(ctx(), v)
		if err != nil {
			t.Fatalf("util=%v: %v", c.util, err)
		}
		if got != c.want {
			t.Errorf("ASD util=%v got %v, want %v", c.util, got, c.want)
		}

		if c.util >= 0 {
			fv.Put("utilization", c.util)
			gotFolded, err := folded.Evaluate(ctx(), fv)
			if err != nil {
				t.Fatalf("folded util=%v: %v", c.util, err)
			}
			if gotFolded != c.want {
				t.Errorf("folded ASD util=%v got %v, want %v", c.util, gotFolded, c.want)
			}
		}
	}
}

// TestDesignBasisLRFD checks the partial-factor inequality
// phi*R_n >= gamma_D*D + gamma_L*L including the exact-equality boundary,
// and proves the deployed folded gate is verdict-identical.
func TestDesignBasisLRFD(t *testing.T) {
	e, _ := NewEvaluator()
	ruleID, expr, vars := DesignGate(DesignBasisLRFD)
	p := mustCompile(t, e, ruleID, expr, vars)
	folded := mustCompile(t, e, "lrfd-folded", DesignLRFDGateFoldedExpression, DesignLRFDFoldedVars())
	v := NewVarSet("phi", "design_resistance", "gamma_d", "dead_load", "gamma_l", "live_load")
	fv := NewVarSet("factored_resistance", "factored_demand")

	const (
		phi    = 0.90
		gammaD = 1.20
		dead   = 50.0
		gammaL = 1.60
		live   = 10.0
	)
	demand := gammaD*dead + gammaL*live // 76.0
	exact := demand / phi               // 84.444...

	cases := []struct {
		name string
		res  float64
		want bool
	}{
		{"strength beyond demand", 100.0, true},
		{"exact factored equality", exact, true},
		{"just below equality", exact - 0.0001, false},
		{"capacity short of demand", demand * 0.9, false},
	}

	for _, c := range cases {
		v.Put("phi", phi)
		v.Put("design_resistance", c.res)
		v.Put("gamma_d", gammaD)
		v.Put("dead_load", dead)
		v.Put("gamma_l", gammaL)
		v.Put("live_load", live)
		got, err := p.Evaluate(ctx(), v)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: res=%v got %v, want %v", c.name, c.res, got, c.want)
		}

		fv.Put("factored_resistance", phi*c.res)
		fv.Put("factored_demand", demand)
		gotFolded, err := folded.Evaluate(ctx(), fv)
		if err != nil {
			t.Fatalf("%s folded: %v", c.name, err)
		}
		if gotFolded != c.want {
			t.Errorf("%s folded: res=%v got %v, want %v", c.name, c.res, gotFolded, c.want)
		}
	}
}

// TestMWSJRPSeaStateFoldingMatchesFullForm proves the AND of the deployed
// per-parameter gates is verdict-identical to the canonical window form for
// physically valid (non-negative) wave inputs.
func TestMWSJRPSeaStateFoldingMatchesFullForm(t *testing.T) {
	e, _ := NewEvaluator()
	full := mustCompile(t, e, "mws-full", MWSJRPSeaStateExpression, MWSJRPSeaStateVars())
	hs := mustCompile(t, e, "mws-hs", MWSJRPSeaStateHsGateExpression, MWSJRPSeaStateVars())
	tp := mustCompile(t, e, "mws-tp", MWSJRPSeaStateTpGateExpression, MWSJRPSeaStateVars())
	fv := NewVarSet("hs_m", "tp_s", "max_hs_m", "max_tp_s")

	const maxHs, maxTp = 1.5, 8.0
	for _, hsM := range []float64{0.0, 0.9, maxHs, maxHs + 0.4} {
		for _, tpS := range []float64{0.0, 6.0, maxTp, maxTp + 0.4} {
			fv.Put("hs_m", hsM)
			fv.Put("tp_s", tpS)
			fv.Put("max_hs_m", maxHs)
			fv.Put("max_tp_s", maxTp)
			gotFull, err := full.Evaluate(ctx(), fv)
			if err != nil {
				t.Fatalf("full hs=%v tp=%v: %v", hsM, tpS, err)
			}
			gotHs, err := hs.Evaluate(ctx(), fv)
			if err != nil {
				t.Fatalf("hw hs=%v: %v", hsM, err)
			}
			gotTp, err := tp.Evaluate(ctx(), fv)
			if err != nil {
				t.Fatalf("tp tp=%v: %v", tpS, err)
			}
			if gotFull != (gotHs && gotTp) {
				t.Errorf("hs=%v tp=%v: full=%v but hsGate=%v tpGate=%v", hsM, tpS, gotFull, gotHs, gotTp)
			}
		}
	}
}

// TestDesignGateSwitching proves the switch returns distinct, compilable CEL
// cards and rejects unknown bases.
func TestDesignGateSwitching(t *testing.T) {
	e, _ := NewEvaluator()
	asdID, asdExpr, asdVars := DesignGate(DesignBasisASD)
	lrfdID, lrfdExpr, lrfdVars := DesignGate(DesignBasisLRFD)

	if asdID == lrfdID || asdExpr == lrfdExpr {
		t.Fatal("ASD and LRFD must select distinct rule cards")
	}
	if _, err := e.Compile(asdID, asdExpr, asdVars); err != nil {
		t.Fatalf("ASD card must compile: %v", err)
	}
	if _, err := e.Compile(lrfdID, lrfdExpr, lrfdVars); err != nil {
		t.Fatalf("LRFD card must compile: %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("DesignGate with an unknown basis must panic, never silently default")
		}
	}()
	DesignGate(DesignBasis("BOGUS"))
}

// TestStructuralBearingPressureGate mirrors the structural ground bearing
// decision (actual <= allowable) at the kPa boundary.
func TestStructuralBearingPressureGate(t *testing.T) {
	e, _ := NewEvaluator()
	p := mustCompile(t, e, StructuralBearingPressureRuleID, BearingPressureGateExpression, BearingPressureGateVars())
	v := NewVarSet("actual_pressure_kpa", "allowable_pressure_kpa")

	cases := []struct {
		actual, allowable float64
		want              bool
	}{
		{150.0, 150.0, true}, // at allowable: permitted (matches <=)
		{150.0001, 150.0, false},
		{120.0, 200.0, true},
		{-1.0, 200.0, false}, // negative pressure fails closed
	}
	for _, c := range cases {
		v.Put("actual_pressure_kpa", c.actual)
		v.Put("allowable_pressure_kpa", c.allowable)
		got, err := p.Evaluate(ctx(), v)
		if err != nil {
			t.Fatalf("actual=%v allowable=%v: %v", c.actual, c.allowable, err)
		}
		if got != c.want {
			t.Errorf("actual=%v allowable=%v: got %v, want %v", c.actual, c.allowable, got, c.want)
		}
	}
}

// TestStructuralSlingSpreadGate mirrors the MultiLegSlingTension lockout:
// cos(theta from vertical) must be > 0.1 (~84.26 deg).
func TestStructuralSlingSpreadGate(t *testing.T) {
	e, _ := NewEvaluator()
	p := mustCompile(t, e, StructuralSlingSpreadRuleID, SlingSpreadGateExpression, SlingSpreadGateVars())
	v := NewVarSet("cos_leg_factor")

	cases := []struct {
		cos  float64
		want bool
	}{
		{1.0, true},   // vertical sling
		{0.707, true}, // 45 deg from vertical
		{0.1000001, true},
		{0.1, false},  // exactly at the lockout threshold: refused
		{0.05, false}, // flat sling
		{0.0, false},  // pure horizontal pull
	}
	for _, c := range cases {
		v.Put("cos_leg_factor", c.cos)
		got, err := p.Evaluate(ctx(), v)
		if err != nil {
			t.Fatalf("cos=%v: %v", c.cos, err)
		}
		if got != c.want {
			t.Errorf("cos=%v: got %v, want %v", c.cos, got, c.want)
		}
	}
}

// TestStructuralCapacityUtilizationGates proves the beam and tandem-crane
// utilization gates shared the capacity-utilization expression and evaluate
// correctly under both rule IDs.
func TestStructuralCapacityUtilizationGates(t *testing.T) {
	e, _ := NewEvaluator()
	for _, ruleID := range []string{StructuralBeamUtilizationRuleID, StructuralTandemCraneUtilizationRuleID} {
		p := mustCompile(t, e, ruleID, CapacityUtilizationGateExpression, CapacityUtilizationVars())
		v := NewVarSet("utilization")

		v.Put("utilization", 0.95)
		got, err := p.Evaluate(ctx(), v)
		if err != nil {
			t.Fatalf("%s: %v", ruleID, err)
		}
		if !got {
			t.Errorf("%s: 0.95 utilization must pass", ruleID)
		}

		v.Put("utilization", 1.05)
		got, err = p.Evaluate(ctx(), v)
		if err != nil {
			t.Fatalf("%s: %v", ruleID, err)
		}
		if got {
			t.Errorf("%s: 1.05 utilization must fail", ruleID)
		}
	}
}

// TestFoldedGateHotPathZeroAllocations asserts the 0 allocs/op contract on
// every deployed folded device gate (DNV SKL, MWS Hs, MWS Tp, ASD, LRFD)
// with a reused VarSet.
func TestFoldedGateHotPathZeroAllocations(t *testing.T) {
	e, _ := NewEvaluator()

	dnv := mustCompile(t, e, "hot-dnv", DNVSKLGateExpression, DNVSKLGateVars())
	vDNV := NewVarSet("skl_factor")
	vDNV.Put("skl_factor", 1.35)

	if allocs := testing.AllocsPerRun(2000, func() {
		if _, err := dnv.Evaluate(ctx(), vDNV); err != nil {
			t.Fatal(err)
		}
	}); allocs != 0 {
		t.Errorf("DNV folded gate: %g allocs/op, want 0", allocs)
	}

	mwsHs := mustCompile(t, e, "hot-mws-hs", MWSJRPSeaStateHsGateExpression, MWSJRPSeaStateVars())
	mwsTp := mustCompile(t, e, "hot-mws-tp", MWSJRPSeaStateTpGateExpression, MWSJRPSeaStateVars())
	vMWS := NewVarSet("hs_m", "tp_s", "max_hs_m", "max_tp_s")
	vMWS.Put("hs_m", 1.2)
	vMWS.Put("tp_s", 6.5)
	vMWS.Put("max_hs_m", 1.5)
	vMWS.Put("max_tp_s", 8.0)

	for name, gate := range map[string]*Program{"Hs": mwsHs, "Tp": mwsTp} {
		if allocs := testing.AllocsPerRun(2000, func() {
			if _, err := gate.Evaluate(ctx(), vMWS); err != nil {
				t.Fatal(err)
			}
		}); allocs != 0 {
			t.Errorf("MWS %s folded gate: %g allocs/op, want 0", name, allocs)
		}
	}

	asdFolded := mustCompile(t, e, "hot-asd", CapacityUtilizationFoldedExpression, CapacityUtilizationVars())
	vASD := NewVarSet("utilization")
	vASD.Put("utilization", 0.85)

	if allocs := testing.AllocsPerRun(2000, func() {
		if _, err := asdFolded.Evaluate(ctx(), vASD); err != nil {
			t.Fatal(err)
		}
	}); allocs != 0 {
		t.Errorf("ASD folded gate: %g allocs/op, want 0", allocs)
	}

	lrfdFolded := mustCompile(t, e, "hot-lrfd", DesignLRFDGateFoldedExpression, DesignLRFDFoldedVars())
	vLRFD := NewVarSet("factored_resistance", "factored_demand")
	vLRFD.Put("factored_resistance", 90.0)
	vLRFD.Put("factored_demand", 76.0)

	if allocs := testing.AllocsPerRun(2000, func() {
		if _, err := lrfdFolded.Evaluate(ctx(), vLRFD); err != nil {
			t.Fatal(err)
		}
	}); allocs != 0 {
		t.Errorf("LRFD folded gate: %g allocs/op, want 0", allocs)
	}
}
