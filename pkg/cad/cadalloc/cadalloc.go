package cadalloc

import (
	"errors"
	"sync/atomic"

	"integin/pkg/cad/galloc"
)

var (
	ErrDoubleFree  = errors.New("double free detected")
	ErrNotPowerOf2 = errors.New("alignment must be a power of 2")
)

type Allocation struct {
	Data  []byte
	freed atomic.Bool
}

// AllocateAligned wraps galloc.AllocateAligned with safety checks
func AllocateAligned(size, alignment uint64) (*Allocation, error) {
	if alignment == 0 || (alignment&(alignment-1)) != 0 {
		return nil, ErrNotPowerOf2
	}

	buf := galloc.AllocateAligned(size, alignment)
	return &Allocation{
		Data: buf,
	}, nil
}

// Free wraps galloc.Free and prevents double-free panics
func (a *Allocation) Free() error {
	if !a.freed.CompareAndSwap(false, true) {
		return ErrDoubleFree
	}
	
	galloc.Free(a.Data)
	return nil
}
