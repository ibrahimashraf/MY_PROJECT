package relativity

import (
	"math"
)

const (
	SpeedOfLight          = 299792458.0 // c (m/s)
	GravitationalConstant = 6.67430e-11 // G (m^3 / (kg * s^2))
	DefaultHubbleConstant = 70.0        // H_0 (km/s/Mpc)
)

// TimeDilation computes Lorentz time dilation: t' = t / sqrt(1 - v^2/c^2)
func TimeDilation(properTimeS float64, velocityMps float64) (dilatedTimeS float64, lorentzFactor float64) {
	beta := math.Abs(velocityMps) / SpeedOfLight
	if beta >= 1.0 {
		return math.Inf(1), math.Inf(1)
	}
	lorentzFactor = 1.0 / math.Sqrt(1.0-beta*beta)
	dilatedTimeS = properTimeS * lorentzFactor
	return dilatedTimeS, lorentzFactor
}

// LengthContraction computes Lorentz spatial contraction in the direction of motion:
// L = L_0 * sqrt(1 - v^2/c^2) = L_0 / gamma
func LengthContraction(restLengthM float64, velocityMps float64) float64 {
	beta := math.Abs(velocityMps) / SpeedOfLight
	if beta >= 1.0 {
		return 0.0
	}
	return restLengthM * math.Sqrt(1.0-beta*beta)
}

// RelativisticVelocityAddition calculates collinear relativistic velocity composition:
// u' = (u + v) / (1 + (u * v) / c^2)
func RelativisticVelocityAddition(uMps, vMps float64) float64 {
	num := uMps + vMps
	denom := 1.0 + (uMps*vMps)/(SpeedOfLight*SpeedOfLight)
	if denom == 0.0 {
		return math.Inf(1)
	}
	return num / denom
}

// RelativisticEnergy holds invariant mass-energy components per E = gamma * m * c^2.
type RelativisticEnergy struct {
	RestEnergyJ   float64 `json:"rest_energy_j"`    // E_0 = m_0 * c^2
	TotalEnergyJ  float64 `json:"total_energy_j"`   // E = gamma * m_0 * c^2
	KineticEnergy float64 `json:"kinetic_energy_j"` // E_k = (gamma - 1) * m_0 * c^2
	MomentumKgmS  float64 `json:"momentum_kg_m_s"`  // p = gamma * m_0 * v
	Gamma         float64 `json:"gamma"`
}

// RelativisticEnergyMomentum computes Einstein mass-energy equivalence and momentum:
// E^2 = (p*c)^2 + (m_0*c^2)^2
func RelativisticEnergyMomentum(restMassKg, velocityMps float64) RelativisticEnergy {
	c := SpeedOfLight
	c2 := c * c
	restE := restMassKg * c2

	beta := math.Abs(velocityMps) / c
	if beta >= 1.0 {
		return RelativisticEnergy{
			RestEnergyJ:   restE,
			TotalEnergyJ:  math.Inf(1),
			KineticEnergy: math.Inf(1),
			MomentumKgmS:  math.Inf(1),
			Gamma:         math.Inf(1),
		}
	}

	gamma := 1.0 / math.Sqrt(1.0-beta*beta)
	totalE := gamma * restE
	kineticE := (gamma - 1.0) * restE
	momentum := gamma * restMassKg * math.Abs(velocityMps)

	return RelativisticEnergy{
		RestEnergyJ:   restE,
		TotalEnergyJ:  totalE,
		KineticEnergy: kineticE,
		MomentumKgmS:  momentum,
		Gamma:         gamma,
	}
}

// RelativisticDoppler computes longitudinal relativistic frequency shift:
// Approaching: f' = f * sqrt((1 + beta) / (1 - beta))
// Receding:    f' = f * sqrt((1 - beta) / (1 + beta))
func RelativisticDoppler(sourceFreqHz float64, velocityMps float64, isApproaching bool) float64 {
	beta := math.Abs(velocityMps) / SpeedOfLight
	if beta >= 1.0 {
		if isApproaching {
			return math.Inf(1)
		}
		return 0.0
	}
	if isApproaching {
		return sourceFreqHz * math.Sqrt((1.0+beta)/(1.0-beta))
	}
	return sourceFreqHz * math.Sqrt((1.0-beta)/(1.0+beta))
}

