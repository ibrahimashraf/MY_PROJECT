// Package cachepad provides cache-line padded atomic primitives.
//
// Each type pads its underlying value to a full 64-byte L1 cache line,
// preventing false sharing when adjacent slots in an array or struct are
// written by different cores/threads on multi-core NUMA machines.
//
// All types are zero-alloc wrappers around sync/atomic primitives with an
// identical method surface.
package cachepad

import "sync/atomic"

// PadSize is the L1 cache line size (bytes) that every padded primitive
// aligns to.
const PadSize = 64

// PaddedUint64 is an atomic uint64 padded to a full 64-byte cache line.
type PaddedUint64 struct {
	v uint64
	_ [PadSize - 8]byte
}

// Load atomically loads the value.
func (p *PaddedUint64) Load() uint64 {
	return atomic.LoadUint64(&p.v)
}

// Store atomically stores val.
func (p *PaddedUint64) Store(val uint64) {
	atomic.StoreUint64(&p.v, val)
}

// Add atomically adds delta to the value and returns the new value.
func (p *PaddedUint64) Add(delta uint64) uint64 {
	return atomic.AddUint64(&p.v, delta)
}

// Swap atomically stores new and returns the previous value.
func (p *PaddedUint64) Swap(new uint64) uint64 {
	return atomic.SwapUint64(&p.v, new)
}

// CompareAndSwap atomically stores new if the current value equals old.
// It reports whether the swap was performed.
func (p *PaddedUint64) CompareAndSwap(old, new uint64) bool {
	return atomic.CompareAndSwapUint64(&p.v, old, new)
}

// PaddedInt64 is an atomic int64 padded to a full 64-byte cache line.
type PaddedInt64 struct {
	v int64
	_ [PadSize - 8]byte
}

// Load atomically loads the value.
func (p *PaddedInt64) Load() int64 {
	return atomic.LoadInt64(&p.v)
}

// Store atomically stores val.
func (p *PaddedInt64) Store(val int64) {
	atomic.StoreInt64(&p.v, val)
}

// Add atomically adds delta to the value and returns the new value.
func (p *PaddedInt64) Add(delta int64) int64 {
	return atomic.AddInt64(&p.v, delta)
}

// Swap atomically stores new and returns the previous value.
func (p *PaddedInt64) Swap(new int64) int64 {
	return atomic.SwapInt64(&p.v, new)
}

// CompareAndSwap atomically stores new if the current value equals old.
// It reports whether the swap was performed.
func (p *PaddedInt64) CompareAndSwap(old, new int64) bool {
	return atomic.CompareAndSwapInt64(&p.v, old, new)
}

// PaddedPointer is an atomic pointer padded to a full 64-byte cache line.
type PaddedPointer[T any] struct {
	v atomic.Pointer[T]
	_ [PadSize - 8]byte
}

// Load atomically loads the value.
func (p *PaddedPointer[T]) Load() *T {
	return p.v.Load()
}

// Store atomically stores val.
func (p *PaddedPointer[T]) Store(val *T) {
	p.v.Store(val)
}

// Swap atomically stores new and returns the previous value.
func (p *PaddedPointer[T]) Swap(new *T) *T {
	return p.v.Swap(new)
}

// CompareAndSwap atomically stores new if the current value equals old.
// It reports whether the swap was performed.
func (p *PaddedPointer[T]) CompareAndSwap(old, new *T) bool {
	return p.v.CompareAndSwap(old, new)
}
