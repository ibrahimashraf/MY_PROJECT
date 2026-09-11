package planet

import (
	"math"
)

// Vec3d represents a double-precision 64-bit vector in geocentric Earth-Centered Earth-Fixed (ECEF) coordinates.
// Float64 is mandatory to prevent precision degradation at planetary scale (R ≈ 6,371,000m).
type Vec3d struct {
	X float64
	Y float64
	Z float64
}

// Length computes Euclidean norm.
func (v Vec3d) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// Normalize projects vector onto unit sphere S^2.
func (v Vec3d) Normalize() Vec3d {
	l := v.Length()
	if l == 0 {
		return Vec3d{0, 1, 0}
	}
	inv := 1.0 / l
	return Vec3d{v.X * inv, v.Y * inv, v.Z * inv}
}

// Sub returns difference vector v - other.
func (v Vec3d) Sub(other Vec3d) Vec3d {
	return Vec3d{v.X - other.X, v.Y - other.Y, v.Z - other.Z}
}

// Add returns vector sum v + other.
func (v Vec3d) Add(other Vec3d) Vec3d {
	return Vec3d{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}

// Scale multiplies vector by scalar s.
func (v Vec3d) Scale(s float64) Vec3d {
	return Vec3d{v.X * s, v.Y * s, v.Z * s}
}

// Dot computes dot product.
func (v Vec3d) Dot(other Vec3d) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// ToFloat32 converts geocentric relative offset to 32-bit floating point for WebGPU/Vulkan vertex buffers.
func (v Vec3d) ToFloat32() [3]float32 {
	return [3]float32{float32(v.X), float32(v.Y), float32(v.Z)}
}

// CameraRelativeTransform subtracts high-precision camera position (Floating Origin)
// to yield local jitter-free float32 coordinates for GPU rendering.
func CameraRelativeTransform(worldPos Vec3d, cameraPos Vec3d) [3]float32 {
	rel := worldPos.Sub(cameraPos)
	return rel.ToFloat32()
}

// CubeFace identifies the 6 faces of the normalized cube-sphere.
type CubeFace int

const (
	FacePX CubeFace = iota // +X
	FaceNX                 // -X
	FacePY                 // +Y
	FaceNY                 // -Y
	FacePZ                 // +Z
	FaceNZ                 // -Z
)

// MapCubeToSphere maps 2D cube face coordinate (u, v in [-1, 1]) to normalized sphere S^2.
// Uses tangent-distortion correction / spherical normalization:
// x' = x * sqrt(1 - y^2/2 - z^2/2 + y^2*z^2/3)
func MapCubeToSphere(face CubeFace, u, v float64) Vec3d {
	var c Vec3d
	switch face {
	case FacePX:
		c = Vec3d{1.0, -v, -u}
	case FaceNX:
		c = Vec3d{-1.0, -v, u}
	case FacePY:
		c = Vec3d{u, 1.0, v}
	case FaceNY:
		c = Vec3d{u, -1.0, -v}
	case FacePZ:
		c = Vec3d{u, -v, 1.0}
	case FaceNZ:
		c = Vec3d{-u, -v, -1.0}
	}

	x2 := c.X * c.X
	y2 := c.Y * c.Y
	z2 := c.Z * c.Z

	// Spherified cube coordinate mapping
	sx := c.X * math.Sqrt(math.Max(0.0, 1.0-(y2/2.0)-(z2/2.0)+(y2*z2/3.0)))
	sy := c.Y * math.Sqrt(math.Max(0.0, 1.0-(z2/2.0)-(x2/2.0)+(z2*x2/3.0)))
	sz := c.Z * math.Sqrt(math.Max(0.0, 1.0-(x2/2.0)-(y2/2.0)+(x2*y2/3.0)))

	return Vec3d{sx, sy, sz}.Normalize()
}

// QuadNode represents a recursive quadtree tile for planetary LOD.
type QuadNode struct {
	Face     CubeFace
	U0, V0   float64 // Bottom-left coordinate in [-1, 1]
	Size     float64 // Node extent in UV space
	Level    int
	Center   Vec3d
	Radius   float64
	Children [4]*QuadNode
}

// NewQuadNode constructs a planetary tile node with computed geocentric center and bounding sphere.
func NewQuadNode(face CubeFace, u0, v0, size float64, level int, planetRadius float64) *QuadNode {
	midU := u0 + size*0.5
	midV := v0 + size*0.5
	centerDir := MapCubeToSphere(face, midU, midV)
	centerPos := centerDir.Scale(planetRadius)

	// Approximate tile bounding radius from corner distance
	cornerDir := MapCubeToSphere(face, u0, v0)
	cornerPos := cornerDir.Scale(planetRadius)
	boundRadius := centerPos.Sub(cornerPos).Length() * 1.15

	return &QuadNode{
		Face:     face,
		U0:       u0,
		V0:       v0,
		Size:     size,
		Level:    level,
		Center:   centerPos,
		Radius:   boundRadius,
	}
}

// Subdivide generates 4 child quadtree nodes.
func (n *QuadNode) Subdivide(planetRadius float64) {
	half := n.Size * 0.5
	n.Children[0] = NewQuadNode(n.Face, n.U0, n.V0, half, n.Level+1, planetRadius)
	n.Children[1] = NewQuadNode(n.Face, n.U0+half, n.V0, half, n.Level+1, planetRadius)
	n.Children[2] = NewQuadNode(n.Face, n.U0, n.V0+half, half, n.Level+1, planetRadius)
	n.Children[3] = NewQuadNode(n.Face, n.U0+half, n.V0+half, half, n.Level+1, planetRadius)
}

// IsHorizonCulled tests if the tile node lies beyond the planetary horizon relative to camera.
func (n *QuadNode) IsHorizonCulled(camPos Vec3d, planetRadius float64) bool {
	camDist := camPos.Length()
	if camDist <= planetRadius {
		return false
	}
	// Distance to geometric horizon: d_h = sqrt(camDist^2 - R^2)
	horizonDistSq := camDist*camDist - planetRadius*planetRadius
	nodeDistSq := n.Center.Sub(camPos).Dot(n.Center.Sub(camPos))

	// If node center is further than horizon distance + bound radius, check dot product
	camDir := camPos.Normalize()
	nodeDir := n.Center.Normalize()
	dot := camDir.Dot(nodeDir)

	// Minimum visible cos angle: cos(alpha) = R / camDist
	minCos := planetRadius / camDist
	return dot < (minCos - n.Radius/planetRadius) && nodeDistSq > horizonDistSq
}
