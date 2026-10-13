package verification

import (
	"testing"
)

func TestDualOracleInvariants(t *testing.T) {
	oracle := InvariantDualOracle{Tol: 1e-12}

	// det(A) = 3, det(B) = -2, det(AB) = -6
	if err := oracle.VerifyDeterminantMultiplicativity(3.0, -2.0, -6.0); err != nil {
		t.Fatalf("Valid det multiplicativity rejected: %v", err)
	}

	if err := oracle.VerifyDeterminantMultiplicativity(3.0, -2.0, -5.99); err == nil {
		t.Fatal("Oracle failed to catch invalid determinant product")
	}

	// Identity matrix inverse
	I := [3][3]float64{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
	if err := oracle.VerifySO3ExponentialInverse(I, I); err != nil {
		t.Fatalf("Valid SO(3) group inverse rejected: %v", err)
	}
}

func TestSymplecticEnergyConservation(t *testing.T) {
	guard := SymplecticIntegratorGuard{MaxEnergyDriftTol: 1e-4}

	// 0.001% drift -> passes
	if err := guard.VerifyEnergyConservation(100.0, 100.001); err != nil {
		t.Fatalf("Acceptable Hamiltonian drift rejected: %v", err)
	}

	// 1% drift -> fails
	if err := guard.VerifyEnergyConservation(100.0, 101.0); err == nil {
		t.Fatal("Guard failed to reject large non-conservative energy drift")
	}
}

func TestConditionNumberGuard(t *testing.T) {
	// Well-conditioned matrix
	if err := CheckConditionNumber2x2(2.0, 0.0, 0.0, 2.0, 1e4); err != nil {
		t.Fatalf("Well-conditioned matrix rejected: %v", err)
	}

	// Near singular ill-conditioned matrix:
	// [ 1.0  1.0 ]
	// [ 1.0  1.0000000001 ]
	err := CheckConditionNumber2x2(1.0, 1.0, 1.0, 1.0000000001, 1e6)
	if err == nil {
		t.Fatal("Condition number guard failed to catch ill-conditioned matrix")
	}
}
