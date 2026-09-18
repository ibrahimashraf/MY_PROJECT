package gltf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestPackGLB(t *testing.T) {
	// Create a sample 3D box mesh with PBR steel material
	cube := MeshData{
		Name: "Industrial_Outrigger_Pad",
		Positions: [][3]float32{
			{-1, 0, -1},
			{1, 0, -1},
			{1, 0, 1},
			{-1, 0, 1},
			{-1, 0.5, -1},
			{1, 0.5, -1},
			{1, 0.5, 1},
			{-1, 0.5, 1},
		},
		Normals: [][3]float32{
			{0, -1, 0}, {0, -1, 0}, {0, -1, 0}, {0, -1, 0},
			{0, 1, 0}, {0, 1, 0}, {0, 1, 0}, {0, 1, 0},
		},
		Indices: []uint32{
			0, 1, 2, 0, 2, 3, // Bottom
			4, 6, 5, 4, 7, 6, // Top
		},
		Material: Material{
			Name: "PBR_Structural_Steel",
			PbrMetallicRoughness: PbrMetallicRoughness{
				BaseColorFactor: [4]float64{0.8, 0.8, 0.85, 1.0},
				MetallicFactor:  0.9,
				RoughnessFactor: 0.25,
			},
			DoubleSided: true,
		},
	}

	glbBytes, err := PackGLB([]MeshData{cube})
	if err != nil {
		t.Fatalf("PackGLB failed: %v", err)
	}

	if len(glbBytes) < 20 {
		t.Fatalf("GLB payload too short: %d bytes", len(glbBytes))
	}

	// 1. Verify 12-byte header
	magic := binary.LittleEndian.Uint32(glbBytes[0:4])
	version := binary.LittleEndian.Uint32(glbBytes[4:8])
	length := binary.LittleEndian.Uint32(glbBytes[8:12])

	if magic != glTFMagic {
		t.Errorf("Expected magic 0x%X, got 0x%X", glTFMagic, magic)
	}
	if version != 2 {
		t.Errorf("Expected glTF version 2, got %d", version)
	}
	if length != uint32(len(glbBytes)) {
		t.Errorf("Header length mismatch: declared %d vs actual %d", length, len(glbBytes))
	}

	// 2. Parse JSON Chunk
	jsonLen := binary.LittleEndian.Uint32(glbBytes[12:16])
	jsonType := binary.LittleEndian.Uint32(glbBytes[16:20])
	if jsonType != glTFJSONChunk {
		t.Errorf("Expected JSON chunk type 0x%X, got 0x%X", glTFJSONChunk, jsonType)
	}

	jsonRaw := glbBytes[20 : 20+jsonLen]
	var doc Document
	if err := json.Unmarshal(bytes.TrimRight(jsonRaw, " "), &doc); err != nil {
		t.Fatalf("Failed to unmarshal glTF JSON chunk: %v", err)
	}

	// Verify Document attributes
	if len(doc.Meshes) != 1 || doc.Meshes[0].Name != "Industrial_Outrigger_Pad" {
		t.Errorf("Mesh not correctly preserved in glTF document: %+v", doc.Meshes)
	}

	if len(doc.Materials) != 1 || doc.Materials[0].Name != "PBR_Structural_Steel" {
		t.Errorf("PBR material not correctly preserved: %+v", doc.Materials)
	}

	mat := doc.Materials[0].PbrMetallicRoughness
	if mat.MetallicFactor != 0.9 || mat.RoughnessFactor != 0.25 {
		t.Errorf("PBR factors unexpected: metallic %f, roughness %f", mat.MetallicFactor, mat.RoughnessFactor)
	}

	// 3. Verify BIN Chunk
	binHeaderOffset := 20 + jsonLen
	binLen := binary.LittleEndian.Uint32(glbBytes[binHeaderOffset : binHeaderOffset+4])
	binType := binary.LittleEndian.Uint32(glbBytes[binHeaderOffset+4 : binHeaderOffset+8])
	if binType != glTFBINChunk {
		t.Errorf("Expected BIN chunk type 0x%X, got 0x%X", glTFBINChunk, binType)
	}
	if binLen == 0 {
		t.Errorf("BIN chunk length is zero")
	}
}
