package engine

import (
	"testing"
)

func TestIngestOBJEnvelopeSuccess(t *testing.T) {
	// Simple column/vessel prism in OBJ
	objData := `
v -1.5 0 -1.5
v 1.5 0 -1.5
v 1.5 20 -1.5
v -1.5 20 -1.5
v -1.5 0 1.5
v 1.5 0 1.5
v 1.5 20 1.5
v -1.5 20 1.5
f 1 2 3
f 1 3 4
f 5 6 7
f 5 7 8
`
	liftPoints := []LiftPoint{
		{ID: "TRUNNION-1", Position: Vertex{X: -1.5, Y: 15, Z: 0}, RatedCapacityT: 35.0, Type: "TRUNNION"},
		{ID: "TRUNNION-2", Position: Vertex{X: 1.5, Y: 15, Z: 0}, RatedCapacityT: 35.0, Type: "TRUNNION"},
	}

	env, err := IngestOBJEnvelope("ENV-COL-01", "C-101-DEETHANIZER", objData, 60.0, liftPoints)
	if err != nil {
		t.Fatalf("expected successful ingestion, got error: %v", err)
	}

	if env.AssetTag != "C-101-DEETHANIZER" {
		t.Errorf("expected asset tag C-101-DEETHANIZER, got %s", env.AssetTag)
	}
	if env.MassSpec.MassTonnes != 60.0 {
		t.Errorf("expected mass 60t, got %.1f", env.MassSpec.MassTonnes)
	}
	if env.MassSpec.LengthM != 20.0 {
		t.Errorf("expected length 20m, got %.1f", env.MassSpec.LengthM)
	}
	if env.MassSpec.RadiusM != 1.5 {
		t.Errorf("expected radius 1.5m, got %.1f", env.MassSpec.RadiusM)
	}
	if len(env.LiftPoints) != 2 {
		t.Errorf("expected 2 lift points, got %d", len(env.LiftPoints))
	}
}

func TestIngestOBJEnvelopeRejectsInvalidInputs(t *testing.T) {
	validOBJ := "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3"
	validLP := []LiftPoint{{ID: "LP-1", Position: Vertex{}, RatedCapacityT: 10, Type: "PADEYE"}}

	// 1. Missing geometry
	if _, err := IngestOBJEnvelope("ENV-1", "TAG", "", 50, validLP); err == nil {
		t.Error("expected error for empty OBJ geometry")
	}

	// 2. Zero/negative mass
	if _, err := IngestOBJEnvelope("ENV-1", "TAG", validOBJ, 0, validLP); err == nil {
		t.Error("expected error for zero mass")
	}

	// 3. Missing lift points
	if _, err := IngestOBJEnvelope("ENV-1", "TAG", validOBJ, 50, nil); err == nil {
		t.Error("expected error for empty lift points")
	}

	// 4. Invalid lift point capacity
	invalidLP := []LiftPoint{{ID: "LP-1", Position: Vertex{}, RatedCapacityT: 0, Type: "PADEYE"}}
	if _, err := IngestOBJEnvelope("ENV-1", "TAG", validOBJ, 50, invalidLP); err == nil {
		t.Error("expected error for non-positive lift point capacity")
	}
}
