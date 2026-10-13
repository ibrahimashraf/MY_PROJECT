package engine

import (
	"errors"
	"strings"
	"testing"

	"integin/pkg/cad/dxf"
)

func TestParseOBJMeshRejectsEmptyGeometry(t *testing.T) {
	if _, err := ParseOBJMesh(""); err == nil {
		t.Fatal("empty OBJ must be rejected")
	}
}

func TestParseOBJMeshRejectsMalformedVertex(t *testing.T) {
	if _, err := ParseOBJMesh("v 1 nope 0\nf 1 1 1"); err == nil {
		t.Fatal("malformed OBJ vertex must be rejected")
	}
}

func TestParseOBJMeshRejectsGeometryWithoutFaces(t *testing.T) {
	if _, err := ParseOBJMesh("v 0 0 0\nv 1 0 0\nv 0 1 0"); err == nil {
		t.Fatal("OBJ without faces must be rejected")
	}
}

func TestCheckClearanceRejectsMissingInputs(t *testing.T) {
	if CheckClearance(nil, &Trajectory4DResult{}, []LiftStage{{}}, CraneKinematics{}, CraneKinematics{}, MassSpec{LengthM: 1, RadiusM: 1}) {
		t.Fatal("nil mesh must fail closed")
	}
	mesh, err := ParseOBJMesh("v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3")
	if err != nil {
		t.Fatal(err)
	}
	if CheckClearance(mesh, nil, []LiftStage{{}}, CraneKinematics{}, CraneKinematics{}, MassSpec{LengthM: 1, RadiusM: 1}) {
		t.Fatal("nil trajectory must fail closed")
	}
}

func TestCheckClearanceRejectsHookCollision(t *testing.T) {
	mesh, err := ParseOBJMesh("v 0 10 0\nv 1 10 0\nv 0 11 0\nf 1 2 3")
	if err != nil {
		t.Fatal(err)
	}
	crane := CraneKinematics{BasePosition: dxf.Point3D{}, BoomLengthMeters: 10, BoomAngleDeg: 90}
	trajectory := &Trajectory4DResult{Passed: true}
	if CheckClearance(mesh, trajectory, []LiftStage{{TimeS: 3, Crane1Angle: 90}}, crane, crane, MassSpec{LengthM: 2, RadiusM: 1}) {
		t.Fatal("hook/mesh collision must fail clearance")
	}
	if trajectory.FailTimeS != 3 {
		t.Fatalf("failure time=%v want 3", trajectory.FailTimeS)
	}
}

func TestCheckClearanceRejectsFaceCentroidCollision(t *testing.T) {
	// Vertices at (-3, 10, 0), (3, 10, 0), (0, 10, -3) are > 2.5m away from hook tip (0, 10, 0)
	// But triangle centroid is (0, 10, -1), which directly penetrates hook/load space
	mesh, err := ParseOBJMesh("v -3 10 0\nv 3 10 0\nv 0 10 -3\nf 1 2 3")
	if err != nil {
		t.Fatal(err)
	}
	crane := CraneKinematics{BasePosition: dxf.Point3D{}, BoomLengthMeters: 10, BoomAngleDeg: 90}
	trajectory := &Trajectory4DResult{Passed: true}
	if CheckClearance(mesh, trajectory, []LiftStage{{TimeS: 5, Crane1Angle: 90}}, crane, crane, MassSpec{LengthM: 4, RadiusM: 1}) {
		t.Fatal("face centroid collision must fail clearance")
	}
	if trajectory.FailTimeS != 5 {
		t.Fatalf("failure time=%v want 5", trajectory.FailTimeS)
	}
}

func TestParseOBJMeshRejectsVertexLimitExceeded(t *testing.T) {
	var sb strings.Builder
	for i := 0; i <= MaxOBJVertices+1; i++ {
		sb.WriteString("v 1.0 1.0 1.0\n")
	}
	sb.WriteString("f 1 2 3\n")
	_, err := ParseOBJMesh(sb.String())
	if err == nil {
		t.Fatal("expected error for exceeding vertex limit")
	}
	if !errors.Is(err, ErrVertexLimitExceeded) {
		t.Fatalf("expected ErrVertexLimitExceeded, got %v", err)
	}
}

func TestParseOBJMeshRejectsFaceLimitExceeded(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("v 0 0 0\nv 1 0 0\nv 0 1 0\n")
	for i := 0; i <= MaxOBJFaces+1; i++ {
		sb.WriteString("f 1 2 3\n")
	}
	_, err := ParseOBJMesh(sb.String())
	if err == nil {
		t.Fatal("expected error for exceeding face limit")
	}
	if !errors.Is(err, ErrFaceLimitExceeded) {
		t.Fatalf("expected ErrFaceLimitExceeded, got %v", err)
	}
}

func TestCheckClearanceAABBDiscardsDistantMeshFast(t *testing.T) {
	mesh, err := ParseOBJMesh("v 100 10 0\nv 101 10 0\nv 100 11 0\nf 1 2 3")
	if err != nil {
		t.Fatal(err)
	}
	crane := CraneKinematics{BasePosition: dxf.Point3D{}, BoomLengthMeters: 10, BoomAngleDeg: 90}
	trajectory := &Trajectory4DResult{Passed: true}
	if !CheckClearance(mesh, trajectory, []LiftStage{{TimeS: 0, Crane1Angle: 90}}, crane, crane, MassSpec{LengthM: 2, RadiusM: 1}) {
		t.Fatal("distant mesh outside AABB must pass clearance")
	}
	if !trajectory.Passed {
		t.Fatal("trajectory must remain passed")
	}
}
