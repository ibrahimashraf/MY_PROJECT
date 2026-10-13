package materials

import "math"

// AnisotropicStiffness represents a 6x6 symmetric elasticity tensor (Voigt notation).
type AnisotropicStiffness [6][6]float64

// IsotropicStiffness builds the 6x6 stiffness tensor for isotropic material.
func IsotropicStiffness(youngsModulusPa, poissonRatio float64) AnisotropicStiffness {
	E, nu := youngsModulusPa, poissonRatio
	lam := (E * nu) / ((1 + nu) * (1 - 2*nu))
	mu := E / (2 * (1 + nu))

	var C AnisotropicStiffness
	// Normal-normal
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i == j {
				C[i][j] = lam + 2*mu
			} else {
				C[i][j] = lam
			}
		}
	}
	// Shear diagonal
	for i := 3; i < 6; i++ {
		C[i][i] = mu
	}
	return C
}

// WohlerFatigue evaluates stress amplitude for N cycles to failure via Wöhler (S-N) curve.
// sigma_a = sigma_u * (N / N_ref)^(-1/b) where b is Basquin exponent.
func WohlerFatigue(ultimateStrengthPa float64, cyclesRef float64, basquinExp float64, cyclesTarget float64) float64 {
	return ultimateStrengthPa * math.Pow(cyclesTarget/cyclesRef, -1.0/basquinExp)
}

// PhaseTransitionCheck determines material state based on temperature thresholds.
type MaterialPhase int

const (
	PhaseSolid MaterialPhase = iota
	PhaseLiquid
	PhaseGas
	PhasePlasma
)

// ClassifyPhase returns thermodynamic phase of material.
func ClassifyPhase(tempK float64, meltingK, boilingK, plasmaK float64) MaterialPhase {
	switch {
	case tempK >= plasmaK:
		return PhasePlasma
	case tempK >= boilingK:
		return PhaseGas
	case tempK >= meltingK:
		return PhaseLiquid
	default:
		return PhaseSolid
	}
}

// CrackTipStressIntensityK1 computes mode-I stress intensity factor (fracture mechanics):
// K_I = sigma * sqrt(pi * a) * F(a/W)
func CrackTipStressIntensityK1(appliedStressPa, crackHalfLengthM, geometryFactor float64) float64 {
	return appliedStressPa * math.Sqrt(math.Pi*crackHalfLengthM) * geometryFactor
}
