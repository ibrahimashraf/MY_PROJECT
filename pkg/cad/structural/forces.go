package structural

import (
	"fmt"
	"math"
)

// Force3D represents a 3-component vector force in Newtons.
type Force3D struct {
	Fx, Fy, Fz float64
}

// Magnitude computes Euclidean force norm ||F||.
func (f Force3D) Magnitude() float64 {
	return math.Sqrt(f.Fx*f.Fx + f.Fy*f.Fy + f.Fz*f.Fz)
}

// Add sums two 3D force vectors.
func (f Force3D) Add(o Force3D) Force3D {
	return Force3D{f.Fx + o.Fx, f.Fy + o.Fy, f.Fz + o.Fz}
}

// TorqueAtPoint computes moment/torque vector M = r x F around pivot point.
func (f Force3D) TorqueAtPoint(leverArm [3]float64) [3]float64 {
	// Cross product r x F:
	// Mx = ry * Fz - rz * Fy
	// My = rz * Fx - rx * Fz
	// Mz = rx * Fy - ry * Fx
	rx, ry, rz := leverArm[0], leverArm[1], leverArm[2]
	return [3]float64{
		ry*f.Fz - rz*f.Fy,
		rz*f.Fx - rx*f.Fz,
		rx*f.Fy - ry*f.Fx,
	}
}

// CentrifugalForce evaluates radial force F_c = m * omega^2 * r during slewing/rotation.
func CentrifugalForce(massKg float64, angularVelocityRadS float64, radiusM float64) float64 {
	return massKg * (angularVelocityRadS * angularVelocityRadS) * radiusM
}

// DynamicHoistForce evaluates total rope/hook tension under acceleration:
// F_dynamic = mass * (g + a_z) * DynamicFactor
func DynamicHoistForce(massKg float64, verticalAccelMps2 float64, dynamicAmplificationFactor float64) float64 {
	const g = 9.80665
	if dynamicAmplificationFactor <= 0 {
		dynamicAmplificationFactor = 1.15 // Standard crane hoisting coefficient
	}
	return massKg * (g + verticalAccelMps2) * dynamicAmplificationFactor
}

// MultiLegSlingTension computes tension per sling leg in multi-leg crane rigging:
// T_leg = (TotalWeight / (N * cos(theta)))
// where theta is angle from vertical (e.g. 0 to 60 degrees).
func MultiLegSlingTension(totalWeightN float64, legCount int, angleFromVerticalRad float64) (tensionPerLegN float64, err error) {
	if legCount <= 0 {
		return 0, fmt.Errorf("invalid leg count: %d", legCount)
	}
	cosTheta := math.Cos(angleFromVerticalRad)
	if cosTheta <= 0.1 { // Over 84 degrees vertical spread causes tension explosion
		return 0, fmt.Errorf("unsafe sling angle: cos(theta)=%v too small (sling angle too flat)", cosTheta)
	}
	tensionPerLegN = totalWeightN / (float64(legCount) * cosTheta)
	return tensionPerLegN, nil
}

// ThreePointStaticEquilibrium solves reaction forces for 3-point outrigger/support system:
// Sum Fz = 0, Sum Mx = 0, Sum My = 0
func ThreePointStaticEquilibrium(totalLoadN float64, loadPos [2]float64, padPositions [3][2]float64) ([3]float64, error) {
	// Linear system:
	// [ 1        1        1      ] [ R1 ]   [ TotalLoad ]
	// [ y1       y2       y3     ] [ R2 ] = [ TotalLoad * loadPos.y ]  (Sum Mx = 0)
	// [ x1       x2       x3     ] [ R3 ]   [ TotalLoad * loadPos.x ]  (Sum My = 0)

	x1, y1 := padPositions[0][0], padPositions[0][1]
	x2, y2 := padPositions[1][0], padPositions[1][1]
	x3, y3 := padPositions[2][0], padPositions[2][1]

	det := (x2*y3 - x3*y2) - (x1*y3 - x3*y1) + (x1*y2 - x2*y1)
	if math.Abs(det) < 1e-6 {
		return [3]float64{}, fmt.Errorf("degenerate/collinear support pads: det=%e", det)
	}

	b0 := totalLoadN
	b1 := totalLoadN * loadPos[1]
	b2 := totalLoadN * loadPos[0]

	// Recompute standard 3x3 Cramer explicitly:
	r1 := (b0*(x2*y3-x3*y2) - b1*(x2-x3) + b2*(y2-y3)) / det
	r2 := (b0*(x3*y1-x1*y3) - b1*(x3-x1) + b2*(y3-y1)) / det
	r3 := (b0*(x1*y2-x2*y1) - b1*(x1-x2) + b2*(y1-y2)) / det

	return [3]float64{r1, r2, r3}, nil
}
