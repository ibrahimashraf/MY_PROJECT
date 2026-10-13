package dxf

import (
	"bytes"
	"testing"
)

func TestDXFRoundTrip(t *testing.T) {
	d := NewDrawing()

	// Add custom layer
	d.Layers["CRANE_RIGGING"] = &Layer{
		Name:      "CRANE_RIGGING",
		Color:     1, // Red
		LineType:  "CONTINUOUS",
		IsVisible: true,
	}

	// Add Entities
	d.AddEntity(&Line{
		BaseEntity: BaseEntity{LayerName: "CRANE_RIGGING", Color: 1},
		Start:      Point3D{X: 0, Y: 0, Z: 0},
		End:        Point3D{X: 25.5, Y: 10.0, Z: 35.0},
	})

	d.AddEntity(&Circle{
		BaseEntity: BaseEntity{LayerName: "CRANE_RIGGING"},
		Center:     Point3D{X: 25.5, Y: 10.0, Z: 0},
		Radius:     5.25,
	})

	d.AddEntity(&Arc{
		BaseEntity: BaseEntity{LayerName: "0"},
		Center:     Point3D{X: 0, Y: 0, Z: 0},
		Radius:     15.0,
		StartAngle: 30.0,
		EndAngle:   90.0,
	})

	d.AddEntity(&Polyline{
		BaseEntity: BaseEntity{LayerName: "0"},
		IsClosed:   true,
		Vertices: []Vertex{
			{Point: Point3D{X: -5, Y: -5, Z: 0}},
			{Point: Point3D{X: 5, Y: -5, Z: 0}},
			{Point: Point3D{X: 5, Y: 5, Z: 0}},
			{Point: Point3D{X: -5, Y: 5, Z: 0}},
		},
	})

	d.AddEntity(&Text{
		BaseEntity: BaseEntity{LayerName: "CRANE_RIGGING"},
		Insertion:  Point3D{X: 10, Y: 20, Z: 0},
		Height:     2.5,
		Value:      "OUTRIGGER PAD #1",
	})

	// Add Block
	shackleBlock := &Block{
		Name:      "SHACKLE_55T",
		BasePoint: Point3D{X: 0, Y: 0, Z: 0},
		LayerName: "0",
		Entities: []Entity{
			&Circle{
				BaseEntity: BaseEntity{LayerName: "0"},
				Center:     Point3D{X: 0, Y: 0, Z: 0},
				Radius:     1.2,
			},
		},
	}
	d.AddBlock(shackleBlock)

	d.AddEntity(&Insert{
		BaseEntity:  BaseEntity{LayerName: "CRANE_RIGGING"},
		BlockName:   "SHACKLE_55T",
		Insertion:   Point3D{X: 25.5, Y: 10.0, Z: 35.0},
		ScaleX:      1.0,
		ScaleY:      1.0,
		ScaleZ:      1.0,
		RotationDeg: 45.0,
	})

	// Serialize to DXF buffer
	var buf bytes.Buffer
	writer := NewWriter(&buf)
	if err := writer.Write(d); err != nil {
		t.Fatalf("DXF Write failed: %v", err)
	}

	// Parse back from buffer
	reader := NewReader(&buf)
	parsed, err := reader.Read()
	if err != nil {
		t.Fatalf("DXF Read failed: %v", err)
	}

	// Assertions
	if len(parsed.Entities) != len(d.Entities) {
		t.Fatalf("Expected %d entities, got %d", len(d.Entities), len(parsed.Entities))
	}

	if _, ok := parsed.Layers["CRANE_RIGGING"]; !ok {
		t.Errorf("Expected CRANE_RIGGING layer to be restored")
	}

	if _, ok := parsed.Blocks["SHACKLE_55T"]; !ok {
		t.Errorf("Expected SHACKLE_55T block definition to be restored")
	}

	// Verify Line coordinates
	l, ok := parsed.Entities[0].(*Line)
	if !ok {
		t.Fatalf("Expected entity 0 to be Line, got %T", parsed.Entities[0])
	}
	if l.End.X != 25.5 || l.End.Z != 35.0 {
		t.Errorf("Line coordinates mismatch: got end %v", l.End)
	}
}
