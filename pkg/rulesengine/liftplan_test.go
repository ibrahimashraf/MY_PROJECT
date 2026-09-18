package rulesengine

import (
	"math"
	"testing"
)

func TestDynamicHookLoad(t *testing.T) {
	// Deck precedent: DHL = W + Wrig, W already carrying the inaccuracy factor.
	w, err := ApplyWeightInaccuracy(20, 0.10)
	if err != nil {
		t.Fatal(err)
	}
	if w != 22 {
		t.Fatalf("W=%v want 22", w)
	}
	dhl, err := DynamicHookLoad(w, 2)
	if err != nil {
		t.Fatal(err)
	}
	if dhl != 24 {
		t.Fatalf("DHL=%v want 24", dhl)
	}
	for _, tc := range []struct {
		name string
		fn   func() error
	}{
		{"zero structure", func() error { _, e := DynamicHookLoad(0, 2); return e }},
		{"zero rigging", func() error { _, e := DynamicHookLoad(22, 0); return e }},
		{"negative rigging", func() error { _, e := DynamicHookLoad(22, -1); return e }},
		{"NaN structure", func() error { _, e := DynamicHookLoad(math.NaN(), 2); return e }},
		{"Inf rigging", func() error { _, e := DynamicHookLoad(22, math.Inf(1)); return e }},
		{"negative inaccuracy", func() error { _, e := ApplyWeightInaccuracy(20, -0.1); return e }},
		{"zero base", func() error { _, e := ApplyWeightInaccuracy(0, 0.1); return e }},
	} {
		if err := tc.fn(); err == nil {
			t.Fatalf("%s: want refusal, got nil", tc.name)
		}
	}
}

