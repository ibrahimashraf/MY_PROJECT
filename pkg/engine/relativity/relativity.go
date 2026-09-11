package relativity

import "math"

const SpeedOfLight = 299792458.0 // m/s

// TimeDilation computes Lorentz time dilation: t' = t / sqrt(1 - v^2/c^2)
func TimeDilation(properTimeS float64, velocityMps float64) (dilatedTimeS float64, lorentzFactor float64) {
	beta := velocityMps / SpeedOfLight
	if beta >= 1.0 {
		return math.Inf(1), math.Inf(1)
	}
	lorentzFactor = 1.0 / math.Sqrt(1.0-beta*beta)
	dilatedTimeS = properTimeS * lorentzFactor
	return dilatedTimeS, lorentzFactor
}

// GravitationalTimeDilation computes gravitational redshift factor: t_inf = t_surface * sqrt(1 - rs/r)
// Schwarzschild radius rs = 2GM/c^2
func GravitationalTimeDilation(massKg, radiusM float64) float64 {
	const G = 6.67430e-11
	rs := 2.0 * G * massKg / (SpeedOfLight * SpeedOfLight)
	if radiusM <= rs {
		return 0.0 // Inside event horizon
	}
	return math.Sqrt(1.0 - rs/radiusM)
}

// KeplerOrbitalPeriod computes satellite orbital period: T = 2*pi * sqrt(a^3 / (G*M))
func KeplerOrbitalPeriod(semiMajorAxisM, centralMassKg float64) float64 {
	const G = 6.67430e-11
	return 2.0 * math.Pi * math.Sqrt(math.Pow(semiMajorAxisM, 3)/(G*centralMassKg))
}

// SchwarzschildRadius computes event horizon: rs = 2GM/c^2
func SchwarzschildRadius(massKg float64) float64 {
	const G = 6.67430e-11
	return 2.0 * G * massKg / (SpeedOfLight * SpeedOfLight)
}

// HubbleRecessionalVelocity computes recessional velocity via Hubble's Law: v = H0 * d
// H0 ≈ 70 km/s/Mpc
func HubbleRecessionalVelocity(distanceMpc float64, h0KmSMpc float64) float64 {
	if h0KmSMpc <= 0 {
		h0KmSMpc = 70.0
	}
	return h0KmSMpc * 1000.0 * distanceMpc // convert to m/s
}
