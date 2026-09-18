package gltf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
)

const (
	glTFMagic     uint32 = 0x46546C67 // "glTF"
	glTFJSONChunk uint32 = 0x4E4F534A // "JSON"
	glTFBINChunk  uint32 = 0x004E4942 // "BIN\0"
)

// MeshData represents a raw 3D mesh ready for glTF packing.
type MeshData struct {
	Name      string
	Positions [][3]float32
	Normals   [][3]float32
	Indices   []uint32
	Material  Material
}

// PackGLB builds a complete, self-contained binary glTF (GLB) file from meshes.
func PackGLB(meshes []MeshData) ([]byte, error) {
	if len(meshes) == 0 {
		return nil, errors.New("at least one mesh is required to pack GLB")
	}

	doc := Document{
		Asset: Asset{
			Version:   "2.0",
			Generator: "INTEGIN Sovereign CAD Engine",
			Extras: map[string]any{
				"integin_audit": map[string]string{
					"status":     "NON_CERTIFIED_SIMULATION",
					"disclaimer": "ENGINEERING STUDY ONLY - NOT VALID FOR FABRICATION OR CRITICAL LIFT WITHOUT LICENSED PE STAMP",
					"standard":   "ASME B30.5 / OSHA 1926",
				},
			},
		},
		Scene:       ptrInt(0),
		Scenes:      []Scene{{Name: "Scene", Nodes: make([]int, 0, len(meshes))}},
		Nodes:       make([]Node, 0, len(meshes)),
		Meshes:      make([]Mesh, 0, len(meshes)),
		Materials:   make([]Material, 0, len(meshes)),
		Accessors:   make([]Accessor, 0, len(meshes)*3),
		BufferViews: make([]BufferView, 0, len(meshes)*3),
		Buffers:     []Buffer{{ByteLength: 0}},
	}

	var binBuffer bytes.Buffer

	for meshIdx, m := range meshes {
		if len(m.Positions) == 0 {
			continue
		}

		// 1. Add Material
		matIdx := len(doc.Materials)
		doc.Materials = append(doc.Materials, m.Material)

		// 2. Pack Positions (Vec3 Float32)
		posOffset := binBuffer.Len()
		minPos := [3]float64{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}
		maxPos := [3]float64{-math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64}

		for _, p := range m.Positions {
			for c := 0; c < 3; c++ {
				val := float64(p[c])
				if val < minPos[c] {
					minPos[c] = val
				}
				if val > maxPos[c] {
					maxPos[c] = val
				}
			}
			if err := binary.Write(&binBuffer, binary.LittleEndian, p); err != nil {
				return nil, err
			}
		}
		posLength := binBuffer.Len() - posOffset
		pad4(&binBuffer, 0x00)

		posViewIdx := len(doc.BufferViews)
		doc.BufferViews = append(doc.BufferViews, BufferView{
			Buffer:     0,
			ByteOffset: posOffset,
			ByteLength: posLength,
			Target:     TargetArrayBuffer,
		})

		posAccessorIdx := len(doc.Accessors)
		doc.Accessors = append(doc.Accessors, Accessor{
			BufferView:    posViewIdx,
			ByteOffset:    0,
			ComponentType: ComponentTypeFloat,
			Count:         len(m.Positions),
			Type:          "VEC3",
			Min:           minPos[:],
			Max:           maxPos[:],
		})

		attribs := map[string]int{
			"POSITION": posAccessorIdx,
		}

		// 3. Pack Normals if present
		if len(m.Normals) == len(m.Positions) {
			normOffset := binBuffer.Len()
			for _, n := range m.Normals {
				if err := binary.Write(&binBuffer, binary.LittleEndian, n); err != nil {
					return nil, err
				}
			}
			normLength := binBuffer.Len() - normOffset
			pad4(&binBuffer, 0x00)

			normViewIdx := len(doc.BufferViews)
			doc.BufferViews = append(doc.BufferViews, BufferView{
				Buffer:     0,
				ByteOffset: normOffset,
				ByteLength: normLength,
				Target:     TargetArrayBuffer,
			})

			normAccessorIdx := len(doc.Accessors)
			doc.Accessors = append(doc.Accessors, Accessor{
				BufferView:    normViewIdx,
				ByteOffset:    0,
				ComponentType: ComponentTypeFloat,
				Count:         len(m.Normals),
				Type:          "VEC3",
			})
			attribs["NORMAL"] = normAccessorIdx
		}

		// 4. Pack Indices if present
		var indicesAccessorIdx *int
		if len(m.Indices) > 0 {
			idxOffset := binBuffer.Len()
			useUint32 := len(m.Positions) >= 65535
			for _, idx := range m.Indices {
				if useUint32 {
					if err := binary.Write(&binBuffer, binary.LittleEndian, idx); err != nil {
						return nil, err
					}
				} else {
					if err := binary.Write(&binBuffer, binary.LittleEndian, uint16(idx)); err != nil {
						return nil, err
					}
				}
			}
			idxLength := binBuffer.Len() - idxOffset
			pad4(&binBuffer, 0x00)

			idxViewIdx := len(doc.BufferViews)
			doc.BufferViews = append(doc.BufferViews, BufferView{
				Buffer:     0,
				ByteOffset: idxOffset,
				ByteLength: idxLength,
				Target:     TargetElementArrayBuffer,
			})

			compType := ComponentTypeUnsignedShort
			if useUint32 {
				compType = ComponentTypeUnsignedInt
			}

			accIdx := len(doc.Accessors)
			doc.Accessors = append(doc.Accessors, Accessor{
				BufferView:    idxViewIdx,
				ByteOffset:    0,
				ComponentType: compType,
				Count:         len(m.Indices),
				Type:          "SCALAR",
			})
			indicesAccessorIdx = &accIdx
		}

		// 5. Create Mesh Primitive
		meshObj := Mesh{
			Name: m.Name,
			Primitives: []Primitive{
				{
					Attributes: attribs,
					Indices:    indicesAccessorIdx,
					Material:   &matIdx,
					Mode:       ModeTriangles,
				},
			},
		}
		doc.Meshes = append(doc.Meshes, meshObj)

		// 6. Create Node
		nodeIdx := len(doc.Nodes)
		doc.Nodes = append(doc.Nodes, Node{
			Name:        m.Name,
			Translation: [3]float64{0, 0, 0},
			Scale:       [3]float64{1, 1, 1},
			Mesh:        ptrInt(meshIdx),
		})
		doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, nodeIdx)
	}

	doc.Buffers[0].ByteLength = binBuffer.Len()

	// 7. Marshal JSON Chunk
	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	jsonPad := (4 - (len(jsonBytes) % 4)) % 4
	jsonChunkLength := uint32(len(jsonBytes) + jsonPad)

	binData := binBuffer.Bytes()
	binPad := (4 - (len(binData) % 4)) % 4
	binChunkLength := uint32(len(binData) + binPad)

	// Total length: 12 header + (8 + jsonChunkLength) + (8 + binChunkLength)
	totalLength := uint32(12 + 8 + jsonChunkLength + 8 + binChunkLength)

	var glb bytes.Buffer

	// 12-byte GLB Header
	binary.Write(&glb, binary.LittleEndian, glTFMagic)
	binary.Write(&glb, binary.LittleEndian, uint32(2))
	binary.Write(&glb, binary.LittleEndian, totalLength)

	// JSON Chunk Header
	binary.Write(&glb, binary.LittleEndian, jsonChunkLength)
	binary.Write(&glb, binary.LittleEndian, glTFJSONChunk)
	glb.Write(jsonBytes)
	for i := 0; i < jsonPad; i++ {
		glb.WriteByte(0x20) // Space padding for JSON
	}

	// BIN Chunk Header
	binary.Write(&glb, binary.LittleEndian, binChunkLength)
	binary.Write(&glb, binary.LittleEndian, glTFBINChunk)
	glb.Write(binData)
	for i := 0; i < binPad; i++ {
		glb.WriteByte(0x00) // Zero padding for BIN
	}

	return glb.Bytes(), nil
}

func pad4(buf *bytes.Buffer, padByte byte) {
	rem := buf.Len() % 4
	if rem != 0 {
		for i := 0; i < 4-rem; i++ {
			buf.WriteByte(padByte)
		}
	}
}

func ptrInt(i int) *int {
	return &i
}
