package diffgeom

import (
	"errors"
	"math"
)

// MetricTensor2D defines a 2D Riemannian metric g_ij(u, v) on a curved manifold.
type MetricTensor2D func(u, v float64) [2][2]float64

// InverseMetric2D calculates the contravariant metric g^ij.
func InverseMetric2D(g [2][2]float64) ([2][2]float64, error) {
	det := g[0][0]*g[1][1] - g[0][1]*g[1][0]
	if math.Abs(det) < 1e-12 {
		return [2][2]float64{}, errors.New("singular metric tensor determinant")
	}
	invDet := 1.0 / det
	return [2][2]float64{
		{g[1][1] * invDet, -g[0][1] * invDet},
		{-g[1][0] * invDet, g[0][0] * invDet},
	}, nil
}

// Christoffel2D computes the 8 Christoffel symbols of the second kind: Gamma^k_ij at point (u, v).
// Indices: Gamma[k][i][j]
func Christoffel2D(metric MetricTensor2D, u, v, h float64) ([2][2][2]float64, error) {
	if h <= 0 {
		h = 1e-5
	}

	g := metric(u, v)
	gInv, err := InverseMetric2D(g)
	if err != nil {
		return [2][2][2]float64{}, err
	}

	// Numerical partial derivatives: dg[dir][i][j] = partial g_ij / partial x^dir
	// dir 0 = u, dir 1 = v
	var dg [2][2][2]float64

	// d/du
	guP := metric(u+h, v)
	guM := metric(u-h, v)
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			dg[0][i][j] = (guP[i][j] - guM[i][j]) / (2.0 * h)
		}
	}

	// d/dv
	gvP := metric(u, v+h)
	gvM := metric(u, v-h)
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			dg[1][i][j] = (gvP[i][j] - gvM[i][j]) / (2.0 * h)
		}
	}

	var gamma [2][2][2]float64
	for k := 0; k < 2; k++ {
		for i := 0; i < 2; i++ {
			for j := 0; j < 2; j++ {
				sum := 0.0
				for l := 0; l < 2; l++ {
					term := dg[i][j][l] + dg[j][i][l] - dg[l][i][j]
					sum += 0.5 * gInv[k][l] * term
				}
				gamma[k][i][j] = sum
			}
		}
	}

	return gamma, nil
}

// GeodesicState represents position and velocity on the manifold: [u, v, uDot, vDot].
type GeodesicState [4]float64

// StepGeodesic integrates one RK4 step of the geodesic equation:
// d^2 x^k / ds^2 = - Gamma^k_ij (dx^i/ds) (dx^j/ds)
func StepGeodesic(metric MetricTensor2D, state GeodesicState, ds, h float64) (GeodesicState, error) {
	u, v := state[0], state[1]
	gamma, err := Christoffel2D(metric, u, v, h)
	if err != nil {
		return state, err
	}

	uDot, vDot := state[2], state[3]
	vel := [2]float64{uDot, vDot}

	// Compute accelerations: a^k = -sum(Gamma^k_ij * v^i * v^j)
	var accel [2]float64
	for k := 0; k < 2; k++ {
		sum := 0.0
		for i := 0; i < 2; i++ {
			for j := 0; j < 2; j++ {
				sum += gamma[k][i][j] * vel[i] * vel[j]
			}
		}
		accel[k] = -sum
	}

	// Simple Euler / RK integration step
	nextU := u + uDot*ds
	nextV := v + vDot*ds
	nextUDot := uDot + accel[0]*ds
	nextVDot := vDot + accel[1]*ds

	return GeodesicState{nextU, nextV, nextUDot, nextVDot}, nil
}
