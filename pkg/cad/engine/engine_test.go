package engine

import (
	"math"
	"testing"

	"integin/pkg/cad/dxf"
)

func TestDynamicBlockEvaluation(t *testing.T) {
	// Define a dynamic crane boom block
	def := &DynamicBlockDef{
		Name: "CRANE_BOOM_2D",
		BaseEntities: []dxf.Entity{
			&dxf.Line{
				BaseEntity: dxf.BaseEntity{LayerName: "BOOM"},
				Start:      dxf.Point3D{X: 0, Y: 0, Z: 0},
				End:        dxf.Point3D{X: 30, Y: 0, Z: 0},
			},
			&dxf.Circle{
				BaseEntity: dxf.BaseEntity{LayerName: "SHEAVE"},
				Center:     dxf.Point3D{X: 30, Y: 0, Z: 0},
				Radius:     1.0,
			},
		},
		Parameters: []Parameter{
			{Name: "TELESCOPE_EXT", Type: ParamLinear, DefaultValue: 0.0, MinValue: 0.0, MaxValue: 20.0},
			{Name: "BOOM_ANGLE", Type: ParamRotation, DefaultValue: 0.0, MinValue: 0.0, MaxValue: 85.0},
			{Name: "SHEAVE_SCALE", Type: ParamLinear, DefaultValue: 1.0, MinValue: 0.5, MaxValue: 3.0},
		},
		Actions: []Action{
			{Name: "EXTEND", Type: ActionStretch, ParameterName: "TELESCOPE_EXT", TargetIndices: []int{0}},
			{Name: "LUFF", Type: ActionRotate, ParameterName: "BOOM_ANGLE", TargetIndices: []int{0}},
			{Name: "RESIZE", Type: ActionScale, ParameterName: "SHEAVE_SCALE", TargetIndices: []int{1}},
		},
	}

	inst := NewDynamicBlockInstance(def, dxf.Point3D{X: 10, Y: 5, Z: 0})

	// 1. Evaluate with defaults
	geom1 := inst.EvaluateGeometry()
	if len(geom1) != 2 {
		t.Fatalf("Expected 2 entities, got %d", len(geom1))
	}
	l1 := geom1[0].(*dxf.Line)
	// Base line was (0,0)->(30,0), translated by (10,5) => (10,5)->(40,5)
	if l1.Start.X != 10 || l1.Start.Y != 5 || l1.End.X != 40 || l1.End.Y != 5 {
		t.Errorf("Unexpected default line coordinates: start %v, end %v", l1.Start, l1.End)
	}

	// 2. Set dynamic parameters: stretch +10m, rotate 90 degrees, sheave scale 2x
	if err := inst.SetParam("TELESCOPE_EXT", 10.0); err != nil {
		t.Fatalf("SetParam TELESCOPE_EXT failed: %v", err)
	}
	if err := inst.SetParam("BOOM_ANGLE", 90.0); err != nil {
		t.Fatalf("SetParam BOOM_ANGLE failed: %v", err)
	}
	if err := inst.SetParam("SHEAVE_SCALE", 2.0); err != nil {
		t.Fatalf("SetParam SHEAVE_SCALE failed: %v", err)
	}

	geom2 := inst.EvaluateGeometry()
	l2 := geom2[0].(*dxf.Line)
	// Line stretched to 40m, BOOM_ANGLE requested 90 deg but clamped to MaxValue 85.0 deg!
	// End should be at pivot (10,5) + (40*cos(85°), 40*sin(85°)) => (13.486230, 44.847788)
	expectedX := 10.0 + 40.0*math.Cos(85.0*math.Pi/180.0)
	expectedY := 5.0 + 40.0*math.Sin(85.0*math.Pi/180.0)
	if math.Abs(l2.End.X-expectedX) > 1e-4 || math.Abs(l2.End.Y-expectedY) > 1e-4 {
		t.Errorf("Expected clamped rotated end at (%f, %f), got (%f, %f)", expectedX, expectedY, l2.End.X, l2.End.Y)
	}

	c2 := geom2[1].(*dxf.Circle)
	if c2.Radius != 2.0 {
		t.Errorf("Expected sheave radius 2.0, got %f", c2.Radius)
	}
}

