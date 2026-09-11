package physics

import (
	"math"
)

// Universal Physical Constants (CODATA standard)
const (
	SpeedOfLightMs        = 299792458.0        // c (m/s)
	StefanBoltzmannConst  = 5.670374419e-8     // sigma (W / (m^2 * K^4))
	AirMolarMassKgMol     = 0.0289644          // M (kg/mol)
	UniversalGasConst     = 8.3144598          // R (J / (mol * K))
	StandardGravity       = 9.80665            // g (m/s^2)
	SeaLevelAirDensity    = 1.225              // rho_0 (kg/m^3)
	SeaLevelPressurePa    = 101325.0           // P_0 (Pa)
	SeaLevelTemperatureK  = 288.15             // T_0 (K) (15°C)
	AdiabaticIndexAir     = 1.4                // gamma (cp/cv)
)

// AtmosphereState captures thermodynamic state at planetary altitude h.
type AtmosphereState struct {
	AltitudeM      float64
	TemperatureK   float64
	PressurePa     float64
	DensityKgM3    float64
	SpeedOfSoundMs float64
}

// EvaluateAtmosphere computes barometric profile with standard adiabatic lapse rate (-6.5 K/km).
func EvaluateAtmosphere(altitudeM float64) AtmosphereState {
	if altitudeM < 0 {
		altitudeM = 0
	}
	const lapseRate = 0.0065 // K/m up to tropopause (11,000m)
	var temp float64
	var press float64

	if altitudeM <= 11000.0 {
		temp = SeaLevelTemperatureK - lapseRate*altitudeM
		// Barometric pressure formula for troposphere:
		exponent := (StandardGravity * AirMolarMassKgMol) / (UniversalGasConst * lapseRate)
		press = SeaLevelPressurePa * math.Pow(temp/SeaLevelTemperatureK, exponent)
	} else {
		// Stratosphere isothermal base
		temp = 216.65
		hDiff := altitudeM - 11000.0
		exponent := -(StandardGravity * AirMolarMassKgMol * hDiff) / (UniversalGasConst * temp)
		p11k := SeaLevelPressurePa * math.Pow(216.65/SeaLevelTemperatureK, (StandardGravity*AirMolarMassKgMol)/(UniversalGasConst*lapseRate))
		press = p11k * math.Exp(exponent)
	}

	density := (press * AirMolarMassKgMol) / (UniversalGasConst * temp)
	// Speed of sound: c = sqrt(gamma * R_spec * T)
	rSpecific := UniversalGasConst / AirMolarMassKgMol
	speedOfSound := math.Sqrt(AdiabaticIndexAir * rSpecific * temp)

	return AtmosphereState{
		AltitudeM:      altitudeM,
		TemperatureK:   temp,
		PressurePa:     press,
		DensityKgM3:    density,
		SpeedOfSoundMs: speedOfSound,
	}
}

// AcousticDopplerShift computes observed frequency under observer/source relative velocities:
// f' = f * (c + v_obs) / (c - v_src)
func AcousticDopplerShift(sourceFreqHz float64, speedOfSound float64, vObserver float64, vSource float64) float64 {
	denom := speedOfSound - vSource
	if math.Abs(denom) < 1e-3 {
		// Mach 1 sonic boom singularity
		return math.Inf(1)
	}
	return sourceFreqHz * (speedOfSound + vObserver) / denom
}

// AcousticSphericalAttenuation computes SPL drop over distance r:
// SPL(r) = SPL_0 - 20 * log10(r / r_0) - alpha * r
func AcousticSphericalAttenuation(initialSPLdB float64, r float64, r0 float64, atmosphericAbsorptionDbPerM float64) float64 {
	if r <= r0 {
		return initialSPLdB
	}
	geomLoss := 20.0 * math.Log10(r/r0)
	atmLoss := atmosphericAbsorptionDbPerM * (r - r0)
	return initialSPLdB - geomLoss - atmLoss
}

// RayleighScatteringCoeff computes wavelength-dependent scattering coefficient:
// beta_R(lambda) = (8 * pi^3 * (n^2 - 1)^2) / (3 * N * lambda^4)
// Demonstrates why the sky is blue (shorter wavelengths scatter dramatically more: ~1/lambda^4).
func RayleighScatteringCoeff(wavelengthM float64) float64 {
	const n = 1.000293 // Refractive index of air
	const N = 2.547e25 // Molecular number density (molecules / m^3)

	n2Minus1 := (n * n) - 1.0
	num := 8.0 * math.Pow(math.Pi, 3) * (n2Minus1 * n2Minus1)
	denom := 3.0 * N * math.Pow(wavelengthM, 4)

	return num / denom
}

// SnellsRefraction computes refracted ray direction via Snell's law:
// n1 * sin(theta1) = n2 * sin(theta2)
// Returns refracted angle and total internal reflection flag.
func SnellsRefraction(theta1Rad float64, n1, n2 float64) (theta2Rad float64, totalInternalReflection bool) {
	sinTheta1 := math.Sin(theta1Rad)
	sinTheta2 := (n1 / n2) * sinTheta1

	if math.Abs(sinTheta2) > 1.0 {
		return 0, true // Total internal reflection
	}
	return math.Asin(sinTheta2), false
}

// BlackbodyRadiantExitance evaluates Stefan-Boltzmann Law:
// M = epsilon * sigma * T^4 (Watts / m^2)
func BlackbodyRadiantExitance(tempK float64, emissivity float64) float64 {
	if tempK <= 0 {
		return 0
	}
	if emissivity <= 0 {
		emissivity = 1.0 // Ideal blackbody
	}
	return emissivity * StefanBoltzmannConst * math.Pow(tempK, 4)
}

// HeatConductionFourier evaluates 1D Fourier thermal conduction:
// q = -k * (T2 - T1) / thicknessM (Watts / m^2)
func HeatConductionFourier(thermalConductivityK float64, tHotK float64, tColdK float64, thicknessM float64) float64 {
	if thicknessM <= 0 {
		return 0
	}
	return thermalConductivityK * (tHotK - tColdK) / thicknessM
}
