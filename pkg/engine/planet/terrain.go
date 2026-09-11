package planet

import (
	"math"
	"sync"
)

// TerrainVertex represents a packed 32-byte GPU-ready vertex buffer element.
// 12 bytes Pos, 12 bytes Normal, 8 bytes UV.
type TerrainVertex struct {
	PosX, PosY, PosZ       float32
	NormX, NormY, NormZ    float32
	U, V                   float32
}

// TerrainChunk represents an evaluated mesh patch ready for WebGPU rendering.
type TerrainChunk struct {
	Face      CubeFace
	Level     int
	Indices   []uint32
	Vertices  []TerrainVertex
	CenterECEF Vec3d
	MinBounds [3]float32
	MaxBounds [3]float32
}

// SimplexNoise2D generates deterministic procedural coherent noise for heightmaps.
func SimplexNoise2D(x, y float64) float64 {
	// Periodic sinusoidal approximation for deterministic pure Go noise without Cgo
	return math.Sin(x*1.5)*math.Cos(y*1.5)*0.5 +
		math.Sin(x*3.7+y*2.1)*0.25 +
		math.Cos(x*7.3-y*5.2)*0.125
}

// GenerateChunkMesh builds an N x N resolution terrain chunk with procedural elevation and normals.
func GenerateChunkMesh(node *QuadNode, resolution int, planetRadius float64, heightScale float64, camPos Vec3d) *TerrainChunk {
	if resolution < 2 {
		resolution = 2
	}

	numVerts := resolution * resolution
	vertices := make([]TerrainVertex, numVerts)
	step := node.Size / float64(resolution-1)

	// Step 1: Compute positions & procedural elevations
	for j := 0; j < resolution; j++ {
		vCoord := node.V0 + float64(j)*step
		for i := 0; i < resolution; i++ {
			uCoord := node.U0 + float64(i)*step
			idx := j*resolution + i

			// Unit sphere normal
			unitDir := MapCubeToSphere(node.Face, uCoord, vCoord)

			// Multi-octave procedural fractal noise
			elevation := SimplexNoise2D(unitDir.X*10.0, unitDir.Y*10.0+unitDir.Z*10.0) * heightScale
			totalR := planetRadius + elevation

			worldPos := unitDir.Scale(totalR)
			// Floating origin camera-relative offset
			relPos := CameraRelativeTransform(worldPos, camPos)

			vertices[idx] = TerrainVertex{
				PosX:  relPos[0],
				PosY:  relPos[1],
				PosZ:  relPos[2],
				NormX: float32(unitDir.X),
				NormY: float32(unitDir.Y),
				NormZ: float32(unitDir.Z),
				U:     float32(float64(i) / float64(resolution-1)),
				V:     float32(float64(j) / float64(resolution-1)),
			}
		}
	}

	// Step 2: Generate triangle indices
	numQuads := (resolution - 1) * (resolution - 1)
	indices := make([]uint32, numQuads*6)
	tIdx := 0
	for j := 0; j < resolution-1; j++ {
		for i := 0; i < resolution-1; i++ {
			row1 := uint32(j * resolution)
			row2 := uint32((j + 1) * resolution)

			v0 := row1 + uint32(i)
			v1 := row1 + uint32(i+1)
			v2 := row2 + uint32(i)
			v3 := row2 + uint32(i+1)

			// Triangle 1
			indices[tIdx] = v0
			indices[tIdx+1] = v2
			indices[tIdx+2] = v1

			// Triangle 2
			indices[tIdx+3] = v1
			indices[tIdx+4] = v2
			indices[tIdx+5] = v3
			tIdx += 6
		}
	}

	return &TerrainChunk{
		Face:       node.Face,
		Level:      node.Level,
		Indices:    indices,
		Vertices:   vertices,
		CenterECEF: node.Center,
	}
}

// MultiThreadedTerrainScheduler coordinates parallel chunk generation across worker goroutines.
type MultiThreadedTerrainScheduler struct {
	Workers int
}

// GeneratePlanetTiles parallel-evaluates visible leaf nodes.
func (s MultiThreadedTerrainScheduler) GeneratePlanetTiles(nodes []*QuadNode, res int, planetRadius, heightScale float64, camPos Vec3d) []*TerrainChunk {
	if s.Workers <= 0 {
		s.Workers = 4
	}

	chunks := make([]*TerrainChunk, len(nodes))
	var wg sync.WaitGroup
	ch := make(chan int, len(nodes))

	for i := 0; i < len(nodes); i++ {
		ch <- i
	}
	close(ch)

	for w := 0; w < s.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range ch {
				chunks[idx] = GenerateChunkMesh(nodes[idx], res, planetRadius, heightScale, camPos)
			}
		}()
	}
	wg.Wait()
	return chunks
}
