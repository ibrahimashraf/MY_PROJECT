package arena

import (
	"sync"
	"testing"
	"unsafe"
)

func ptrOf(b []byte) uintptr {
	return uintptr(unsafe.Pointer(&b[0]))
}

func TestAllocAlign8(t *testing.T) {
	a := NewArena(128)
	lens := []int{1, 8, 16, 24, 3, 64}
	for i, want := range lens {
		b := a.Alloc(want)
		if len(b) != want {
			t.Fatalf("alloc %d: len=%d want=%d", i, len(b), want)
		}
		if ptrOf(b)%8 != 0 {
			t.Fatalf("alloc %d: ptr %#x not 8-aligned", i, ptrOf(b))
		}
		for j := range b {
			b[j] = byte(i + 1)
		}
	}
	off := 0
	for i, wantLen := range lens {
		wantTag := byte(i + 1)
		for j := 0; j < wantLen; j++ {
			if a.buf[off+j] != wantTag {
				t.Fatalf("alloc %d byte %d: got %d want %d", i, j, a.buf[off+j], wantTag)
			}
		}
		off += (wantLen + 7) &^ 7
	}
	if off != 128 {
		t.Fatalf("expected all 128 slab bytes consumed, consumed %d", off)
	}
}

func TestAllocOverflow(t *testing.T) {
	a := NewArena(64)
	if b := a.Alloc(64); len(b) != 64 {
		t.Fatalf("full alloc: len=%d", len(b))
	}
	if a.AllocatedBytes() != 64 {
		t.Fatalf("AllocatedBytes=%d want 64", a.AllocatedBytes())
	}
	if b := a.Alloc(16); len(b) != 16 {
		t.Fatalf("overflow alloc: len=%d want 16", len(b))
	}
	if a.OverflowCount() != 1 {
		t.Fatalf("OverflowCount=%d want 1", a.OverflowCount())
	}
	if a.AllocatedBytes() != 64 {
		t.Fatalf("AllocatedBytes=%d want 64 (overflow must not advance slab)", a.AllocatedBytes())
	}
	b := a.Alloc(1)
	if len(b) < 1 {
		t.Fatalf("overflow alloc of 1 byte must return usable buffer")
	}
	if a.OverflowCount() != 2 {
		t.Fatalf("OverflowCount=%d want 2", a.OverflowCount())
	}
}

func TestResetReuse(t *testing.T) {
	a := NewArena(8)
	if s := a.AllocString("abcdef"); s != "abcdef" {
		t.Fatalf("AllocString got %q", s)
	}
	if a.AllocatedBytes() != 8 {
		t.Fatalf("AllocatedBytes=%d want 8", a.AllocatedBytes())
	}
	if b := a.Alloc(1); len(b) != 8 {
		t.Fatalf("expected overflow alloc of aligned 8, got len=%d", len(b))
	}
	if a.OverflowCount() != 1 {
		t.Fatalf("OverflowCount=%d want 1", a.OverflowCount())
	}
	a.Reset()
	if a.AllocatedBytes() != 0 {
		t.Fatalf("After Reset AllocatedBytes=%d want 0", a.AllocatedBytes())
	}
	if a.OverflowCount() != 0 {
		t.Fatalf("After Reset overflow=%d want 0", a.OverflowCount())
	}
	if s := a.AllocString("xyz"); s != "xyz" {
		t.Fatalf("reuse after Reset got %q", s)
	}
	if ptrOf(a.buf[:4])%8 != 0 {
		t.Fatalf("after Reset base not 8-aligned")
	}
}

func TestPoolConcurrent(t *testing.T) {
	p := NewPool(0)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			<-start
			for i := 0; i < 200; i++ {
				ar := p.Get()
				s := ar.AllocString("tenant-value")
				if s != "tenant-value" {
					t.Errorf("goroutine %d iter %d: corrupted string", g, i)
				}
				if b := ar.Alloc(3); ptrOf(b)%8 != 0 {
					t.Errorf("goroutine %d iter %d: misaligned", g, i)
				}
				ar.Reset()
				p.Put(ar)
			}
		}(g)
	}
	close(start)
	wg.Wait()
}

var benchSink []byte

func BenchmarkMakeAlloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = make([]byte, 1024)
	}
}

func BenchmarkArenaAlloc(b *testing.B) {
	a := NewArena(1 << 20)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = a.Alloc(1024)
	}
}
