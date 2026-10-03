package engine

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"integin/pkg/cad/dxf"
)

var ErrInvalidOBJ = errors.New("invalid OBJ geometry")

const (
	maxOBJVertices = 100000
	maxOBJFaces    = 200000

	// DefaultHookClearanceRadiusM defines the spatial clearance boundary around the crane hook.
	DefaultHookClearanceRadiusM = 2.0
)

type Vertex struct {
	X, Y, Z float64
}

type Face struct {
	V1, V2, V3 int // 0-based indices into Vertices
}

type OBJMesh struct {
	Vertices      []Vertex
	Faces         int
	FaceTriangles []Face
}

func parseVertexIndex(field string, totalVertices int) (int, error) {
	slashIdx := strings.IndexByte(field, '/')
	token := field
	if slashIdx != -1 {
		token = field[:slashIdx]
	}
	idx, err := strconv.Atoi(token)
	if err != nil {
		return 0, err
	}
	if idx > 0 {
		idx = idx - 1
	} else if idx < 0 {
		idx = totalVertices + idx
	} else {
		return 0, errors.New("zero index invalid in OBJ")
	}
	if idx < 0 || idx >= totalVertices {
		return 0, errors.New("vertex index out of range")
	}
	return idx, nil
}

func ParseOBJMesh(data string) (*OBJMesh, error) {
	if strings.TrimSpace(data) == "" {
		return nil, ErrInvalidOBJ
	}
	mesh := &OBJMesh{}
	faceCount := 0
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		switch fields[0] {
		case "v":
			if len(fields) < 4 {
				return nil, fmt.Errorf("%w: vertex requires three coordinates", ErrInvalidOBJ)
			}
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("%w: vertex x coordinate", ErrInvalidOBJ)
			}
			y, err := strconv.ParseFloat(fields[2], 64)
			if err != nil {
				return nil, fmt.Errorf("%w: vertex y coordinate", ErrInvalidOBJ)
			}
			z, err := strconv.ParseFloat(fields[3], 64)
			if err != nil {
				return nil, fmt.Errorf("%w: vertex z coordinate", ErrInvalidOBJ)
			}
			if !finiteOBJValue(x) || !finiteOBJValue(y) || !finiteOBJValue(z) {
				return nil, fmt.Errorf("%w: vertex coordinates must be finite", ErrInvalidOBJ)
			}
			mesh.Vertices = append(mesh.Vertices, Vertex{X: x, Y: y, Z: z})
			if len(mesh.Vertices) > maxOBJVertices {
				return nil, fmt.Errorf("%w: vertex limit exceeded", ErrInvalidOBJ)
			}
		case "f":
			if len(fields) < 4 {
				return nil, fmt.Errorf("%w: face requires at least three vertices", ErrInvalidOBJ)
			}
			faceCount++
			if faceCount > maxOBJFaces {
				return nil, fmt.Errorf("%w: face limit exceeded", ErrInvalidOBJ)
			}
			indices := make([]int, len(fields)-1)
			for i := 1; i < len(fields); i++ {
				idx, err := parseVertexIndex(fields[i], len(mesh.Vertices))
				if err != nil {
					return nil, fmt.Errorf("%w: face vertex index: %v", ErrInvalidOBJ, err)
				}
				indices[i-1] = idx
			}
			for i := 1; i < len(indices)-1; i++ {
				mesh.FaceTriangles = append(mesh.FaceTriangles, Face{
					V1: indices[0],
					V2: indices[i],
					V3: indices[i+1],
				})
			}
		}
	}
	if len(mesh.Vertices) == 0 || faceCount == 0 {
		return nil, fmt.Errorf("%w: geometry has no vertices or faces", ErrInvalidOBJ)
	}
	mesh.Faces = faceCount
	return mesh, nil
}

func pointCollides(v Vertex, hookTip dxf.Point3D, loadLengthM, loadRadiusSq, hookRadiusSq float64) bool {
	dx := v.X - hookTip.X
	dy := v.Y - hookTip.Y
	dz := v.Z - hookTip.Z
	if dz <= 0 && dz >= -loadLengthM {
		if dx*dx+dy*dy < loadRadiusSq {
			return true
		}
	} else if dx*dx+dy*dy+dz*dz < hookRadiusSq {
		return true
	}
	return false
}

