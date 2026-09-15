package arena

import "sync"

const DefaultSlabBytes = 64 << 10

type Arena struct {
	buf      []byte
	offset   int
	overflow int
}

func NewArena(capacity int) *Arena {
	if capacity < 8 {
		panic("arena: slab capacity must be >= 8 bytes")
	}
	return &Arena{buf: make([]byte, capacity)}
}

func (a *Arena) Alloc(size int) []byte {
	if size < 0 {
		panic("arena: negative allocation size")
	}
	aligned := (size + 7) &^ 7
	if aligned < size {
		panic("arena: allocation size overflow")
	}
	if remaining := len(a.buf) - a.offset; aligned <= remaining {
		b := a.buf[a.offset : a.offset+size : a.offset+aligned]
		a.offset += aligned
		return b
	}
	b := make([]byte, aligned)
	a.overflow++
	return b
}

func (a *Arena) AllocString(s string) string {
	b := a.Alloc(len(s))
	copy(b, s)
	return string(b)
}

func (a *Arena) Reset() {
	a.offset = 0
	a.overflow = 0
}

func (a *Arena) AllocatedBytes() int { return a.offset }

func (a *Arena) Capacity() int { return len(a.buf) }

func (a *Arena) OverflowCount() int { return a.overflow }

type Pool struct {
	slabSize int
	pool     sync.Pool
}

func NewPool(slabSize int) *Pool {
	p := &Pool{slabSize: slabSize}
	p.pool.New = func() any {
		return NewArena(p.normSize())
	}
	return p
}

func (p *Pool) normSize() int {
	if p.slabSize < 8 {
		return DefaultSlabBytes
	}
	return p.slabSize
}

func (p *Pool) Get() *Arena {
	return p.pool.Get().(*Arena)
}

func (p *Pool) Put(a *Arena) {
	a.Reset()
	p.pool.Put(a)
}
