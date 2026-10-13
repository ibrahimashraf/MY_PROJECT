package rigging

import (
	"math"
)

// IKSolverConfig holds tuning parameters for iterative IK solvers.
type IKSolverConfig struct {
	MaxIterations int
	ToleranceM    float64
}

// DefaultIKConfig provides standard configuration for crane/robotic IK.
func DefaultIKConfig() IKSolverConfig {
	return IKSolverConfig{
		MaxIterations: 40,
		ToleranceM:    0.005, // 5 mm precision
	}
}

// SolveCCD solves Inverse Kinematics using Cyclic Coordinate Descent.
// boneChain: slice of bone IDs from base to end-effector (e.g. [0, 1, 2]).
func (s *Skeleton) SolveCCD(
	boneChain []int,
	target [3]float64,
	endEffectorLength float64,
	cfg IKSolverConfig,
) (bool, int) {
	if len(boneChain) == 0 {
		return false, 0
	}

	endBoneID := boneChain[len(boneChain)-1]
	s.ComputeFK()

	for iter := 0; iter < cfg.MaxIterations; iter++ {
		endPos := s.GetEndEffectorPos(endBoneID, endEffectorLength)
		dist := dist3D(endPos, target)
		if dist <= cfg.ToleranceM {
			return true, iter
		}

		// Iterate backwards from the parent of end-effector to base
		for i := len(boneChain) - 1; i >= 0; i-- {
			boneID := boneChain[i]
			bone := &s.Bones[boneID]

			// Current joint position in world space
			jointPos := bone.GlobalTransform.TransformPoint([3]float64{0, 0, 0})
			currentEnd := s.GetEndEffectorPos(endBoneID, endEffectorLength)

			// Vectors from joint to end-effector and target
			vEnd := [3]float64{currentEnd[0] - jointPos[0], currentEnd[1] - jointPos[1], currentEnd[2] - jointPos[2]}
			vTgt := [3]float64{target[0] - jointPos[0], target[1] - jointPos[1], target[2] - jointPos[2]}

			lenEnd := math.Sqrt(vEnd[0]*vEnd[0] + vEnd[1]*vEnd[1] + vEnd[2]*vEnd[2])
			lenTgt := math.Sqrt(vTgt[0]*vTgt[0] + vTgt[1]*vTgt[1] + vTgt[2]*vTgt[2])

			if lenEnd < 1e-6 || lenTgt < 1e-6 {
				continue
			}

			// Normalize
			vEnd[0] /= lenEnd
			vEnd[1] /= lenEnd
			vEnd[2] /= lenEnd

			vTgt[0] /= lenTgt
			vTgt[1] /= lenTgt
			vTgt[2] /= lenTgt

			// Dot product for rotation angle
			dot := vEnd[0]*vTgt[0] + vEnd[1]*vTgt[1] + vEnd[2]*vTgt[2]
			dot = math.Max(-1.0, math.Min(1.0, dot))
			angle := math.Acos(dot)

			if angle < 1e-5 {
				continue
			}

			// Cross product for rotation axis
			cross := [3]float64{
				vEnd[1]*vTgt[2] - vEnd[2]*vTgt[1],
				vEnd[2]*vTgt[0] - vEnd[0]*vTgt[2],
				vEnd[0]*vTgt[1] - vEnd[1]*vTgt[0],
			}
			crossLen := math.Sqrt(cross[0]*cross[0] + cross[1]*cross[1] + cross[2]*cross[2])
			if crossLen < 1e-6 {
				continue
			}

			// Unit rotation axis with damping factor for smooth convergence
			axisY := (cross[1] / crossLen) * 0.8
			axisZ := (cross[2] / crossLen) * 0.8

			// Adjust Z-rotation (Pitch / Luff)
			if math.Abs(axisZ) > 1e-4 {
				bone.LocalRot[2] += axisZ * angle
				clampAngle(&bone.LocalRot[2], bone.Limits.MinZ, bone.Limits.MaxZ)
			}

			// Adjust Y-rotation (Yaw / Slew)
			if math.Abs(axisY) > 1e-4 {
				bone.LocalRot[1] += axisY * angle
				clampAngle(&bone.LocalRot[1], bone.Limits.MinY, bone.Limits.MaxY)
			}

			s.ComputeFK()
		}
	}

	finalEnd := s.GetEndEffectorPos(endBoneID, endEffectorLength)
	return dist3D(finalEnd, target) <= cfg.ToleranceM, cfg.MaxIterations
}

func dist3D(a, b [3]float64) float64 {
	dx := a[0] - b[0]
	dy := a[1] - b[1]
	dz := a[2] - b[2]
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func clampAngle(val *float64, min, max float64) {
	if min != 0 || max != 0 {
		if *val < min {
			*val = min
		}
		if *val > max {
			*val = max
		}
	}
}
