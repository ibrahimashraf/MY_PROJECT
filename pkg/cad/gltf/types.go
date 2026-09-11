package gltf

// GLTF 2.0 standard component types
const (
	ComponentTypeByte          = 5120
	ComponentTypeUnsignedByte  = 5121
	ComponentTypeShort         = 5122
	ComponentTypeUnsignedShort = 5123
	ComponentTypeUnsignedInt   = 5125
	ComponentTypeFloat         = 5126
)

// GLTF 2.0 buffer view targets
const (
	TargetArrayBuffer        = 34962
	TargetElementArrayBuffer = 34963
)

// GLTF 2.0 primitive modes
const (
	ModePoints        = 0
	ModeLines         = 1
	ModeLineLoop      = 2
	ModeLineStrip     = 3
	ModeTriangles     = 4
	ModeTriangleStrip = 5
	ModeTriangleFan   = 6
)

// Document is the root object for a glTF 2.0 asset.
type Document struct {
	Asset       Asset        `json:"asset"`
	Scene       *int         `json:"scene,omitempty"`
	Scenes      []Scene      `json:"scenes,omitempty"`
	Nodes       []Node       `json:"nodes,omitempty"`
	Meshes      []Mesh       `json:"meshes,omitempty"`
	Materials   []Material   `json:"materials,omitempty"`
	Accessors   []Accessor   `json:"accessors,omitempty"`
	BufferViews []BufferView `json:"bufferViews,omitempty"`
	Buffers     []Buffer     `json:"buffers,omitempty"`
}

// Asset contains metadata about the glTF asset.
type Asset struct {
	Version   string         `json:"version"`
	Generator string         `json:"generator,omitempty"`
	Extras    map[string]any `json:"extras,omitempty"`
}

// Scene contains indices of root nodes.
type Scene struct {
	Name  string `json:"name,omitempty"`
	Nodes []int  `json:"nodes"`
}

// Node represents an object in the scene hierarchy.
type Node struct {
	Name        string     `json:"name,omitempty"`
	Translation [3]float64 `json:"translation,omitempty"`
	Rotation    [4]float64 `json:"rotation,omitempty"` // Quaternion (x, y, z, w)
	Scale       [3]float64 `json:"scale,omitempty"`
	Mesh        *int       `json:"mesh,omitempty"`
	Children    []int      `json:"children,omitempty"`
}

// Mesh represents a set of geometric primitives to be rendered.
type Mesh struct {
	Name       string      `json:"name,omitempty"`
	Primitives []Primitive `json:"primitives"`
}

// Primitive defines geometry attributes, material, and draw mode.
type Primitive struct {
	Attributes map[string]int `json:"attributes"` // "POSITION": accessorIdx, "NORMAL": accessorIdx
	Indices    *int           `json:"indices,omitempty"`
	Material   *int           `json:"material,omitempty"`
	Mode       int            `json:"mode"`
}

// Material defines PBR shading attributes.
type Material struct {
	Name                 string               `json:"name,omitempty"`
	PbrMetallicRoughness PbrMetallicRoughness `json:"pbrMetallicRoughness"`
	DoubleSided          bool                 `json:"doubleSided,omitempty"`
}

// PbrMetallicRoughness defines physical properties for PBR lighting.
type PbrMetallicRoughness struct {
	BaseColorFactor [4]float64 `json:"baseColorFactor"` // RGBA [0.0 - 1.0]
	MetallicFactor  float64    `json:"metallicFactor"`  // 0.0 (dielectric) to 1.0 (pure metal)
	RoughnessFactor float64    `json:"roughnessFactor"` // 0.0 (mirror) to 1.0 (diffuse rough)
}

// Accessor indexes a typed view into a bufferView.
type Accessor struct {
	BufferView    int       `json:"bufferView"`
	ByteOffset    int       `json:"byteOffset"`
	ComponentType int       `json:"componentType"`
	Count         int       `json:"count"`
	Type          string    `json:"type"` // "SCALAR", "VEC2", "VEC3", "VEC4"
	Min           []float64 `json:"min,omitempty"`
	Max           []float64 `json:"max,omitempty"`
}

// BufferView represents a subset of a buffer.
type BufferView struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset"`
	ByteLength int `json:"byteLength"`
	Target     int `json:"target,omitempty"`
}

// Buffer points to binary payload.
type Buffer struct {
	ByteLength int `json:"byteLength"`
}