// TransverseRelativisticDoppler computes transverse Doppler effect (pure time dilation):
// f' = f / gamma = f * sqrt(1 - beta^2)
func TransverseRelativisticDoppler(sourceFreqHz float64, velocityMps float64) float64 {
	beta := math.Abs(velocityMps) / SpeedOfLight
	if beta >= 1.0 {
		return 0.0
	}
	return sourceFreqHz * math.Sqrt(1.0-beta*beta)
}

// GravitationalTimeDilation computes gravitational redshift factor: t_inf = t_surface * sqrt(1 - rs/r)
// Schwarzschild radius rs = 2GM/c^2
func GravitationalTimeDilation(massKg, radiusM float64) float64 {
	rs := SchwarzschildRadius(massKg)
	if radiusM <= rs {
		return 0.0 // Inside or at event horizon
	}
	return math.Sqrt(1.0 - rs/radiusM)
}

// GravitationalRedshift computes spectral frequency shift z_g = 1 / sqrt(1 - rs/r) - 1.
func GravitationalRedshift(massKg, radiusM float64) float64 {
	factor := GravitationalTimeDilation(massKg, radiusM)
	if factor <= 0.0 {
		return math.Inf(1)
	}
	return (1.0 / factor) - 1.0
}

// EinsteinLightDeflection computes light deflection angle by curved spacetime:
// theta = 4 * G * M / (c^2 * b)
// Exact General Relativity prediction verified in 1919 eclipse (2x Newtonian value).
func EinsteinLightDeflection(centralMassKg, impactParameterM float64) float64 {
	if impactParameterM <= 0 {
		return 0.0
	}
	rs := SchwarzschildRadius(centralMassKg)
	return 2.0 * rs / impactParameterM
}

// EinsteinCouplingConstant evaluates gravitational coupling kappa in Einstein Field Equations:
// G_uv + Lambda * g_uv = kappa * T_uv, where kappa = 8 * pi * G / c^4
func EinsteinCouplingConstant() float64 {
	c4 := math.Pow(SpeedOfLight, 4)
	return (8.0 * math.Pi * GravitationalConstant) / c4
}

// GravitationalWaveQuadrupoleLuminosity computes gravitational radiation power (Einstein 1918):
// P = (G / (5 * c^5)) * (d^3 I / dt^3)^2
func GravitationalWaveQuadrupoleLuminosity(thirdDerivativeMomentInertiaKgM2S3 float64) float64 {
	c5 := math.Pow(SpeedOfLight, 5)
	prefactor := GravitationalConstant / (5.0 * c5)
	return prefactor * math.Pow(thirdDerivativeMomentInertiaKgM2S3, 2)
}

// GeodeticPrecession evaluates geodetic (de Sitter) precession rate (rad/s):
// Omega = (3 * G * M) / (2 * c^2 * r^2) * omega_orbit
func GeodeticPrecession(massKg, orbitalRadiusM, orbitalSpeedMps float64) float64 {
	if orbitalRadiusM <= 0 {
		return 0.0
	}
	c2 := SpeedOfLight * SpeedOfLight
	return (3.0 * GravitationalConstant * massKg * orbitalSpeedMps) / (2.0 * c2 * math.Pow(orbitalRadiusM, 2))
}

// KeplerOrbitalPeriod computes satellite orbital period: T = 2*pi * sqrt(a^3 / (G*M))
func KeplerOrbitalPeriod(semiMajorAxisM, centralMassKg float64) float64 {
	if centralMassKg <= 0 || semiMajorAxisM <= 0 {
		return 0.0
	}
	return 2.0 * math.Pi * math.Sqrt(math.Pow(semiMajorAxisM, 3)/(GravitationalConstant*centralMassKg))
}

// SchwarzschildRadius computes event horizon: rs = 2GM/c^2
func SchwarzschildRadius(massKg float64) float64 {
	if massKg <= 0 {
		return 0.0
	}
	return 2.0 * GravitationalConstant * massKg / (SpeedOfLight * SpeedOfLight)
}

// HubbleRecessionalVelocity computes recessional velocity via Hubble's Law: v = H0 * d
// H0 ≈ 70 km/s/Mpc
func HubbleRecessionalVelocity(distanceMpc float64, h0KmSMpc float64) float64 {
	if h0KmSMpc <= 0 {
		h0KmSMpc = DefaultHubbleConstant
	}
	return h0KmSMpc * 1000.0 * distanceMpc // convert to m/s
}
