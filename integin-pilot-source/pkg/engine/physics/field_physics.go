package physics

import (
	"math"
)

// Universal Physical Constants (CODATA standard)
const (
	SpeedOfLightMs          = 299792458.0    // c (m/s)
	StefanBoltzmannConst    = 5.670374419e-8 // sigma (W / (m^2 * K^4))
	AirMolarMassKgMol       = 0.0289644      // M (kg/mol)
	UniversalGasConst       = 8.3144598      // R (J / (mol * K))
	StandardGravity         = 9.80665        // g (m/s^2)
	SeaLevelAirDensity      = 1.225          // rho_0 (kg/m^3)
	SeaLevelPressurePa      = 101325.0       // P_0 (Pa)
	SeaLevelTemperatureK    = 288.15         // T_0 (K) (15°C)
	AdiabaticIndexAir       = 1.4            // gamma (cp/cv)
	StandardSeawaterDensity = 1025.0         // rho_sea (kg/m^3) at surface 15°C
	StandardSalinityPSU     = 35.0           // practical salinity units (g/kg)
	SeawaterRefractiveIndex = 1.333          // n_water at optical frequencies

	// Planetary & Deep Earth Constants (PREM model standard)
	EarthRadiusM                = 6371000.0   // Mean planetary radius (m)
	EarthCoreRadiusM            = 3480000.0   // Core radius (boundary at depth ~2891 km)
	EarthInnerCoreRadiusM       = 1220000.0   // Solid inner core radius (m)
	EarthMassKg                 = 5.972e24    // Total mass (kg)
	EarthCenterPressurePa       = 3.64e11     // Central hydrostatic pressure ~364 GPa
	EarthCenterTempK            = 5700.0      // Central temperature ~5700 K
	UniversalGravitationalConst = 6.67430e-11 // G (m^3 / (kg * s^2))
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

// HydrostaticState captures physical and acoustic state at ocean depth d.
type HydrostaticState struct {
	DepthM         float64
	PressurePa     float64
	DensityKgM3    float64
	SpeedOfSoundMs float64
}

// EvaluateHydrostatic computes hydrostatic pressure and underwater acoustic velocity
// based on standard seawater density and the 9-term Mackenzie sound speed equation.
func EvaluateHydrostatic(depthM float64, tempC float64, salinityPSU float64) HydrostaticState {
	if depthM < 0 {
		depthM = 0
	}
	if salinityPSU <= 0 {
		salinityPSU = StandardSalinityPSU
	}

	// Hydrostatic pressure: P = P0 + rho * g * depth
	pressurePa := SeaLevelPressurePa + (StandardSeawaterDensity * StandardGravity * depthM)

	// Mackenzie (1981) 9-term equation for sound speed in seawater (m/s)
	// c(T,S,D) = 1448.96 + 4.591*T - 5.304e-2*T^2 + 2.374e-4*T^3
	//          + 1.340*(S - 35) + 1.630e-2*D + 1.675e-7*D^2
	//          - 1.025e-2*T*(S - 35) - 7.139e-13*T*D^3
	t := tempC
	t2 := t * t
	t3 := t2 * t
	d := depthM
	d2 := d * d
	d3 := d2 * d
	sDiff := salinityPSU - 35.0

	cMackenzie := 1448.96 +
		4.591*t -
		5.304e-2*t2 +
		2.374e-4*t3 +
		1.340*sDiff +
		1.630e-2*d +
		1.675e-7*d2 -
		1.025e-2*t*sDiff -
		7.139e-13*t*d3

	return HydrostaticState{
		DepthM:         depthM,
		PressurePa:     pressurePa,
		DensityKgM3:    StandardSeawaterDensity,
		SpeedOfSoundMs: cMackenzie,
	}
}

// PlanetaryInteriorState describes geodynamic state at radial distance r from Earth center.
type PlanetaryInteriorState struct {
	RadiusM         float64 // Distance from planet center (m)
	DepthM          float64 // Depth beneath surface (m)
	LayerName       string  // PREM shell name
	DensityKgM3     float64 // Material density
	GravityMs2      float64 // Local gravitational acceleration g(r)
	PressurePa      float64 // Hydrostatic lithostatic pressure P(r)
	TemperatureK    float64 // Geothermal gradient temperature
	PWaveVelocityMs float64 // Compressional seismic velocity V_p
	SWaveVelocityMs float64 // Shear seismic velocity V_s (0 in liquid outer core)
}

// EvaluatePlanetaryCore computes radial geophysical profile from surface to center (PREM model).
func EvaluatePlanetaryCore(radiusFromCenterM float64) PlanetaryInteriorState {
	if radiusFromCenterM < 0 {
		radiusFromCenterM = 0
	}
	if radiusFromCenterM > EarthRadiusM {
		radiusFromCenterM = EarthRadiusM
	}

	depthM := EarthRadiusM - radiusFromCenterM
	rRatio := radiusFromCenterM / EarthRadiusM

	var layer string
	var density, gravity, pressure, temp, vp, vs float64

	switch {
	case radiusFromCenterM <= EarthInnerCoreRadiusM:
		// Solid Inner Core (Fe-Ni alloy crystal)
		layer = "INNER_CORE_SOLID"
		density = 12800.0 + 300.0*(1.0-radiusFromCenterM/EarthInnerCoreRadiusM)
		gravity = StandardGravity * (radiusFromCenterM / EarthInnerCoreRadiusM) * 0.45
		pressure = EarthCenterPressurePa - 3.4e10*(radiusFromCenterM/EarthInnerCoreRadiusM)
		temp = EarthCenterTempK - 300.0*(radiusFromCenterM/EarthInnerCoreRadiusM)
		vp = 11200.0
		vs = 3700.0

	case radiusFromCenterM <= EarthCoreRadiusM:
		// Liquid Outer Core (MHD geodynamo source)
		layer = "OUTER_CORE_LIQUID"
		frac := (radiusFromCenterM - EarthInnerCoreRadiusM) / (EarthCoreRadiusM - EarthInnerCoreRadiusM)
		density = 12100.0 - 2200.0*frac
		gravity = 4.4 + 6.3*frac
		pressure = 3.3e11 - 1.94e11*frac
		temp = 5400.0 - 1400.0*frac
		vp = 10000.0 - 2000.0*frac
		vs = 0.0 // Liquid outer core cannot transmit shear S-waves

	case depthM >= 35000.0:
		// Solid Mantle (Silicate peridotite)
		layer = "MANTLE_SOLID"
		frac := (radiusFromCenterM - EarthCoreRadiusM) / (EarthRadiusM - 35000.0 - EarthCoreRadiusM)
		density = 5500.0 - 2100.0*frac
		gravity = 10.7 - 0.9*frac
		pressure = 1.36e11 * (1.0 - frac)
		temp = 4000.0 - 2500.0*frac
		vp = 13700.0 - 5500.0*frac
		vs = 7300.0 - 2800.0*frac

	default:
		// Lithospheric Crust
		layer = "CRUST_LITHOSPHERE"
		density = 2700.0
		gravity = StandardGravity
		pressure = SeaLevelPressurePa + (density * StandardGravity * depthM)
		temp = SeaLevelTemperatureK + (depthM * 0.025) // ~25 K/km geothermal gradient
		vp = 6000.0
		vs = 3500.0
	}

	_ = rRatio

	return PlanetaryInteriorState{
		RadiusM:         radiusFromCenterM,
		DepthM:          depthM,
		LayerName:       layer,
		DensityKgM3:     density,
		GravityMs2:      gravity,
		PressurePa:      pressure,
		TemperatureK:    temp,
		PWaveVelocityMs: vp,
		SWaveVelocityMs: vs,
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
