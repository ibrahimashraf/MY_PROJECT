package rulesengine

import (
	"fmt"
	"math"
)

// GroundBearingPressure models the outrigger / mat bearing check:
//
//	sigma_actual = P_i / A_mat   (must be <= sigma_allowable)
//
// where P_i is the vertical load carried by one outrigger pad and A_mat is the
// effective bearing area of the outrigger mat (or directly of the pad when no
// spreader mat is used). A zero or negative area is a hard refusal: a rig
// without bearing area sinks into the ground.
func GroundBearingPressure(padLoadT float64, bearingAreaM2 float64) (actualKPa float64, passed bool, err error) {
	if math.IsNaN(padLoadT) || math.IsNaN(bearingAreaM2) ||
		math.IsInf(padLoadT, 0) || math.IsInf(bearingAreaM2, 0) {
		return 0, false, fmt.Errorf("geotech: NaN/Inf input rejected")
	}
	if padLoadT < 0 {
		return 0, false, fmt.Errorf("geotech: pad load must be >= 0, got %g", padLoadT)
	}
	if bearingAreaM2 <= 0 {
		return 0, false, fmt.Errorf("geotech: bearing area must be > 0 m^2, got %g (no ground support)", bearingAreaM2)
	}
	// 1 tonne-force ~= 9.80665 kN; pressure in kPa = kN / m^2.
	actual := (padLoadT * 9.80665) / bearingAreaM2
	return actual, actual > 0 && !math.IsInf(actual, 0), nil
}

// BearingPressureResult wraps the geotech check together with the allowable
// bound so callers get a single deterministic verdict.
type BearingPressureResult struct {
	ActualKPa      float64 `json:"actual_kpa"`
	AllowableKPa   float64 `json:"allowable_kpa"`
	Passed         bool    `json:"passed"`
	FactorOfSafety float64 `json:"factor_of_safety"`
}

// CheckBearingPressure performs the full ground bearing pressure gate:
// actual pressure must be below allowable with the configured factor of
// safety already folded into `allowable` by the caller.
func CheckBearingPressure(padLoadT float64, bearingAreaM2 float64, allowableKPa float64) (BearingPressureResult, error) {
	actual, ok, err := GroundBearingPressure(padLoadT, bearingAreaM2)
	if err != nil {
		return BearingPressureResult{}, err
	}
	if !ok || allowableKPa <= 0 {
		return BearingPressureResult{
			ActualKPa: actual, AllowableKPa: allowableKPa,
			Passed: false, FactorOfSafety: 0,
		}, nil
	}
	fos := allowableKPa / actual
	if math.IsNaN(fos) || math.IsInf(fos, 0) {
		return BearingPressureResult{ActualKPa: actual, AllowableKPa: allowableKPa, Passed: false, FactorOfSafety: 0}, nil
	}
	return BearingPressureResult{
		ActualKPa: actual, AllowableKPa: allowableKPa,
		Passed: actual <= allowableKPa, FactorOfSafety: fos,
	}, nil
}