func CheckClearance(mesh *OBJMesh, trajectory *Trajectory4DResult, stages []LiftStage, c1, c2 CraneKinematics, load MassSpec) bool {
	if mesh == nil || len(mesh.Vertices) == 0 || trajectory == nil || len(stages) == 0 {
		return false
	}
	if !finiteOBJValue(load.LengthM) || load.LengthM <= 0 || !finiteOBJValue(load.RadiusM) || load.RadiusM <= 0 {
		return false
	}
	loadRadiusSq := load.RadiusM * load.RadiusM
	hookRadiusSq := DefaultHookClearanceRadiusM * DefaultHookClearanceRadiusM

	for _, stage := range stages {
		c1Temp := c1
		c1Temp.BoomAngleDeg = stage.Crane1Angle
		c1Temp.SlewAngleDeg = stage.Crane1Slew
		h1 := c1Temp.ComputeHookPosition()

		c2Temp := c2
		c2Temp.BoomAngleDeg = stage.Crane2Angle
		c2Temp.SlewAngleDeg = stage.Crane2Slew
		h2 := c2Temp.ComputeHookPosition()

		// 1. Direct vertex check
		for _, v := range mesh.Vertices {
			if pointCollides(v, h1.HookTip, load.LengthM, loadRadiusSq, hookRadiusSq) ||
				pointCollides(v, h2.HookTip, load.LengthM, loadRadiusSq, hookRadiusSq) {
				trajectory.FailTimeS = stage.TimeS
				return false
			}
		}

		// 2. Face interior (centroid) & edge midpoint check (resolves OBS-1)
		for _, f := range mesh.FaceTriangles {
			if f.V1 >= len(mesh.Vertices) || f.V2 >= len(mesh.Vertices) || f.V3 >= len(mesh.Vertices) {
				continue
			}
			v1, v2, v3 := mesh.Vertices[f.V1], mesh.Vertices[f.V2], mesh.Vertices[f.V3]
			centroid := Vertex{
				X: (v1.X + v2.X + v3.X) / 3.0,
				Y: (v1.Y + v2.Y + v3.Y) / 3.0,
				Z: (v1.Z + v2.Z + v3.Z) / 3.0,
			}
			if pointCollides(centroid, h1.HookTip, load.LengthM, loadRadiusSq, hookRadiusSq) ||
				pointCollides(centroid, h2.HookTip, load.LengthM, loadRadiusSq, hookRadiusSq) {
				trajectory.FailTimeS = stage.TimeS
				return false
			}

			// Edge midpoints
			m1 := Vertex{X: (v1.X + v2.X) * 0.5, Y: (v1.Y + v2.Y) * 0.5, Z: (v1.Z + v2.Z) * 0.5}
			m2 := Vertex{X: (v2.X + v3.X) * 0.5, Y: (v2.Y + v3.Y) * 0.5, Z: (v2.Z + v3.Z) * 0.5}
			m3 := Vertex{X: (v3.X + v1.X) * 0.5, Y: (v3.Y + v1.Y) * 0.5, Z: (v3.Z + v1.Z) * 0.5}
			if pointCollides(m1, h1.HookTip, load.LengthM, loadRadiusSq, hookRadiusSq) ||
				pointCollides(m1, h2.HookTip, load.LengthM, loadRadiusSq, hookRadiusSq) ||
				pointCollides(m2, h1.HookTip, load.LengthM, loadRadiusSq, hookRadiusSq) ||
				pointCollides(m2, h2.HookTip, load.LengthM, loadRadiusSq, hookRadiusSq) ||
				pointCollides(m3, h1.HookTip, load.LengthM, loadRadiusSq, hookRadiusSq) ||
				pointCollides(m3, h2.HookTip, load.LengthM, loadRadiusSq, hookRadiusSq) {
				trajectory.FailTimeS = stage.TimeS
				return false
			}
		}
	}
	return true
}

func finiteOBJValue(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
