package planet

import (
	"math"
	"testing"
)

func TestTerrainMeshGeneration(t *testing.T) {
	rEarth := 6371000.0
	node := NewQuadNode(FacePX, -1.0, -1.0, 2.0, 0, rEarth)
	camPos := Vec3d{X: rEarth + 1000.0, Y: 0, Z: 0}

	res := 16
	chunk := GenerateChunkMesh(node, res, rEarth, 500.0, camPos)

	expectedVerts := res * res
	if len(chunk.Vertices) != expectedVerts {
		t.Fatalf("Expected %d vertices, got %d", expectedVerts, len(chunk.Vertices))
	}

	expectedIndices := (res - 1) * (res - 1) * 6
	if len(chunk.Indices) != expectedIndices {
		t.Fatalf("Expected %d indices, got %d", expectedIndices, len(chunk.Indices))
	}

	// Verify normals are normalized
	for i, v := range chunk.Vertices {
		l := math.Sqrt(float64(v.NormX*v.NormX + v.NormY*v.NormY + v.NormZ*v.NormZ))
		if math.Abs(l-1.0) > 1e-4 {
			t.Fatalf("Vertex %d normal not unit length: %v", i, l)
		}
	}
}

func TestMultiThreadedScheduler(t *testing.T) {
	rEarth := 6371000.0
	nodes := []*QuadNode{
		NewQuadNode(FacePX, -1.0, -1.0, 2.0, 0, rEarth),
		NewQuadNode(FaceNX, -1.0, -1.0, 2.0, 0, rEarth),
		NewQuadNode(FacePY, -1.0, -1.0, 2.0, 0, rEarth),
		NewQuadNode(FaceNY, -1.0, -1.0, 2.0, 0, rEarth),
	}

	camPos := Vec3d{X: 0, Y: 0, Z: rEarth + 50000.0}
	scheduler := MultiThreadedTerrainScheduler{Workers: 4}

	chunks := scheduler.GeneratePlanetTiles(nodes, 8, rEarth, 200.0, camPos)
	if len(chunks) != len(nodes) {
		t.Fatalf("Expected %d chunks, got %d", len(nodes), len(chunks))
	}
	for i, c := range chunks {
		if c == nil {
			t.Fatalf("Chunk %d is nil", i)
		}
	}
}
