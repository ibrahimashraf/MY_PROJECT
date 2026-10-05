package engine

import (
	"math"
	"testing"

	"integin/pkg/cad/dxf"
	"integin/pkg/engine/physics"
)

func TestPadeyeEvaluation(t *testing.T) {
	// 50 tonne padeye: pin 80mm, hole 82mm, main 50mm, cheeks 2x25mm (50mm), e=120mm, yield 355 MPa
	params := PadeyeParams{
		PinDiameterM:     0.080,
		HoleDiameterM:    0.082,
		MainPlateThickM:  0.050,
		CheekPlateThickM: 0.050,
		EdgeDistanceM:    0.120,
		YieldStrengthPa:  355e6,
		WeldLengthM:      0.60,
		WeldLegSizeM:     0.015,
	}

	loadN := 490500.0 // 50 tonnes
	res, err := EvaluatePadeyeStress(params, loadN)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.PassAllChecks {
		t.Fatalf("expected 50t padeye to pass checks")
	}
	if res.BearingStressPa <= 0 || res.TearOutStressPa <= 0 || res.WeldShearStressPa <= 0 {
		t.Fatalf("expected positive stresses")
	}

	// Error path
	invParams := params
	invParams.EdgeDistanceM = 0.02 // Less than hole radius (0.041m)
	if _, err := EvaluatePadeyeStress(invParams, loadN); err == nil {
		t.Fatalf("expected error when edge distance is less than hole radius")
	}
}

func TestSpreaderBeamEvaluation(t *testing.T) {
	// 12m spreader beam: CHS 406.4 x 16 (A = 0.0196 m^2, I = 0.000378 m^4), Steel S355
	params := SpreaderBeamParams{
		SpanLengthM:      12.0,
		CrossSectionArea: 0.0196,
		MomentOfInertia:  0.000378,
		ElasticModulusPa: 2.1e11,
		YieldStrengthPa:  355e6,
		SelfWeightKgM:    154.0,
	}

	res, err := EvaluateSpreaderBeam(params, 500000.0) // 500 kN compression
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.PassCheck {
		t.Fatalf("expected spreader beam to pass capacity check")
	}
	if res.CombinedUtilization <= 0 || res.CombinedUtilization > 1.0 {
		t.Fatalf("utilization out of range: %v", res.CombinedUtilization)
	}
}

