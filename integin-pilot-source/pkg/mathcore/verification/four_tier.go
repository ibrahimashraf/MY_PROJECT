package verification

import (
	"fmt"
	"math"
)

// InvariantDualOracle verifies algebraic and group-theoretic identities between dual mathematical implementations.
type InvariantDualOracle struct {
	Tol float64
}

// VerifyDeterminantMultiplicativity asserts det(A * B) == det(A) * det(B).
func (o InvariantDualOracle) VerifyDeterminantMultiplicativity(detA, detB, detAB float64) error {
	expected := detA * detB
	diff := math.Abs(detAB - expected)
	scale := math.Max(1.0, math.Abs(expected))
	if diff/scale > o.Tol {
		return fmt.Errorf("dual oracle violation: det(AB)=%v != det(A)*det(B)=%v (diff=%e)", detAB, expected, diff)
	}
	return nil
}

// VerifySO3ExponentialInverse asserts exp(omega) * exp(-omega) == I.
func (o InvariantDualOracle) VerifySO3ExponentialInverse(R, Rinv [3][3]float64) error {
	// Multiply R * Rinv
	var prod [3][3]float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			sum := 0.0
			for k := 0; k < 3; k++ {
				sum += R[i][k] * Rinv[k][j]
			}
			prod[i][j] = sum
		}
	}

	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			expected := 0.0
			if i == j {
				expected = 1.0
			}
			diff := math.Abs(prod[i][j] - expected)
			if diff > o.Tol {
				return fmt.Errorf("group inverse violation at (%d,%d): got %v, expected %v", i, j, prod[i][j], expected)
			}
		}
	}
	return nil
}

// SymplecticIntegratorGuard enforces energy conservation (Hamiltonian drift bounds) during dynamic steps.
type SymplecticIntegratorGuard struct {
	MaxEnergyDriftTol float64
}

// VerifyEnergyConservation asserts |(E_final - E_initial) / E_initial| <= MaxEnergyDriftTol.
func (g SymplecticIntegratorGuard) VerifyEnergyConservation(initialEnergy, finalEnergy float64) error {
	if initialEnergy == 0 {
		if math.Abs(finalEnergy) > g.MaxEnergyDriftTol {
			return fmt.Errorf("energy conservation violation: E_0=0, E_t=%e > tol=%e", finalEnergy, g.MaxEnergyDriftTol)
		}
		return nil
	}
	drift := math.Abs(finalEnergy-initialEnergy) / math.Abs(initialEnergy)
	if drift > g.MaxEnergyDriftTol {
		return fmt.Errorf("Hamiltonian energy drift violation: drift=%e > tol=%e (E0=%v, Et=%v)",
			drift, g.MaxEnergyDriftTol, initialEnergy, finalEnergy)
	}
	return nil
}

// ConditionNumberGuard guards against matrix inversion when condition number kappa(A) diverges.
func CheckConditionNumber2x2(a00, a01, a10, a11 float64, maxKappa float64) error {
	// Frobenius norm: ||A||_F = sqrt(sum a_ij^2)
	normA := math.Sqrt(a00*a00 + a01*a01 + a10*a10 + a11*a11)
	det := a00*a11 - a01*a10
	if math.Abs(det) < 1e-15 {
		return fmt.Errorf("matrix is strictly singular: det=%e", det)
	}

	// Inverse matrix elements:
	invDet := 1.0 / det
	i00 := a11 * invDet
	i01 := -a01 * invDet
	i10 := -a10 * invDet
	i11 := a00 * invDet

	normInvA := math.Sqrt(i00*i00 + i01*i01 + i10*i10 + i11*i11)
	kappa := normA * normInvA

	if kappa > maxKappa {
		return fmt.Errorf("matrix condition number ill-conditioned: kappa=%e > max=%e", kappa, maxKappa)
	}
	return nil
}
