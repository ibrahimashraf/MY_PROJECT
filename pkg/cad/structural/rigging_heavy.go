package structural

import (
	"fmt"
	"math"
)

// FourPointStaticEquilibrium solves reaction forces for a 4-point outrigger configuration
// using structural flexibility formulation on elastic support springs (Winkler foundation).
//
// When four outriggers contact the ground, the system is statically indeterminate (degree 1).
// To find realistic individual pad reaction forces without artificial singularity:
//
// Reactions satisfy:
// 1) Sum(Fz) = R1 + R2 + R3 + R4 = TotalLoad
// 2) Sum(Mx) = R1*y1 + R2*y2 + R3*y3 + R4*y4 = TotalLoad * loadPos.y
// 3) Sum(My) = R1*x1 + R2*x2 + R3*x3 + R4*x4 = TotalLoad * loadPos.x
// 4) Planar displacement compatibility: pads displace elastically according to vertical spring stiffness k:
//    delta_i = R_i / k_i.
//    Assuming a rigid crane chassis: delta(x, y) = w0 + theta_x * y + theta_y * x
func FourPointStaticEquilibrium(
	totalLoadN float64,
	loadPos [2]float64,
	padPositions [4][2]float64,
	padStiffnessNPerM [4]float64,
) ([4]float64, error) {
	if totalLoadN <= 0 {
		return [4]float64{}, fmt.Errorf("invalid non-positive total load: %v", totalLoadN)
	}

	for i, k := range padStiffnessNPerM {
		if k <= 0 {
			return [4]float64{}, fmt.Errorf("invalid non-positive pad stiffness at index %d: %v", i, k)
		}
	}

	// Matrix formulation for rigid chassis displacement:
	// delta_i = w0 + theta_x * y_i + theta_y * x_i
	// R_i = k_i * delta_i
	//
	// Equilibrium equations in terms of [w0, theta_x, theta_y]^T:
	// A * [w0, theta_x, theta_y]^T = B
	//
	// A[0][0] = sum(k_i),          A[0][1] = sum(k_i * y_i),        A[0][2] = sum(k_i * x_i)
	// A[1][0] = sum(k_i * y_i),    A[1][1] = sum(k_i * y_i^2),      A[1][2] = sum(k_i * x_i * y_i)
	// A[2][0] = sum(k_i * x_i),    A[2][1] = sum(k_i * x_i * y_i),  A[2][2] = sum(k_i * x_i^2)
	//
	// B = [totalLoadN, totalLoadN * loadPos[1], totalLoadN * loadPos[0]]^T

	var a00, a01, a02 float64
	var a11, a12 float64
	var a22 float64

	for i := 0; i < 4; i++ {
		k := padStiffnessNPerM[i]
		x := padPositions[i][0]
		y := padPositions[i][1]

		a00 += k
		a01 += k * y
		a02 += k * x

		a11 += k * y * y
		a12 += k * x * y

		a22 += k * x * x
	}

	b0 := totalLoadN
	b1 := totalLoadN * loadPos[1]
	b2 := totalLoadN * loadPos[0]

	// Solve 3x3 symmetric positive definite system via Cramer:
	det := a00*(a11*a22-a12*a12) - a01*(a01*a22-a12*a02) + a02*(a01*a12-a11*a02)
	if math.Abs(det) < 1e-9 {
		return [4]float64{}, fmt.Errorf("singular stiffness matrix for 4-point outriggers (collinear or coincident pads)")
	}

	w0 := (b0*(a11*a22-a12*a12) - a01*(b1*a22-a12*b2) + a02*(b1*a12-a11*b2)) / det
	thetaX := (a00*(b1*a22-a12*b2) - b0*(a01*a22-a12*a02) + a02*(a01*b2-b1*a02)) / det
	thetaY := (a00*(a11*b2-b1*a12) - a01*(a01*b2-b1*a02) + b0*(a01*a12-a11*a02)) / det

	var reactions [4]float64
	for i := 0; i < 4; i++ {
		k := padStiffnessNPerM[i]
		x := padPositions[i][0]
		y := padPositions[i][1]
		delta := w0 + thetaX*y + thetaY*x
		reactions[i] = k * delta
	}

	return reactions, nil
}

// WireRopeBendingDerating computes ASME B30.9 D/d bending efficiency factor.
// D: sheave/pin diameter, d: rope diameter.
// Efficiency drops as D/d decreases below 25.
func WireRopeBendingDerating(pinDiameterM, ropeDiameterM float64) (efficiency float64, err error) {
	if ropeDiameterM <= 0 {
		return 0, fmt.Errorf("invalid rope diameter: %v", ropeDiameterM)
	}
	ratio := pinDiameterM / ropeDiameterM
	if ratio < 1.0 {
		return 0, fmt.Errorf("unsafe pin/rope ratio D/d = %v (< 1.0)", ratio)
	}
	if ratio >= 25.0 {
		return 1.0, nil
	}
	// ASME empirical curve: E = 1 - 0.5 / sqrt(D/d)
	eff := 1.0 - (0.5 / math.Sqrt(ratio))
	if eff < 0.5 {
		eff = 0.5
	}
	return eff, nil
}

