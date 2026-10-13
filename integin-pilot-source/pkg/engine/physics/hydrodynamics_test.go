package physics

import (
	"math"
	"testing"
)

func TestEvaluateOffshoreDAF(t *testing.T) {
	// Lift 50 tonnes (50,000 kg), K_rig = 2.5e7 N/m, v_rel = 1.5 m/s
	params := OffshoreLiftParams{
		LiftedMassKg:       50000.0,
		RiggingStiffnessNm: 2.5e7,
		RelativeVelocityMs: 1.5,
	}

	res, err := EvaluateOffshoreDAF(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.DAF <= 1.0 {
		t.Fatalf("expected DAF > 1.0, got %v", res.DAF)
	}
	if res.DynamicHookLoadN <= res.StaticHookLoadN {
		t.Fatalf("dynamic load must exceed static load")
	}
	expectedStatic := 50000.0 * StandardGravity
	if math.Abs(res.StaticHookLoadN-expectedStatic) > 1.0 {
		t.Fatalf("expected static load %v, got %v", expectedStatic, res.StaticHookLoadN)
	}

	// Validation guards
	inv1 := params
	inv1.LiftedMassKg = 0
	if _, err := EvaluateOffshoreDAF(inv1); err == nil {
		t.Fatalf("expected error on zero lifted mass")
	}
	inv2 := params
	inv2.RiggingStiffnessNm = -10
	if _, err := EvaluateOffshoreDAF(inv2); err == nil {
		t.Fatalf("expected error on negative stiffness")
	}
	inv3 := params
	inv3.RelativeVelocityMs = -1.0
	if _, err := EvaluateOffshoreDAF(inv3); err == nil {
		t.Fatalf("expected error on negative relative velocity")
	}
}

func TestEvaluateSplashZoneTransition(t *testing.T) {
	params := SplashZoneParams{
		EntryVelocityMs:   2.0,
		ProjectedAreaM2:   10.0,
		DisplacedVolumeM3: 5.0,
		WaterDensityKgM3:  1025.0,
		SlammingCoeff:     math.Pi,
		AddedMassCoeff:    1.2,
		VerticalAccelMs2:  1.5,
	}

	res, err := EvaluateSplashZoneTransition(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSlamming := 0.5 * 1025.0 * math.Pi * 10.0 * 4.0
	if math.Abs(res.SlammingForceN-expectedSlamming) > 1.0 {
		t.Fatalf("expected slamming %v, got %v", expectedSlamming, res.SlammingForceN)
	}

	expectedAddedMass := 1025.0 * 1.2 * 5.0 * 1.5
	if math.Abs(res.AddedMassForceN-expectedAddedMass) > 1.0 {
		t.Fatalf("expected added mass %v, got %v", expectedAddedMass, res.AddedMassForceN)
	}

	// Default fallback values
	paramsDefaults := params
	paramsDefaults.WaterDensityKgM3 = 0
	paramsDefaults.SlammingCoeff = 0
	paramsDefaults.AddedMassCoeff = 0
	resDef, err := EvaluateSplashZoneTransition(paramsDefaults)
	if err != nil {
		t.Fatalf("unexpected error with default fallbacks: %v", err)
	}
	if resDef.SlammingForceN <= 0 || resDef.AddedMassForceN <= 0 {
		t.Fatalf("expected positive default slamming and added mass")
	}

	// Guards
	inv := params
	inv.ProjectedAreaM2 = 0
	if _, err := EvaluateSplashZoneTransition(inv); err == nil {
		t.Fatalf("expected error on zero area")
	}
}

func TestMorisonAndSlackSling(t *testing.T) {
	morison, err := EvaluateMorisonForce(1025.0, 1.2, 4.0, 2.0, 2.0, 1.5, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if morison.TotalForceN <= 0 {
		t.Fatalf("expected positive total Morison force, got %v", morison.TotalForceN)
	}

	// Morison guard
	if _, err := EvaluateMorisonForce(1025.0, 1.2, 0, 2.0, 2.0, 1.5, 0.5); err == nil {
		t.Fatalf("expected error on zero Morison area")
	}
	if _, err := EvaluateMorisonForce(1025.0, 1.2, 4.0, 2.0, 0, 1.5, 0.5); err == nil {
		t.Fatalf("expected error on zero Morison volume")
	}
	morisonDef, err := EvaluateMorisonForce(0, 1.2, 4.0, 2.0, 2.0, 1.5, 0.5)
	if err != nil || morisonDef.TotalForceN <= 0 {
		t.Fatalf("expected valid evaluation with default water density")
	}

	// Slack sling test
	slackRisk, err := EvaluateSlackSlingSnapRisk(10000.0, 8.0, 1025.0, 3.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slackRisk.IsAtRisk {
		t.Fatalf("expected slack risk to be true for high buoyancy/wave acceleration")
	}

	safeRisk, err := EvaluateSlackSlingSnapRisk(50000.0, 2.0, 0, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if safeRisk.IsAtRisk {
		t.Fatalf("expected heavy load to be safe from slack sling")
	}

	// Slack sling guard
	if _, err := EvaluateSlackSlingSnapRisk(0, 2.0, 1025.0, 1.0); err == nil {
		t.Fatalf("expected error on zero lifted mass in slack sling")
	}
}
