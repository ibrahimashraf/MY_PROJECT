package mathcore_test

import (
	"math"
	"math/cmplx"
	"testing"

	"integin/pkg/mathcore/diffgeom"
	"integin/pkg/mathcore/fft"
	"integin/pkg/mathcore/infotheory"
	"integin/pkg/mathcore/lie"
	"integin/pkg/mathcore/topology"
)

func TestLieAlgebra(t *testing.T) {
	// Rotate 90 degrees around Z axis: w = [0, 0, pi/2]
	w := [3]float64{0, 0, math.Pi * 0.5}
	r := lie.So3Exp(w)

	// In SO(3), 90 deg around Z should map (1, 0, 0) to (0, 1, 0)
	pX := r[0][0]*1.0 + r[0][1]*0.0 + r[0][2]*0.0
	pY := r[1][0]*1.0 + r[1][1]*0.0 + r[1][2]*0.0
	if math.Abs(pX-0.0) > 1e-4 || math.Abs(pY-1.0) > 1e-4 {
		t.Errorf("Expected rotated point (0, 1, 0), got (%f, %f)", pX, pY)
	}

	// Logarithm round-trip: log(exp(w)) should equal w
	recoveredW, err := lie.So3Log(r)
	if err != nil {
		t.Fatalf("So3Log failed: %v", err)
	}
	if math.Abs(recoveredW[2]-w[2]) > 1e-4 {
		t.Errorf("Expected recovered wz=%f, got %f", w[2], recoveredW[2])
	}
}

func TestDifferentialGeometry(t *testing.T) {
	// 2D Unit Sphere metric: ds^2 = dtheta^2 + sin^2(theta) dphi^2
	// u = theta (colatitude), v = phi (longitude)
	sphereMetric := func(u, v float64) [2][2]float64 {
		sinU := math.Sin(u)
		return [2][2]float64{
			{1.0, 0.0},
			{0.0, sinU * sinU},
		}
	}

	// At theta = pi/4, Christoffel symbol Gamma^u_vv should be -sin(u)*cos(u) = -0.5
	gamma, err := diffgeom.Christoffel2D(sphereMetric, math.Pi*0.25, 0.0, 1e-5)
	if err != nil {
		t.Fatalf("Christoffel2D failed: %v", err)
	}

	expectedGammaU_VV := -math.Sin(math.Pi*0.25) * math.Cos(math.Pi*0.25) // -0.5
	if math.Abs(gamma[0][1][1]-expectedGammaU_VV) > 1e-3 {
		t.Errorf("Expected Gamma^u_vv=%f, got %f", expectedGammaU_VV, gamma[0][1][1])
	}
}

func TestFFTRoundTrip(t *testing.T) {
	// 8-point complex pulse
	input := []complex128{
		complex(1, 0), complex(2, 0), complex(3, 0), complex(4, 0),
		complex(5, 0), complex(6, 0), complex(7, 0), complex(8, 0),
	}

	fwd, err := fft.FFT(input)
	if err != nil {
		t.Fatalf("FFT failed: %v", err)
	}

	inv, err := fft.IFFT(fwd)
	if err != nil {
		t.Fatalf("IFFT failed: %v", err)
	}

	for i := 0; i < len(input); i++ {
		diff := cmplx.Abs(inv[i] - input[i])
		if diff > 1e-10 {
			t.Errorf("IFFT round-trip failed at idx %d: expected %v, got %v (diff=%e)", i, input[i], inv[i], diff)
		}
	}
}

func TestAlgebraicTopology(t *testing.T) {
	// 1. Tetrahedron (2-sphere): V=4, E=6, F=4 => chi = 2, genus = 0
	tetra := topology.TetrahedronComplex()
	chiTetra := tetra.EulerCharacteristic()
	if chiTetra != 2 {
		t.Errorf("Expected tetrahedron chi=2, got %d", chiTetra)
	}
	genusTetra, err := tetra.Genus()
	if err != nil || genusTetra != 0 {
		t.Errorf("Expected tetrahedron genus=0, got %d (err=%v)", genusTetra, err)
	}

	// 2. Torus: V=7, E=21, F=14 => chi = 0, genus = 1
	torus := topology.TorusComplex()
	chiTorus := torus.EulerCharacteristic()
	if chiTorus != 0 {
		t.Errorf("Expected torus chi=0, got %d", chiTorus)
	}
	genusTorus, err := torus.Genus()
	if err != nil || genusTorus != 1 {
		t.Errorf("Expected torus genus=1, got %d (err=%v)", genusTorus, err)
	}
}

func TestInformationTheoryAndGaloisField(t *testing.T) {
	// 1. Shannon Entropy of fair coin: H([0.5, 0.5]) = 1.0 bit
	h, err := infotheory.ShannonEntropy([]float64{0.5, 0.5})
	if err != nil || math.Abs(h-1.0) > 1e-6 {
		t.Errorf("Expected fair coin entropy 1.0 bit, got %f (err=%v)", h, err)
	}

	// 2. Galois Field GF(2^8) arithmetic
	var a infotheory.GF256 = 0x57 // Standard AES test vector
	var b infotheory.GF256 = 0x83

	prod := a.Mul(b)
	if prod != 0xC1 {
		t.Errorf("Expected GF(2^8) 0x57 * 0x83 = 0xC1, got 0x%X", prod)
	}

	// Multiplicative inverse: a * a^-1 = 1
	invA, err := a.Inverse()
	if err != nil {
		t.Fatalf("GF256 Inverse failed: %v", err)
	}
	if a.Mul(invA) != 1 {
		t.Errorf("Expected a * a^-1 = 1, got 0x%X", a.Mul(invA))
	}
}
