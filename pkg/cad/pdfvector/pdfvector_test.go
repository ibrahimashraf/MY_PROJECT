package pdfvector

import (
	"testing"

	"integin/pkg/cad/dxf"
)

func TestPDFVectorExtraction(t *testing.T) {
	// Sample PDF drawing stream with lines, bezier curves, and a rectangle
	stream := `
		q
		10 20 m
		50 60 l
		S
		100 100 200 150 re
		f
		300 300 m
		320 350 380 350 400 300 c
		s
		Q
	`

	extractor := NewExtractor()
	entities, err := extractor.ParseStream(stream)
	if err != nil {
		t.Fatalf("ParseStream failed: %v", err)
	}

	if len(entities) != 3 {
		t.Fatalf("Expected 3 CAD entities from PDF stream, got %d", len(entities))
	}

	// 1. Check Line
	line, ok := entities[0].(*dxf.Line)
	if !ok {
		t.Fatalf("Entity 0 should be *dxf.Line, got %T", entities[0])
	}
	if line.Start.X != 10 || line.Start.Y != 20 || line.End.X != 50 || line.End.Y != 60 {
		t.Errorf("Line coordinates unexpected: start %v, end %v", line.Start, line.End)
	}

	// 2. Check Rectangle (Closed Polyline)
	rect, ok := entities[1].(*dxf.Polyline)
	if !ok {
		t.Fatalf("Entity 1 should be *dxf.Polyline, got %T", entities[1])
	}
	if !rect.IsClosed || len(rect.Vertices) != 4 {
		t.Errorf("Rectangle polyline expected 4 vertices and closed, got %d vertices, closed=%v", len(rect.Vertices), rect.IsClosed)
	}

	// 3. Check Bézier Curve (Segmented Polyline)
	bezier, ok := entities[2].(*dxf.Polyline)
	if !ok {
		t.Fatalf("Entity 2 should be *dxf.Polyline, got %T", entities[2])
	}
	if len(bezier.Vertices) < 5 {
		t.Errorf("Bézier polyline expected to be subdivided into at least 5 segments, got %d", len(bezier.Vertices))
	}
	// Verify start and end points
	start := bezier.Vertices[0].Point
	end := bezier.Vertices[len(bezier.Vertices)-1].Point
	if start.X != 300 || start.Y != 300 {
		t.Errorf("Bézier start mismatch: %v", start)
	}
	if end.X != 400 || end.Y != 300 {
		t.Errorf("Bézier end mismatch: %v", end)
	}
}

func TestAffineMatrixTransform(t *testing.T) {
	// PDF stream with translation and scaling: 2 0 0 2 50 50 cm
	stream := `
		2 0 0 2 50 50 cm
		10 10 m
		20 20 l
		S
	`

	extractor := NewExtractor()
	entities, err := extractor.ParseStream(stream)
	if err != nil {
		t.Fatalf("ParseStream failed: %v", err)
	}

	if len(entities) != 1 {
		t.Fatalf("Expected 1 entity, got %d", len(entities))
	}

	line := entities[0].(*dxf.Line)
	// (10 * 2) + 50 = 70; (20 * 2) + 50 = 90
	if line.Start.X != 70 || line.Start.Y != 70 {
		t.Errorf("Expected transformed start (70, 70), got (%f, %f)", line.Start.X, line.Start.Y)
	}
	if line.End.X != 90 || line.End.Y != 90 {
		t.Errorf("Expected transformed end (90, 90), got (%f, %f)", line.End.X, line.End.Y)
	}
}
