package rigging

import (
	"math"
	"testing"
)

func TestForwardKinematics(t *testing.T) {
	skel := NewSkeleton()

	// Create 2-bone arm: Base at (0,0,0), Bone 1 length 10 along X
	// Bone 2 attached at (10,0,0), length 10 along X
	b0 := skel.AddBone("Base", -1, [3]float64{0, 0, 0}, DefaultLimits())
	b1 := skel.AddBone("Forearm", b0, [3]float64{10, 0, 0}, DefaultLimits())

	// Rotate Base by 90 degrees around Z (points up along Y)
	skel.Bones[b0].LocalRot[2] = math.Pi * 0.5 // 90 deg

	skel.ComputeFK()

	// Effector at tip of Bone 1 (length 10)
	tipPos := skel.GetEndEffectorPos(b1, 10.0)

	// Since Base rotated 90 deg up:
	// Joint 1 is at (0, 10, 0)
	// Tip of Bone 1 is at (0, 20, 0)
	if math.Abs(tipPos[0]-0.0) > 1e-4 || math.Abs(tipPos[1]-20.0) > 1e-4 {
		t.Errorf("Expected FK tip at (0, 20, 0), got (%f, %f, %f)", tipPos[0], tipPos[1], tipPos[2])
	}
}

func TestCCDSolverConvergence(t *testing.T) {
	skel := NewSkeleton()

	// Crane Boom (Bone 0, 20m) and Jib (Bone 1, 10m)
	limits := AngleLimits{
		MinZ: -math.Pi * 0.45, MaxZ: math.Pi * 0.45, // -81 to +81 degrees articulation
		MinY: -math.Pi, MaxY: math.Pi, // 360 deg slewing
	}
	b0 := skel.AddBone("Boom", -1, [3]float64{0, 0, 0}, limits)
	b1 := skel.AddBone("Jib", b0, [3]float64{20, 0, 0}, limits)

	// Target coordinate to reach with hook (tip of Jib)
	// For 20m + 10m arm with 81° limit, reachable radius is [23.72m, 30.0m].
	// Target at (15, 20) has radius = sqrt(225 + 400) = 25.0m.
	target := [3]float64{15.0, 20.0, 0.0}

	cfg := IKSolverConfig{
		MaxIterations: 50,
		ToleranceM:    0.05, // 5 cm convergence
	}

	converged, iters := skel.SolveCCD([]int{b0, b1}, target, 10.0, cfg)

	finalPos := skel.GetEndEffectorPos(b1, 10.0)
	errDist := dist3D(finalPos, target)

	if !converged || errDist > 0.05 {
		t.Fatalf("CCD failed to converge within tolerance: errDist=%f m, iters=%d, finalPos=%v", errDist, iters, finalPos)
	}
}

func TestLinearBlendSkinning(t *testing.T) {
	// Single bone translated to (5, 10, 0)
	boneTransforms := []Mat4{
		TranslationMat4(5, 10, 0),
	}

	vert := SkinVertex{
		BasePos: [3]float32{1, 2, 3},
		Joints:  [4]uint16{0, 0, 0, 0},
		Weights: [4]float32{2.0, 0.0, 0.0, 0.0}, // Un-normalized
	}
	vert.NormalizeWeights()

	if vert.Weights[0] != 1.0 {
		t.Fatalf("Expected normalized weight 1.0, got %f", vert.Weights[0])
	}

	outDeformed := make([][3]float32, 1)
	LinearBlendSkinning([]SkinVertex{vert}, boneTransforms, outDeformed)

	// Expected: (1+5, 2+10, 3+0) = (6, 12, 3)
	if outDeformed[0][0] != 6.0 || outDeformed[0][1] != 12.0 || outDeformed[0][2] != 3.0 {
		t.Errorf("LBS deformation unexpected: got %v", outDeformed[0])
	}
}
