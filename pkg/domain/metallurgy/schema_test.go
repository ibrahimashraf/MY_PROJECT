package metallurgy

import (
	"math"
	"strings"
	"testing"
	"time"
)

func validTestCertificate() MillHeatCertificate {
	return MillHeatCertificate{
		HeatDID:             "did:integin:heat:SA-HADEED-2026-X8199",
		HeatNumber:          "HT-819920",
		SteelMillName:       "HADEED / SABIC",
		MillCountryISO2:     "SA",
		Standard:            CertEN10204_3_1,
		SteelGrade:          "S355ML (Offshore Structural)",
		MeltDate:            time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
		MillInspectorSigner: "QA-INSP-HADEED-041",
		Chemistry: ChemicalComposition{
			CarbonPercent:     0.12,
			SiliconPercent:    0.35,
			ManganesePercent:  1.45,
			PhosphorusPercent: 0.015,
			SulfurPercent:     0.005,
			ChromiumPercent:   0.05,
			NickelPercent:     0.25,
			MolybdenumPercent: 0.02,
			VanadiumPercent:   0.04,
		},
		Mechanicals: MechanicalProperties{
			YieldStrengthMPa:   385.0,
			TensileStrengthMPa: 520.0,
			ElongationPercent:  26.5,
			CharpyImpactJoules: 85.0,
			TestTemperatureC:   -40.0,
		},
	}
}

func TestMillHeatCertificate_Validation(t *testing.T) {
	cert := validTestCertificate()
	if err := cert.Validate(); err != nil {
		t.Fatalf("expected valid certificate, got: %v", err)
	}

	// Verify Carbon Equivalent calculation
	cev := cert.Chemistry.CarbonEquivalent()
	if cev <= 0.30 || cev >= 0.50 {
		t.Errorf("unexpected carbon equivalent calculation: %.4f", cev)
	}

	// Negative path: EN 10204 3.2 without third party inspector
	cert.Standard = CertEN10204_3_2
	cert.ThirdPartyInspector = ""
	if err := cert.Validate(); err == nil || !strings.Contains(err.Error(), "third party") {
		t.Errorf("expected EN 10204 3.2 validation failure for missing 3rd party inspector, got: %v", err)
	}

	// Positive path with third party inspector
	cert.ThirdPartyInspector = "DNV-INSP-2026"
	if err := cert.Validate(); err != nil {
		t.Errorf("expected valid EN 10204 3.2 cert, got: %v", err)
	}
}

func TestPalmgrenMiner_FatigueAccumulation(t *testing.T) {
	fatigueState := CumulativeFatigueState{
		AssetDID: "did:integin:asset:boom-section-01",
		CyclesHistory: []StressCycle{
			{CycleCount: 50000, StressRangeMPa: 120.0, AllowableCyclesNi: 500000.0}, // d1 = 0.10
			{CycleCount: 150000, StressRangeMPa: 100.0, AllowableCyclesNi: 1000000.0}, // d2 = 0.15
			{CycleCount: 20000, StressRangeMPa: 160.0, AllowableCyclesNi: 100000.0},  // d3 = 0.20
		},
	}

	damageIndex, err := fatigueState.CalculateDamageIndex()
	if err != nil {
		t.Fatalf("unexpected damage calculation error: %v", err)
	}
	expectedDamage := 0.10 + 0.15 + 0.20 // 0.45
	if math.Abs(damageIndex-expectedDamage) > 1e-6 {
		t.Errorf("expected damage index %.2f, got %.4f", expectedDamage, damageIndex)
	}

	remainingLife, err := fatigueState.RemainingSafeWorkingLifeFraction()
	if err != nil {
		t.Fatalf("unexpected remaining life error: %v", err)
	}
	expectedRemaining := 0.55
	if math.Abs(remainingLife-expectedRemaining) > 1e-6 {
		t.Errorf("expected remaining life %.2f, got %.4f", expectedRemaining, remainingLife)
	}

	// Exceed structural fatigue failure limit (D >= 1.0)
	fatigueState.CyclesHistory = append(fatigueState.CyclesHistory, StressCycle{
		CycleCount:        60000,
		StressRangeMPa:    160.0,
		AllowableCyclesNi: 100000.0, // d4 = 0.60 -> Total D = 1.05
	})

	_, err = fatigueState.RemainingSafeWorkingLifeFraction()
	if err == nil || !strings.Contains(err.Error(), "damage index") {
		t.Errorf("expected fatigue life exceeded failure error, got: %v", err)
	}
}

func BenchmarkMinerFatigueCalculation_HotPath(b *testing.B) {
	fatigueState := CumulativeFatigueState{
		AssetDID: "did:integin:asset:boom-section-01",
		CyclesHistory: []StressCycle{
			{CycleCount: 50000, StressRangeMPa: 120.0, AllowableCyclesNi: 500000.0},
			{CycleCount: 150000, StressRangeMPa: 100.0, AllowableCyclesNi: 1000000.0},
			{CycleCount: 20000, StressRangeMPa: 160.0, AllowableCyclesNi: 100000.0},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fatigueState.CalculateDamageIndex()
		if err != nil {
			b.Fatal(err)
		}
	}
}
