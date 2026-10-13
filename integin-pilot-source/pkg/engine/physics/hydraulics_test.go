package physics

import (
	"math"
	"testing"
)

func TestHydraulicRam(t *testing.T) {
	// Standard 100mm bore, 50mm rod cylinder, 210 bar (21 MPa)
	params := HydraulicRamParams{
		BoreDiameterM:        0.10, // 100 mm
		RodDiameterM:         0.05, // 50 mm
		StrokeM:              1.0,
		MechanicalEfficiency: 0.95,
	}
	pressurePa := 21e6 // 210 bar

	out, err := EvaluateHydraulicRam(params, pressurePa)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedExtendArea := (math.Pi * 0.01) / 4.0
	if math.Abs(out.ExtendAreaM2-expectedExtendArea) > 1e-6 {
		t.Fatalf("expected extend area %v, got %v", expectedExtendArea, out.ExtendAreaM2)
	}

	expectedThrust := pressurePa * expectedExtendArea * 0.95
	if math.Abs(out.ExtendThrustN-expectedThrust) > 1.0 {
		t.Fatalf("expected thrust %v, got %v", expectedThrust, out.ExtendThrustN)
	}

	// Reverse check: ComputeRequiredRamPressure
	calcPressure, err := ComputeRequiredRamPressure(params, expectedThrust)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(calcPressure-pressurePa) > 1.0 {
		t.Fatalf("expected pressure %v, got %v", pressurePa, calcPressure)
	}

	// Error path tests
	invalidParams := params
	invalidParams.BoreDiameterM = -1.0
	if _, err := EvaluateHydraulicRam(invalidParams, pressurePa); err == nil {
		t.Fatalf("expected error on negative bore diameter")
	}
	if _, err := ComputeRequiredRamPressure(invalidParams, 1000); err == nil {
		t.Fatalf("expected error on negative bore diameter in ComputeRequiredRamPressure")
	}

	invalidParams2 := params
	invalidParams2.RodDiameterM = 0.20 // Rod > bore
	if _, err := EvaluateHydraulicRam(invalidParams2, pressurePa); err == nil {
		t.Fatalf("expected error on rod > bore")
	}

	if _, err := EvaluateHydraulicRam(params, -5.0); err == nil {
		t.Fatalf("expected error on negative pressure")
	}
	if _, err := ComputeRequiredRamPressure(params, -100); err == nil {
		t.Fatalf("expected error on negative thrust")
	}

	// Default efficiency test
	paramsDefEff := params
	paramsDefEff.MechanicalEfficiency = 0
	outDefEff, err := EvaluateHydraulicRam(paramsDefEff, pressurePa)
	if err != nil || outDefEff.ExtendThrustN <= 0 {
		t.Fatalf("expected valid evaluation with default efficiency")
	}
	pDefEff, err := ComputeRequiredRamPressure(paramsDefEff, 1000)
	if err != nil || pDefEff <= 0 {
		t.Fatalf("expected valid pressure with default efficiency")
	}
}

