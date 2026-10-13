package inspection

import (
	"math"
	"testing"

	"integin/internal/shared/types"
)

func floatPtr(value float64) *float64 { return &value }

func TestValidUnit(t *testing.T) {
	for _, valid := range []string{"Pa", "kPa", "kN", "m", "mm", "cm", "in", "ft", "psi", "ksi", "m/s", "km/h", "deg", "rad", "C", "F", "K", "kg", "lb", "s", "min", "h", "Hz", "J", "bar"} {
		if !ValidUnit(valid) {
			t.Fatalf("unit %q should be recognized", valid)
		}
	}
	for _, invalid := range []string{"", "watts", "P", " mm", "m ", "Kg", "mp/h", "psi**", "strong"} {
		if ValidUnit(invalid) {
			t.Fatalf("unit %q must be rejected", invalid)
		}
	}
}

func TestCanonicalSIConversion(t *testing.T) {
	cases := []struct {
		value  float64
		unit   string
		want   float64
		symbol string
	}{
		{1000, "mm", 1, "m"},
		{1, "in", 0.0254, "m"},
		{1, "ft", 0.3048, "m"},
		{1, "km", 1000, "m"},
		{50, "kN", 50000, "N"},
		{2, "kip", 2 * 4448.2216152605, "N"},
		{1, "psi", psiInPa, "Pa"},
		{1, "ksi", 1e3 * psiInPa, "Pa"},
		{1, "bar", 1e5, "Pa"},
		{60, "km/h", 1000.0 / 60.0, "m/s"},
		{180, "deg", math.Pi, "rad"},
		{2.5, "t", 2500, "kg"},
		{100, "C", 373.15, "K"},
		{32, "F", 273.15, "K"},
		{0, "R", 0, "K"},
		{90, "min", 5400, "s"},
	}
	for _, tc := range cases {
		got, ok := CanonicalSI(tc.value, tc.unit)
		if !ok {
			t.Fatalf("CanonicalSI(%g, %q) rejected valid input", tc.value, tc.unit)
		}
		if got.Unit != tc.symbol {
			t.Fatalf("CanonicalSI(%g, %q) symbol = %q, want %q", tc.value, tc.unit, got.Unit, tc.symbol)
		}
		if math.Abs(got.Value-tc.want) > 1e-9 {
			t.Fatalf("CanonicalSI(%g, %q) = %g, want %g", tc.value, tc.unit, got.Value, tc.want)
		}
		if got.SourceUnit != tc.unit {
			t.Fatalf("CanonicalSI(%g, %q) lost source unit %q", tc.value, tc.unit, got.SourceUnit)
		}
	}
}

func TestCanonicalSIRejectsInvalidInputs(t *testing.T) {
	if _, ok := CanonicalSI(10, "not-a-unit"); ok {
		t.Fatal("unrecognized unit must fail")
	}
	if _, ok := CanonicalSI(math.NaN(), "Pa"); ok {
		t.Fatal("NaN value must fail")
	}
	if _, ok := CanonicalSI(math.Inf(1), "m"); ok {
		t.Fatal("Infinite value must fail")
	}
	if _, ok := CanonicalSI(-300, "C"); ok {
		t.Fatal("temperature below absolute zero must fail")
	}
}

func TestValidateMeasuredValue(t *testing.T) {
	if err := ValidateMeasuredValue(355e6, "Pa"); err != nil {
		t.Fatalf("valid measurement rejected: %v", err)
	}
	if err := ValidateMeasuredValue(5, "kN"); err != nil {
		t.Fatalf("valid measurement rejected: %v", err)
	}
	for _, invalid := range []struct {
		value float64
		unit  string
	}{
		{0, "watts"},
		{math.NaN(), "N"},
		{math.Inf(1), "m/s"},
	} {
		if err := ValidateMeasuredValue(invalid.value, invalid.unit); err == nil {
			t.Fatalf("invalid measurement %g %q accepted", invalid.value, invalid.unit)
		} else {
			domainErr, ok := err.(DomainError)
			if !ok || domainErr.Code != types.ErrValidation {
				t.Fatalf("expected validation DomainError, got %v", err)
			}
		}
	}
}

func TestFindingMeasuredSI(t *testing.T) {
	finding := Finding{ID: "finding-1", MeasuredValue: floatPtr(9.85), MeasuredUnit: "mm"}
	got, ok := finding.MeasuredSI()
	if !ok {
		t.Fatal("valid finding measurement rejected")
	}
	if got.Unit != "m" || math.Abs(got.Value-0.00985) > 1e-12 {
		t.Fatalf("MeasuredSI = %g %s, want 0.00985 m", got.Value, got.Unit)
	}

	var unitless Finding
	if _, ok := unitless.MeasuredSI(); ok {
		t.Fatal("finding without a measurement must fail")
	}

	badUnit := Finding{MeasuredValue: floatPtr(1), MeasuredUnit: "ftlb"}
	if _, ok := badUnit.MeasuredSI(); ok {
		t.Fatal("finding with unrecognized unit must fail")
	}
}
