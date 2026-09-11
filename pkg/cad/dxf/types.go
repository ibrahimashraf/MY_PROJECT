package dxf

import "fmt"

// Point3D represents a 3D coordinate point in CAD space.
type Point3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (p Point3D) String() string {
	return fmt.Sprintf("(%.4f, %.4f, %.4f)", p.X, p.Y, p.Z)
}

// Layer defines a CAD organization layer with color and line properties.
type Layer struct {
	Name      string `json:"name"`
	Color     int16  `json:"color"`     // AutoCAD Color Index (ACI 1-255)
	LineType  string `json:"line_type"` // e.g. CONTINUOUS, DASHED, HIDDEN
	IsVisible bool   `json:"is_visible"`
	IsLocked  bool   `json:"is_locked"`
}

// EntityType identifies the type of CAD entity.
type EntityType string

const (
	EntityLine     EntityType = "LINE"
	EntityPolyline EntityType = "LWPOLYLINE"
	EntityCircle   EntityType = "CIRCLE"
	EntityArc      EntityType = "ARC"
	EntityText     EntityType = "TEXT"
	EntityInsert   EntityType = "INSERT"
)

// Entity is the interface implemented by all geometric CAD elements.
type Entity interface {
	Type() EntityType
	Layer() string
	BoundingBox() (min Point3D, max Point3D)
}

// BaseEntity holds common attributes for all CAD entities.
type BaseEntity struct {
	LayerName string `json:"layer"`
	Color     int16  `json:"color,omitempty"`
}

func (b BaseEntity) Layer() string {
	if b.LayerName == "" {
		return "0"
	}
	return b.LayerName
}

// Line represents a 2D/3D line segment.
type Line struct {
	BaseEntity
	Start Point3D `json:"start"`
	End   Point3D `json:"end"`
}

func (l *Line) Type() EntityType { return EntityLine }
func (l *Line) BoundingBox() (min Point3D, max Point3D) {
	min = Point3D{
		X: minF(l.Start.X, l.End.X),
		Y: minF(l.Start.Y, l.End.Y),
		Z: minF(l.Start.Z, l.End.Z),
	}
	max = Point3D{
		X: maxF(l.Start.X, l.End.X),
		Y: maxF(l.Start.Y, l.End.Y),
		Z: maxF(l.Start.Z, l.End.Z),
	}
	return min, max
}

// Vertex represents a 2D vertex in a lightweight polyline.
type Vertex struct {
	Point Point3D `json:"point"`
	Bulge float64 `json:"bulge,omitempty"` // Arc curvature tangent of 1/4 included angle
}

// Polyline represents a lightweight 2D polyline (LWPOLYLINE).
type Polyline struct {
	BaseEntity
	Vertices []Vertex `json:"vertices"`
	IsClosed bool     `json:"is_closed"`
}

func (p *Polyline) Type() EntityType { return EntityPolyline }
func (p *Polyline) BoundingBox() (min Point3D, max Point3D) {
	if len(p.Vertices) == 0 {
		return Point3D{}, Point3D{}
	}
	min = p.Vertices[0].Point
	max = p.Vertices[0].Point
	for _, v := range p.Vertices[1:] {
		min.X = minF(min.X, v.Point.X)
		min.Y = minF(min.Y, v.Point.Y)
		min.Z = minF(min.Z, v.Point.Z)
		max.X = maxF(max.X, v.Point.X)
		max.Y = maxF(max.Y, v.Point.Y)
		max.Z = maxF(max.Z, v.Point.Z)
	}
	return min, max
}

// Circle represents a 2D/3D circular entity.
type Circle struct {
	BaseEntity
	Center Point3D `json:"center"`
	Radius float64 `json:"radius"`
}

func (c *Circle) Type() EntityType { return EntityCircle }
func (c *Circle) BoundingBox() (min Point3D, max Point3D) {
	return Point3D{X: c.Center.X - c.Radius, Y: c.Center.Y - c.Radius, Z: c.Center.Z},
		Point3D{X: c.Center.X + c.Radius, Y: c.Center.Y + c.Radius, Z: c.Center.Z}
}

// Arc represents a circular arc.
type Arc struct {
	BaseEntity
	Center     Point3D `json:"center"`
	Radius     float64 `json:"radius"`
	StartAngle float64 `json:"start_angle"` // in degrees
	EndAngle   float64 `json:"end_angle"`   // in degrees
}

func (a *Arc) Type() EntityType { return EntityArc }
func (a *Arc) BoundingBox() (min Point3D, max Point3D) {
	// Conservative bounding box using radius envelope
	return Point3D{X: a.Center.X - a.Radius, Y: a.Center.Y - a.Radius, Z: a.Center.Z},
		Point3D{X: a.Center.X + a.Radius, Y: a.Center.Y + a.Radius, Z: a.Center.Z}
}

// Text represents single-line text in a drawing.
type Text struct {
	BaseEntity
	Insertion Point3D `json:"insertion"`
	Height    float64 `json:"height"`
	Value     string  `json:"value"`
	Rotation  float64 `json:"rotation,omitempty"`
}

func (t *Text) Type() EntityType { return EntityText }
func (t *Text) BoundingBox() (min Point3D, max Point3D) {
	return t.Insertion, Point3D{X: t.Insertion.X + float64(len(t.Value))*t.Height*0.6, Y: t.Insertion.Y + t.Height, Z: t.Insertion.Z}
}

// Block represents a reusable definition of grouped entities.
type Block struct {
	Name      string   `json:"name"`
	BasePoint Point3D  `json:"base_point"`
	Entities  []Entity `json:"entities"`
	LayerName string   `json:"layer"`
}

// Insert represents an instance insertion of a Block definition.
type Insert struct {
	BaseEntity
	BlockName   string  `json:"block_name"`
	Insertion   Point3D `json:"insertion"`
	ScaleX      float64 `json:"scale_x"`
	ScaleY      float64 `json:"scale_y"`
	ScaleZ      float64 `json:"scale_z"`
	RotationDeg float64 `json:"rotation_deg"`
}

func (i *Insert) Type() EntityType { return EntityInsert }
func (i *Insert) BoundingBox() (min Point3D, max Point3D) {
	return i.Insertion, i.Insertion
}

// Drawing represents a complete CAD drawing containing layers, blocks, and entities.
type Drawing struct {
	Layers   map[string]*Layer `json:"layers"`
	Blocks   map[string]*Block `json:"blocks"`
	Entities []Entity          `json:"entities"`
}

// NewDrawing initializes an empty CAD drawing with default layer "0".
func NewDrawing() *Drawing {
	d := &Drawing{
		Layers:   make(map[string]*Layer),
		Blocks:   make(map[string]*Block),
		Entities: make([]Entity, 0),
	}
	d.Layers["0"] = &Layer{
		Name:      "0",
		Color:     7, // White/Black standard
		LineType:  "CONTINUOUS",
		IsVisible: true,
	}
	return d
}

// AddEntity adds a geometric entity to the drawing.
func (d *Drawing) AddEntity(e Entity) {
	d.Entities = append(d.Entities, e)
	if _, exists := d.Layers[e.Layer()]; !exists {
		d.Layers[e.Layer()] = &Layer{
			Name:      e.Layer(),
			Color:     7,
			LineType:  "CONTINUOUS",
			IsVisible: true,
		}
	}
}

// AddBlock registers a reusable block definition.
func (d *Drawing) AddBlock(b *Block) {
	d.Blocks[b.Name] = b
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
