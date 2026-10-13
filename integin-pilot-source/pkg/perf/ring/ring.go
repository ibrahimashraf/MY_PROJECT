// Package ring provides a lock-free, zero-allocation bounded ring buffer for
// event dispatch and audit streaming.
//
// RingBuffer implements Dmitry Vyukov's bounded MPMC queue using per-slot
// sequence numbers and cache-line padded head/tail indices. The same
// implementation is correct (and uncontended, wait-free) for SPSC/SPMC modes.
// Push and Pop never allocate: the backing slot array is allocated once in
// NewRingBuffer, and the hot paths touch only atomic operations and the
// caller-provided values.
package ring

import (
	"errors"
	"math/bits"
	"runtime"
	"sync/atomic"

	"integin/pkg/perf/cachepad"
)

// cell is a single ring slot. The sequence number hands the slot between the
// producer and consumer side; the padding prevents false sharing between
// adjacent slots written by different cores.
type cell[T any] struct {
	seq atomic.Uint64
	val T
	_   [cachepad.PadSize - 8]byte
}

// RingBuffer is a bounded, multi-producer multi-consumer FIFO. Capacity is
// rounded up to a power of two so slot indexing uses a single bitwise mask.
type RingBuffer[T any] struct {
	cap   uint32
	mask  uint64
	head  cachepad.PaddedUint64
	tail  cachepad.PaddedUint64
	cells []cell[T]
}

// NewRingBuffer allocates a ring with room for at least capacity items.
// Capacity is rounded up to the next power of two.
func NewRingBuffer[T any](capacity int) (*RingBuffer[T], error) {
	if capacity <= 0 {
		return nil, errors.New("ring: capacity must be positive")
	}
	n := 1 << bits.Len64(uint64(capacity-1))
	rb := &RingBuffer[T]{
		cap:   uint32(n),
		mask:  uint64(n - 1),
		cells: make([]cell[T], n),
	}
	for i := range rb.cells {
		rb.cells[i].seq.Store(uint64(i))
	}
	return rb, nil
}

// Cap returns the ring's internal (power-of-two) capacity.
func (r *RingBuffer[T]) Cap() int {
	return int(r.cap)
}

// TryPush enqueues item. It reports false when the ring is full; the item is
// not placed.
func (r *RingBuffer[T]) TryPush(item T) bool {
	var c *cell[T]
	pos := r.head.Load()
	for {
		c = &r.cells[pos&r.mask]
		seq := c.seq.Load()
		switch {
		case seq == pos:
			if !r.head.CompareAndSwap(pos, pos+1) {
				runtime.Gosched()
				pos = r.head.Load()
				continue
			}
			c.val = item
			c.seq.Store(pos + 1)
			return true
		case int64(seq)-int64(pos) < 0:
			return false
		default:
			runtime.Gosched()
			pos = r.head.Load()
		}
	}
}

// TryPop dequeues the next item. It reports false when the ring is empty.
func (r *RingBuffer[T]) TryPop() (T, bool) {
	var zero T
	var c *cell[T]
	pos := r.tail.Load()
	for {
		c = &r.cells[pos&r.mask]
		seq := c.seq.Load()
		switch {
		case seq == pos+1:
			if !r.tail.CompareAndSwap(pos, pos+1) {
				runtime.Gosched()
				pos = r.tail.Load()
				continue
			}
			val := c.val
			c.seq.Store(pos + uint64(r.cap))
			return val, true
		case int64(seq)-int64(pos+1) < 0:
			return zero, false
		default:
			runtime.Gosched()
			pos = r.tail.Load()
		}
	}
}

// PushBatch pushes as many items from items as fit, in order. It returns the
// number pushed (0 when full before the first item).
func (r *RingBuffer[T]) PushBatch(items []T) int {
	n := 0
	for _, it := range items {
		if !r.TryPush(it) {
			break
		}
		n++
	}
	return n
}

// PopBatch pops up to len(dst) items in FIFO order into dst. It returns the
// number popped (0 when empty).
func (r *RingBuffer[T]) PopBatch(dst []T) int {
	for i := range dst {
		v, ok := r.TryPop()
		if !ok {
			return i
		}
		dst[i] = v
	}
	return len(dst)
}

// Len returns a best-effort snapshot of the number of items in the ring.
// Under concurrent push/pop it is not exact but is bounded by Cap().
func (r *RingBuffer[T]) Len() int {
	head := r.head.Load()
	tail := r.tail.Load()
	if tail > head {
		return 0
	}
	n := head - tail
	if n > uint64(r.cap) {
		n = uint64(r.cap)
	}
	return int(n)
}

// IsEmpty reports whether the ring is empty (snapshot).
func (r *RingBuffer[T]) IsEmpty() bool {
	return r.head.Load() == r.tail.Load()
}

// IsFull reports whether the ring is full (snapshot).
func (r *RingBuffer[T]) IsFull() bool {
	head := r.head.Load()
	tail := r.tail.Load()
	return tail <= head && head-tail >= uint64(r.cap)
}
