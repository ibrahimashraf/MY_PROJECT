package ring

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

func TestFIFOOrdering(t *testing.T) {
	rb, err := NewRingBuffer[int](16)
	if err != nil {
		t.Fatal(err)
	}
	const n = 16
	for i := 1; i <= n; i++ {
		if !rb.TryPush(i) {
			t.Fatalf("push %d failed", i)
		}
	}
	if !rb.IsFull() {
		t.Error("expected full after filling capacity")
	}
	if got := rb.Len(); got != n {
		t.Errorf("Len()=%d want %d", got, n)
	}
	for want := 1; want <= n; want++ {
		got, ok := rb.TryPop()
		if !ok {
			t.Fatalf("pop %d reported empty", want)
		}
		if got != want {
			t.Fatalf("FIFO violated: got %d want %d", got, want)
		}
	}
	if !rb.IsEmpty() {
		t.Error("expected empty after draining")
	}
	if got := rb.Len(); got != 0 {
		t.Errorf("Len()=%d want 0", got)
	}
}

func TestFullEmptyBoundaries(t *testing.T) {
	rb, err := NewRingBuffer[int](8)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if !rb.TryPush(i) {
			t.Fatalf("push %d failed", i)
		}
	}
	if rb.TryPush(99) {
		t.Error("push on full ring must report false")
	}
	for i := 0; i < 8; i++ {
		if _, ok := rb.TryPop(); !ok {
			t.Fatalf("pop %d reported empty", i)
		}
	}
	if _, ok := rb.TryPop(); ok {
		t.Error("pop on empty ring must report false")
	}
}

func TestBatch(t *testing.T) {
	rb, err := NewRingBuffer[int](8)
	if err != nil {
		t.Fatal(err)
	}
	if got := rb.PushBatch([]int{1, 2, 3, 4, 5}); got != 5 {
		t.Fatalf("PushBatch=%d want 5", got)
	}
	if got := rb.PushBatch([]int{6, 7, 8, 9}); got != 3 {
		t.Fatalf("PushBatch=%d want 3 (ring holds 8)", got)
	}
	dst := make([]int, 4)
	if got := rb.PopBatch(dst); got != 4 {
		t.Fatalf("PopBatch=%d want 4", got)
	}
	if dst[0] != 1 || dst[1] != 2 || dst[2] != 3 || dst[3] != 4 {
		t.Fatalf("PopBatch order wrong: %v", dst)
	}
	if got := rb.PopBatch(dst); got != 4 {
		t.Fatalf("second PopBatch=%d want 4", got)
	}
	if got := rb.PopBatch(dst); got != 0 {
		t.Fatalf("PopBatch on empty=%d want 0", got)
	}
}

func TestCapacityRounding(t *testing.T) {
	rb, err := NewRingBuffer[int](5)
	if err != nil {
		t.Fatal(err)
	}
	if got := rb.Cap(); got != 8 {
		t.Errorf("Cap()=%d want 8", got)
	}
	rb2, err := NewRingBuffer[int](8)
	if err != nil {
		t.Fatal(err)
	}
	if got := rb2.Cap(); got != 8 {
		t.Errorf("Cap()=%d want 8", got)
	}
	rb3, err := NewRingBuffer[int](1)
	if err != nil {
		t.Fatal(err)
	}
	if got := rb3.Cap(); got != 1 {
		t.Errorf("Cap()=%d want 1", got)
	}
}

func TestNewRingBufferInvalidCapacity(t *testing.T) {
	if _, err := NewRingBuffer[int](0); err == nil {
		t.Error("capacity 0 must error")
	}
	if _, err := NewRingBuffer[int](-3); err == nil {
		t.Error("negative capacity must error")
	}
}

func TestZeroAllocHotPaths(t *testing.T) {
	rb, err := NewRingBuffer[int](64)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 64; i++ {
		if !rb.TryPush(i) {
			t.Fatal("prefill failed")
		}
	}
	allocs := testing.AllocsPerRun(5000, func() {
		if _, ok := rb.TryPop(); !ok {
			t.Error("pop failed")
		}
		if !rb.TryPush(1) {
			t.Error("push failed")
		}
	})
	if allocs != 0 {
		t.Fatalf("hot path allocated %v bytes/op, want 0", allocs)
	}
}

