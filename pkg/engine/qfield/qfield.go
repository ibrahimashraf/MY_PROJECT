package qfield

import "math"

// Planck and related quantum constants
const (
	HBar              = 1.054571817e-34 // ℏ (J·s)
	ElectronMassKg    = 9.1093837015e-31
	ElementaryCharge  = 1.602176634e-19  // C
	VacuumPermittivity = 8.8541878e-12   // ε₀ (F/m)
	BohrRadius        = 5.29177210903e-11 // a₀ (m)
)

// CasimirForce computes attractive force per unit area between two parallel conducting plates:
// F/A = -(π²ℏc) / (240 d^4)
func CasimirForce(separationM float64) float64 {
	if separationM <= 0 {
		return 0
	}
	numerator := math.Pi * math.Pi * HBar * 299792458.0
	denominator := 240.0 * math.Pow(separationM, 4)
	return -(numerator / denominator)
}

// QuantumTunnelingProbability computes WKB tunneling probability through a square potential barrier:
// T ≈ exp(-2 * kappa * L), kappa = sqrt(2m(V0-E)) / ℏ
func QuantumTunnelingProbability(barrierHeightJoules, particleEnergyJoules, barrierWidthM, massKg float64) float64 {
	if particleEnergyJoules >= barrierHeightJoules {
		return 1.0 // Classical transmission (above barrier)
	}
	deltaE := barrierHeightJoules - particleEnergyJoules
	kappa := math.Sqrt(2.0*massKg*deltaE) / HBar
	exponent := -2.0 * kappa * barrierWidthM
	return math.Exp(exponent)
}

// ZeroPointEnergyDensity computes vacuum zero-point energy density for a cavity:
// ρ = (π² ℏ c) / (720 L^4) per unit volume — Casimir energy density form
func ZeroPointEnergyDensity(cavityLengthM float64) float64 {
	if cavityLengthM <= 0 {
		return 0
	}
	numerator := math.Pi * math.Pi * HBar * 299792458.0
	denominator := 720.0 * math.Pow(cavityLengthM, 4)
	return numerator / denominator
}

// DecoherenceTime estimates quantum coherence lifetime in thermal environment:
// tau_d = ℏ / (kB * T) for a single qubit coupled to thermal bath
func DecoherenceTime(temperatureK float64) float64 {
	const kB = 1.380649e-23 // Boltzmann constant
	if temperatureK <= 0 {
		return math.Inf(1)
	}
	return HBar / (kB * temperatureK)
}
