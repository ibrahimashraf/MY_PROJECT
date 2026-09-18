package engine

import (
	"errors"
	"fmt"
	"math"

	"integin/pkg/cad/dxf"
)

// Lift-view layers. One drawing, fixed layer set — no per-crane-type files.
const (
	LayerCranes    = "CRANES"
	LayerRigging   = "RIGGING"
	LayerClearance = "CLEARANCE"
	LayerAnnot     = "ANNOT"
)

func liftViewLayer(d *dxf.Drawing, name string, color int16) {
	d.Layers[name] = &dxf.Layer{Name: name, Color: color, LineType: "CONTINUOUS", IsVisible: true}
}

func newLiftViewDrawing() *dxf.Drawing {
	d := dxf.NewDrawing()
	liftViewLayer(d, LayerCranes, 30)
	liftViewLayer(d, LayerRigging, 1)
	liftViewLayer(d, LayerClearance, 4)
	liftViewLayer(d, LayerAnnot, 7)
	return d
}

func liftText(layer string, at dxf.Point3D, height float64, value string) *dxf.Text {
	return &dxf.Text{BaseEntity: dxf.BaseEntity{LayerName: layer}, Insertion: at, Height: height, Value: value}
}

func liftLine(layer string, from, to dxf.Point3D) *dxf.Line {
	return &dxf.Line{BaseEntity: dxf.BaseEntity{LayerName: layer}, Start: from, End: to}
}

func liftCircle(layer string, at dxf.Point3D, radius float64) *dxf.Circle {
	return &dxf.Circle{BaseEntity: dxf.BaseEntity{LayerName: layer}, Center: at, Radius: radius}
}

func checkLiftViewInputs(c1, c2 CraneKinematics, res *TandemLiftResult) error {
	if res == nil {
		return errors.New("liftview: nil tandem result")
	}
	for _, v := range []float64{
		c1.BoomLengthMeters, c1.BoomAngleDeg, c1.SlewAngleDeg,
		c2.BoomLengthMeters, c2.BoomAngleDeg, c2.SlewAngleDeg,
		res.HookSpanMeters, res.MinBoomClearanceM,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return errors.New("liftview: NaN/Inf input rejected")
		}
	}
	if res.HookSpanMeters <= 0 {
		return fmt.Errorf("liftview: hook span must be > 0, got %g", res.HookSpanMeters)
	}
	return nil
}

// outriggerRect returns the four outrigger pad corners in plan (X-Z) for a
// crane, derived from its base position and spreads.
func outriggerRect(c CraneKinematics) [4]dxf.Point3D {
	hx := c.OutriggerSpreadXM / 2
	hz := c.OutriggerSpreadZM / 2
	b := c.BasePosition
	return [4]dxf.Point3D{
		{X: b.X - hx, Y: b.Y, Z: b.Z - hz},
		{X: b.X + hx, Y: b.Y, Z: b.Z - hz},
		{X: b.X + hx, Y: b.Y, Z: b.Z + hz},
		{X: b.X - hx, Y: b.Y, Z: b.Z + hz},
	}
}