// ShacklePointLoadingDerating computes ASME B30.26 load capacity derating
// when a shackle is loaded out of plane or with an off-center sling hook angle.
func ShacklePointLoadingDerating(angleDeg float64) (capacityFactor float64, err error) {
	if angleDeg < 0 || angleDeg > 90 {
		return 0, fmt.Errorf("angle must be between 0 and 90 degrees: got %v", angleDeg)
	}
	// In-line (0 deg): 100%
	// 45 deg: 70% capacity
	// 90 deg: 50% capacity
	switch {
	case angleDeg <= 5.0:
		return 1.0, nil
	case angleDeg <= 45.0:
		// Linear drop from 1.0 to 0.7
		return 1.0 - (0.30 * (angleDeg / 45.0)), nil
	default:
		// Linear drop from 0.7 to 0.5
		return 0.70 - (0.20 * ((angleDeg - 45.0) / 45.0)), nil
	}
}

// TandemLiftResult contains the resolved static hook reactions for a 2-crane lift.
type TandemLiftResult struct {
	Crane1LoadN   float64
	Crane2LoadN   float64
	SharePct1     float64
	SharePct2     float64
	SafeCapacity1 bool
	SafeCapacity2 bool
}

// TandemLiftLoadDistribution calculates static hook load split for a dual-crane tandem lift:
// W1 = TotalWeight * (L2 / (L1 + L2))
// W2 = TotalWeight * (L1 / (L1 + L2))
func TandemLiftLoadDistribution(
	totalWeightN float64,
	distFromHook1ToCoGM float64,
	distFromHook2ToCoGM float64,
	crane1RatedCapacityN float64,
	crane2RatedCapacityN float64,
) (TandemLiftResult, error) {
	totalSpan := distFromHook1ToCoGM + distFromHook2ToCoGM
	if totalSpan <= 0.01 {
		return TandemLiftResult{}, fmt.Errorf("invalid hook span: %v m", totalSpan)
	}
	if totalWeightN <= 0 {
		return TandemLiftResult{}, fmt.Errorf("invalid non-positive total weight: %v", totalWeightN)
	}

	w1 := totalWeightN * (distFromHook2ToCoGM / totalSpan)
	w2 := totalWeightN * (distFromHook1ToCoGM / totalSpan)

	pct1 := (w1 / totalWeightN) * 100.0
	pct2 := (w2 / totalWeightN) * 100.0

	return TandemLiftResult{
		Crane1LoadN:   w1,
		Crane2LoadN:   w2,
		SharePct1:     pct1,
		SharePct2:     pct2,
		SafeCapacity1: w1 <= crane1RatedCapacityN,
		SafeCapacity2: w2 <= crane2RatedCapacityN,
	}, nil
}

// VesselUpendingKinematics calculates the Main Crane (Head) and Tail Crane loads
// as an industrial vessel is tilted from horizontal (0°) to vertical (90°).
//
// thetaDeg: Angle of the vessel from horizontal (0° = horizontal on transport, 90° = vertical on foundation)
// vesselLengthM: Distance between Head trunnion and Tail lifting lug.
// cogDistanceM: Distance from Tail lug to Center of Gravity along vessel centerline.
func VesselUpendingKinematics(
	vesselWeightN float64,
	vesselLengthM float64,
	cogDistanceM float64,
	thetaDeg float64,
) (headLoadN, tailLoadN float64, err error) {
	if vesselLengthM <= 0.01 || cogDistanceM < 0 || cogDistanceM > vesselLengthM {
		return 0, 0, fmt.Errorf("invalid vessel dimensions: length=%v, cogDist=%v", vesselLengthM, cogDistanceM)
	}
	if thetaDeg < 0 || thetaDeg > 90 {
		return 0, 0, fmt.Errorf("upending angle must be between 0 and 90 degrees: got %v", thetaDeg)
	}

	// When horizontal (0 deg):
	// TailLoad = W * (Length - CoG) / Length
	// HeadLoad = W * CoG / Length
	//
	// As angle tilts up, the effective horizontal lever arms scale with cos(theta):
	// At 90 deg (fully vertical), Head Crane takes 100% of the load, Tail Load drops to 0.
	rad := thetaDeg * (math.Pi / 180.0)
	cosTheta := math.Cos(rad)

	if thetaDeg >= 89.9 {
		return vesselWeightN, 0.0, nil
	}

	// Moment equilibrium around tail pivot with horizontal projection:
	headLoadN = vesselWeightN * (cogDistanceM / vesselLengthM)
	// Additional load shifts to head crane proportionally as tail unloads:
	unloadingFactor := 1.0 - cosTheta
	headLoadN += (vesselWeightN - headLoadN) * unloadingFactor
	tailLoadN = vesselWeightN - headLoadN
	if tailLoadN < 0 {
		tailLoadN = 0
	}

	return headLoadN, tailLoadN, nil
}

