package engine

import "math"

// RigidBodyNode represents a 64-byte CPU cache-line aligned physical node.
// Layout: [32]byte tag + 4*float32 coords (16 bytes) + float32 radius (4) + float32 mass (4) + uint64 id (8) = 64 bytes
type RigidBodyNode struct {
	Tag      [32]byte `json:"tag"`       // Fixed 32-byte identifier
	X        float32  `json:"x"`         // 4 bytes
	Y        float32  `json:"y"`         // 4 bytes
	Z        float32  `json:"z"`         // 4 bytes
	Radius   float32  `json:"radius"`    // 4 bytes
	Mass     float32  `json:"mass"`      // 4 bytes
	Flags    uint32   `json:"flags"`     // 4 bytes
	EntityID uint64   `json:"entity_id"` // 8 bytes
}

// SoAPhysicsBuffer stores flat contiguous arrays for 60Hz vector clearance and collision sweeps.
// Eliminates interface pointer indirection and guarantees sequential L1 cache line prefetching.
type SoAPhysicsBuffer struct {
	PosX   []float32
	PosY   []float32
	PosZ   []float32
	Radius []float32
	Count  int
}

// NewSoAPhysicsBuffer pre-allocates contiguous memory for N geometric nodes.
func NewSoAPhysicsBuffer(capacity int) *SoAPhysicsBuffer {
	if capacity <= 0 {
		capacity = 64
	}
	return &SoAPhysicsBuffer{
		PosX:   make([]float32, 0, capacity),
		PosY:   make([]float32, 0, capacity),
		PosZ:   make([]float32, 0, capacity),
		Radius: make([]float32, 0, capacity),
		Count:  0,
	}
}

// Reset clears the buffer without releasing underlying memory allocations.
func (b *SoAPhysicsBuffer) Reset() {
	b.PosX = b.PosX[:0]
	b.PosY = b.PosY[:0]
	b.PosZ = b.PosZ[:0]
	b.Radius = b.Radius[:0]
	b.Count = 0
}

// AddNode appends a 3D bounding sphere into the contiguous arrays.
func (b *SoAPhysicsBuffer) AddNode(x, y, z, radius float32) {
	b.PosX = append(b.PosX, x)
	b.PosY = append(b.PosY, y)
	b.PosZ = append(b.PosZ, z)
	b.Radius = append(b.Radius, radius)
	b.Count++
}

// CollisionPair contains the indices of two colliding spheres and their penetration depth.
type CollisionPair struct {
	IndexA      int     `json:"index_a"`
	IndexB      int     `json:"index_b"`
	Penetration float32 `json:"penetration"`
}

// SweepClearance checks all pairs for spatial clearance against minMargin.
// Operates on contiguous memory with 0 heap allocations on the hot path.
func (b *SoAPhysicsBuffer) SweepClearance(minMargin float32, out *[]CollisionPair) bool {
	hasCollision := false
	n := b.Count

	for i := 0; i < n; i++ {
		xi, yi, zi := b.PosX[i], b.PosY[i], b.PosZ[i]
		ri := b.Radius[i]

		for j := i + 1; j < n; j++ {
			dx := b.PosX[j] - xi
			dy := b.PosY[j] - yi
			dz := b.PosZ[j] - zi

			distSq := dx*dx + dy*dy + dz*dz
			threshold := ri + b.Radius[j] + minMargin
			threshSq := threshold * threshold

			if distSq < threshSq {
				dist := float32(math.Sqrt(float64(distSq)))
				penetration := threshold - dist
				hasCollision = true
				if out != nil {
					*out = append(*out, CollisionPair{
						IndexA:      i,
						IndexB:      j,
						Penetration: penetration,
					})
				}
			}
		}
	}

	return hasCollision
}
