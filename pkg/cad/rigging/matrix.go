package rigging

import "math"

// Mat4 represents a standard 4x4 column-major transformation matrix.
// Indexing: m[col*4 + row]
type Mat4 [16]float64

// IdentityMat4 returns a 4x4 identity matrix.
func IdentityMat4() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// TranslationMat4 creates a translation matrix.
func TranslationMat4(x, y, z float64) Mat4 {
	m := IdentityMat4()
	m[12] = x
	m[13] = y
	m[14] = z
	return m
}

// Mul multiplies two 4x4 matrices: a * b.
func (a Mat4) Mul(b Mat4) Mat4 {
	var out Mat4
	for c := 0; c < 4; c++ {
		for r := 0; r < 4; r++ {
			sum := 0.0
			for k := 0; k < 4; k++ {
				sum += a[k*4+r] * b[c*4+k]
			}
			out[c*4+r] = sum
		}
	}
	return out
}

// TransformPoint multiplies matrix by a 3D point (assuming w=1).
func (a Mat4) TransformPoint(p [3]float64) [3]float64 {
	x := a[0]*p[0] + a[4]*p[1] + a[8]*p[2] + a[12]
	y := a[1]*p[0] + a[5]*p[1] + a[9]*p[2] + a[13]
	z := a[2]*p[0] + a[6]*p[1] + a[10]*p[2] + a[14]
	w := a[3]*p[0] + a[7]*p[1] + a[11]*p[2] + a[15]

	if w != 1.0 && w != 0.0 {
		invW := 1.0 / w
		return [3]float64{x * invW, y * invW, z * invW}
	}
	return [3]float64{x, y, z}
}

// RotationX creates a rotation matrix around X axis (angle in radians).
func RotationX(rad float64) Mat4 {
	m := IdentityMat4()
	c := math.Cos(rad)
	s := math.Sin(rad)
	m[5] = c
	m[6] = s
	m[9] = -s
	m[10] = c
	return m
}

// RotationY creates a rotation matrix around Y axis (angle in radians).
func RotationY(rad float64) Mat4 {
	m := IdentityMat4()
	c := math.Cos(rad)
	s := math.Sin(rad)
	m[0] = c
	m[2] = -s
	m[8] = s
	m[10] = c
	return m
}

// RotationZ creates a rotation matrix around Z axis (angle in radians).
func RotationZ(rad float64) Mat4 {
	m := IdentityMat4()
	c := math.Cos(rad)
	s := math.Sin(rad)
	m[0] = c
	m[1] = s
	m[4] = -s
	m[5] = c
	return m
}
