package verification

import (
	"fmt"
	"math"
	"math/big"
)

// Interval represents a rigorous bounding interval [Lo, Hi] for interval arithmetic.
// Used to track floating-point precision loss and roundoff accumulation.
type Interval struct {
	Lo float64
	Hi float64
}

// NewInterval creates an interval enclosing x ± eps.
func NewInterval(x, eps float64) Interval {
	return Interval{Lo: x - eps, Hi: x + eps}
}

// Width returns interval uncertainty width.
func (i Interval) Width() float64 {
	return i.Hi - i.Lo
}

// Add returns interval sum [a.Lo + b.Lo, a.Hi + b.Hi].
func (i Interval) Add(b Interval) Interval {
	return Interval{Lo: i.Lo + b.Lo, Hi: i.Hi + b.Hi}
}

// Sub returns interval difference [a.Lo - b.Hi, a.Hi - b.Lo].
func (i Interval) Sub(b Interval) Interval {
	return Interval{Lo: i.Lo - b.Hi, Hi: i.Hi - b.Lo}
}

// Mul returns interval multiplication considering all 4 endpoint bounds.
func (i Interval) Mul(b Interval) Interval {
	p1 := i.Lo * b.Lo
	p2 := i.Lo * b.Hi
	p3 := i.Hi * b.Lo
	p4 := i.Hi * b.Hi
	return Interval{
		Lo: math.Min(math.Min(p1, p2), math.Min(p3, p4)),
		Hi: math.Max(math.Max(p1, p2), math.Max(p3, p4)),
	}
}

// ResidualWitness represents a cryptographic or numeric proof certificate for linear system Ax = b.
// Verification takes O(n^2) instead of O(n^3) solving.
type ResidualWitness struct {
	ResidualVector []float64
	ResidualNorm   float64
	RelativeError  float64
	ConditionBound float64
	Passed         bool
}

// VerifyLinearSystemWitness verifies if computed solution x solves Ax = b within tolerance.
// Returns an explicit ResidualWitness certificate.
func VerifyLinearSystemWitness(A [][]float64, b []float64, x []float64, tol float64) (*ResidualWitness, error) {
	n := len(A)
	if n == 0 || len(b) != n || len(x) != n {
		return nil, fmt.Errorf("dimension mismatch: A(%d), b(%d), x(%d)", n, len(b), len(x))
	}

	res := make([]float64, n)
	bNorm := 0.0
	resNorm := 0.0

	for i := 0; i < n; i++ {
		bNorm += b[i] * b[i]
		rowDot := 0.0
		for j := 0; j < n; j++ {
			rowDot += A[i][j] * x[j]
		}
		res[i] = b[i] - rowDot
		resNorm += res[i] * res[i]
	}

	bNorm = math.Sqrt(bNorm)
	resNorm = math.Sqrt(resNorm)

	relErr := 0.0
	if bNorm > 1e-15 {
		relErr = resNorm / bNorm
	} else {
		relErr = resNorm
	}

	witness := &ResidualWitness{
		ResidualVector: res,
		ResidualNorm:   resNorm,
		RelativeError:  relErr,
		Passed:         relErr <= tol,
	}

	if !witness.Passed {
		return witness, fmt.Errorf("residual witness failed: relErr=%e > tol=%e", relErr, tol)
	}

	return witness, nil
}

// MetamorphicTransform verifies coordinate frame invariance f(R * x) = R * f(x)
// under arbitrary 2D/3D rotations within tolerance.
type MetamorphicTester struct {
	Epsilon float64
}

// Verify2DRotationInvariance checks if vector transform f is equivariant under rotation by angle theta.
func (m MetamorphicTester) Verify2DRotationInvariance(f func(x, y float64) (float64, float64), x, y, theta float64) error {
	cosT := math.Cos(theta)
	sinT := math.Sin(theta)

	// Rotate input
	rx := cosT*x - sinT*y
	ry := sinT*x + cosT*y

	// Apply function to rotated input: f(R x)
	fx_r, fy_r := f(rx, ry)

	// Apply function to unrotated input: f(x)
	fx, fy := f(x, y)

	// Rotate unrotated output: R f(x)
	r_fx := cosT*fx - sinT*fy
	r_fy := sinT*fx + cosT*fy

	diffX := math.Abs(fx_r - r_fx)
	diffY := math.Abs(fy_r - r_fy)

	if diffX > m.Epsilon || diffY > m.Epsilon {
		return fmt.Errorf("metamorphic rotation symmetry broken: diff=(%e, %e) > eps=%e", diffX, diffY, m.Epsilon)
	}
	return nil
}

// BigRatExactLinearSolve computes exact rational solution for 2x2 linear system using arbitrary-precision big.Rat.
// Used as fallback oracle when floating-point condition number diverges.
func BigRatExactLinearSolve(A [2][2]int64, b [2]int64) (*big.Rat, *big.Rat, error) {
	det := A[0][0]*A[1][1] - A[0][1]*A[1][0]
	if det == 0 {
		return nil, nil, fmt.Errorf("singular matrix: det=0")
	}

	// Cramer's rule in exact rational arithmetic
	detX := b[0]*A[1][1] - A[0][1]*b[1]
	detY := A[0][0]*b[1] - b[0]*A[1][0]

	x := big.NewRat(detX, det)
	y := big.NewRat(detY, det)

	return x, y, nil
}
