package bce

import (
	"bytes"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var (
	sinkU64 uint64
	sinkN   int
	sinkOK  bool
)

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*2654435761%251 + i/3)
	}
	return b
}

func readCheck(t *testing.T, name string, got uint64, ok bool, want uint64, wantOK bool) {
	t.Helper()
	if ok != wantOK {
		t.Fatalf("%s: ok = %v, want %v", name, ok, wantOK)
	}
	if ok && got != want {
		t.Fatalf("%s: got %#x, want %#x", name, got, want)
	}
}

func TestReadUint64LE(t *testing.T) {
	for i := 0; i < 256; i++ {
		want := rand.Uint64()
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, want)
		got, ok := ReadUint64LE(buf)
		readCheck(t, "LE", got, ok, want, true)
	}
	for n := 0; n < 8; n++ {
		got, ok := ReadUint64LE(randBytes(n))
		readCheck(t, "LE short", got, ok, 0, false)
	}
}

func TestReadUint64BE(t *testing.T) {
	for i := 0; i < 256; i++ {
		want := rand.Uint64()
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, want)
		got, ok := ReadUint64BE(buf)
		readCheck(t, "BE", got, ok, want, true)
	}
	for n := 0; n < 8; n++ {
		got, ok := ReadUint64BE(randBytes(n))
		readCheck(t, "BE short", got, ok, 0, false)
	}
}

func TestWriteUint64LE(t *testing.T) {
	for i := 0; i < 256; i++ {
		v := rand.Uint64()
		buf := make([]byte, 8)
		if !WriteUint64LE(buf, v) {
			t.Fatalf("WriteUint64LE ok = false")
		}
		if got := binary.LittleEndian.Uint64(buf); got != v {
			t.Fatalf("LE roundtrip: got %#x, want %#x", got, v)
		}
	}
	if WriteUint64LE(make([]byte, 7), 0) {
		t.Fatal("WriteUint64LE short buffer reported ok")
	}
}

func TestWriteUint64BE(t *testing.T) {
	for i := 0; i < 256; i++ {
		v := rand.Uint64()
		buf := make([]byte, 8)
		if !WriteUint64BE(buf, v) {
			t.Fatalf("WriteUint64BE ok = false")
		}
		if got := binary.BigEndian.Uint64(buf); got != v {
			t.Fatalf("BE roundtrip: got %#x, want %#x", got, v)
		}
	}
	if WriteUint64BE(make([]byte, 7), 0) {
		t.Fatal("WriteUint64BE short buffer reported ok")
	}
}

func TestXORBytes(t *testing.T) {
	for i := 0; i < 128; i++ {
		a := randBytes(32)
		b := randBytes(32)
		dst := make([]byte, 40)
		n := XORBytes(dst, a, b)
		if n != 32 {
			t.Fatalf("XORBytes n = %d, want 32", n)
		}
		for j := 0; j < 32; j++ {
			if want := a[j] ^ b[j]; dst[j] != want {
				t.Fatalf("XORBytes dst[%d] = %#x, want %#x", j, dst[j], want)
			}
		}
		if !bytes.Equal(dst[32:], make([]byte, 8)) {
			t.Fatal("XORBytes wrote past the min length")
		}
	}
	// Uneven lengths: writes min(len(a), len(b)) bytes.
	a := randBytes(16)
	short := randBytes(7)
	dst := make([]byte, 16)
	if n := XORBytes(dst, a, short); n != 7 {
		t.Fatalf("XORBytes uneven n = %d, want 7", n)
	}
	// Empty inputs write zero bytes without touching dst.
	for j := 0; j < len(dst); j++ {
		dst[j] = 0xEE
	}
	if n := XORBytes(dst, nil, short); n != 0 {
		t.Fatalf("XORBytes nil a n = %d, want 0", n)
	}
	if !bytes.Equal(dst, bytes.Repeat([]byte{0xEE}, 16)) {
		t.Fatal("XORBytes modified dst on empty input")
	}
	// Short destination panics, matching crypto/subtle.XORBytes semantics.
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("XORBytes short dst did not panic")
			}
		}()
		XORBytes(make([]byte, 3), make([]byte, 4), make([]byte, 4))
	}()
}

