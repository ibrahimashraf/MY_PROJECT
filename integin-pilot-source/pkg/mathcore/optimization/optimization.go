package optimization

import (
	"errors"
	"math"
)

// NewtonRaphson finds a root of f(x) = 0 given f(x) and its derivative df(x).
func NewtonRaphson(
	f func(float64) float64,
	df func(float64) float64,
	x0 float64,
	tolerance float64,
	maxIter int,
) (float64, int, error) {
	if maxIter <= 0 {
		maxIter = 100
	}
	if tolerance <= 0 {
		tolerance = 1e-7
	}

	x := x0
	for i := 0; i < maxIter; i++ {
		y := f(x)
		if math.Abs(y) < tolerance {
			return x, i, nil
		}
		dy := df(x)
		if math.Abs(dy) < 1e-12 {
			return x, i, errors.New("zero derivative encountered in Newton-Raphson")
		}
		xNext := x - y/dy
		if math.Abs(xNext-x) < tolerance {
			return xNext, i + 1, nil
		}
		x = xNext
	}

	return x, maxIter, errors.New("max iterations reached without convergence")
}

// GradientDescentMomentum minimizes a multivariate objective function f(x).
func GradientDescentMomentum(
	f func([]float64) float64,
	grad func([]float64) []float64,
	x0 []float64,
	learningRate, momentum float64,
	maxIter int,
	tolerance float64,
) ([]float64, error) {
	dim := len(x0)
	x := make([]float64, dim)
	copy(x, x0)
	v := make([]float64, dim)

	for iter := 0; iter < maxIter; iter++ {
		g := grad(x)
		gNorm := 0.0
		for d := 0; d < dim; d++ {
			gNorm += g[d] * g[d]
		}
		if math.Sqrt(gNorm) < tolerance {
			break
		}

		for d := 0; d < dim; d++ {
			v[d] = momentum*v[d] + learningRate*g[d]
			x[d] -= v[d]
		}
	}

	return x, nil
}

// GoldenSectionSearch finds the minimum of a unimodal function f(x) on interval [a, b].
func GoldenSectionSearch(
	f func(float64) float64,
	a, b float64,
	tolerance float64,
) (float64, float64) {
	invPhi := (math.Sqrt(5) - 1) * 0.5   // ~0.6180339887
	invPhiSq := (3 - math.Sqrt(5)) * 0.5 // ~0.3819660113

	h := b - a
	if h <= tolerance {
		mid := (a + b) * 0.5
		return mid, f(mid)
	}

	n := int(math.Ceil(math.Log(tolerance/h) / math.Log(invPhi)))

	c := a + invPhiSq*h
	d := a + invPhi*h
	yc := f(c)
	yd := f(d)

	for k := 0; k < n; k++ {
		if yc < yd {
			b = d
			d = c
			yd = yc
			h = invPhi * h
			c = a + invPhiSq*h
			yc = f(c)
		} else {
			a = c
			c = d
			yc = yd
			h = invPhi * h
			d = a + invPhi*h
			yd = f(d)
		}
	}

	if yc < yd {
		return (a + d) * 0.5, yc
	}
	return (c + b) * 0.5, yd
}
