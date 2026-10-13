package engine

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"integin/pkg/cad/dxf"
)

// Simulator default scenario mirror (tools/lifting-simulator/3d/app.js):
// Alpha x=-15 boom 30 m @65 deg slew 15, Beta x=+15 boom 28 m @60 deg slew
// -20, 45 t shared load. The views must render exactly what the solver
// computes — never redrawn numbers.
func simScenario() (CraneKinematics, CraneKinematics, float64) {
	c1 := CraneKinematics{
		BasePosition:       dxf.Point3D{X: -15},
		BoomLengthMeters:   30,
		BoomAngleDeg:       65,
		SlewAngleDeg:       15,
		CounterweightTonne: 20,
		OutriggerSpreadXM:  8.5,
		OutriggerSpreadZM:  8.5,
		ChassisWeightTonne: 60,
	}
	c2 := CraneKinematics{
		BasePosition:       dxf.Point3D{X: 15},
		BoomLengthMeters:   28,
		BoomAngleDeg:       60,
		SlewAngleDeg:       -20,
		CounterweightTonne: 20,
		OutriggerSpreadXM:  8.5,
		OutriggerSpreadZM:  8.5,
		ChassisWeightTonne: 60,
	}
	return c1, c2, 45.0
}

func countType(d *dxf.Drawing, t dxf.EntityType) int {
	n := 0
	for _, e := range d.Entities {
		if e.Type() == t {
			n++
		}
	}
	return n
}

func TestPlanViewGolden(t *testing.T) {
	c1, c2, load := simScenario()
	res, err := SolveTandemLift(c1, c2, load, 0)
	if err != nil {
		t.Fatal(err)
	}
	d, err := PlanView(c1, c2, res)
	if err != nil {
		t.Fatal(err)
	}
	// Per crane: 4 outrigger lines + 2 hook marks = 6; plus span line.
	if got := countType(d, dxf.EntityLine); got != 13 {
		t.Fatalf("plan lines=%d want 13", got)
	}
	if got := countType(d, dxf.EntityCircle); got != 2 {
		t.Fatalf("plan circles=%d want 2", got)
	}
	if got := countType(d, dxf.EntityText); got != 3 {
		t.Fatalf("plan texts=%d want 3", got)
	}
	for _, e := range d.Entities {
		c, ok := e.(*dxf.Circle)
		if !ok {
			continue
		}
		dx1 := c.Center.X - c1.BasePosition.X
		dz1 := c.Center.Y - c1.BasePosition.Z
		dx2 := c.Center.X - c2.BasePosition.X
		dz2 := c.Center.Y - c2.BasePosition.Z
		r1 := math.Hypot(dx1, dz1)
		r2 := math.Hypot(dx2, dz2)
		if math.Abs(r1) < 1e-9 && math.Abs(c.Radius-res.Hook1State.WorkingRadius) > 1e-9 {
			t.Fatalf("crane-1 circle r=%g want solver %g", c.Radius, res.Hook1State.WorkingRadius)
		}
		if math.Abs(r2) < 1e-9 && math.Abs(c.Radius-res.Hook2State.WorkingRadius) > 1e-9 {
			t.Fatalf("crane-2 circle r=%g want solver %g", c.Radius, res.Hook2State.WorkingRadius)
		}
	}
	var buf bytes.Buffer
	if err := dxf.NewWriter(&buf).Write(d); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "CIRCLE") || !strings.Contains(buf.String(), "CRANE-1") {
		t.Fatal("DXF bytes must contain circle entities and crane callouts")
	}
}

func TestElevationViewGolden(t *testing.T) {
	c1, c2, load := simScenario()
	res, err := SolveTandemLift(c1, c2, load, 0)
	if err != nil {
		t.Fatal(err)
	}
	d, err := ElevationView(c1, c2, res)
	if err != nil {
		t.Fatal(err)
	}
	// ground + 2 booms + 2 drops + 1 span = 6 lines, 1 title text.
	if got := countType(d, dxf.EntityLine); got != 6 {
		t.Fatalf("elevation lines=%d want 6", got)
	}
	if got := countType(d, dxf.EntityText); got != 1 {
		t.Fatalf("elevation texts=%d want 1", got)
	}
	// Boom lines must run base(hook x, 0) -> hook tip.
	found := 0
	for _, e := range d.Entities {
		l, ok := e.(*dxf.Line)
		if !ok || l.Layer() != LayerCranes {
			continue
		}
		if l.Start.Y == 0 && l.End.Y == 0 {
			continue // ground line
		}
		found++
		if math.Abs(l.End.Y-res.Hook1State.HookTip.Y) > 1e-9 && math.Abs(l.End.Y-res.Hook2State.HookTip.Y) > 1e-9 {
			t.Fatalf("boom tip height %g matches neither hook", l.End.Y)
		}
	}
	if found != 2 {
		t.Fatalf("want 2 boom lines, got %d", found)
	}
	var buf bytes.Buffer
	if err := dxf.NewWriter(&buf).Write(d); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "LINE") {
		t.Fatal("DXF bytes must contain line entities")
	}
}

func TestLiftViewsRefuse(t *testing.T) {
	c1, c2, load := simScenario()
	res, err := SolveTandemLift(c1, c2, load, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PlanView(c1, c2, nil); err == nil {
		t.Fatal("nil result must refuse")
	}
	bad := *res
	bad.HookSpanMeters = math.NaN()
	if _, err := PlanView(c1, c2, &bad); err == nil {
		t.Fatal("NaN span must refuse")
	}
	if _, err := ElevationView(c1, c2, nil); err == nil {
		t.Fatal("nil result must refuse")
	}
}