func TestConstantTimeCompare32(t *testing.T) {
	var x, y [32]byte
	if !ConstantTimeCompare32(&x, &y) {
		t.Fatal("equal [32]byte reported unequal")
	}
	for i := 0; i < 32; i++ {
		y = x
		y[i] = 0xFF
		if ConstantTimeCompare32(&x, &y) {
			t.Fatalf("differing byte %d reported equal", i)
		}
		if got, want := subtle.ConstantTimeCompare(x[:], y[:]), 0; got != want {
			t.Fatalf("byte %d: subtle compare = %d, want %d", i, got, want)
		}
	}
}

func TestConstantTimeCompare64(t *testing.T) {
	var x, y [64]byte
	if !ConstantTimeCompare64(&x, &y) {
		t.Fatal("equal [64]byte reported unequal")
	}
	for i := 0; i < 64; i++ {
		y = x
		y[i] = 1
		if ConstantTimeCompare64(&x, &y) {
			t.Fatalf("differing byte %d reported equal", i)
		}
	}
}

func TestHexEncode32(t *testing.T) {
	var src [32]byte
	cases := [][]byte{
		{0x00},
		{0xFF},
		{0x01},
		{0x12, 0x34, 0xAB, 0xCD, 0xEF, 0x89, 0x0A, 0xBC},
	}
	for _, c := range cases {
		clear(src[:])
		copy(src[:], c)
		dst := make([]byte, 64)
		if !HexEncode32(dst, &src) {
			t.Fatalf("HexEncode32 ok = false")
		}
		want := make([]byte, 64)
		hex.Encode(want, src[:])
		if !bytes.Equal(dst, want) {
			t.Fatalf("HexEncode32 = %q, want %q", dst, want)
		}
	}
	// Bytes beyond the first 64 of dst are untouched.
	full := make([]byte, 96)
	for i := range full {
		full[i] = 0xEE
	}
	if !HexEncode32(full[:64], &src) {
		t.Fatal("HexEncode32 ok = false on 64-byte dst")
	}
	if !bytes.Equal(full[64:], bytes.Repeat([]byte{0xEE}, 32)) {
		t.Fatal("HexEncode32 wrote past its 64-byte prefix")
	}
	if HexEncode32(make([]byte, 63), &src) {
		t.Fatal("HexEncode32 short dst reported ok")
	}
}

func TestZeroAllocations(t *testing.T) {
	buf8 := make([]byte, 8)
	a := make([]byte, 16)
	b := make([]byte, 15)
	dst := make([]byte, 16)
	var x, y [32]byte
	var d [64]byte
	var src [32]byte
	hbuf := make([]byte, 64)

	check := func(name string, f func()) {
		t.Helper()
		if allocs := testing.AllocsPerRun(1000, f); allocs != 0 {
			t.Errorf("%s: %v allocs/op, want 0", name, allocs)
		}
	}
	check("ReadUint64LE", func() {
		sinkU64, sinkOK = ReadUint64LE(buf8)
	})
	check("ReadUint64BE", func() {
		sinkU64, sinkOK = ReadUint64BE(buf8)
	})
	check("WriteUint64LE", func() {
		sinkOK = WriteUint64LE(buf8, 0x0123456789abcdef)
	})
	check("WriteUint64BE", func() {
		sinkOK = WriteUint64BE(buf8, 0x0123456789abcdef)
	})
	check("XORBytes", func() {
		sinkN = XORBytes(dst, a, b)
	})
	check("ConstantTimeCompare32", func() {
		sinkOK = ConstantTimeCompare32(&x, &y)
	})
	check("ConstantTimeCompare64", func() {
		sinkOK = ConstantTimeCompare64(&d, &d)
	})
	check("HexEncode32", func() {
		sinkOK = HexEncode32(hbuf, &src)
	})
}

