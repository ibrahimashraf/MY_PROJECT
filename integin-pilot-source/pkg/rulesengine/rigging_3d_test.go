package rulesengine

import (
	"testing"
)

func Test3DRiggingSymmetric4Leg(t *testing.T) {
	// 4 symmetric padeyes at +/- 3m X, +/- 2m Y, Z=0
	padeyes := []PadeyeSpec{
		{
			PadeyeID:              "PE-1",
			Position:              Point3D{X: -3, Y: -2, Z: 0},
			InPlaneAzimuthDeg:     33.69,
			MainPlateThicknessMM:  25,
			CheekPlateThicknessMM: 10,
			CheekOuterDiameterMM:  120,
			CheekFilletLegMM:      8,
			OuterRadiusMM:         90,
			PinHoleDiameterMM:     45,
			BaseWidthMM:           200,
			HoleHeightFromBaseMM:  160,
			BaseWeldThroatMM:      10,
			YieldStrengthMPa:      355,
			UltimateStrengthMPa:   490,
		},
		{
			PadeyeID:              "PE-2",
			Position:              Point3D{X: 3, Y: -2, Z: 0},
			InPlaneAzimuthDeg:     146.31,
			MainPlateThicknessMM:  25,
			CheekPlateThicknessMM: 10,
			CheekOuterDiameterMM:  120,
			CheekFilletLegMM:      8,
			OuterRadiusMM:         90,
			PinHoleDiameterMM:     45,
			BaseWidthMM:           200,
			HoleHeightFromBaseMM:  160,
			BaseWeldThroatMM:      10,
			YieldStrengthMPa:      355,
			UltimateStrengthMPa:   490,
		},
		{
			PadeyeID:              "PE-3",
			Position:              Point3D{X: 3, Y: 2, Z: 0},
			InPlaneAzimuthDeg:     213.69,
			MainPlateThicknessMM:  25,
			CheekPlateThicknessMM: 10,
			CheekOuterDiameterMM:  120,
			CheekFilletLegMM:      8,
			OuterRadiusMM:         90,
			PinHoleDiameterMM:     45,
			BaseWidthMM:           200,
			HoleHeightFromBaseMM:  160,
			BaseWeldThroatMM:      10,
			YieldStrengthMPa:      355,
			UltimateStrengthMPa:   490,
		},
		{
			PadeyeID:              "PE-4",
			Position:              Point3D{X: -3, Y: 2, Z: 0},
			InPlaneAzimuthDeg:     326.31,
			MainPlateThicknessMM:  25,
			CheekPlateThicknessMM: 10,
			CheekOuterDiameterMM:  120,
			CheekFilletLegMM:      8,
			OuterRadiusMM:         90,
			PinHoleDiameterMM:     45,
			BaseWidthMM:           200,
			HoleHeightFromBaseMM:  160,
			BaseWeldThroatMM:      10,
			YieldStrengthMPa:      355,
			UltimateStrengthMPa:   490,
		},
	}

	shackle := ShackleSpec{
		ModelName:        "25t Safety Shackle",
		WLLTonnes:        25.0,
		PinDiameterMM:    42.0,
		InsideJawWidthMM: 55.0,
		SpacerWashersMM:  10.0,
	}

	legs := make([]SlingLegInput, 4)
	for i := 0; i < 4; i++ {
		legs[i] = SlingLegInput{
			LegID:                    padeyes[i].PadeyeID,
			Padeye:                   padeyes[i],
			Shackle:                  shackle,
			SlingWLLTonnes:           30.0,
			AxialStiffnessEAkN:       65000.0,
			LengthToleranceShortenMM: 10.0,
		}
	}

	in := Rigging3DInput{
		NetLoadMassKg:            20000.0, // 20 tonnes
		DynamicAmplification:     1.15,
		MaterialResistanceGammaM: 1.15,
		LateralLoadFraction:      0.05,
		HookElevationZM:          8.0, // Hook at 8m
		CoGEnvelope: CoGEnvelopeInput{
			NominalCoG:        Point3D{X: 0, Y: 0, Z: 0},
			PlusMinusDeltaXM:  0.2, // 200mm CoG shift
			PlusMinusDeltaYM:  0.2,
			PlusMinusDeltaZM:  0.0,
			WeightContingency: 1.05,
		},
		Legs: legs,
	}

	report, err := Solve3DRiggingAndPadeyes(in)
	if err != nil {
		t.Fatalf("Solve3DRiggingAndPadeyes failed: %v", err)
	}

	if !report.IsSafeAndCompliant {
		t.Errorf("Expected compliant rigging, got violations: %v", report.GlobalViolations)
	}

	// Verify that each leg has realistic design tension (between 50 kN and 150 kN for 20t factored lift)
	for _, lr := range report.LegResults {
		if lr.GoverningDesignTensionKN <= 0 {
			t.Errorf("Leg %s: tension must be > 0, got %g", lr.LegID, lr.GoverningDesignTensionKN)
		}
		if lr.SlingUtilizationRatio > 1.0 {
			t.Errorf("Leg %s: sling utilization %.2f exceeds 1.0", lr.LegID, lr.SlingUtilizationRatio)
		}
		if !lr.PadeyeEvaluation.PassedAllChecks {
			t.Errorf("Leg %s: padeye checks failed: %v", lr.LegID, lr.PadeyeEvaluation.WarningsAndViolations)
		}
	}
}

func TestPadeyePinTooLargeRejected(t *testing.T) {
	padeye := PadeyeSpec{
		PadeyeID:             "PE-FAIL",
		Position:             Point3D{X: 0, Y: 0, Z: 0},
		MainPlateThicknessMM: 25,
		PinHoleDiameterMM:    35.0, // Smaller than pin
		BaseWidthMM:          150,
		HoleHeightFromBaseMM: 120,
		OuterRadiusMM:        70,
		YieldStrengthMPa:     355,
		UltimateStrengthMPa:  490,
	}
	shackle := ShackleSpec{
		ModelName:        "Large Shackle",
		WLLTonnes:        25.0,
		PinDiameterMM:    40.0, // 40mm pin into 35mm hole
		InsideJawWidthMM: 50.0,
	}
	res := evaluateDNVPadeye(padeye, shackle, 100.0, 60.0, 0.0, 0.05, 1.15)
	if res.PassedAllChecks {
		t.Errorf("Expected failure when pin diameter > hole diameter")
	}
}
