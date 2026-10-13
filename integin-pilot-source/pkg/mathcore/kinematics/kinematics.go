package kinematics

import (
	"math"
)

// Mat4 represents a 4x4 transformation matrix stored in row-major order.
// Zero-allocation, stack-allocated value type for maximum CPU cache locality.
type Mat4 [16]float64

// Vec3 represents a 3D coordinate vector [X, Y, Z].
type Vec3 [3]float64

// IdentityMat4 returns a 4x4 identity matrix.
func IdentityMat4() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// TranslationMat4 creates a translation transformation matrix.
func TranslationMat4(tx, ty, tz float64) Mat4 {
	return Mat4{
		1, 0, 0, tx,
		0, 1, 0, ty,
		0, 0, 1, tz,
		0, 0, 0, 1,
	}
}

// RotationZMat4 creates a rotation matrix around the Z axis (yaw/slew).
func RotationZMat4(angleRad float64) Mat4 {
	c := math.Cos(angleRad)
	s := math.Sin(angleRad)
	return Mat4{
		c, -s, 0, 0,
		s, c, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Mul multiplies two 4x4 matrices (a * b) with zero heap allocation.
func (a Mat4) Mul(b Mat4) Mat4 {
	var res Mat4
	for r := 0; r < 4; r++ {
		r4 := r * 4
		for c := 0; c < 4; c++ {
			res[r4+c] = a[r4]*b[c] +
				a[r4+1]*b[4+c] +
				a[r4+2]*b[8+c] +
				a[r4+3]*b[12+c]
		}
	}
	return res
}

// TransformVec3 transforms a 3D point by this 4x4 matrix (assuming W=1).
func (a Mat4) TransformVec3(v Vec3) Vec3 {
	return Vec3{
		a[0]*v[0] + a[1]*v[1] + a[2]*v[2] + a[3],
		a[4]*v[0] + a[5]*v[1] + a[6]*v[2] + a[7],
		a[8]*v[0] + a[9]*v[1] + a[10]*v[2] + a[11],
	}
}

// KinematicNode represents an articulated link in a crane or rigging assembly.
type KinematicNode struct {
	ID       int
	ParentID int // -1 for root
	Local    Mat4
	World    Mat4
}

// ForwardKinematics evaluates the global world poses across an ordered kinematic tree.
// Pre-ordered breadth-first / topological list ensures O(N) evaluation with 0 heap escapes.
func ForwardKinematics(nodes []KinematicNode) {
	for i := range nodes {
		p := nodes[i].ParentID
		if p < 0 || p >= i {
			nodes[i].World = nodes[i].Local
		} else {
			nodes[i].World = nodes[p].World.Mul(nodes[i].Local)
		}
	}
}

// BoussinesqStress calculates vertical soil stress (Ground Bearing Pressure)
// at depth z and radial distance r from an outrigger point load P (in Newtons or Tonnes-force).
// Equation: sigma_z = (3P / 2*pi*z^2) * [1 + (r/z)^2]^(-5/2)
func BoussinesqStress(P, r, z float64) float64 {
	if z <= 0.0001 {
		z = 0.0001 // prevent singularity at surface
	}
	rz := r / z
	factor := 1.0 + rz*rz
	coeff := (3.0 * P) / (2.0 * math.Pi * z * z)
	return coeff * math.Pow(factor, -2.5)
}

// EvaluateGroundBearingPressureGrid computes vertical stress values across a 2D soil grid
// with size X x Y (total points = nx * ny) for an outrigger reaction load P at (cx, cy).
func EvaluateGroundBearingPressureGrid(P, cx, cy, z, step float64, nx, ny int, out []float64) {
	idx := 0
	for iy := 0; iy < ny; iy++ {
		y := float64(iy)*step - cy
		for ix := 0; ix < nx; ix++ {
			x := float64(ix)*step - cx
			r := math.Hypot(x, y)
			out[idx] = BoussinesqStress(P, r, z)
			idx++
		}
	}
}
