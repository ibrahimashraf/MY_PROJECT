package physics

import (
	"math"
	"testing"
)

func TestEvaluateWindProfile(t *testing.T) {
	// 10 m/s at 10m height, evaluate at 80m boom tip in open country (alpha = 0.14)
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

	// vz = 10 * (80 / 10)^0.14 = 10 * 8^0.14 = 10 * 1.3379 ≈ 13.38 m/s
	expectedVz := 10.0 * math.Pow(8.0, 0.14)
	if math.Abs(vz-expectedVz) > 1e-2 {
		t.Fatalf("expected wind speed %v, got %v", expectedVz, vz)
	}
}

func TestEvaluateAeroDrag(t *testing.T) {
	// Standard sea-level air, 20 m/s wind, Cd = 1.2 (flat plate/box container), area = 25 m^2
	res, err := EvaluateAeroDrag(1.225, 20.0, 1.2, 25.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// q = 0.5 * 1.225 * 400 = 245 Pa
	// Fd = 245 * 1.2 * 25 = 7350 N
	expectedDynP := 0.5 * 1.225 * 400.0
	expectedDrag := expectedDynP * 1.2 * 25.0

	if math.Abs(res.DynamicPressurePa-expectedDynP) > 1e-3 {
		t.Fatalf("expected dyn pressure %v, got %v", expectedDynP, res.DynamicPressurePa)
	}
	if math.Abs(res.DragForceN-expectedDrag) > 1e-3 {
		t.Fatalf("expected drag force %v, got %v", expectedDrag, res.DragForceN)
	}
}

func TestEvaluateVIVLockIn(t *testing.T) {
	// Circular boom chord: D = 0.5m, wind = 10 m/s, fn = 4.0 Hz, St = 0.20
	// fs = 0.20 * 10 / 0.5 = 4.0 Hz
	// ratio = 4.0 / 4.0 = 1.0 -> LOCK-IN RISK!
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
}

func TestEvaluatePDeltaAmplification(t *testing.T) {
	// Steel tubular boom: L = 40m, E = 2.1e11 Pa, I = 0.005 m^4, K = 1.0
	// P_Euler = pi^2 * 2.1e11 * 0.005 / 40^2 = 1.036e10 / 1600 = 6,476,707 N
	p := PDeltaParams{
		BoomLengthM:               40.0,
		ElasticModulusPa:          2.1e11,
		MomentOfInertiaM4:         0.005,
		EffectiveLengthFactor:     1.0,
		AxialCompressiveLoadN:     3.0e6, // ~3000 kN (less than buckling load)
		InitialLateralDeflectionM: 0.10,  // 100 mm initial wind sag
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

	// Test buckling failure condition
	pFail := p
	pFail.AxialCompressiveLoadN = 7.0e6 // Exceeds ~6.48 MN
	resFail, err := EvaluatePDeltaAmplification(pFail)
	if err == nil {
		t.Fatalf("expected error on Euler buckling exceedance")
	}
	if !resFail.IsBucklingUnstable {
		t.Fatalf("expected unstable flag")
	}
}
