package piping

import (
	"testing"
)

func sampleCircuit() CircuitParameters {
	return CircuitParameters{
		CircuitDID:           "did:integin:circuit:refinery-pipe-01",
		OuterDiameterMM:      168.3, // 6-inch NPS pipe
		DesignPressureBar:    50.0,  // 5.0 MPa
		AllowableStressMPa:   138.0, // ASTM A106 Grade B
		JointEfficiencyE:     1.0,   // Seamless
		CoefficientY:         0.4,
		CorrosionAllowanceMM: 3.0,
		NominalThicknessMM:   7.11, // Schedule 40
	}
}

func TestEvaluatePipingCircuit_Safe(t *testing.T) {
	cfg := sampleCircuit()
	pt := MeasurementPoint{
		PointID:              "CML-01",
		CurrentThicknessMM:   6.5,
		PreviousThicknessMM:  6.8,
		YearsBetweenReadings: 2.0, // CR = 0.15 mm/yr
	}

	res, err := EvaluatePipingCircuit(cfg, pt)
	if err != nil {
		t.Fatalf("unexpected assessment error: %v", err)
	}

	if !res.IsSafeToOperate || res.QuarantineRequired {
		t.Fatalf("expected safe operation, got %+v", res)
	}

	if res.MinimumRequiredTMM <= 0 || res.MinimumRequiredTMM >= 6.5 {
		t.Fatalf("unexpected t_min: %f", res.MinimumRequiredTMM)
	}

	if res.RemainingLifeYears <= 0 {
		t.Fatalf("expected positive remaining life, got %f", res.RemainingLifeYears)
	}
}

func TestEvaluatePipingCircuit_QuarantineBreach(t *testing.T) {
	cfg := sampleCircuit()
	// Extreme wall thinning below t_min (~3.0 mm)
	pt := MeasurementPoint{
		PointID:              "CML-02",
		CurrentThicknessMM:   2.5,
		PreviousThicknessMM:  4.0,
		YearsBetweenReadings: 1.0,
	}

	res, err := EvaluatePipingCircuit(cfg, pt)
	if err != ErrWallThicknessCritical {
		t.Fatalf("expected ErrWallThicknessCritical, got %v", err)
	}
	if res.IsSafeToOperate || !res.QuarantineRequired {
		t.Fatalf("expected quarantine required, got %+v", res)
	}
}

func BenchmarkEvaluatePipingCircuit_HotPath(b *testing.B) {
	cfg := sampleCircuit()
	pt := MeasurementPoint{
		PointID:              "CML-BENCH",
		CurrentThicknessMM:   6.0,
		PreviousThicknessMM:  6.5,
		YearsBetweenReadings: 2.0,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = EvaluatePipingCircuit(cfg, pt)
	}
}
