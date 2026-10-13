package lie

import (
	"math"
)

// So3Hat converts a 3D angular velocity vector into a 3x3 skew-symmetric matrix.
// omega = [wx, wy, wz] -> [[0, -wz, wy], [wz, 0, -wx], [-wy, wx, 0]]
func So3Hat(w [3]float64) [3][3]float64 {
	return [3][3]float64{
		{0, -w[2], w[1]},
		{w[2], 0, -w[0]},
		{-w[1], w[0], 0},
	}
}

// So3Vee extracts the 3D angular velocity vector from a 3x3 skew-symmetric matrix.
func So3Vee(m [3][3]float64) [3]float64 {
	return [3]float64{m[2][1], m[0][2], m[1][0]}
}

// So3Exp computes the matrix exponential exp(hat(w)) in SO(3) via Rodrigues' formula.
func So3Exp(w [3]float64) [3][3]float64 {
	theta := math.Sqrt(w[0]*w[0] + w[1]*w[1] + w[2]*w[2])
	if theta < 1e-12 {
		return [3][3]float64{
			{1, 0, 0},
			{0, 1, 0},
			{0, 0, 1},
		}
	}

	k := [3]float64{w[0] / theta, w[1] / theta, w[2] / theta}
	kHat := So3Hat(k)

	sinT := math.Sin(theta)
	cosT := math.Cos(theta)
	oneMinusCos := 1.0 - cosT

	var r [3][3]float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			// I + sin(theta)*K + (1-cos(theta))*K^2
			term1 := 0.0
			if i == j {
				term1 = 1.0
			}
			term2 := sinT * kHat[i][j]
			term3 := 0.0
			for m := 0; m < 3; m++ {
				term3 += kHat[i][m] * kHat[m][j]
			}
			r[i][j] = term1 + term2 + oneMinusCos*term3
		}
	}
	return r
}

// So3Log computes the matrix logarithm log(R) mapping from SO(3) to so(3) vector.
func So3Log(r [3][3]float64) ([3]float64, error) {
	// Trace(R) = 1 + 2*cos(theta)
	tr := r[0][0] + r[1][1] + r[2][2]
	cosTheta := (tr - 1.0) * 0.5
	cosTheta = math.Max(-1.0, math.Min(1.0, cosTheta))
	theta := math.Acos(cosTheta)

	if theta < 1e-12 {
		return [3]float64{0, 0, 0}, nil
	}

	sinTheta := math.Sin(theta)
	if math.Abs(sinTheta) < 1e-6 {
		// Close to pi: diagonal extraction
		return [3]float64{
			theta * math.Sqrt(math.Max(0, (r[0][0]+1)*0.5)),
			theta * math.Sqrt(math.Max(0, (r[1][1]+1)*0.5)),
			theta * math.Sqrt(math.Max(0, (r[2][2]+1)*0.5)),
		}, nil
	}

	scale := theta / (2.0 * sinTheta)
	return [3]float64{
		(r[2][1] - r[1][2]) * scale,
		(r[0][2] - r[2][0]) * scale,
		(r[1][0] - r[0][1]) * scale,
	}, nil
}

// Se3Twist represents a 6-DOF spatial twist vector: [vx, vy, vz, wx, wy, wz].
type Se3Twist struct {
	V [3]float64 // Linear velocity
	W [3]float64 // Angular velocity
}

// Se3Exp computes the matrix exponential mapping from se(3) twist to SE(3) homogeneous transform.
func Se3Exp(twist Se3Twist) [4][4]float64 {
	r := So3Exp(twist.W)
	theta := math.Sqrt(twist.W[0]*twist.W[0] + twist.W[1]*twist.W[1] + twist.W[2]*twist.W[2])

	var t [3]float64
	if theta < 1e-12 {
		t = twist.V
	} else {
		wHat := So3Hat(twist.W)
		// V_matrix = I + (1-cos(theta))/theta^2 * wHat + (theta - sin(theta))/theta^3 * wHat^2
		sinT := math.Sin(theta)
		cosT := math.Cos(theta)
		c1 := (1.0 - cosT) / (theta * theta)
		c2 := (theta - sinT) / (theta * theta * theta)

		for i := 0; i < 3; i++ {
			sum := 0.0
			for j := 0; j < 3; j++ {
				term1 := 0.0
				if i == j {
					term1 = 1.0
				}
				term2 := c1 * wHat[i][j]
				term3 := 0.0
				for k := 0; k < 3; k++ {
					term3 += wHat[i][k] * wHat[k][j]
				}
				vMatIJ := term1 + term2 + c2*term3
				sum += vMatIJ * twist.V[j]
			}
			t[i] = sum
		}
	}

	var se3 [4][4]float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			se3[i][j] = r[i][j]
		}
		se3[i][3] = t[i]
	}
	se3[3][3] = 1.0

	return se3
}
