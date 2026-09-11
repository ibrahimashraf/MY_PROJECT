package planet

import (
	"math"
	"testing"
)

func TestVec3dPrecisionAndCameraRelative(t *testing.T) {
	// Earth radius: 6,371,000 meters
	rEarth := 6371000.0

	// Station at sea level
	groundStation := Vec3d{X: rEarth, Y: 0, Z: 0}
	// Observer 2mm away
	caliper := Vec3d{X: rEarth + 0.002, Y: 0, Z: 0}

	// Double precision preserves delta
	delta := caliper.Sub(groundStation)
	if math.Abs(delta.X-0.002) > 1e-9 {
		t.Fatalf("Float64 precision loss at planetary scale: expected 0.002, got %v", delta.X)
	}

	// Camera relative transform
	camRel := CameraRelativeTransform(caliper, groundStation)
	if math.Abs(float64(camRel[0])-0.002) > 1e-6 {
		t.Fatalf("Camera relative transform failed: got %v", camRel[0])
	}
}

func TestCubeSphereNormalization(t *testing.T) {
	faces := []CubeFace{FacePX, FaceNX, FacePY, FaceNY, FacePZ, FaceNZ}
	for _, f := range faces {
		dir := MapCubeToSphere(f, 0.0, 0.0)
		l := dir.Length()
		if math.Abs(l-1.0) > 1e-7 {
			t.Fatalf("Face %v center vector length != 1: %v", f, l)
		}

		cornerDir := MapCubeToSphere(f, 1.0, 1.0)
		if math.Abs(cornerDir.Length()-1.0) > 1e-7 {
			t.Fatalf("Face %v corner vector length != 1: %v", f, cornerDir.Length())
		}
	}
}

func TestQuadtreeSubdivisionAndHorizonCulling(t *testing.T) {
	rEarth := 6371000.0
	rootNode := NewQuadNode(FacePZ, -1.0, -1.0, 2.0, 0, rEarth)
	if rootNode.Center.Length() == 0 {
		t.Fatal("Root node center is zero")
	}

	rootNode.Subdivide(rEarth)
	for i, ch := range rootNode.Children {
		if ch == nil {
			t.Fatalf("Child %d is nil", i)
		}
		if ch.Level != 1 {
			t.Fatalf("Child %d expected level 1, got %d", i, ch.Level)
		}
	}

	// Camera positioned high above +Z pole at altitude 1,000,000m (total R = 7,371,000m)
	camPos := Vec3d{X: 0, Y: 0, Z: rEarth + 1000000.0}

	// +Z face tile should NOT be horizon culled
	if rootNode.IsHorizonCulled(camPos, rEarth) {
		t.Fatal("Front-facing node was incorrectly culled by horizon test")
	}

	// Node on opposite side (-Z face) should be horizon culled
	backNode := NewQuadNode(FaceNZ, -1.0, -1.0, 2.0, 0, rEarth)
	if !backNode.IsHorizonCulled(camPos, rEarth) {
		t.Fatal("Back-facing node should have been culled by horizon test")
	}
}