// PlanView renders the top-down lift drawing (Parker P4 equivalent) from the
// solved tandem state: outrigger rectangles, working-radius circles, hook
// positions, hook-span dimension, and per-crane load/max-outrigger callouts.
// DXF X = world X, DXF Y = world Z.
func PlanView(c1, c2 CraneKinematics, res *TandemLiftResult) (*dxf.Drawing, error) {
	if err := checkLiftViewInputs(c1, c2, res); err != nil {
		return nil, err
	}
	d := newLiftViewDrawing()
	th := math.Max(0.4, res.HookSpanMeters*0.02)
	cranes := []struct {
		c    CraneKinematics
		hook dxf.Point3D
		load float64
		maxP float64
		name string
	}{
		{c1, res.Hook1State.HookTip, res.Crane1LoadTonnes, res.Crane1Outriggers.MaxLoad, "CRANE-1"},
		{c2, res.Hook2State.HookTip, res.Crane2LoadTonnes, res.Crane2Outriggers.MaxLoad, "CRANE-2"},
	}
	for _, k := range cranes {
		b := dxf.Point3D{X: k.c.BasePosition.X, Y: k.c.BasePosition.Z}
		rect := outriggerRect(k.c)
		for i := 0; i < 4; i++ {
			a := rect[i]
			e := rect[(i+1)%4]
			d.Entities = append(d.Entities, liftLine(LayerCranes,
				dxf.Point3D{X: a.X, Y: a.Z}, dxf.Point3D{X: e.X, Y: e.Z}))
		}
		radius := math.Hypot(k.hook.X-k.c.BasePosition.X, k.hook.Z-k.c.BasePosition.Z)
		d.Entities = append(d.Entities, liftCircle(LayerClearance, b, radius))
		h := dxf.Point3D{X: k.hook.X, Y: k.hook.Z}
		d.Entities = append(d.Entities,
			liftLine(LayerRigging, dxf.Point3D{X: h.X - 1, Y: h.Y}, dxf.Point3D{X: h.X + 1, Y: h.Y}),
			liftLine(LayerRigging, dxf.Point3D{X: h.X, Y: h.Y - 1}, dxf.Point3D{X: h.X, Y: h.Y + 1}),
		)
		d.Entities = append(d.Entities, liftText(LayerAnnot,
			dxf.Point3D{X: b.X, Y: b.Y + radius + th},
			th,
			fmt.Sprintf("%s LOAD %.1ft SHARE %.0f%% MAX-PAD %.1ft", k.name, k.load,
				100*k.load/(res.Crane1LoadTonnes+res.Crane2LoadTonnes), k.maxP)))
	}
	h1 := dxf.Point3D{X: res.Hook1State.HookTip.X, Y: res.Hook1State.HookTip.Z}
	h2 := dxf.Point3D{X: res.Hook2State.HookTip.X, Y: res.Hook2State.HookTip.Z}
	d.Entities = append(d.Entities, liftLine(LayerRigging, h1, h2))
	mid := dxf.Point3D{X: (h1.X + h2.X) / 2, Y: (h1.Y + h2.Y) / 2}
	d.Entities = append(d.Entities, liftText(LayerAnnot,
		dxf.Point3D{X: mid.X, Y: mid.Y + th}, th,
		fmt.Sprintf("SPAN %.2fm CLEAR %.2fm", res.HookSpanMeters, res.MinBoomClearanceM)))
	return d, nil
}

// ElevationView renders the side lift drawing (Parker P5 equivalent): ground
// line, both booms as base-to-tip lines, hook drops, and the suspended span.
// DXF X = world X, DXF Y = world height.
func ElevationView(c1, c2 CraneKinematics, res *TandemLiftResult) (*dxf.Drawing, error) {
	if err := checkLiftViewInputs(c1, c2, res); err != nil {
		return nil, err
	}
	d := newLiftViewDrawing()
	th := math.Max(0.4, res.HookSpanMeters*0.02)
	minX := math.Min(c1.BasePosition.X, c2.BasePosition.X) - res.HookSpanMeters*0.3
	maxX := math.Max(c1.BasePosition.X, c2.BasePosition.X) + res.HookSpanMeters*0.3
	d.Entities = append(d.Entities, liftLine(LayerCranes,
		dxf.Point3D{X: minX, Y: 0}, dxf.Point3D{X: maxX, Y: 0}))
	hooks := []dxf.Point3D{res.Hook1State.HookTip, res.Hook2State.HookTip}
	bases := []dxf.Point3D{c1.BasePosition, c2.BasePosition}
	for i := 0; i < 2; i++ {
		base := dxf.Point3D{X: bases[i].X, Y: 0}
		tip := dxf.Point3D{X: hooks[i].X, Y: hooks[i].Y}
		d.Entities = append(d.Entities, liftLine(LayerCranes, base, tip))
		d.Entities = append(d.Entities, liftLine(LayerRigging, tip, dxf.Point3D{X: tip.X, Y: 0}))
	}
	d.Entities = append(d.Entities, liftLine(LayerRigging,
		dxf.Point3D{X: hooks[0].X, Y: hooks[0].Y}, dxf.Point3D{X: hooks[1].X, Y: hooks[1].Y}))
	top := math.Max(hooks[0].Y, hooks[1].Y)
	d.Entities = append(d.Entities, liftText(LayerAnnot,
		dxf.Point3D{X: minX, Y: top + th}, th,
		fmt.Sprintf("TANDEM SPAN %.2fm CLEAR %.2fm C1 %.1ft C2 %.1ft",
			res.HookSpanMeters, res.MinBoomClearanceM, res.Crane1LoadTonnes, res.Crane2LoadTonnes)))
	return d, nil
}
