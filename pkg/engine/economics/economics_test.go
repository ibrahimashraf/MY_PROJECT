package economics

import (
	"math"
	"testing"
)

func TestNashEquilibrium(t *testing.T) {
	// Prisoner's Dilemma: Nash equilibrium = (Defect, Defect) = (0, 0)
	game := BimatrixGame{
		RowPayoffs: [][]float64{{3, 0}, {5, 1}},
		ColPayoffs: [][]float64{{3, 5}, {0, 1}},
	}
	nash := game.FindPureNashEquilibria()
	if len(nash) != 1 || nash[0] != [2]int{1, 1} {
		t.Fatalf("Expected single NE at (1,1), got %v", nash)
	}
}

func TestDijkstraTerrainPath(t *testing.T) {
	// Simple 4-node graph: 0-1-2-3 with direct path cost
	graph := [][]float64{
		{0, 1, 0, 0},
		{1, 0, 2, 0},
		{0, 2, 0, 1},
		{0, 0, 1, 0},
	}
	dist := DijkstraShortestPath(graph, 0)
	// dist[3] should be 0->1->2->3 = 1+2+1 = 4
	if math.Abs(dist[3]-4.0) > 1e-9 {
		t.Fatalf("Dijkstra shortest path mismatch: expected 4, got %v", dist[3])
	}
}

func TestDemandElasticity(t *testing.T) {
	// Elastic demand (eta=2): price rises 10% -> quantity falls ~19%
	q := ExponentialDemandElasticity(100.0, 10.0, 11.0, 2.0)
	// Q = 100 * (11/10)^-2 = 100 * 0.8264 = 82.64
	expected := 100.0 * math.Pow(1.1, -2.0)
	if math.Abs(q-expected) > 1e-6 {
		t.Fatalf("Demand elasticity mismatch: expected %v, got %v", expected, q)
	}
}
