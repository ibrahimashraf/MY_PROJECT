package calculus

import (
	"errors"
)

// DerivativeCentral computes numerical 1st derivative using central difference: (f(x+h) - f(x-h)) / (2h).
func DerivativeCentral(f func(float64) float64, x, h float64) float64 {
	if h <= 0 {
		h = 1e-5
	}
	return (f(x+h) - f(x-h)) / (2.0 * h)
}

// SecondDerivative computes numerical 2nd derivative: (f(x+h) - 2f(x) + f(x-h)) / h^2.
func SecondDerivative(f func(float64) float64, x, h float64) float64 {
	if h <= 0 {
		h = 1e-4
	}
	return (f(x+h) - 2.0*f(x) + f(x-h)) / (h * h)
}

// IntegrateSimpson computes definite integral of f(x) from a to b using composite Simpson's rule.
func IntegrateSimpson(f func(float64) float64, a, b float64, n int) (float64, error) {
	if n <= 0 || n%2 != 0 {
		n = 100 // Force even number of intervals
	}
	if a == b {
		return 0, nil
	}

	h := (b - a) / float64(n)
	sum := f(a) + f(b)

	for i := 1; i < n; i++ {
		x := a + float64(i)*h
		if i%2 == 0 {
			sum += 2.0 * f(x)
		} else {
			sum += 4.0 * f(x)
		}
	}

	return (h / 3.0) * sum, nil
}

// ODESystemFunc defines a system of first-order differential equations: dy/dt = f(t, y).
type ODESystemFunc func(t float64, y []float64) []float64

// SolveRK4 integrates an ODE system from t0 to tEnd using 4th-order Runge-Kutta.
func SolveRK4(
	f ODESystemFunc,
	t0, tEnd float64,
	y0 []float64,
	steps int,
) ([][]float64, []float64, error) {
	if steps <= 0 {
		return nil, nil, errors.New("steps must be > 0")
	}
	dim := len(y0)
	dt := (tEnd - t0) / float64(steps)

	tHist := make([]float64, steps+1)
	yHist := make([][]float64, steps+1)

	curY := make([]float64, dim)
	copy(curY, y0)

	tHist[0] = t0
	yHist[0] = make([]float64, dim)
	copy(yHist[0], curY)

	k1 := make([]float64, dim)
	k2 := make([]float64, dim)
	k3 := make([]float64, dim)
	k4 := make([]float64, dim)
	tempY := make([]float64, dim)

	curT := t0
	for step := 1; step <= steps; step++ {
		// k1 = f(t, y)
		k1Vals := f(curT, curY)
		copy(k1, k1Vals)

		// k2 = f(t + dt/2, y + dt/2 * k1)
		for d := 0; d < dim; d++ {
			tempY[d] = curY[d] + 0.5*dt*k1[d]
		}
		k2Vals := f(curT+0.5*dt, tempY)
		copy(k2, k2Vals)

		// k3 = f(t + dt/2, y + dt/2 * k2)
		for d := 0; d < dim; d++ {
			tempY[d] = curY[d] + 0.5*dt*k2[d]
		}
		k3Vals := f(curT+0.5*dt, tempY)
		copy(k3, k3Vals)

		// k4 = f(t + dt, y + dt * k3)
		for d := 0; d < dim; d++ {
			tempY[d] = curY[d] + dt*k3[d]
		}
		k4Vals := f(curT+dt, tempY)
		copy(k4, k4Vals)

		// y_{n+1} = y_n + dt/6 * (k1 + 2*k2 + 2*k3 + k4)
		for d := 0; d < dim; d++ {
			curY[d] += (dt / 6.0) * (k1[d] + 2.0*k2[d] + 2.0*k3[d] + k4[d])
		}

		curT = t0 + float64(step)*dt
		tHist[step] = curT
		yHist[step] = make([]float64, dim)
		copy(yHist[step], curY)
	}

	return yHist, tHist, nil
}
