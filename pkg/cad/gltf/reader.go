package gltf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
)

var (
	ErrOversizedAsset = errors.New("GLB exceeds safety limits")
)

const (
	MaxFileSize = 128 * 1024 * 1024 // 128 MB
	MaxVertices = 500000
	MaxIndices  = 1500000
)

func ReadGLB(data []byte) (*Document, []byte, error) {
	if len(data) > MaxFileSize {
		return nil, nil, ErrOversizedAsset
	}
	if len(data) < 20 {
		return nil, nil, errors.New("GLB too short")
	}

	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != glTFMagic {
		return nil, nil, errors.New("not a GLB file")
	}

	length := binary.LittleEndian.Uint32(data[8:12])
	if uint32(len(data)) < length {
		return nil, nil, ErrMalformedAccessor
	}

	jsonLen := binary.LittleEndian.Uint32(data[12:16])
	jsonType := binary.LittleEndian.Uint32(data[16:20])
	if jsonType != glTFJSONChunk {
		return nil, nil, errors.New("missing JSON chunk")
	}

	if 20+jsonLen > uint32(len(data)) {
		return nil, nil, errors.New("JSON chunk truncated")
	}

	jsonRaw := bytes.TrimRight(data[20:20+jsonLen], " ")
	var doc Document
	if err := json.Unmarshal(jsonRaw, &doc); err != nil {
		return nil, nil, err
	}

	binOffset := 20 + jsonLen
	if binOffset+8 > uint32(len(data)) {
		return &doc, nil, nil // No BIN chunk
	}

	binLen := binary.LittleEndian.Uint32(data[binOffset : binOffset+4])
	binType := binary.LittleEndian.Uint32(data[binOffset+4 : binOffset+8])
	if binType != glTFBINChunk {
		return &doc, nil, nil
	}

	if binOffset+8+binLen > uint32(len(data)) {
		return nil, nil, errors.New("BIN chunk truncated")
	}

	binData := data[binOffset+8 : binOffset+8+binLen]

	totalVertices := 0
	totalIndices := 0

	for _, acc := range doc.Accessors {
		if acc.Type == "VEC3" {
			totalVertices += acc.Count
		} else if acc.Type == "SCALAR" {
			totalIndices += acc.Count
		}
		
		if totalVertices > MaxVertices || totalIndices > MaxIndices {
			return nil, nil, ErrOversizedAsset
		}

		if acc.BufferView >= len(doc.BufferViews) {
			return nil, nil, ErrMalformedAccessor
		}
		bv := doc.BufferViews[acc.BufferView]
		
		accTotal := acc.ByteOffset
		if acc.Type == "VEC3" {
			accTotal += acc.Count * 12
		} else if acc.Type == "SCALAR" && acc.ComponentType == ComponentTypeUnsignedShort {
			accTotal += acc.Count * 2
		} else if acc.Type == "SCALAR" && acc.ComponentType == ComponentTypeUnsignedInt {
			accTotal += acc.Count * 4
		} else if acc.Type == "SCALAR" && acc.ComponentType == ComponentTypeFloat {
			accTotal += acc.Count * 4
		}
		
		if accTotal > bv.ByteLength {
			return nil, nil, ErrMalformedAccessor
		}
		if bv.ByteOffset+bv.ByteLength > len(binData) {
			return nil, nil, ErrMalformedAccessor
		}
	}

	return &doc, binData, nil
}
