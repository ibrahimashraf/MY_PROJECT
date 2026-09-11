package verification

import (
	"fmt"
	"math"
)

// NISTBenchmarkVector provides certified reference data for math verification.
type NISTBenchmarkVector struct {
	Input    float64
	Expected float64
	Tol      float64
}

// VerifyNISTExponential tests e^x against certified NIST high-precision benchmarks.
func VerifyNISTExponential() error {
	// Reference values from NIST DLMF (Digital Library of Mathematical Functions)
	benchmarks := []NISTBenchmarkVector{
		{Input: 0.0, Expected: 1.0, Tol: 1e-15},
		{Input: 1.0, Expected: 2.71828182845904523536, Tol: 1e-15},
		{Input: -1.0, Expected: 0.36787944117144232159, Tol: 1e-15},
		{Input: 2.0, Expected: 7.38905609893065022723, Tol: 1e-14},
		{Input: -10.0, Expected: 4.53999297624848515355e-5, Tol: 1e-18},
	}

	for _, bm := range benchmarks {
		actual := math.Exp(bm.Input)
		diff := math.Abs(actual - bm.Expected)
		if diff > bm.Tol {
			return fmt.Errorf("NIST Exp mismatch at x=%v: expected %v, got %v, diff=%e", bm.Input, bm.Expected, actual, diff)
		}
	}
	return nil
}

// MeshTopology tracks vertices, edges, and faces to enforce Euler characteristic preservation.
type MeshTopology struct {
	V int
	E int
	F int
}

// EulerCharacteristic calculates chi = V - E + F.
func (m MeshTopology) EulerCharacteristic() int {
	return m.V - m.E + m.F
}

// Genus calculates g = 1 - chi/2 for orientable closed surfaces.
func (m MeshTopology) Genus() int {
	return 1 - m.EulerCharacteristic()/2
}

// VerifyTopologicalContinuity asserts that deformation/subdivision preserves genus without tearing.
func VerifyTopologicalContinuity(initial, deformed MeshTopology) error {
	chiInit := initial.EulerCharacteristic()
	chiDef := deformed.EulerCharacteristic()
	if chiInit != chiDef {
		return fmt.Errorf("topological tear detected: initial chi=%d (genus %d) != deformed chi=%d (genus %d)",
			chiInit, initial.Genus(), chiDef, deformed.Genus())
	}
	return nil
}

// RegularizedInvert2x2 applies Levenberg-Marquardt damping (A + lambda*I) to near-singular matrices.
func RegularizedInvert2x2(a00, a01, a10, a11, lambda float64) ([2][2]float64, error) {
	// Damped matrix: A + lambda * I
	d00 := a00 + lambda
	d11 := a11 + lambda
	det := d00*d11 - a01*a10

	if math.Abs(det) < 1e-15 {
		return [2][2]float64{}, fmt.Errorf("singular matrix even after regularization: det=%e", det)
	}

	invDet := 1.0 / det
	return [2][2]float64{
		{d11 * invDet, -a01 * invDet},
		{-a10 * invDet, d00 * invDet},
	}, nil
}