func TestBCEVerified(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	srcdir := filepath.Dir(file)
	src := filepath.Join(srcdir, "bce.go")

	// The go command wraps the compiler, preserving GOROOT/GOTOOLDIR.
	cmd := exec.Command("go", "tool", "compile", "-p", "bce",
		"-d=ssa/check_bce/debug=1", "-o", filepath.Join(t.TempDir(), "bce.o"), src)
	cmd.Dir = srcdir
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Skipf("compiler BCE debug pass unavailable: %v\n%s", err, out.String())
	}

	hot := map[string]bool{
		"ReadUint64LE": true, "ReadUint64BE": true,
		"WriteUint64LE": true, "WriteUint64BE": true,
		"XORBytes": true, "ConstantTimeCompare32": true,
		"ConstantTimeCompare64": true, "HexEncode32": true,
	}
	spans := funcSpans(t, src)

	lineRe := regexp.MustCompile(`^bce\.go:(\d+):`)
	for _, line := range strings.Split(out.String(), "\n") {
		if !strings.Contains(line, "IsInBounds") {
			continue
		}
		m := lineRe.FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("unparseable BCE report: %q", line)
		}
		ln, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("bad line number in BCE report %q: %v", line, err)
		}
		for fn, rng := range spans {
			if hot[fn] && ln >= rng[0] && ln <= rng[1] {
				t.Errorf("%s: surviving bounds check at %s", fn, line)
			}
		}
	}
}

// funcSpans maps each top-level function name to its [start, end] line range.
func funcSpans(t *testing.T, src string) map[string][2]int {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	lines := strings.Split(string(data), "\n")
	re := regexp.MustCompile(`^func\s+(\w+)`)
	spans := map[string][2]int{}
	var order []string
	for i, l := range lines {
		if m := re.FindStringSubmatch(l); m != nil {
			spans[m[1]] = [2]int{i + 1, i + 1}
			order = append(order, m[1])
		}
	}
	for i := 0; i+1 < len(order); i++ {
		cur := spans[order[i]]
		cur[1] = spans[order[i+1]][0] - 1
		spans[order[i]] = cur
	}
	last := spans[order[len(order)-1]]
	last[1] = len(lines)
	spans[order[len(order)-1]] = last
	return spans
}

func BenchmarkReadUint64LE(b *testing.B) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, 0x0123456789abcdef)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkU64, _ = ReadUint64LE(buf)
	}
}

func BenchmarkReadUint64BE(b *testing.B) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, 0x0123456789abcdef)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkU64, _ = ReadUint64BE(buf)
	}
}

func BenchmarkWriteUint64LE(b *testing.B) {
	buf := make([]byte, 8)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkOK = WriteUint64LE(buf, uint64(i))
	}
}

func BenchmarkWriteUint64BE(b *testing.B) {
	buf := make([]byte, 8)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkOK = WriteUint64BE(buf, uint64(i))
	}
}

func BenchmarkXORBytes(b *testing.B) {
	a := randBytes(8192)
	x := randBytes(8192)
	dst := make([]byte, 8192)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkN = XORBytes(dst, a, x)
	}
}

func BenchmarkConstantTimeCompare32(b *testing.B) {
	var x, y [32]byte
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkOK = ConstantTimeCompare32(&x, &y)
	}
}

func BenchmarkConstantTimeCompare64(b *testing.B) {
	var x, y [64]byte
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkOK = ConstantTimeCompare64(&x, &y)
	}
}

func BenchmarkHexEncode32(b *testing.B) {
	var src [32]byte
	dst := make([]byte, 64)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkOK = HexEncode32(dst, &src)
	}
}

func BenchmarkHexEncodeStdlib(b *testing.B) {
	var src [32]byte
	dst := make([]byte, 64)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hex.Encode(dst, src[:])
	}
}

func BenchmarkConstantTimeCompareStdlib(b *testing.B) {
	x := make([]byte, 32)
	y := make([]byte, 32)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkN = subtle.ConstantTimeCompare(x, y)
	}
}