func TestTandemLiftKinematicsAndEquilibrium(t *testing.T) {
	crane1 := CraneKinematics{
		BasePosition:       dxf.Point3D{X: -15, Y: 0, Z: 0},
		BoomLengthMeters:   35.0,
		BoomAngleDeg:       60.0,
		SlewAngleDeg:       0.0,
		CounterweightTonne: 40.0,
		OutriggerSpreadXM:  8.0,
		OutriggerSpreadZM:  8.0,
		ChassisWeightTonne: 60.0,
	}

	crane2 := CraneKinematics{
		BasePosition:       dxf.Point3D{X: 15, Y: 0, Z: 0},
		BoomLengthMeters:   35.0,
		BoomAngleDeg:       60.0,
		SlewAngleDeg:       0.0,
		CounterweightTonne: 40.0,
		OutriggerSpreadXM:  8.0,
		OutriggerSpreadZM:  8.0,
		ChassisWeightTonne: 60.0,
	}

	// 1. Verify individual crane hook position
	// R = 35 * cos(60) = 17.5m
	// Height = 35 * sin(60) = 30.3108m
	h1 := crane1.ComputeHookPosition()
	expectedR := 35.0 * 0.5
	if math.Abs(h1.WorkingRadius-expectedR) > 1e-4 {
		t.Errorf("Expected radius %f, got %f", expectedR, h1.WorkingRadius)
	}

	// 2. Solve tandem lift: 100 tonne load, CoG centered
	res, err := SolveTandemLift(crane1, crane2, 100.0, -1)
	if err != nil {
		t.Fatalf("SolveTandemLift failed: %v", err)
	}

	// With CoG centered, both cranes must share 50%
	if math.Abs(res.LoadShareCrane1-50.0) > 1e-4 || math.Abs(res.LoadShareCrane2-50.0) > 1e-4 {
		t.Errorf("Expected 50/50 load share, got %.2f / %.2f", res.LoadShareCrane1, res.LoadShareCrane2)
	}
	if res.Crane1LoadTonnes != 50.0 || res.Crane2LoadTonnes != 50.0 {
		t.Errorf("Expected 50t each, got %.2f and %.2f", res.Crane1LoadTonnes, res.Crane2LoadTonnes)
	}

	// Verify outrigger loads are non-zero and positive
	if res.Crane1Outriggers.MaxLoad <= 0 || res.Crane2Outriggers.MaxLoad <= 0 {
		t.Errorf("Outrigger max load must be positive")
	}

	// 3. Shift CoG towards crane 1 (25% from Crane 1, 75% from Crane 2)
	// Span is 30m. CoG at 7.5m from Crane 1.
	resOffset, err := SolveTandemLift(crane1, crane2, 100.0, res.HookSpanMeters*0.25)
	if err != nil {
		t.Fatalf("SolveTandemLift offset failed: %v", err)
	}

	// Crane 1 should carry 75% of the load; Crane 2 carries 25%
	if math.Abs(resOffset.LoadShareCrane1-75.0) > 1e-4 {
		t.Errorf("Expected Crane 1 to carry 75%%, got %.2f%%", resOffset.LoadShareCrane1)
	}
	if math.Abs(resOffset.LoadShareCrane2-25.0) > 1e-4 {
		t.Errorf("Expected Crane 2 to carry 25%%, got %.2f%%", resOffset.LoadShareCrane2)
	}
}

func TestSoAPhysicsBufferClearance(t *testing.T) {
	buf := NewSoAPhysicsBuffer(16)

	// Add 3 nodes: Node 0 & Node 1 close (collision), Node 2 distant
	buf.AddNode(0, 0, 0, 1.0)     // Sphere at origin, r=1.0
	buf.AddNode(2.5, 0, 0, 1.0)   // Sphere at (2.5, 0, 0), r=1.0 (margin 1.0 -> 1+1+1=3.0 > 2.5 -> collision)
	buf.AddNode(100.0, 0, 0, 1.0) // Distant sphere

	var collisions []CollisionPair
	hasCollision := buf.SweepClearance(1.0, &collisions)

	if !hasCollision || len(collisions) != 1 {
		t.Fatalf("Expected 1 collision between node 0 and 1, got %d (hasCollision=%v)", len(collisions), hasCollision)
	}

	if collisions[0].IndexA != 0 || collisions[0].IndexB != 1 {
		t.Errorf("Expected collision between indices 0 and 1, got %d and %d", collisions[0].IndexA, collisions[0].IndexB)
	}

	expectedPenetration := float32(3.0 - 2.5) // (1.0 + 1.0 + 1.0) - 2.5 = 0.5
	if math.Abs(float64(collisions[0].Penetration-expectedPenetration)) > 1e-4 {
		t.Errorf("Expected penetration %f, got %f", expectedPenetration, collisions[0].Penetration)
	}
}
