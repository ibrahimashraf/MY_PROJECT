package structural

import (
	"math"
	"testing"
)

func TestStructuralAssemblyWeightAndCoG(t *testing.T) {
	steel := Material{
		Name:            "S355 Structural Steel",
		DensityKgM3:     7850.0,
		YieldStrengthPa: 355e6,
		SafetyFactor:    1.5,
	}

	// Body 1: 2m x 1m x 1m block at origin (Volume = 2 m^3, Mass = 15,700 kg, CoG = (1, 0.5, 0.5))
	b1 := StructuralBody{
		ID:       "counterweight-1",
		Material: steel,
		Size:     BoundingBox{LengthM: 2.0, WidthM: 1.0, HeightM: 1.0},
		Position: [3]float64{0, 0, 0},
		LocalCoG: [3]float64{1.0, 0.5, 0.5},
	}

	// Body 2: Identical block placed at X=2 (Volume = 2 m^3, Mass = 15,700 kg, CoG = (3, 0.5, 0.5))
	b2 := StructuralBody{
		ID:       "counterweight-2",
		Material: steel,
		Size:     BoundingBox{LengthM: 2.0, WidthM: 1.0, HeightM: 1.0},
		Position: [3]float64{2.0, 0, 0},
		LocalCoG: [3]float64{1.0, 0.5, 0.5},
	}

	assembly := MultiBodyAssembly{Bodies: []StructuralBody{b1, b2}}
	mass, weight, cog, err := assembly.AggregateProperties()
	if err != nil {
		t.Fatalf("AggregateProperties failed: %v", err)
	}

	expectedMass := 15700.0 * 2.0
	if math.Abs(mass-expectedMass) > 1e-3 {
		t.Fatalf("Mass mismatch: expected %v, got %v", expectedMass, mass)
	}

	expectedWeight := expectedMass * 9.80665
	if math.Abs(weight-expectedWeight) > 1e-3 {
		t.Fatalf("Weight mismatch: expected %v, got %v", expectedWeight, weight)
	}

	// Combined CoG X must be exactly at midpoint (2.0)
	if math.Abs(cog[0]-2.0) > 1e-6 || math.Abs(cog[1]-0.5) > 1e-6 || math.Abs(cog[2]-0.5) > 1e-6 {
		t.Fatalf("CoG mismatch: expected (2.0, 0.5, 0.5), got %v", cog)
	}
}

func TestGroundBearingPressure(t *testing.T) {
	// Pad: 2m x 2m steel spreader mat = 4.0 m^2
	// Allowable soil bearing capacity = 150 kPa = 150,000 Pa
	pad := OutriggerPad{
		ID:          "pad-front-left",
		AreaM2:      4.0,
		AllowablePa: 150000.0,
	}

	// Reaction force 400 kN -> 100 kPa -> SAFE
	press, safe := GroundBearingPressure(400000.0, pad)
	if !safe || math.Abs(press-100000.0) > 1e-3 {
		t.Fatalf("Expected safe 100 kPa, got %v (safe=%v)", press, safe)
	}

	// Reaction force 800 kN -> 200 kPa -> UNSAFE (ground failure hazard)
	pressOver, safeOver := GroundBearingPressure(800000.0, pad)
	if safeOver || math.Abs(pressOver-200000.0) > 1e-3 {
		t.Fatalf("Expected unsafe 200 kPa, got %v (safe=%v)", pressOver, safeOver)
	}
}

func TestWindLoadAssessment(t *testing.T) {
	// Wind speed: 20 m/s (~72 km/h gale force)
	// Exposed boom area: 25 m^2
	// Drag coefficient: 1.2
	windSpeed := 20.0
	area := 25.0
	cd := 1.2
	rho := 1.225

	forceN, qPa := WindLoadAssessment(windSpeed, rho, area, cd)

	// q = 0.5 * 1.225 * 400 = 245 Pa
	expectedQ := 245.0
	if math.Abs(qPa-expectedQ) > 1e-3 {
		t.Fatalf("Dynamic pressure mismatch: expected %v, got %v", expectedQ, qPa)
	}

	// F = 245 * 25 * 1.2 = 7350 N
	expectedForce := 7350.0
	if math.Abs(forceN-expectedForce) > 1e-3 {
		t.Fatalf("Wind force mismatch: expected %v, got %v", expectedForce, forceN)
	}

	// Combined overturning moment with 15m lever arm
	weightN := 200000.0 // 200 kN
	eccM := 1.5         // 1.5m eccentricity
	heightM := 15.0     // 15m wind center of pressure

	moment := CombinedOverturningMoment(weightN, eccM, forceN, heightM)
	// Gravity moment = 300,000 Nm; Wind moment = 7350 * 15 = 110,250 Nm -> Total = 410,250 Nm
	expectedMoment := 410250.0
	if math.Abs(moment-expectedMoment) > 1e-3 {
		t.Fatalf("Overturning moment mismatch: expected %v, got %v", expectedMoment, moment)
	}
}

func TestBeamStressAnalysis(t *testing.T) {
	steel := Material{
		Name:            "S355 Steel",
		DensityKgM3:     7850.0,
		YieldStrengthPa: 355e6, // 355 MPa
		SafetyFactor:    1.5,   // Allowable stress = 355/1.5 = 236.67 MPa
	}

	// Rectangular beam: width b = 0.3m, height h = 0.6m
	// W = b * h^2 / 6 = 0.3 * 0.36 / 6 = 0.018 m^3
	bendingMoment := 2e6 // 2 MNm -> sigma = 2e6 / 0.018 ≈ 111.11 MPa
	stress, util, safe := BeamStressAnalysis(bendingMoment, 0.3, 0.6, steel)

	if !safe {
		t.Fatalf("Beam should be safe under 2 MNm moment: stress=%v, util=%v", stress, util)
	}
	expectedStress := 2e6 / 0.018
	if math.Abs(stress-expectedStress) > 1.0 {
		t.Fatalf("Stress calculation mismatch: expected %v, got %v", expectedStress, stress)
	}
}
