package structural

import "math"

// WindSpeedAtHeight scales a reference wind speed up the atmospheric
// boundary layer using the ASTM/ASCE 7 power-law profile:
//
//	v(z) = vRef * (z / z0)^alpha
//
// z0 is the height at which vRef was measured (e.g. a 10 m anemometer
// mast); alpha is the terrain exponent (~0.143 open terrain, ~0.22 urban,
// ~0.33 large cities). Below the reference height the profile is clamped
// to vRef. Physically invalid inputs evaluate to a zero wind speed rather
// than NaN (Hazard 31).
func WindSpeedAtHeight(vRef float64, z float64, z0 float64, alpha float64) float64 {
	if vRef <= 0 || z0 <= 0 || alpha <= 0 {
		return 0.0
	}
	if z <= z0 {
		return vRef
	}
	return vRef * math.Pow(z/z0, alpha)
}

// TurbulenceIntensity evaluates the Eurocode 1 longitudinal turbulence
// intensity I_v(z) = 1 / ln(z / z0) for a roughness length z0. At or below
// the roughness height the boundary layer is fully turbulent and I_v is
// capped at 1.0.
func TurbulenceIntensity(z float64, z0 float64) float64 {
	if z0 <= 0 || z <= z0 {
		return 1.0
	}
	iv := 1.0 / math.Log(z/z0)
	if iv > 1.0 {
		return 1.0
	}
	return iv
}

// GustPeakPressure evaluates the peak dynamic pressure produced by a gust
// factor G = 1 + 7*I_v acting on the mean wind speed:
//
//	q_p = 0.5 * rho * (G * vMean)^2
//
// This is the Eurocode 1 peak pressure used as input to WindLoadAssessment.
func GustPeakPressure(airDensityKgM3 float64, meanSpeedMps float64, turbulenceIntensity float64) float64 {
	if airDensityKgM3 <= 0 {
		airDensityKgM3 = 1.225 // standard sea level air density
	}
	gustFactor := 1.0 + 7.0*turbulenceIntensity
	peakSpeed := gustFactor * meanSpeedMps
	return 0.5 * airDensityKgM3 * peakSpeed * peakSpeed
}
