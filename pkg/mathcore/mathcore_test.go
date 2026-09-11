package mathcore_test

import (
	"math"
	"testing"

	"integin/pkg/mathcore/calculus"
	"integin/pkg/mathcore/chaos"
	"integin/pkg/mathcore/discrete"
	"integin/pkg/mathcore/linear"
	"integin/pkg/mathcore/optimization"
	"integin/pkg/mathcore/quantum"
	"integin/pkg/mathcore/stats"
)

func TestLinearAlgebra(t *testing.T) {
	// A = [ [1, 2], [3, 4] ], det(A) = 1*4 - 2*3 = -2
	m, err := linear.NewMatrixFromData(2, 2, []float64{1, 2, 3, 4})
	if err != nil {
		t.Fatalf("Failed to create matrix: %v", err)
	}

	det, err := m.Determinant()
	if err != nil || math.Abs(det-(-2.0)) > 1e-4 {
		t.Fatalf("Expected determinant -2.0, got %f (err=%v)", det, err)
	}

	// Solve Ax = b: [ [2, 1], [1, 3] ] * [x, y] = [5, 5] => x = 2, y = 1
	a, _ := linear.NewMatrixFromData(2, 2, []float64{2, 1, 1, 3})
	b := []float64{5, 5}
	x, err := linear.SolveLinear(a, b)
	if err != nil {
		t.Fatalf("SolveLinear failed: %v", err)
	}
	if math.Abs(x[0]-2.0) > 1e-4 || math.Abs(x[1]-1.0) > 1e-4 {
		t.Errorf("Expected solution [2, 1], got %v", x)
	}
}

func TestCalculus(t *testing.T) {
	// 1. Derivative of x^2 at x = 3 is 2*x = 6.0
	f := func(x float64) float64 { return x * x }
	df := calculus.DerivativeCentral(f, 3.0, 1e-5)
	if math.Abs(df-6.0) > 1e-4 {
		t.Errorf("Expected derivative 6.0, got %f", df)
	}

	// 2. Integral of sin(x) from 0 to pi is 2.0
	sinFunc := func(x float64) float64 { return math.Sin(x) }
	integral, err := calculus.IntegrateSimpson(sinFunc, 0, math.Pi, 100)
	if err != nil || math.Abs(integral-2.0) > 1e-4 {
		t.Errorf("Expected integral 2.0, got %f (err=%v)", integral, err)
	}
}

func TestOptimization(t *testing.T) {
	// Find root of f(x) = x^2 - 2 => sqrt(2) ~ 1.41421356
	f := func(x float64) float64 { return x*x - 2.0 }
	df := func(x float64) float64 { return 2.0 * x }

	root, _, err := optimization.NewtonRaphson(f, df, 1.0, 1e-6, 50)
	if err != nil || math.Abs(root-math.Sqrt(2)) > 1e-5 {
		t.Errorf("Expected root %f, got %f (err=%v)", math.Sqrt(2), root, err)
	}
}

func TestStatisticsAndEconometrics(t *testing.T) {
	// y = 2*x + 1 with perfect correlation
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{3, 5, 7, 9, 11}

	reg, err := stats.LinearRegression(x, y)
	if err != nil {
		t.Fatalf("LinearRegression failed: %v", err)
	}
	if math.Abs(reg.Beta-2.0) > 1e-4 || math.Abs(reg.Alpha-1.0) > 1e-4 || math.Abs(reg.R2-1.0) > 1e-4 {
		t.Errorf("Expected Beta=2, Alpha=1, R2=1, got Beta=%f, Alpha=%f, R2=%f", reg.Beta, reg.Alpha, reg.R2)
	}
}

func TestChaosAndFractals(t *testing.T) {
	// 1. Lorenz Attractor integration
	params := chaos.DefaultLorenz()
	traj, err := chaos.SimulateLorenz(params, 1.0, 1.0, 1.0, 1.0, 100)
	if err != nil || len(traj) != 101 {
		t.Fatalf("Lorenz simulation failed: %v", err)
	}

	// 2. Mandelbrot: Origin (0,0) does not escape
	esc0 := chaos.MandelbrotEscape(0, 0, 50)
	if esc0 != 50 {
		t.Errorf("Expected origin to stay bounded (50 iters), got %d", esc0)
	}

	// Point (2, 2) escapes immediately
	esc2 := chaos.MandelbrotEscape(2, 2, 50)
	if esc2 >= 50 {
		t.Errorf("Expected point (2,2) to escape, got %d", esc2)
	}
}

func TestQuantumComputing(t *testing.T) {
	// Create Bell state |Phi+> = (|00> + |11>) / sqrt(2)
	// Start with 2 qubits in |00>
	sv := quantum.NewMultiQubitState(2)

	// Apply Hadamard to qubit 0: (|00> + |10>) / sqrt(2)
	if err := sv.ApplyHadamard(0); err != nil {
		t.Fatalf("Hadamard failed: %v", err)
	}

	// Apply CNOT with control=0, target=1: (|00> + |11>) / sqrt(2)
	if err := sv.ApplyCNOT(0, 1); err != nil {
		t.Fatalf("CNOT failed: %v", err)
	}

	// Verify probabilities: P(00)=0.5, P(01)=0, P(10)=0, P(11)=0.5
	p00 := sv.Probability(0) // Basis 0 = |00>
	p01 := sv.Probability(1) // Basis 1 = |01>
	p10 := sv.Probability(2) // Basis 2 = |10>
	p11 := sv.Probability(3) // Basis 3 = |11>

	if math.Abs(p00-0.5) > 1e-4 || math.Abs(p11-0.5) > 1e-4 {
		t.Errorf("Expected Bell state P(00)=0.5 and P(11)=0.5, got P(00)=%f, P(11)=%f", p00, p11)
	}
	if p01 > 1e-4 || p10 > 1e-4 {
		t.Errorf("Expected 0 probability for |01> and |10>, got P(01)=%f, P(10)=%f", p01, p10)
	}
}

func TestDiscreteMathAndKnots(t *testing.T) {
	// Poker combinations: 52C5 = 2,598,960
	nCr, err := discrete.Combinations(52, 5)
	if err != nil || nCr.String() != "2598960" {
		t.Errorf("Expected 52C5 = 2598960, got %s (err=%v)", nCr.String(), err)
	}

	// Logic evaluation: IMPLIES (true -> false is false; false -> true is true)
	if discrete.EvaluateLogic(discrete.OpImplies, true, false) != false {
		t.Errorf("Expected True -> False to be False")
	}
	if discrete.EvaluateLogic(discrete.OpImplies, false, true) != true {
		t.Errorf("Expected False -> True to be True")
	}

	// Trefoil knot minimal crossing number = 3
	trefoil := discrete.TrefoilKnot()
	if trefoil.CrossingNumber() != 3 {
		t.Errorf("Expected Trefoil crossing number 3, got %d", trefoil.CrossingNumber())
	}
}
