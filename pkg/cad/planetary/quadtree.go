package planetary

import (
	"math"
)

// CubeFace represents one of the 6 faces of the cube-sphere.
type CubeFace int

const (
	FaceFront CubeFace = iota
	FaceBack
	FaceLeft
	FaceRight
	FaceTop
	FaceBottom
)

// QuadNode is a node in the cube-sphere quadtree.
type QuadNode struct {
	Face     CubeFace
	Level    int
	Bounds   [4]float64
	Children [4]*QuadNode
}

// HorizonCull checks if a bounding sphere is completely behind the planetary horizon from the camera's perspective.
func HorizonCull(cam Camera64, nodeCenter [3]float64, nodeRadius float64) bool {
	camDistSq := cam.X*cam.X + cam.Y*cam.Y + cam.Z*cam.Z
	rPlanetSq := WGS84b * WGS84b

	if camDistSq <= rPlanetSq {
		return false // camera at or inside planet surface
	}

	horizonDistSq := camDistSq - rPlanetSq
	horizonDist := math.Sqrt(horizonDistSq)

	dx := nodeCenter[0] - cam.X
	dy := nodeCenter[1] - cam.Y
	dz := nodeCenter[2] - cam.Z
	distToNode := math.Sqrt(dx*dx + dy*dy + dz*dz)

	// If sphere is completely behind the horizon line of sight
	if distToNode-nodeRadius > horizonDist {
		// Verify angle between camera direction and node position from planet center
		camDotNode := cam.X*nodeCenter[0] + cam.Y*nodeCenter[1] + cam.Z*nodeCenter[2]
		if camDotNode < 0 {
			return true // beyond the limb on the opposite hemisphere
		}
	}
	return false
}

// BuildTagScheduler schedules the building of quadtree nodes based on priority/visibility.
type BuildTagScheduler struct {
	Queue []*QuadNode
}

// Enqueue adds a node to the scheduling queue.
func (s *BuildTagScheduler) Enqueue(node *QuadNode) {
	s.Queue = append(s.Queue, node)
}

// Process processes the build queue by invoking the provided handler.
func (s *BuildTagScheduler) Process(handler func(node *QuadNode)) {
	for len(s.Queue) > 0 {
		node := s.Queue[0]
		s.Queue = s.Queue[1:]
		if handler != nil {
			handler(node)
		}
	}
}
