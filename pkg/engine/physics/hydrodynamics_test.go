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

	// term = sqrt(2.5e7 / (50000 * 9.80665^2)) = sqrt(2.5e7 / 4.8085e6) = sqrt(5.199) ≈ 2.280
	// DAF = 1 + 1.5 * 2.280 ≈ 4.42
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

	// Slamming = 0.5 * 1025 * pi * 10 * 4 = 64402.6 N
	expectedSlamming := 0.5 * 1025.0 * math.Pi * 10.0 * 4.0
	if math.Abs(res.SlammingForceN-expectedSlamming) > 1.0 {
		t.Fatalf("expected slamming %v, got %v", expectedSlamming, res.SlammingForceN)
	}

	// Added mass = 1025 * 1.2 * 5 * 1.5 = 9225 N
	expectedAddedMass := 1025.0 * 1.2 * 5.0 * 1.5
	if math.Abs(res.AddedMassForceN-expectedAddedMass) > 1.0 {
		t.Fatalf("expected added mass %v, got %v", expectedAddedMass, res.AddedMassForceN)
	}
}

func TestMorisonAndSlackSling(t *testing.T) {
	// Morison force
	morison, err := EvaluateMorisonForce(1025.0, 1.2, 4.0, 2.0, 2.0, 1.5, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if morison.TotalForceN <= 0 {
		t.Fatalf("expected positive total Morison force, got %v", morison.TotalForceN)
	}

	// Slack sling test: 10,000 kg load, submerged vol 8.0 m^3 (buoyancy ~ 80,443 N)
	// Gravity = 98,066 N. If upward wave acceleration is 3 m/s^2, inertial = 30,000 N.
	// Net tension = 98,066 - 30,000 - 80,443 = -12,377 N -> SLACK AT RISK!
	slackRisk, err := EvaluateSlackSlingSnapRisk(10000.0, 8.0, 1025.0, 3.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slackRisk.IsAtRisk {
		t.Fatalf("expected slack risk to be true for high buoyancy/wave acceleration")
	}

	// Heavy load: 50,000 kg load, submerged vol 2.0 m^3, upward accel 1.0 m/s^2 -> NOT at risk
	safeRisk, err := EvaluateSlackSlingSnapRisk(50000.0, 2.0, 1025.0, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if safeRisk.IsAtRisk {
		t.Fatalf("expected heavy load to be safe from slack sling")
	}
}
