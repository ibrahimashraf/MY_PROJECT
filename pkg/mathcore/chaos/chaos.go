package chaos

import (
	"math"

	"integin/pkg/mathcore/calculus"
)

// LorenzParams holds parameters for the canonical Lorenz chaotic system.
type LorenzParams struct {
	Sigma float64 // typically 10.0 (Prandtl number)
	Rho   float64 // typically 28.0 (Rayleigh number)
	Beta  float64 // typically 8.0/3.0 (Geometric factor)
}

// DefaultLorenz returns standard chaotic parameter values.
func DefaultLorenz() LorenzParams {
	return LorenzParams{
		Sigma: 10.0,
		Rho:   28.0,
		Beta:  8.0 / 3.0,
	}
}

// SimulateLorenz integrates the 3D Lorenz attractor using RK4.
func SimulateLorenz(
	p LorenzParams,
	x0, y0, z0 float64,
	tEnd float64,
	steps int,
) ([][3]float64, error) {
	odeFunc := func(t float64, y []float64) []float64 {
		x, yVal, z := y[0], y[1], y[2]
		dx := p.Sigma * (yVal - x)
		dy := x*(p.Rho-z) - yVal
		dz := x*yVal - p.Beta*z
		return []float64{dx, dy, dz}
	}

	rawHist, _, err := calculus.SolveRK4(odeFunc, 0.0, tEnd, []float64{x0, y0, z0}, steps)
	if err != nil {
		return nil, err
	}

	trajectory := make([][3]float64, len(rawHist))
	for i, pt := range rawHist {
		trajectory[i] = [3]float64{pt[0], pt[1], pt[2]}
	}

	return trajectory, nil
}

// MandelbrotEscape evaluates escape time for complex coordinate c = cr + ci*i.
// Returns iteration count in [0, maxIter].
func MandelbrotEscape(cr, ci float64, maxIter int) int {
	zr, zi := 0.0, 0.0
	for iter := 0; iter < maxIter; iter++ {
		zr2 := zr * zr
		zi2 := zi * zi
		if zr2+zi2 > 4.0 {
			return iter
		}
		newZi := 2.0*zr*zi + ci
		zr = zr2 - zi2 + cr
		zi = newZi
	}
	return maxIter
}

// EstimateLyapunov computes the maximal Lyapunov exponent of a 1D chaotic map f(x).
func EstimateLyapunov(f func(float64) float64, df func(float64) float64, x0 float64, n int) float64 {
	sum := 0.0
	x := x0
	valid := 0

	for i := 0; i < n; i++ {
		d := math.Abs(df(x))
		if d > 1e-12 {
			sum += math.Log(d)
			valid++
		}
		x = f(x)
	}

	if valid == 0 {
		return 0
	}
	return sum / float64(valid)
}
