package engine

import (
	"fmt"
	"math"

	"integin/pkg/cad/dxf"
)

// ParameterType defines the kind of dynamic parameter on a block.
type ParameterType string

const (
	ParamLinear   ParameterType = "LINEAR"
	ParamRotation ParameterType = "ROTATION"
	ParamPoint    ParameterType = "POINT"
)

// Parameter represents a dynamic grip or constraint on a CAD block.
type Parameter struct {
	Name         string        `json:"name"`
	Type         ParameterType `json:"type"`
	DefaultValue float64       `json:"default_value"`
	MinValue     float64       `json:"min_value,omitempty"`
	MaxValue     float64       `json:"max_value,omitempty"`
}

// ActionType defines transformations driven by parameters.
type ActionType string

const (
	ActionStretch ActionType = "STRETCH"
	ActionRotate  ActionType = "ROTATE"
	ActionScale   ActionType = "SCALE"
)

// Action defines how entities transform when a parameter changes.
type Action struct {
	Name          string     `json:"name"`
	Type          ActionType `json:"type"`
	ParameterName string     `json:"parameter_name"`
	TargetIndices []int      `json:"target_indices"` // Indices of affected base entities
}

// DynamicBlockDef defines a reusable parametric dynamic block template.
type DynamicBlockDef struct {
	Name         string        `json:"name"`
	BaseEntities []dxf.Entity  `json:"base_entities"`
	Parameters   []Parameter   `json:"parameters"`
	Actions      []Action      `json:"actions"`
}

// DynamicBlockInstance is an instantiated block with evaluated parameter values.
type DynamicBlockInstance struct {
	DefName       string             `json:"def_name"`
	ParamValues   map[string]float64 `json:"param_values"`
	Insertion     dxf.Point3D        `json:"insertion"`
	Def           *DynamicBlockDef   `json:"-"`
}

// NewDynamicBlockInstance creates an instance with default parameters.
func NewDynamicBlockInstance(def *DynamicBlockDef, insertion dxf.Point3D) *DynamicBlockInstance {
	values := make(map[string]float64)
	for _, p := range def.Parameters {
		values[p.Name] = p.DefaultValue
	}
	return &DynamicBlockInstance{
		DefName:     def.Name,
		ParamValues: values,
		Insertion:   insertion,
		Def:         def,
	}
}

// SetParam updates a parameter value within its min/max limits.
func (dbi *DynamicBlockInstance) SetParam(name string, val float64) error {
	var targetParam *Parameter
	for i := range dbi.Def.Parameters {
		if dbi.Def.Parameters[i].Name == name {
			targetParam = &dbi.Def.Parameters[i]
			break
		}
	}
	if targetParam == nil {
		return fmt.Errorf("parameter %q not found on block %q", name, dbi.DefName)
	}

	if targetParam.MinValue != 0 && val < targetParam.MinValue {
		val = targetParam.MinValue
	}
	if targetParam.MaxValue != 0 && val > targetParam.MaxValue {
		val = targetParam.MaxValue
	}

	dbi.ParamValues[name] = val
	return nil
}

// EvaluateGeometry computes the final transformed geometric entities.
func (dbi *DynamicBlockInstance) EvaluateGeometry() []dxf.Entity {
	evaluated := make([]dxf.Entity, 0, len(dbi.Def.BaseEntities))

	for i, e := range dbi.Def.BaseEntities {
		transformed := cloneEntity(e)

		// Apply associated actions
		for _, action := range dbi.Def.Actions {
			if !containsInt(action.TargetIndices, i) {
				continue
			}

			paramVal := dbi.ParamValues[action.ParameterName]

			switch action.Type {
			case ActionStretch:
				transformed = applyStretch(transformed, paramVal)
			case ActionRotate:
				transformed = applyRotate(transformed, paramVal)
			case ActionScale:
				transformed = applyScale(transformed, paramVal)
			}
		}

		// Apply final insertion translation
		transformed = applyTranslate(transformed, dbi.Insertion)
		evaluated = append(evaluated, transformed)
	}

	return evaluated
}

func cloneEntity(e dxf.Entity) dxf.Entity {
	switch v := e.(type) {
	case *dxf.Line:
		return &dxf.Line{BaseEntity: v.BaseEntity, Start: v.Start, End: v.End}
	case *dxf.Circle:
		return &dxf.Circle{BaseEntity: v.BaseEntity, Center: v.Center, Radius: v.Radius}
	case *dxf.Polyline:
		verts := make([]dxf.Vertex, len(v.Vertices))
		copy(verts, v.Vertices)
		return &dxf.Polyline{BaseEntity: v.BaseEntity, IsClosed: v.IsClosed, Vertices: verts}
	default:
		return e
	}
}

func applyStretch(e dxf.Entity, deltaX float64) dxf.Entity {
	if l, ok := e.(*dxf.Line); ok {
		l.End.X += deltaX
		return l
	}
	return e
}

func applyRotate(e dxf.Entity, angleDeg float64) dxf.Entity {
	rad := angleDeg * math.Pi / 180.0
	cosA, sinA := math.Cos(rad), math.Sin(rad)

	if l, ok := e.(*dxf.Line); ok {
		l.End = rotatePoint(l.End, l.Start, cosA, sinA)
		return l
	}
	return e
}

func applyScale(e dxf.Entity, scaleFactor float64) dxf.Entity {
	if scaleFactor <= 0 {
		scaleFactor = 1.0
	}
	if c, ok := e.(*dxf.Circle); ok {
		c.Radius *= scaleFactor
		return c
	}
	return e
}

func applyTranslate(e dxf.Entity, offset dxf.Point3D) dxf.Entity {
	switch v := e.(type) {
	case *dxf.Line:
		v.Start.X += offset.X
		v.Start.Y += offset.Y
		v.Start.Z += offset.Z
		v.End.X += offset.X
		v.End.Y += offset.Y
		v.End.Z += offset.Z
	case *dxf.Circle:
		v.Center.X += offset.X
		v.Center.Y += offset.Y
		v.Center.Z += offset.Z
	case *dxf.Polyline:
		for i := range v.Vertices {
			v.Vertices[i].Point.X += offset.X
			v.Vertices[i].Point.Y += offset.Y
			v.Vertices[i].Point.Z += offset.Z
		}
	}
	return e
}

func rotatePoint(p, pivot dxf.Point3D, cosA, sinA float64) dxf.Point3D {
	dx := p.X - pivot.X
	dy := p.Y - pivot.Y
	return dxf.Point3D{
		X: pivot.X + (dx*cosA - dy*sinA),
		Y: pivot.Y + (dx*sinA + dy*cosA),
		Z: p.Z,
	}
}

func containsInt(slice []int, val int) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
