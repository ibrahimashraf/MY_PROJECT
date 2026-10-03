package engine

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var ErrInvalidOBJ = errors.New("invalid OBJ geometry")

const (
	maxOBJVertices = 100000
	maxOBJFaces    = 200000
)

type Vertex struct {
	X, Y, Z float64
}

type OBJMesh struct {
	Vertices []Vertex
	Faces    int
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
		}
	}
	if len(mesh.Vertices) == 0 || faceCount == 0 {
		return nil, fmt.Errorf("%w: geometry has no vertices or faces", ErrInvalidOBJ)
	}
	mesh.Faces = faceCount
	return mesh, nil
}

func CheckClearance(mesh *OBJMesh, trajectory *Trajectory4DResult, stages []LiftStage, c1, c2 CraneKinematics, load MassSpec) bool {
	if mesh == nil || len(mesh.Vertices) == 0 || trajectory == nil || len(stages) == 0 {
		return false
	}
	if !finiteOBJValue(load.LengthM) || load.LengthM <= 0 || !finiteOBJValue(load.RadiusM) || load.RadiusM <= 0 {
		return false
	}
	for _, stage := range stages {
		c1Temp := c1
		c1Temp.BoomAngleDeg = stage.Crane1Angle
		c1Temp.SlewAngleDeg = stage.Crane1Slew
		h1 := c1Temp.ComputeHookPosition()

		c2Temp := c2
		c2Temp.BoomAngleDeg = stage.Crane2Angle
		c2Temp.SlewAngleDeg = stage.Crane2Slew
		h2 := c2Temp.ComputeHookPosition()

		for _, v := range mesh.Vertices {
			dx1 := v.X - h1.HookTip.X
			dy1 := v.Y - h1.HookTip.Y
			dz1 := v.Z - h1.HookTip.Z
			if dz1 <= 0 && dz1 >= -load.LengthM {
				if dx1*dx1+dy1*dy1 < load.RadiusM*load.RadiusM {
					trajectory.FailTimeS = stage.TimeS
					return false
				}
			} else if dx1*dx1+dy1*dy1+dz1*dz1 < 4.0 {
				trajectory.FailTimeS = stage.TimeS
				return false
			}

			dx2 := v.X - h2.HookTip.X
			dy2 := v.Y - h2.HookTip.Y
			dz2 := v.Z - h2.HookTip.Z
			if dz2 <= 0 && dz2 >= -load.LengthM {
				if dx2*dx2+dy2*dy2 < load.RadiusM*load.RadiusM {
					trajectory.FailTimeS = stage.TimeS
					return false
				}
			} else if dx2*dx2+dy2*dy2+dz2*dz2 < 4.0 {
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
