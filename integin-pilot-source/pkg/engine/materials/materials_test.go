package materials

import (
	"math"
	"testing"
)

func TestIsotropicStiffnessTensor(t *testing.T) {
	// Steel: E = 200 GPa, nu = 0.3
	C := IsotropicStiffness(200e9, 0.3)

	// C11 = lambda + 2*mu = E*(1-nu) / ((1+nu)*(1-2*nu))
	expected11 := 200e9 * (1.0 - 0.3) / ((1 + 0.3) * (1 - 0.6))
	if math.Abs(C[0][0]-expected11) > 1e3 {
		t.Fatalf("C11 mismatch: expected %v, got %v", expected11, C[0][0])
	}
	// Shear modulus: G = E / (2*(1+nu)) = 200e9 / 2.6 ≈ 76.92 GPa
	expectedShear := 200e9 / (2.0 * 1.3)
	if math.Abs(C[3][3]-expectedShear) > 1e6 {
		t.Fatalf("Shear C44 mismatch: expected %v, got %v", expectedShear, C[3][3])
	}
}

func TestWohlerFatigueCurve(t *testing.T) {
	// S355 Steel: ultimate 490 MPa, b=10, ref N=1. At N=1e6: sigma = 490e6 * (1e6)^(-0.1) ≈ 123 MPa
	sigma := WohlerFatigue(490e6, 1.0, 10.0, 1e6)
	if sigma < 100e6 || sigma > 200e6 {
		t.Fatalf("Wöhler fatigue stress out of expected range: %v MPa", sigma/1e6)
	}
}

func TestPhaseTransition(t *testing.T) {
	// Iron: melting 1811 K, boiling 3134 K, plasma > 20000 K
	if ClassifyPhase(300, 1811, 3134, 20000) != PhaseSolid {
		t.Fatal("Room temperature iron should be solid")
	}
	if ClassifyPhase(2000, 1811, 3134, 20000) != PhaseLiquid {
		t.Fatal("Molten iron should be liquid")
	}
	if ClassifyPhase(4000, 1811, 3134, 20000) != PhaseGas {
		t.Fatal("Iron vapor should be gas")
	}
}
