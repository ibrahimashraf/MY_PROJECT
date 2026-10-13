package physics

import (
	"math"
	"testing"
)

func TestEvaluateWindProfile(t *testing.T) {
	// Open country (alpha = 0.14)
	params := WindProfileParams{
		RefVelocityMs: 10.0,
		RefHeightM:    10.0,
		TargetHeightM: 80.0,
		Terrain:       TerrainOpenCountry,
	}

	vz, err := EvaluateWindProfile(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedVz := 10.0 * math.Pow(8.0, 0.14)
	if math.Abs(vz-expectedVz) > 1e-2 {
		t.Fatalf("expected wind speed %v, got %v", expectedVz, vz)
	}

	// Offshore (alpha = 0.11)
	params.Terrain = TerrainOffshore
	vzOffshore, err := EvaluateWindProfile(params)
	if err != nil || vzOffshore <= 0 {
		t.Fatalf("expected offshore profile result, err: %v", err)
	}

	// Suburban (alpha = 0.22)
	params.Terrain = TerrainSuburban
	vzSuburban, err := EvaluateWindProfile(params)
	if err != nil || vzSuburban <= 0 {
		t.Fatalf("expected suburban profile result, err: %v", err)
	}

	// Urban (alpha = 0.33)
	params.Terrain = TerrainUrban
	vzUrban, err := EvaluateWindProfile(params)
	if err != nil || vzUrban <= 0 {
		t.Fatalf("expected urban profile result, err: %v", err)
	}

	// Default fallback terrain
	params.Terrain = "UNKNOWN_TERRAIN"
	vzDef, err := EvaluateWindProfile(params)
	if err != nil || vzDef <= 0 {
		t.Fatalf("expected default profile result, err: %v", err)
	}

	// Guards
	inv1 := params
	inv1.RefVelocityMs = -1.0
	if _, err := EvaluateWindProfile(inv1); err == nil {
		t.Fatalf("expected error on negative ref velocity")
	}
	inv2 := params
	inv2.RefHeightM = 0
	if _, err := EvaluateWindProfile(inv2); err == nil {
		t.Fatalf("expected error on zero ref height")
	}
	inv3 := params
	inv3.TargetHeightM = 0
	if _, err := EvaluateWindProfile(inv3); err == nil {
		t.Fatalf("expected error on zero target height")
	}
}

func TestEvaluateAeroDrag(t *testing.T) {
	// Standard sea-level air, 20 m/s wind, Cd = 1.2 (flat plate/box container), area = 25 m^2
	res, err := EvaluateAeroDrag(1.225, 20.0, 1.2, 25.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDynP := 0.5 * 1.225 * 400.0
	expectedDrag := expectedDynP * 1.2 * 25.0

	if math.Abs(res.DynamicPressurePa-expectedDynP) > 1e-3 {
		t.Fatalf("expected dyn pressure %v, got %v", expectedDynP, res.DynamicPressurePa)
	}
	if math.Abs(res.DragForceN-expectedDrag) > 1e-3 {
		t.Fatalf("expected drag force %v, got %v", expectedDrag, res.DragForceN)
	}

	// Default air density
	resDef, err := EvaluateAeroDrag(0, 20.0, 1.2, 25.0)
	if err != nil || resDef.DragForceN <= 0 {
		t.Fatalf("expected default density drag calculation")
	}

	// Guards
	if _, err := EvaluateAeroDrag(1.225, 20.0, 1.2, 0); err == nil {
		t.Fatalf("expected error on zero area")
	}
	if _, err := EvaluateAeroDrag(1.225, 20.0, 0, 25.0); err == nil {
		t.Fatalf("expected error on zero drag coeff")
	}
}

func TestEvaluateVIVLockIn(t *testing.T) {
	p := VIVParams{
		DiameterM:          0.5,
		WindVelocityMs:     10.0,
		NaturalFrequencyHz: 4.0,
		StrouhalNumber:     0.20,
	}

	res, err := EvaluateVIVLockIn(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.LockInRisk {
		t.Fatalf("expected lock-in risk to be true for ratio 1.0")
	}
	if math.Abs(res.CriticalLockInVelocityMs-10.0) > 1e-3 {
		t.Fatalf("expected critical lock-in velocity 10.0 m/s, got %v", res.CriticalLockInVelocityMs)
	}

	// Non-resonant velocity
	pSafe := p
	pSafe.WindVelocityMs = 2.0 // fs = 0.8 Hz, ratio = 0.2 -> safe
	resSafe, err := EvaluateVIVLockIn(pSafe)
	if err != nil || resSafe.LockInRisk {
		t.Fatalf("expected no lock in risk for low wind velocity")
	}

	// Default Strouhal number
	pDefSt := p
	pDefSt.StrouhalNumber = 0
	resDefSt, err := EvaluateVIVLockIn(pDefSt)
	if err != nil || resDefSt.SheddingFrequencyHz <= 0 {
		t.Fatalf("expected default Strouhal number evaluation")
	}

	// Guards
	inv1 := p
	inv1.DiameterM = 0
	if _, err := EvaluateVIVLockIn(inv1); err == nil {
		t.Fatalf("expected error on zero diameter")
	}
	inv2 := p
	inv2.NaturalFrequencyHz = 0
	if _, err := EvaluateVIVLockIn(inv2); err == nil {
		t.Fatalf("expected error on zero frequency")
	}
	inv3 := p
	inv3.WindVelocityMs = -1.0
	if _, err := EvaluateVIVLockIn(inv3); err == nil {
		t.Fatalf("expected error on negative wind velocity")
	}
}

func TestEvaluatePDeltaAmplification(t *testing.T) {
	p := PDeltaParams{
		BoomLengthM:               40.0,
		ElasticModulusPa:          2.1e11,
		MomentOfInertiaM4:         0.005,
		EffectiveLengthFactor:     1.0,
		AxialCompressiveLoadN:     3.0e6,
		InitialLateralDeflectionM: 0.10,
	}

	res, err := EvaluatePDeltaAmplification(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsBucklingUnstable {
		t.Fatalf("expected stable boom")
	}
	if res.AmplificationFactor <= 1.0 {
		t.Fatalf("expected amplification > 1.0, got %v", res.AmplificationFactor)
	}
	if res.TotalDeflectionM <= 0.10 {
		t.Fatalf("expected total deflection > initial deflection")
	}

	pDefK := p
	pDefK.EffectiveLengthFactor = 0
	resDefK, err := EvaluatePDeltaAmplification(pDefK)
	if err != nil || resDefK.EulerBucklingLoadN <= 0 {
		t.Fatalf("expected valid evaluation with default effective length factor")
	}

	// Test buckling failure condition
	pFail := p
	pFail.AxialCompressiveLoadN = 7.0e6
	resFail, err := EvaluatePDeltaAmplification(pFail)
	if err == nil {
		t.Fatalf("expected error on Euler buckling exceedance")
	}
	if !resFail.IsBucklingUnstable {
		t.Fatalf("expected unstable flag")
	}

	// Guards
	inv1 := p
	inv1.BoomLengthM = 0
	if _, err := EvaluatePDeltaAmplification(inv1); err == nil {
		t.Fatalf("expected error on zero boom length")
	}
	inv2 := p
	inv2.ElasticModulusPa = 0
	if _, err := EvaluatePDeltaAmplification(inv2); err == nil {
		t.Fatalf("expected error on zero modulus")
	}
	inv3 := p
	inv3.AxialCompressiveLoadN = -1
	if _, err := EvaluatePDeltaAmplification(inv3); err == nil {
		t.Fatalf("expected error on negative axial load")
	}
}
