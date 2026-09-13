package inspection

import (
	"fmt"
	"math"

	"integin/internal/shared/types"
)

// Imperially exact conversion factors (SI definitions, no rounding).
const (
	inPerM  = 0.0254
	lbfInN  = 4.4482216152605
	kgPerLb = 0.45359237
	psiInPa = lbfInN / (inPerM * inPerM)
)

// unitSpec describes how to convert a recorded unit symbol into canonical SI:
//
//	canonical = (measured + offset) * factor
//
// offset is non-zero only for affine temperature scales (Celsius, Fahrenheit).
type unitSpec struct {
	symbol string  // canonical SI unit symbol (e.g. "m", "Pa", "K")
	factor float64 // multiplier to canonical SI
	offset float64 // affine offset to canonical SI
}

// unitTable is the ingress allowlist of engineering unit symbols recognized by
// pkg/units and convertible to canonical SI without loss of precision.
var unitTable = map[string]unitSpec{
	// Length (canonical m)
	"mm": {"m", 1e-3, 0},
	"cm": {"m", 1e-2, 0},
	"m":  {"m", 1, 0},
	"km": {"m", 1e3, 0},
	"in": {"m", inPerM, 0},
	"ft": {"m", 12 * inPerM, 0},
	"yd": {"m", 36 * inPerM, 0},
	"mi": {"m", 12 * inPerM * 5280, 0},
	// Mass (canonical kg)
	"mg": {"kg", 1e-6, 0},
	"g":  {"kg", 1e-3, 0},
	"kg": {"kg", 1, 0},
	"t":  {"kg", 1e3, 0},
	"lb": {"kg", kgPerLb, 0},
	// Force (canonical N)
	"N":   {"N", 1, 0},
	"kN":  {"N", 1e3, 0},
	"MN":  {"N", 1e6, 0},
	"lbf": {"N", lbfInN, 0},
	"kip": {"N", 4448.2216152605, 0},
	// Pressure / stress (canonical Pa)
	"Pa":  {"Pa", 1, 0},
	"kPa": {"Pa", 1e3, 0},
	"MPa": {"Pa", 1e6, 0},
	"GPa": {"Pa", 1e9, 0},
	"bar": {"Pa", 1e5, 0},
	"psi": {"Pa", psiInPa, 0},
	"ksi": {"Pa", 1e3 * psiInPa, 0},
	// Velocity (canonical m/s)
	"m/s":  {"m/s", 1, 0},
	"mm/s": {"m/s", 1e-3, 0},
	"km/h": {"m/s", 5.0 / 18.0, 0},
	"ft/s": {"m/s", 12 * inPerM, 0},
	"mph":  {"m/s", 1609.344 / 3600.0, 0},
	// Time (canonical s)
	"ms":  {"s", 1e-3, 0},
	"s":   {"s", 1, 0},
	"min": {"s", 60, 0},
	"h":   {"s", 3600, 0},
	// Frequency (canonical Hz)
	"Hz":  {"Hz", 1, 0},
	"kHz": {"Hz", 1e3, 0},
	"MHz": {"Hz", 1e6, 0},
	// Angle (canonical rad)
	"rad":  {"rad", 1, 0},
	"deg":  {"rad", math.Pi / 180.0, 0},
	"grad": {"rad", math.Pi / 200.0, 0},
	// Temperature (canonical K)
	"K": {"K", 1, 0},
	"C": {"K", 1, 273.15},
	"F": {"K", 5.0 / 9.0, 459.67},
	"R": {"K", 5.0 / 9.0, 0},
	// Energy (canonical J)
	"J":  {"J", 1, 0},
	"kJ": {"J", 1e3, 0},
	"MJ": {"J", 1e6, 0},
}

// Measurement is a validated measurement expressed in canonical SI units.
type Measurement struct {
	Value      float64 // magnitude in canonical SI
	Unit       string  // canonical SI unit symbol (e.g. "m", "Pa", "K")
	SourceUnit string  // unit symbol as recorded by the inspector
}

// ValidUnit reports whether unit is a recognized engineering unit symbol
// convertible to canonical SI.
func ValidUnit(unit string) bool {
	_, ok := unitTable[unit]
	return ok
}

// CanonicalSI converts a measurement recorded in unit to canonical SI without
// loss of precision. It returns ok=false when the unit is unrecognized, the
// value is non-finite, or the result is physically impossible (below absolute
// zero).
func CanonicalSI(value float64, unit string) (Measurement, bool) {
	spec, ok := unitTable[unit]
	if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
		return Measurement{}, false
	}
	canonical := (value + spec.offset) * spec.factor
	if spec.symbol == "K" && canonical < 0 {
		return Measurement{}, false
	}
	return Measurement{Value: canonical, Unit: spec.symbol, SourceUnit: unit}, true
}

// ValidateMeasuredValue validates a measurement ingressing into the system.
func ValidateMeasuredValue(value float64, unit string) error {
	if _, ok := CanonicalSI(value, unit); !ok {
		return DomainError{Code: types.ErrValidation, Message: fmt.Sprintf("invalid measurement: value=%v unit=%q", value, unit)}
	}
	return nil
}

// MeasuredSI converts the finding's recorded measurement to canonical SI.
// Returns ok=false when the finding carries no measurement or its unit is not
// recognized.
func (f Finding) MeasuredSI() (Measurement, bool) {
	if f.MeasuredValue == nil {
		return Measurement{}, false
	}
	return CanonicalSI(*f.MeasuredValue, f.MeasuredUnit)
}
