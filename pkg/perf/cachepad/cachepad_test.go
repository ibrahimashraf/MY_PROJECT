package cachepad

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

func testPadding[T any](t *testing.T, name string, one, two *T) {
	t.Helper()
	if got := unsafe.Sizeof(*one); got < PadSize {
		t.Errorf("%s size = %d, want >= %d", name, got, PadSize)
	}
	if got := uintptr(unsafe.Pointer(two)) - uintptr(unsafe.Pointer(one)); got < PadSize {
		t.Errorf("%s element stride = %d, want >= %d", name, got, PadSize)
	}
}

func TestPaddedSizes(t *testing.T) {
	var u [2]PaddedUint64
	testPadding(t, "PaddedUint64", &u[0], &u[1])
	var i [2]PaddedInt64
	testPadding(t, "PaddedInt64", &i[0], &i[1])
	var p [2]PaddedPointer[int]
	testPadding(t, "PaddedPointer", &p[0], &p[1])
}

func TestPaddedUint64(t *testing.T) {
	var p PaddedUint64
	if v := p.Load(); v != 0 {
		t.Fatalf("initial Load = %d, want 0", v)
	}
	want := uint64(1)
	if got := p.Add(want); got != want {
		t.Fatalf("Add(1) = %d, want %d", got, want)
	}
	p.Store(10)
	if v := p.Swap(20); v != 10 {
		t.Fatalf("Swap = %d, want 10", v)
	}
	if !p.CompareAndSwap(20, 30) {
		t.Fatal("CompareAndSwap(20, 30) = false, want true")
	}
	if p.CompareAndSwap(0, 999) {
		t.Fatal("CompareAndSwap(0, 999) = true on stale value, want false")
	}
	if v := p.Load(); v != 30 {
		t.Fatalf("final Load = %d, want 30", v)
	}
}

func TestPaddedInt64(t *testing.T) {
	var p PaddedInt64
	if v := p.Load(); v != 0 {
		t.Fatalf("initial Load = %d, want 0", v)
	}
	delta := int64(-7)
	if got := p.Add(delta); got != delta {
		t.Fatalf("Add(-7) = %d, want %d", got, delta)
	}
	p.Store(10)
	if v := p.Swap(20); v != 10 {
		t.Fatalf("Swap = %d, want 10", v)
	}
	if !p.CompareAndSwap(20, 30) {
		t.Fatal("CompareAndSwap(20, 30) = false, want true")
	}
	if p.CompareAndSwap(0, 999) {
		t.Fatal("CompareAndSwap(0, 999) = true on stale value, want false")
	}
	if v := p.Load(); v != 30 {
		t.Fatalf("final Load = %d, want 30", v)
	}
}

func TestPaddedPointer(t *testing.T) {
	var p PaddedPointer[int]
	if v := p.Load(); v != nil {
		t.Fatalf("initial Load = %p, want nil", v)
	}
	a, b, c := new(int), new(int), new(int)
	p.Store(a)
	if p.Load() != a {
		t.Fatal("Load after Store != a")
	}
	if v := p.Swap(b); v != a {
		t.Fatalf("Swap = %p, want a", v)
	}
	if !p.CompareAndSwap(b, c) {
		t.Fatal("CompareAndSwap(b, c) = false, want true")
	}
	if p.CompareAndSwap(a, c) {
		t.Fatal("CompareAndSwap(a, c) = true on stale value, want false")
	}
	if p.Load() != c {
		t.Fatalf("final Load = %p, want c (%p)", p.Load(), c)
	}
}

func TestConcurrentPaddedUint64Array(t *testing.T) {
	const (
		nSlots = 64
		iters  = 100_000
	)
	var slots [nSlots]PaddedUint64
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				slots[i].Add(1)
			}
		}(i)
	}
	wg.Wait()
	for i := 0; i < nSlots; i++ {
		if v := slots[i].Load(); v != iters {
			t.Errorf("slot %d = %d, want %d", i, v, iters)
		}
	}
}

func TestConcurrentSameSlot(t *testing.T) {
	const (
		nIters   = 100_000
		nWorkers = 16
	)
	var p PaddedUint64
	var wg sync.WaitGroup
	for i := 0; i < nWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < nIters; j++ {
				p.Add(1)
			}
		}()
	}
	wg.Wait()
	if v := p.Load(); v != nIters*nWorkers {
		t.Errorf("total = %d, want %d", v, nIters*nWorkers)
	}
}

func BenchmarkPaddedContention(b *testing.B) {
	var p PaddedUint64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p.Add(1)
		}
	})
}

func BenchmarkUnpaddedContention(b *testing.B) {
	var v uint64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.AddUint64(&v, 1)
		}
	})
}

func ExamplePaddedUint64() {
	var counter PaddedUint64
	counter.Add(1)
	fmt.Println(counter.Load())
	// Output: 1
}
