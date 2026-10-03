package engine

import (
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
