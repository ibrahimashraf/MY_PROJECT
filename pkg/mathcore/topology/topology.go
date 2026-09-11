package topology

import "fmt"

// Simplex2D represents a triangle face defined by 3 vertex indices.
type Simplex2D [3]int

// Edge represents an undirected connection between two vertex indices.
type Edge [2]int

// SimplicialComplex2D represents a 2D triangulated mesh topology.
type SimplicialComplex2D struct {
	NumVertices int
	Edges       map[string]Edge
	Faces       []Simplex2D
}

// NewSimplicialComplex initializes an empty 2D simplicial complex.
func NewSimplicialComplex(numVertices int) *SimplicialComplex2D {
	return &SimplicialComplex2D{
		NumVertices: numVertices,
		Edges:       make(map[string]Edge),
		Faces:       make([]Simplex2D, 0),
	}
}

// AddFace inserts a 2-simplex (triangle) and extracts its 1-simplices (edges).
func (sc *SimplicialComplex2D) AddFace(v0, v1, v2 int) {
	face := Simplex2D{v0, v1, v2}
	sc.Faces = append(sc.Faces, face)

	sc.addEdge(v0, v1)
	sc.addEdge(v1, v2)
	sc.addEdge(v2, v0)
}

func (sc *SimplicialComplex2D) addEdge(u, v int) {
	if u > v {
		u, v = v, u
	}
	key := fmt.Sprintf("%d-%d", u, v)
	sc.Edges[key] = Edge{u, v}
}

// EulerCharacteristic calculates chi = V - E + F.
func (sc *SimplicialComplex2D) EulerCharacteristic() int {
	v := sc.NumVertices
	e := len(sc.Edges)
	f := len(sc.Faces)
	return v - e + f
}

// Genus calculates the topological genus g = 1 - chi/2 for closed orientable surfaces.
func (sc *SimplicialComplex2D) Genus() (int, error) {
	chi := sc.EulerCharacteristic()
	if chi%2 != 0 {
		return 0, fmt.Errorf("euler characteristic %d is odd; manifold may have boundary or non-orientable topology", chi)
	}
	return 1 - chi/2, nil
}

// TetrahedronComplex builds a closed simplicial 2-sphere (V=4, E=6, F=4, chi=2, genus=0).
func TetrahedronComplex() *SimplicialComplex2D {
	sc := NewSimplicialComplex(4)
	sc.AddFace(0, 1, 2)
	sc.AddFace(0, 2, 3)
	sc.AddFace(0, 3, 1)
	sc.AddFace(1, 3, 2)
	return sc
}

// TorusComplex builds a minimal triangulated 2-torus (V=7, E=21, F=14, chi=0, genus=1).
func TorusComplex() *SimplicialComplex2D {
	sc := NewSimplicialComplex(7)
	// Canonical Mobius 7-vertex triangulation of the 2-torus
	faces := [][3]int{
		{0, 1, 3}, {1, 2, 4}, {2, 3, 5}, {3, 4, 6}, {4, 5, 0}, {5, 6, 1}, {6, 0, 2},
		{3, 2, 0}, {4, 3, 1}, {5, 4, 2}, {6, 5, 3}, {0, 6, 4}, {1, 0, 5}, {2, 1, 6},
	}
	for _, f := range faces {
		sc.AddFace(f[0], f[1], f[2])
	}
	return sc
}