func TestSPMTStability(t *testing.T) {
	// 400t load, 16m wheelbase, 4m track width, 2.5m CoG height
	cfg3 := SPMTConfiguration{
		SupportType: SPMTSupport3Point,
		TotalLoadKg: 400000.0,
		CoGOffsetXM: 0.0,
		CoGOffsetYM: 0.2, // slight transverse eccentricity
		CoGHeightM:  2.5,
		WheelbaseM:  16.0,
		TrackWidthM: 4.0,
	}

	out3, err := EvaluateSPMTStability(cfg3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out3.IsStable {
		t.Fatalf("expected stable 3-point configuration")
	}
	if len(out3.PointLoadsKg) != 3 {
		t.Fatalf("expected 3 point loads, got %d", len(out3.PointLoadsKg))
	}
	sum3 := out3.PointLoadsKg[0] + out3.PointLoadsKg[1] + out3.PointLoadsKg[2]
	if math.Abs(sum3-400000.0) > 1.0 {
		t.Fatalf("expected total load sum 400,000kg, got %v", sum3)
	}

	// 4-point configuration test
	cfg4 := SPMTConfiguration{
		SupportType: SPMTSupport4Point,
		TotalLoadKg: 400000.0,
		CoGOffsetXM: 1.0,
		CoGOffsetYM: 0.0,
		CoGHeightM:  2.5,
		WheelbaseM:  16.0,
		TrackWidthM: 4.0,
	}
	out4, err := EvaluateSPMTStability(cfg4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out4.IsStable {
		t.Fatalf("expected stable 4-point configuration")
	}
	sum4 := out4.PointLoadsKg[0] + out4.PointLoadsKg[1] + out4.PointLoadsKg[2] + out4.PointLoadsKg[3]
	if math.Abs(sum4-400000.0) > 1.0 {
		t.Fatalf("expected total load sum 400,000kg, got %v", sum4)
	}

	// Error & boundary guards
	invalidCfg := cfg4
	invalidCfg.TotalLoadKg = -10
	if _, err := EvaluateSPMTStability(invalidCfg); err == nil {
		t.Fatalf("expected error on negative total load")
	}
	invalidCfg2 := cfg4
	invalidCfg2.WheelbaseM = 0
	if _, err := EvaluateSPMTStability(invalidCfg2); err == nil {
		t.Fatalf("expected error on zero wheelbase")
	}
	invalidCfg3 := cfg4
	invalidCfg3.CoGHeightM = 0
	if _, err := EvaluateSPMTStability(invalidCfg3); err == nil {
		t.Fatalf("expected error on zero cog height")
	}

	// Outside envelope check
	outOfBoundsCfg := cfg4
	outOfBoundsCfg.CoGOffsetXM = 100.0
	if _, err := EvaluateSPMTStability(outOfBoundsCfg); err == nil {
		t.Fatalf("expected error on out of bounds CoG")
	}

	// Lift-off detection test
	cfgLiftOff := cfg4
	cfgLiftOff.CoGOffsetXM = 7.9 // Large offset within envelope but causes quadrant lift-off
	cfgLiftOff.CoGOffsetYM = 1.9
	outLiftOff, err := EvaluateSPMTStability(cfgLiftOff)
	if err != nil {
		t.Fatalf("unexpected error on lift off test: %v", err)
	}
	if outLiftOff.IsStable {
		t.Fatalf("expected unstable due to lift-off")
	}
	if outLiftOff.CriticalAxis != "LIFT_OFF_DETECTED" {
		t.Fatalf("expected LIFT_OFF_DETECTED critical axis, got %s", outLiftOff.CriticalAxis)
	}

	// Unsupported support type check
	unsuppCfg := cfg4
	unsuppCfg.SupportType = "INVALID-TYPE"
	if _, err := EvaluateSPMTStability(unsuppCfg); err == nil {
		t.Fatalf("expected error on unsupported SPMT support type")
	}
}

func TestDarcyWeisbachPressureDrop(t *testing.T) {
	// Laminar flow
	deltaP, re, f, err := DarcyWeisbachPressureDrop(10.0, 0.025, 3.0, 870.0, 0.04, 0.00005)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(re-1631.25) > 0.1 {
		t.Fatalf("expected Re 1631.25, got %v", re)
	}
	expectedF := 64.0 / 1631.25
	if math.Abs(f-expectedF) > 1e-4 {
		t.Fatalf("expected friction factor %v, got %v", expectedF, f)
	}
	if deltaP <= 0 {
		t.Fatalf("expected positive deltaP, got %v", deltaP)
	}

	// Turbulent flow: Water D=0.1m, L=50m, v=2.5m/s, rho=1000, mu=0.001
	// Re = 1000 * 2.5 * 0.1 / 0.001 = 250,000 (turbulent)
	deltaPTurb, reTurb, fTurb, err := DarcyWeisbachPressureDrop(50.0, 0.1, 2.5, 1000.0, 0.001, 0.000045)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reTurb < 2300.0 {
		t.Fatalf("expected turbulent Re, got %v", reTurb)
	}
	if fTurb <= 0 || deltaPTurb <= 0 {
		t.Fatalf("expected positive values for turbulent pipe drop")
	}

	// Invalid input guards
	if _, _, _, err := DarcyWeisbachPressureDrop(0, 0.1, 1, 1000, 0.001, 0); err == nil {
		t.Fatalf("expected error on zero length")
	}
}