func TestSPSCStress(t *testing.T) {
	const n = 100_000
	rb, err := NewRingBuffer[int](1024)
	if err != nil {
		t.Fatal(err)
	}
	var prod sync.WaitGroup
	prod.Add(1)
	go func() {
		defer prod.Done()
		for i := 1; i <= n; i++ {
			for !rb.TryPush(i) {
				runtime.Gosched()
			}
		}
	}()
	errCh := make(chan error, 1)
	go func() {
		want := 1
		for want <= n {
			v, ok := rb.TryPop()
			if !ok {
				runtime.Gosched()
				continue
			}
			if v != want {
				errCh <- fmt.Errorf("SPSC order: got %d want %d", v, want)
				return
			}
			want++
		}
		close(errCh)
	}()
	prod.Wait()
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentMPMCNoDropsNoDuplicates(t *testing.T) {
	const (
		producers = 4
		consumers = 4
		each      = 20_000
		cap       = 256
	)
	total := producers * each
	rb, err := NewRingBuffer[int64](cap)
	if err != nil {
		t.Fatal(err)
	}

	var prodWg sync.WaitGroup
	prodWg.Add(producers)
	done := make(chan struct{})
	go func() {
		prodWg.Wait()
		close(done)
	}()

	for p := 0; p < producers; p++ {
		go func(p int) {
			defer prodWg.Done()
			for i := 0; i < each; {
				v := int64(p)<<32 | int64(i)
				if rb.TryPush(v) {
					i++
					continue
				}
				runtime.Gosched()
			}
		}(p)
	}

	var consWg sync.WaitGroup
	consWg.Add(consumers)
	results := make([]map[int64]struct{}, consumers)
	for c := 0; c < consumers; c++ {
		seen := make(map[int64]struct{}, each)
		results[c] = seen
		go func() {
			defer consWg.Done()
			for {
				v, ok := rb.TryPop()
				if !ok {
					select {
					case <-done:
						if rb.IsEmpty() {
							return
						}
						runtime.Gosched()
					default:
						runtime.Gosched()
					}
					continue
				}
				if _, dup := seen[v]; dup {
					t.Errorf("consumer saw duplicate value %d", v)
					return
				}
				seen[v] = struct{}{}
			}
		}()
	}
	consWg.Wait()

	all := make(map[int64]struct{}, total)
	for _, seen := range results {
		for v := range seen {
			if _, dup := all[v]; dup {
				t.Errorf("value %d consumed more than once", v)
			}
			all[v] = struct{}{}
		}
	}
	if got := len(all); got != total {
		t.Errorf("unique consumed = %d, want %d (drops or duplicates)", got, total)
	}
	if !rb.IsEmpty() {
		t.Errorf("ring should be drained, Len()=%d", rb.Len())
	}
}

func BenchmarkRingPushPop(b *testing.B) {
	rb, err := NewRingBuffer[int](1024)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !rb.TryPush(i) {
			b.Fatal("push reported full")
		}
		if _, ok := rb.TryPop(); !ok {
			b.Fatal("pop reported empty")
		}
	}
}

func BenchmarkChanPushPop(b *testing.B) {
	ch := make(chan int, 1024)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		select {
		case ch <- i:
		default:
			b.Fatal("chan full")
		}
		select {
		case <-ch:
		default:
			b.Fatal("chan empty")
		}
	}
}

func BenchmarkRingTryPop(b *testing.B) {
	rb, err := NewRingBuffer[int](1024)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 1024; i++ {
		if !rb.TryPush(i) {
			b.Fatal("prefill failed")
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := rb.TryPop(); !ok {
			for j := 0; j < 1024; j++ {
				if !rb.TryPush(j) {
					b.Fatal("refill failed")
				}
			}
		}
	}
}

func BenchmarkChanTryPop(b *testing.B) {
	ch := make(chan int, 1024)
	for i := 0; i < 1024; i++ {
		ch <- i
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case <-ch:
		default:
			for j := 0; j < 1024; j++ {
				ch <- j
			}
		}
	}
}
