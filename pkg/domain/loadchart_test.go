package domain

import (
	"errors"
	"math"
	"testing"
	"time"
)

func validLoadChart() LoadChart {
	effectiveTo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	return LoadChart{
		ID:     "chart-liebherr-ltm1500-r1",
		OEM:    "Liebherr",
		Model:  "LTM 1500",
		Unit:   LoadChartUnitTonne,
		Status: LoadChartDataVerified,
		Provenance: LoadChartProvenance{
			Source:          "OEM load chart",
			DocumentID:      "CH-1500-R1",
			Revision:        "R1",
			RightsReference: "owner-supplied-oem-evidence",
			SHA256:          "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveTo:   &effectiveTo,
		Configurations: []LoadChartConfiguration{
			{
				ID:              "boom-30-cw-20",
				BoomLengthM:     30,
				JibLengthM:      0,
				CounterweightT:  20,
				OutriggerSpanXM: 8.5,
				OutriggerSpanZM: 8.5,
				Points: []LoadChartPoint{
					{RadiusM: 3, CapacityT: 120},
					{RadiusM: 6, CapacityT: 80},
					{RadiusM: 9, CapacityT: 50},
				},
			},
		},
	}
}

func TestLoadChartValidateForAuthority(t *testing.T) {
	chart := validLoadChart()
	if err := chart.ValidateForAuthority(); err != nil {
		t.Fatalf("valid chart rejected: %v", err)
	}
	capacity, err := chart.LookupCapacityExact("boom-30-cw-20", 6)
	if err != nil {
		t.Fatalf("exact lookup: %v", err)
	}
	if capacity != 80 {
		t.Fatalf("capacity = %g, want 80", capacity)
	}
}

func TestLoadChartRefusesDraftAndDemoForAuthority(t *testing.T) {
	for _, status := range []LoadChartDataStatus{LoadChartDataDraft, LoadChartDataDemo} {
		chart := validLoadChart()
		chart.Status = status
		if err := chart.Validate(); err != nil {
			t.Fatalf("%s structural validation: %v", status, err)
		}
		if err := chart.ValidateForAuthority(); !errors.Is(err, ErrLoadChartNotVerified) {
			t.Fatalf("%s authority error = %v", status, err)
		}
	}
}

func TestLoadChartRejectsInvalidProvenance(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*LoadChart)
		want   error
	}{
		{
			name: "missing source",
			mutate: func(chart *LoadChart) {
				chart.Provenance.Source = ""
			},
			want: ErrLoadChartProvenanceRequired,
		},
		{
			name: "short hash",
			mutate: func(chart *LoadChart) {
				chart.Provenance.SHA256 = "abc"
			},
			want: ErrLoadChartHashInvalid,
		},
		{
			name: "uppercase hash",
			mutate: func(chart *LoadChart) {
				chart.Provenance.SHA256 = "0123456789ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef"
			},
			want: ErrLoadChartHashInvalid,
		},
		{
			name: "bad unit",
			mutate: func(chart *LoadChart) {
				chart.Unit = "kg"
			},
			want: ErrLoadChartUnitUnsupported,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chart := validLoadChart()
			test.mutate(&chart)
			if err := chart.Validate(); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestLoadChartRejectsInvalidDatesAndConfigurations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*LoadChart)
		want   error
	}{
		{
			name: "missing effective from",
			mutate: func(chart *LoadChart) {
				chart.EffectiveFrom = time.Time{}
			},
			want: ErrLoadChartEffectiveDateInvalid,
		},
		{
			name: "effective to before from",
			mutate: func(chart *LoadChart) {
				before := chart.EffectiveFrom.Add(-time.Hour)
				chart.EffectiveTo = &before
			},
			want: ErrLoadChartEffectiveDateInvalid,
		},
		{
			name: "duplicate configuration",
			mutate: func(chart *LoadChart) {
				chart.Configurations = append(chart.Configurations, chart.Configurations[0])
			},
			want: ErrLoadChartConfigurationDuplicate,
		},
		{
			name: "invalid span",
			mutate: func(chart *LoadChart) {
				chart.Configurations[0].OutriggerSpanXM = 0
			},
			want: ErrLoadChartConfigurationValues,
		},
		{
			name: "unordered points",
			mutate: func(chart *LoadChart) {
				chart.Configurations[0].Points[0], chart.Configurations[0].Points[1] = chart.Configurations[0].Points[1], chart.Configurations[0].Points[0]
			},
			want: ErrLoadChartPointOrder,
		},
		{
			name: "nonfinite point",
			mutate: func(chart *LoadChart) {
				chart.Configurations[0].Points[0].CapacityT = math.NaN()
			},
			want: ErrLoadChartPointValues,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chart := validLoadChart()
			test.mutate(&chart)
			if err := chart.Validate(); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestLoadChartEffectiveAtAndExactLookupRefusal(t *testing.T) {
	chart := validLoadChart()
	active, err := chart.EffectiveAt(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || !active {
		t.Fatalf("effective date: active=%v err=%v", active, err)
	}
	active, err = chart.EffectiveAt(time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || active {
		t.Fatalf("expired chart: active=%v err=%v", active, err)
	}
	invalid := validLoadChart()
	invalid.Unit = "kg"
	if _, err := invalid.EffectiveAt(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)); !errors.Is(err, ErrLoadChartUnitUnsupported) {
		t.Fatalf("invalid chart effective-date error = %v", err)
	}
	if _, err := chart.LookupCapacityExact("missing", 6); !errors.Is(err, ErrLoadChartConfigurationNotFound) {
		t.Fatalf("missing configuration error = %v", err)
	}
	if _, err := chart.LookupCapacityExact("boom-30-cw-20", 7); !errors.Is(err, ErrLoadChartPointNotFound) {
		t.Fatalf("missing point error = %v", err)
	}
	if _, err := chart.LookupCapacityExact("boom-30-cw-20", math.NaN()); !errors.Is(err, ErrLoadChartPointValues) {
		t.Fatalf("invalid radius error = %v", err)
	}
}
