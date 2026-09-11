package verification

import (
	"math"
	"testing"
)

func TestNISTExponentialBenchmarks(t *testing.T) {
	if err := VerifyNISTExponential(); err != nil {
		t.Fatalf("NIST benchmark verification failed: %v", err)
	}
}

func TestTopologicalContinuityGuard(t *testing.T) {
	// Sphere/Tetrahedron: V=4, E=6, F=4 -> chi=2, genus=0
	initialSphere := MeshTopology{V: 4, E: 6, F: 4}
	// Valid subdivision (e.g. 1 edge split -> +1 V, +3 E, +2 F) -> V=5, E=9, F=6 -> chi=2
	subdividedSphere := MeshTopology{V: 5, E: 9, F: 6}

	if err := VerifyTopologicalContinuity(initialSphere, subdividedSphere); err != nil {
		t.Fatalf("Valid topological subdivision flagged as failure: %v", err)
	}

	// Invalid puncture/tear (e.g. removed face -> F=3) -> chi = 1 -> genus change
	tornMesh := MeshTopology{V: 4, E: 6, F: 3}
	if err := VerifyTopologicalContinuity(initialSphere, tornMesh); err == nil {
		t.Fatal("Topological continuity guard failed to catch mesh tear")
	}
}

func TestRegularizedInversion(t *testing.T) {
	// Singular matrix:
	// [ 1  1 ]
	// [ 1  1 ] det = 0
	_, err := RegularizedInvert2x2(1.0, 1.0, 1.0, 1.0, 0.0)
	if err == nil {
		t.Fatal("Zero damping should fail for singular matrix")
	}

	// Levenberg-Marquardt regularized damping lambda = 0.1
	// [ 1.1  1.0 ]
	// [ 1.0  1.1 ] det = 1.21 - 1.0 = 0.21 != 0
	inv, err := RegularizedInvert2x2(1.0, 1.0, 1.0, 1.0, 0.1)
	if err != nil {
		t.Fatalf("Regularized inversion failed: %v", err)
	}

	det := 1.1*1.1 - 1.0
	expected00 := 1.1 / det
	if math.Abs(inv[0][0]-expected00) > 1e-12 {
		t.Fatalf("Inversion value mismatch: expected %v, got %v", expected00, inv[0][0])
	}
}
