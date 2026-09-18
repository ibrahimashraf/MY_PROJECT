package gltf

import (
	"encoding/binary"
	"errors"
	"math"
)

var (
	ErrMalformedAccessor = errors.New("malformed accessor")
)

func ReadFloat32(data []byte, count int) ([]float32, error) {
	if len(data) < count*4 {
		return nil, ErrMalformedAccessor
	}
	res := make([]float32, count)
	for i := 0; i < count; i++ {
		bits := binary.LittleEndian.Uint32(data[i*4 : i*4+4])
		res[i] = math.Float32frombits(bits)
	}
	return res, nil
}

func ReadUint16(data []byte, count int) ([]uint16, error) {
	if len(data) < count*2 {
		return nil, ErrMalformedAccessor
	}
	res := make([]uint16, count)
	for i := 0; i < count; i++ {
		res[i] = binary.LittleEndian.Uint16(data[i*2 : i*2+2])
	}
	return res, nil
}

func ReadUint32(data []byte, count int) ([]uint32, error) {
	if len(data) < count*4 {
		return nil, ErrMalformedAccessor
	}
	res := make([]uint32, count)
	for i := 0; i < count; i++ {
		res[i] = binary.LittleEndian.Uint32(data[i*4 : i*4+4])
	}
	return res, nil
}