func TestSlingGrommetEvaluation(t *testing.T) {
	params := SlingGrommetParams{
		NominalMBLN:     1000000.0, // 100t MBL -> 20t nominal WLL
		IsCableLaid:     true,      // 0.85 factor -> 17t WLL
		ChokeAngleDeg:   90.0,      // 0.80 factor -> 13.6t WLL (~133.4 kN)
		OperatingTempC:  25.0,
		IsSyntheticHMPE: false,
	}

	res, err := EvaluateSlingGrommet(params, 120000.0) // 120 kN tension
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.PassCheck {
		t.Fatalf("expected sling to pass check")
	}
	if math.Abs(res.EffectiveFactor-(0.85*0.80)) > 1e-4 {
		t.Fatalf("expected effective factor 0.68, got %v", res.EffectiveFactor)
	}

	// Test HMPE thermal cutoff (>65°C)
	paramsHMPE := params
	paramsHMPE.IsSyntheticHMPE = true
	paramsHMPE.OperatingTempC = 70.0
	resHMPE, err := EvaluateSlingGrommet(paramsHMPE, 10000.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resHMPE.PassCheck || !resHMPE.ThermalDerated {
		t.Fatalf("expected HMPE sling to fail due to thermal cutoff")
	}
}

func TestSheaveReevingEvaluation(t *testing.T) {
	params := SheaveReevingParams{
		SheaveDiameterM: 0.80,  // 800 mm
		RopeDiameterM:   0.032, // 32 mm
		FleetAngleDeg:   1.2,   // Safe (< 1.5°)
		IsGroovedDrum:   false,
	}

	res, err := EvaluateSheaveReeving(params, 100000.0) // 100 kN line pull
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.PassAllChecks {
		t.Fatalf("expected sheave to pass checks")
	}

	// Excessive fleet angle
	paramsExcess := params
	paramsExcess.FleetAngleDeg = 2.5
	resExcess, _ := EvaluateSheaveReeving(paramsExcess, 100000.0)
	if resExcess.PassAllChecks || resExcess.FleetAngleValid {
		t.Fatalf("expected fleet angle violation")
	}
}

func TestHookWinklerBachEvaluation(t *testing.T) {
	params := HookWinklerBachParams{
		ThroatRadiusM:   0.15, // 150 mm
		DepthM:          0.20, // 200 mm
		WidthM:          0.12, // 120 mm
		YieldStrengthPa: 400e6,
	}

	res, err := EvaluateHookCrossSection(params, 250000.0) // 25 tonne hook load
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.PassCheck {
		t.Fatalf("expected hook to pass stress check")
	}
	if math.Abs(res.InnerFiberStressPa) <= math.Abs(res.OuterFiberStressPa) {
		t.Fatalf("curved beam theory dictates inner fiber stress exceeds outer fiber stress")
	}
}

func TestIndeterminateRelaxationAndCoupledEnvironmentalLift(t *testing.T) {
	crane := CraneKinematics{
		BasePosition:       dxf.Point3D{X: 0, Y: 0, Z: 0},
		BoomLengthMeters:   50.0,
		BoomAngleDeg:       60.0,
		SlewAngleDeg:       45.0, // Slewed diagonally
		CounterweightTonne: 30.0,
		OutriggerSpreadXM:  10.0,
		OutriggerSpreadZM:  10.0,
		ChassisWeightTonne: 50.0,
	}

	envCfg := CoupledEnvironmentalCraneConfig{
		WindSpeedAt10mMs:   15.0,
		WindTerrain:        physics.TerrainOffshore,
		BoomDragCoeff:      1.2,
		BoomProjectedAreaM: 15.0,
		LoadDragCoeff:      1.4,
		LoadProjectedAreaM: 20.0,
		OffshoreRelativeVM: 1.5,
		RiggingStiffnessNm: 3.0e7,
		IsOffshoreLift:     true,
	}

	res, err := SolveCoupledEnvironmentalLift(crane, 40.0, envCfg, 9.0, 0.30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify offshore DAF was computed dynamically
	if res.DAF <= 1.0 {
		t.Fatalf("expected dynamic offshore DAF > 1.0, got %v", res.DAF)
	}
	if res.EffectiveHookLoadTonne <= 40.0 {
		t.Fatalf("effective hook load must reflect DAF amplification")
	}

	// Verify wind forces computed
	if res.WindDragBoomN <= 0 || res.WindDragLoadN <= 0 {
		t.Fatalf("expected positive aerodynamic drag forces")
	}

	// Verify outrigger sum preserves exact vertical equilibrium
	totalExpected := crane.ChassisWeightTonne + crane.CounterweightTonne + res.EffectiveHookLoadTonne
	totalActual := res.OutriggerReactions.FrontLeft + res.OutriggerReactions.FrontRight +
		res.OutriggerReactions.RearLeft + res.OutriggerReactions.RearRight
	if math.Abs(totalExpected-totalActual) > 1e-4 {
		t.Fatalf("vertical equilibrium violated: expected %v t, got %v t", totalExpected, totalActual)
	}

	// Verify crane mat punch shear
	if res.CraneMatPunchShearPa <= 0 {
		t.Fatalf("expected positive crane mat punch shear")
	}

	// Verify formal proof witness
	if !res.Proof.IsValid {
		t.Fatalf("formal proof witness must be valid: %+v", res.Proof)
	}
	if res.Proof.Residual > 1e-6 {
		t.Fatalf("proof residual %v exceeds tolerance 1e-6", res.Proof.Residual)
	}
}
