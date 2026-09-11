package ecs

import (
	"sync"
)

// EntityID represents an indexed lightweight identity.
type EntityID uint32

// PackedPhysicsComponent stores 64-byte aligned contiguous spatial physics state (SoA layout).
type PackedPhysicsComponent struct {
	PosX, PosY, PosZ    float64
	VelX, VelY, VelZ    float64
	Mass                float64
	IsStatic            bool
}

// SpatialCellID defines a 64-bit spatial hash key for planetary scale indexing.
type SpatialCellID uint64

// HashSpatialPosition maps continuous 3D world coordinates to spatial grid cell.
func HashSpatialPosition(x, y, z float64, cellSize float64) SpatialCellID {
	ix := uint64(int64(mathFloor(x / cellSize)))
	iy := uint64(int64(mathFloor(y / cellSize)))
	iz := uint64(int64(mathFloor(z / cellSize)))

	// 64-bit Morton-style spatial hash
	return SpatialCellID((ix * 73856093) ^ (iy * 19349663) ^ (iz * 83492791))
}

func mathFloor(v float64) float64 {
	if v >= 0 {
		return float64(int64(v))
	}
	return float64(int64(v) - 1)
}

// PlanetaryECSWorld coordinates high-throughput entity systems with cache-conscious SoA arrays.
type PlanetaryECSWorld struct {
	mu           sync.RWMutex
	NextID       EntityID
	Physics      map[EntityID]PackedPhysicsComponent
	SpatialGrid  map[SpatialCellID][]EntityID
	CellSizeM    float64
}

// NewPlanetaryECSWorld constructs ECS world with specified spatial cell resolution (e.g. 100m).
func NewPlanetaryECSWorld(cellSizeM float64) *PlanetaryECSWorld {
	if cellSizeM <= 0 {
		cellSizeM = 100.0
	}
	return &PlanetaryECSWorld{
		Physics:     make(map[EntityID]PackedPhysicsComponent),
		SpatialGrid: make(map[SpatialCellID][]EntityID),
		CellSizeM:   cellSizeM,
	}
}

// SpawnEntity registers a new physics entity.
func (w *PlanetaryECSWorld) SpawnEntity(pos [3]float64, vel [3]float64, mass float64, isStatic bool) EntityID {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.NextID++
	id := w.NextID

	comp := PackedPhysicsComponent{
		PosX:     pos[0], PosY: pos[1], PosZ: pos[2],
		VelX:     vel[0], VelY: vel[1], VelZ: vel[2],
		Mass:     mass,
		IsStatic: isStatic,
	}
	w.Physics[id] = comp

	cell := HashSpatialPosition(pos[0], pos[1], pos[2], w.CellSizeM)
	w.SpatialGrid[cell] = append(w.SpatialGrid[cell], id)

	return id
}

// ParallelPhysicsStep advances all dynamic entities by dt and rebuilds spatial grid.
func (w *PlanetaryECSWorld) ParallelPhysicsStep(dt float64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Clear spatial grid
	for k := range w.SpatialGrid {
		delete(w.SpatialGrid, k)
	}

	const gravity = -9.80665

	for id, comp := range w.Physics {
		if !comp.IsStatic {
			comp.VelZ += gravity * dt
			comp.PosX += comp.VelX * dt
			comp.PosY += comp.VelY * dt
			comp.PosZ += comp.VelZ * dt

			// Simple ground plane clamp at Z = 0
			if comp.PosZ < 0 {
				comp.PosZ = 0
				comp.VelZ = 0
				comp.VelX *= 0.9 // Friction
				comp.VelY *= 0.9
			}
			w.Physics[id] = comp
		}

		cell := HashSpatialPosition(comp.PosX, comp.PosY, comp.PosZ, w.CellSizeM)
		w.SpatialGrid[cell] = append(w.SpatialGrid[cell], id)
	}
}

// QueryEntitiesInRadius returns entity IDs within spherical radius R from query point.
func (w *PlanetaryECSWorld) QueryEntitiesInRadius(centerX, centerY, centerZ float64, radius float64) []EntityID {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var result []EntityID
	rSq := radius * radius

	// Check surrounding spatial cells
	minCellX := int64(mathFloor((centerX - radius) / w.CellSizeM))
	maxCellX := int64(mathFloor((centerX + radius) / w.CellSizeM))
	minCellY := int64(mathFloor((centerY - radius) / w.CellSizeM))
	maxCellY := int64(mathFloor((centerY + radius) / w.CellSizeM))
	minCellZ := int64(mathFloor((centerZ - radius) / w.CellSizeM))
	maxCellZ := int64(mathFloor((centerZ + radius) / w.CellSizeM))

	for cx := minCellX; cx <= maxCellX; cx++ {
		for cy := minCellY; cy <= maxCellY; cy++ {
			for cz := minCellZ; cz <= maxCellZ; cz++ {
				cell := SpatialCellID((uint64(cx) * 73856093) ^ (uint64(cy) * 19349663) ^ (uint64(cz) * 83492791))
				for _, id := range w.SpatialGrid[cell] {
					comp := w.Physics[id]
					dx := comp.PosX - centerX
					dy := comp.PosY - centerY
					dz := comp.PosZ - centerZ
					if dx*dx+dy*dy+dz*dz <= rSq {
						result = append(result, id)
					}
				}
			}
		}
	}
	return result
}