func TestEvaluateCapacityGates(t *testing.T) {
	// Crane class cross-check: 20 t chart capacity (B30.5 proof tier 1.25).
	if B30_5ProofLoad(20) != 25 {
		t.Fatal("B30.5 tier changed under the capacity test")
	}
	pass, err := EvaluateCapacityGates(CapacityGates{
		CraneLoadT: 18, CraneCapT: 20,
		RiggingLoadT: 9, RiggingCapT: 12.5,
		SteelLoadT: 18, SteelCapT: 25,
		ObjectLoadT: 18, ObjectCapT: 22,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !pass.Passed || pass.CraneUtil != 0.9 {
		t.Fatalf("want pass util 0.9, got %+v", pass)
	}
	fail, err := EvaluateCapacityGates(CapacityGates{
		CraneLoadT: 21, CraneCapT: 20,
		RiggingLoadT: 9, RiggingCapT: 12.5,
		SteelLoadT: 18, SteelCapT: 25,
		ObjectLoadT: 18, ObjectCapT: 22,
	})
	if err != nil {
		t.Fatal(err)
	}
	if fail.Passed {
		t.Fatalf("want fail at crane util 1.05, got %+v", fail)
	}
	// Boundary: exactly 1.0 passes on every gate.
	edge, err := EvaluateCapacityGates(CapacityGates{
		CraneLoadT: 20, CraneCapT: 20,
		RiggingLoadT: 12.5, RiggingCapT: 12.5,
		SteelLoadT: 25, SteelCapT: 25,
		ObjectLoadT: 22, ObjectCapT: 22,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !edge.Passed {
		t.Fatalf("want pass at util 1.0, got %+v", edge)
	}
	if _, err := EvaluateCapacityGates(CapacityGates{CraneLoadT: 1, CraneCapT: 0}); err == nil {
		t.Fatal("zero capacity must refuse (divisor guard)")
	}
	if _, err := EvaluateCapacityGates(CapacityGates{CraneLoadT: math.NaN(), CraneCapT: 20}); err == nil {
		t.Fatal("NaN load must refuse")
	}
}

func TestShackleLateralLoad(t *testing.T) {
	lat, err := ShackleLateralLoad(100)
	if err != nil {
		t.Fatal(err)
	}
	if lat != 3 {
		t.Fatalf("lateral=%v want 3 (3%% of 100 t)", lat)
	}
	if _, err := ShackleLateralLoad(0); err == nil {
		t.Fatal("zero design load must refuse")
	}
	if _, err := ShackleLateralLoad(math.Inf(1)); err == nil {
		t.Fatal("Inf design load must refuse")
	}
}

func TestGateDAFTableApplicability(t *testing.T) {
	for _, hs := range []float64{0, 1.5, 2.0, 2.5} {
		ok, err := GateDAFTableApplicability(hs)
		if err != nil || !ok {
			t.Fatalf("Hs=%v: want applicable, got %v %v", hs, ok, err)
		}
	}
	// Above the DNV minor-sea-state ceiling: fail-closed, estimate separately.
	if _, err := GateDAFTableApplicability(2.6); err == nil {
		t.Fatal("Hs=2.6 must refuse tabulated DAF")
	}
	if _, err := GateDAFTableApplicability(-1); err == nil {
		t.Fatal("negative Hs must refuse")
	}
	if _, err := GateDAFTableApplicability(math.NaN()); err == nil {
		t.Fatal("NaN Hs must refuse")
	}
}

func TestLookupDAF(t *testing.T) {
	// Illustrative band structure only: real values enter as cited
	// owner-supplied data (the deck's table is raster-locked). The engine
	// enforces structure, ordering, and the Hs discipline — never values.
	table := DAFTable{
		Source: "illustrative-bands/v1",
		Bands:  []DAFBand{{MaxHsM: 1.0, DAF: 1.10}, {MaxHsM: 2.0, DAF: 1.20}, {MaxHsM: 2.5, DAF: 1.30}},
	}
	for _, tc := range []struct {
		hs   float64
		want float64
	}{
		{0, 1.10}, {1.0, 1.10}, {1.5, 1.20}, {2.5, 1.30},
	} {
		got, err := LookupDAF(table, tc.hs)
		if err != nil || got != tc.want {
			t.Fatalf("Hs=%v: want %v, got %v %v", tc.hs, tc.want, got, err)
		}
	}
	for _, tc := range []struct {
		name  string
		table DAFTable
		hs    float64
	}{
		{"above table", table, 2.6},
		{"negative Hs", table, -1},
		{"NaN Hs", table, math.NaN()},
		{"missing source", DAFTable{Bands: table.Bands}, 1.0},
		{"empty bands", DAFTable{Source: "x"}, 1.0},
		{"non-ascending", DAFTable{Source: "x", Bands: []DAFBand{{2.0, 1.20}, {1.0, 1.10}}}, 1.0},
		{"daf below one", DAFTable{Source: "x", Bands: []DAFBand{{2.5, 0.95}}}, 1.0},
	} {
		if _, err := LookupDAF(tc.table, tc.hs); err == nil {
			t.Fatalf("%s: want refusal, got nil", tc.name)
		}
	}
}

func TestApplyDAF(t *testing.T) {
	dyn, err := ApplyDAF(20, 1.20)
	if err != nil {
		t.Fatal(err)
	}
	if dyn != 24 {
		t.Fatalf("dynamic=%v want 24", dyn)
	}
	if _, err := ApplyDAF(20, 0.95); err == nil {
		t.Fatal("DAF below 1.0 must refuse")
	}
	if _, err := ApplyDAF(0, 1.2); err == nil {
		t.Fatal("zero static load must refuse")
	}
	if _, err := ApplyDAF(math.NaN(), 1.2); err == nil {
		t.Fatal("NaN static load must refuse")
	}
}

// TestLorryLoaderXLSGroundPressure cross-validates CheckBearingPressure
// against independently cached results digitized from the reference
// "Loadings - Lorry Loader Calculator.xls" (BIFF FORMULA records): four
// (leg load, ground pressure) pairs, each on an exact 1 m² pad, must
// reproduce within 0.5%.
func TestLorryLoaderXLSGroundPressure(t *testing.T) {
	pairs := []struct {
		legKg      float64
		pressurePa float64
	}{
		{16647.2892, 163.1434},
		{7129.9975, 69.874},
		{28519.9902, 279.7811},
		{26011.3893, 255.1717},
	}
	for _, tc := range pairs {
		got, err := CheckBearingPressure(tc.legKg/1000, 1.0, 1000)
		if err != nil {
			t.Fatal(err)
		}
		if !got.Passed {
			t.Fatalf("leg %.4f kg: gate must pass, got %+v", tc.legKg, got)
		}
		rel := (got.ActualKPa - tc.pressurePa) / tc.pressurePa
		if rel < -0.005 || rel > 0.005 {
			t.Fatalf("leg %.4f kg: engine %.4f kPa vs xls %.4f kPa (rel %.4f)",
				tc.legKg, got.ActualKPa, tc.pressurePa, rel)
		}
	}
}

func TestLiftPlanCELGates(t *testing.T) {
	e, err := NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	dhl := mustCompile(t, e, LiftPlanDHLRuleID, LiftPlanDHLExpression, LiftPlanDHLGateVars())
	v := NewVarSet("dhl_t", "structure_t", "rigging_t")
	v.Put("dhl_t", 24.0)
	v.Put("structure_t", 22.0)
	v.Put("rigging_t", 2.0)
	got, err := dhl.Evaluate(ctx(), v)
	if err != nil || !got {
		t.Fatalf("DHL gate: want true, got %v %v", got, err)
	}
	cap := mustCompile(t, e, LiftPlanCapacityRuleID, LiftPlanCapacityExpression, LiftPlanCapacityGateVars())
	cv := NewVarSet("crane_util", "rigging_util", "steel_util", "object_util")
	cv.Put("crane_util", 0.9)
	cv.Put("rigging_util", 0.72)
	cv.Put("steel_util", 0.72)
	cv.Put("object_util", 0.82)
	got, err = cap.Evaluate(ctx(), cv)
	if err != nil || !got {
		t.Fatalf("capacity gate: want true, got %v %v", got, err)
	}
	cv.Put("crane_util", 1.05)
	got, err = cap.Evaluate(ctx(), cv)
	if err != nil || got {
		t.Fatalf("capacity gate: want false at 1.05, got %v %v", got, err)
	}
	lat := mustCompile(t, e, LiftPlanLateralRuleID, LiftPlanLateralExpression, LiftPlanLateralGateVars())
	lv := NewVarSet("lateral_t", "design_t")
	lv.Put("lateral_t", 3.0)
	lv.Put("design_t", 100.0)
	got, err = lat.Evaluate(ctx(), lv)
	if err != nil || !got {
		t.Fatalf("lateral gate: want true, got %v %v", got, err)
	}
	daf := mustCompile(t, e, LiftPlanDAFRuleID, LiftPlanDAFExpression, LiftPlanDAFGateVars())
	dv := NewVarSet("hs_m", "daf")
	dv.Put("hs_m", 2.0)
	dv.Put("daf", 1.2)
	got, err = daf.Evaluate(ctx(), dv)
	if err != nil || !got {
		t.Fatalf("DAF gate: want true at Hs 2.0, got %v %v", got, err)
	}
	dv.Put("hs_m", 3.0)
	dv.Put("daf", 1.3)
	got, err = daf.Evaluate(ctx(), dv)
	if err != nil || got {
		t.Fatalf("DAF gate: want false at Hs 3.0, got %v %v", got, err)
	}
}

func TestCurzonStAndBBVLiftPlans(t *testing.T) {
	// 1. BBV Curzon St HIAB X-HIPRO 658 EP-6 Lift Plan (BBV LP AR013)
	// Lift A: Pre-cast concrete blocks 2.29 t @ 15m with 2-leg 5t SWL sling at 60 deg
	tension, passSling, lockout, err := SlingTension2Leg(2.29, 60.0)
	if err != nil || !passSling || lockout {
		t.Fatalf("BBV HIAB Block lift sling tension failed: %v", err)
	}
	expectedTension := 2.29 / (2.0 * math.Sin(60.0*math.Pi/180.0))
	if math.Abs(tension-expectedTension) > 1e-6 {
		t.Fatalf("Sling tension got %.4f, want %.4f", tension, expectedTension)
	}
	if tension > 5.0 {
		t.Fatalf("Sling tension %.2f t exceeds 5t SWL", tension)
	}

	// Lift B: 5t Roller (4.92 t @ 8.5m radius vs 6.00 t capacity chart)
	verdictRoller, err := EvaluateCapacityGates(CapacityGates{
		CraneLoadT:   4.92,
		CraneCapT:    6.00,
		RiggingLoadT: 4.92,
		RiggingCapT:  6.70, // 6.7t 4-leg chain
		SteelLoadT:   4.92,
		SteelCapT:    6.00,
		ObjectLoadT:  4.92,
		ObjectCapT:   5.00,
	})
	if err != nil {
		t.Fatalf("EvaluateCapacityGates roller: %v", err)
	}
	if !verdictRoller.Passed {
		t.Fatalf("BBV HIAB 5t roller lift must pass capacity gate, got %+v", verdictRoller)
	}
	if math.Abs(verdictRoller.CraneUtil-0.82) > 0.01 {
		t.Fatalf("Crane utilization got %.4f, want 0.82", verdictRoller.CraneUtil)
	}

	// 2. Curzon St 3 CAT 320 GC Excavator Lift Plan (LP AR016)
	// Concrete Manhole Rings: 6,350 kg (6.35 t) @ 4.5m radius vs 9,900 kg (9.9 t) capacity
	verdictExcavator, err := EvaluateCapacityGates(CapacityGates{
		CraneLoadT:   6.35,
		CraneCapT:    9.90,
		RiggingLoadT: 6.35,
		RiggingCapT:  16.00, // Miller Quick Hitch 16t SWL
		SteelLoadT:   6.35,
		SteelCapT:    9.90,
		ObjectLoadT:  6.35,
		ObjectCapT:   6.40,
	})
	if err != nil {
		t.Fatalf("EvaluateCapacityGates excavator: %v", err)
	}
	if !verdictExcavator.Passed {
		t.Fatalf("CAT 320 GC lift must pass capacity gate, got %+v", verdictExcavator)
	}
	// Verify utilization is ~64% (plan states 64%)
	if math.Abs(verdictExcavator.CraneUtil-0.6414) > 0.005 {
		t.Fatalf("Excavator crane util got %.4f, want 0.6414 (64%%)", verdictExcavator.CraneUtil)
	}

	// 3. Braemar Subsea Deck Lift (pdfcoffee.com_offshore-lifting-for-subsea-equipment)
	// Base 20t + 10% weight inaccuracy -> 22t, Rigging 2t -> DHL = 24t
	wInacc, err := ApplyWeightInaccuracy(20.0, 0.10)
	if err != nil || wInacc != 22.0 {
		t.Fatalf("Weight inaccuracy failed: %v, got %.2f", err, wInacc)
	}
	dhl, err := DynamicHookLoad(wInacc, 2.0)
	if err != nil || dhl != 24.0 {
		t.Fatalf("DHL failed: %v, got %.2f", err, dhl)
	}
	lateral, err := ShackleLateralLoad(dhl)
	if err != nil || math.Abs(lateral-0.72) > 1e-6 {
		t.Fatalf("Lateral load failed: %v, got %.2f", err, lateral)
	}
}
