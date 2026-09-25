package engine

import (
	"fmt"
	"strings"
)

type Vertex struct {
	X, Y, Z float64
}

type OBJMesh struct {
	Vertices []Vertex
	Faces    int
}

func ParseOBJMesh(data string) (*OBJMesh, error) {
	lines := strings.Split(data, "\n")
	mesh := &OBJMesh{}
	fCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "v ") {
			var v Vertex
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				fmt.Sscanf(parts[1]+" "+parts[2]+" "+parts[3], "%f %f %f", &v.X, &v.Y, &v.Z)
				mesh.Vertices = append(mesh.Vertices, v)
			}
		} else if strings.HasPrefix(line, "f ") {
			fCount++
		}
	}
	mesh.Faces = fCount
	return mesh, nil
}

func CheckClearance(mesh *OBJMesh, trajectory *Trajectory4DResult, stages []LiftStage, c1, c2 CraneKinematics, load MassSpec) bool {
	if len(mesh.Vertices) == 0 {
		return true
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
