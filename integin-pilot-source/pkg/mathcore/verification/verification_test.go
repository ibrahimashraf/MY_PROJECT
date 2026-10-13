package verification

import (
	"math"
	"math/big"
	"testing"
	"testing/quick"
)

func TestIntervalArithmeticBounds(t *testing.T) {
	// a in [1.9, 2.1], b in [2.9, 3.1]
	a := NewInterval(2.0, 0.1)
	b := NewInterval(3.0, 0.1)

	sum := a.Add(b) // [4.8, 5.2]
	if math.Abs(sum.Lo-4.8) > 1e-9 || math.Abs(sum.Hi-5.2) > 1e-9 {
		t.Fatalf("Interval add failed: expected [4.8, 5.2], got [%v, %v]", sum.Lo, sum.Hi)
	}

	prod := a.Mul(b) // [1.9*2.9, 2.1*3.1] = [5.51, 6.51]
	if math.Abs(prod.Lo-5.51) > 1e-9 || math.Abs(prod.Hi-6.51) > 1e-9 {
		t.Fatalf("Interval mul failed: expected [5.51, 6.51], got [%v, %v]", prod.Lo, prod.Hi)
	}
}

func TestLinearSystemResidualWitness(t *testing.T) {
	// Ax = b
	// [ 2  1 ] [ 1 ] = [ 4 ]
	// [ 1  3 ] [ 2 ]   [ 7 ]
	A := [][]float64{
		{2.0, 1.0},
		{1.0, 3.0},
	}
	b := []float64{4.0, 7.0}
	x := []float64{1.0, 2.0} // exact solution

	witness, err := VerifyLinearSystemWitness(A, b, x, 1e-12)
	if err != nil {
		t.Fatalf("Witness rejected exact solution: %v", err)
	}
	if !witness.Passed || witness.ResidualNorm > 1e-14 {
		t.Fatalf("Witness failed: norm=%e", witness.ResidualNorm)
	}

	// Perturbed solution (buggy solver output)
	badX := []float64{1.05, 2.0}
	_, badErr := VerifyLinearSystemWitness(A, b, badX, 1e-12)
	if badErr == nil {
		t.Fatal("Witness failed to catch perturbed/corrupted solution")
	}
}

func TestMetamorphicRotationInvariance(t *testing.T) {
	tester := MetamorphicTester{Epsilon: 1e-9}

	// Scaling function f(v) = 2.5 * v is invariant/equivariant under rotation
	linearScale := func(x, y float64) (float64, float64) {
		return 2.5 * x, 2.5 * y
	}

	angles := []float64{0.1, math.Pi / 4.0, math.Pi / 2.0, 1.2345, math.Pi}
	for _, th := range angles {
		if err := tester.Verify2DRotationInvariance(linearScale, 3.0, 4.0, th); err != nil {
			t.Fatalf("Metamorphic invariance failed for theta=%v: %v", th, err)
		}
	}
}

func TestBigRatExactFallback(t *testing.T) {
	// Near-singular ill-conditioned system
	// [ 1000  1001 ] [ x ] = [ 2001 ]
	// [ 1001  1002 ] [ y ]   [ 2003 ]
	// det = 1000*1002 - 1001^2 = 1002000 - 1002001 = -1
	A := [2][2]int64{
		{1000, 1001},
		{1001, 1002},
	}
	b := [2]int64{2001, 2003}

	x, y, err := BigRatExactLinearSolve(A, b)
	if err != nil {
		t.Fatalf("BigRat solver failed: %v", err)
	}

	// Exact solution is x = 1, y = 1
	one := big.NewRat(1, 1)
	if x.Cmp(one) != 0 || y.Cmp(one) != 0 {
		t.Fatalf("BigRat exact solution mismatch: x=%v, y=%v", x.String(), y.String())
	}
}

func TestPropertyFuzzTriangleInequality(t *testing.T) {
	// Quick check: ||a + b|| <= ||a|| + ||b|| + eps across arbitrary floats
	fn := func(ax, ay, bx, by float64) bool {
		if math.IsNaN(ax) || math.IsNaN(ay) || math.IsNaN(bx) || math.IsNaN(by) {
			return true
		}
		if math.IsInf(ax, 0) || math.IsInf(ay, 0) || math.IsInf(bx, 0) || math.IsInf(by, 0) {
			return true
		}
		normA := math.Hypot(ax, ay)
		normB := math.Hypot(bx, by)
		normSum := math.Hypot(ax+bx, ay+by)
		// Account for machine float rounding:
		eps := 1e-12 * (normA + normB + 1.0)
		return normSum <= (normA + normB + eps)
	}

	if err := quick.Check(fn, &quick.Config{MaxCount: 10000}); err != nil {
		t.Fatalf("Property fuzzing caught triangle inequality violation: %v", err)
	}
}
