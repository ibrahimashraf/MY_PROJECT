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

	// Area = pi * 0.1^2 / 4 ≈ 0.00785398 m^2
	expectedExtendArea := (math.Pi * 0.01) / 4.0
	if math.Abs(out.ExtendAreaM2-expectedExtendArea) > 1e-6 {
		t.Fatalf("expected extend area %v, got %v", expectedExtendArea, out.ExtendAreaM2)
	}

	// Thrust = 21e6 * 0.00785398 * 0.95 ≈ 156.68 kN
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
	// Total load sum check
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
}

func TestDarcyWeisbachPressureDrop(t *testing.T) {
	// Hydraulic oil flow: D=0.025m (1 inch), L=10m, v=3m/s, rho=870 kg/m3, mu=0.04 Pa.s (ISO VG 46)
	deltaP, re, f, err := DarcyWeisbachPressureDrop(10.0, 0.025, 3.0, 870.0, 0.04, 0.00005)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Re = 870 * 3 * 0.025 / 0.04 = 1631.25 (laminar flow)
	if math.Abs(re-1631.25) > 0.1 {
		t.Fatalf("expected Re 1631.25, got %v", re)
	}
	// Laminar f = 64 / 1631.25 ≈ 0.0392336
	expectedF := 64.0 / 1631.25
	if math.Abs(f-expectedF) > 1e-4 {
		t.Fatalf("expected friction factor %v, got %v", expectedF, f)
	}
	if deltaP <= 0 {
		t.Fatalf("expected positive deltaP, got %v", deltaP)
	}
}
