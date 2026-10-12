package loadchart_test

import (
	"testing"

	"integin/pkg/rulesengine/loadchart"
)

func sampleTestChart() loadchart.LoadChart {
	return loadchart.LoadChart{
		CraneModel:        "Grove-GMK6350L",
		CounterweightT:    100.0,
		OutriggerPosition: "100%",
		SlewSectorDeg:     360.0,
		Configurations: []loadchart.BoomConfiguration{
			{
				BoomLengthM: 15.6,
				Ratings: []loadchart.RatingEntry{
					{RadiusM: 3.0, CapacityT: 120.0},
					{RadiusM: 5.0, CapacityT: 85.0},
					{RadiusM: 8.0, CapacityT: 52.0},
					{RadiusM: 12.0, CapacityT: 30.0},
				},
			},
			{
				BoomLengthM: 25.0,
				Ratings: []loadchart.RatingEntry{
					{RadiusM: 5.0, CapacityT: 70.0},
					{RadiusM: 8.0, CapacityT: 45.0},
					{RadiusM: 12.0, CapacityT: 25.0},
					{RadiusM: 16.0, CapacityT: 16.0},
				},
			},
		},
	}
}

func TestLoadChartValidation(t *testing.T) {
	chart := sampleTestChart()
	if err := chart.Validate(); err != nil {
		t.Fatalf("valid chart failed: %v", err)
	}

	invalid := chart
	invalid.CraneModel = ""
	if err := invalid.Validate(); err == nil {
		t.Fatal("empty crane_model must fail")
	}
}

func TestConservativeStepDownLookup(t *testing.T) {
	chart := sampleTestChart()

	// Exact coordinate: Boom 15.6m, Radius 5.0m -> Capacity 85.0t
	capExact, err := chart.LookupConservativeCapacity(15.6, 5.0)
	if err != nil || capExact != 85.0 {
		t.Fatalf("expected exact 85.0t, got %v (err=%v)", capExact, err)
	}

	// Between published radii (Radius = 6.2m between 5.0m & 8.0m):
	// Must step down to larger radius (8.0m) capacity = 52.0t (conservative, no overestimation)
	capStepRadius, err := chart.LookupConservativeCapacity(15.6, 6.2)
	if err != nil || capStepRadius != 52.0 {
		t.Fatalf("expected conservative step-down 52.0t for r=6.2m, got %v (err=%v)", capStepRadius, err)
	}

	// Intermediate boom length (Boom = 20.0m between 15.6m & 25.0m):
	// Must select longer boom (25.0m), and for Radius 6.0m step down to r=8.0m -> 45.0t
	capStepBoom, err := chart.LookupConservativeCapacity(20.0, 6.0)
	if err != nil || capStepBoom != 45.0 {
		t.Fatalf("expected conservative longer-boom 45.0t for boom=20m r=6m, got %v (err=%v)", capStepBoom, err)
	}

	// Exceeded boom length -> must fail
	if _, err := chart.LookupConservativeCapacity(30.0, 5.0); err == nil {
		t.Fatal("requested boom > max boom must fail")
	}

	// Exceeded radius -> must fail
	if _, err := chart.LookupConservativeCapacity(15.6, 20.0); err == nil {
		t.Fatal("requested radius > max radius must fail")
	}
}
