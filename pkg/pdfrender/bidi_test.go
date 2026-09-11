package pdfrender

import (
	"strings"
	"testing"
)

func concatRuns(r ReorderResult) string {
	var sb strings.Builder
	for _, rn := range r.Runs {
		sb.WriteString(rn.Text)
	}
	return sb.String()
}

func TestReorderPureLTR(t *testing.T) {
	in := "Certificate 2026"
	r, err := Reorder(in)
	if err != nil {
		t.Fatalf("Reorder(%q): unexpected error: %v", in, err)
	}
	if len(r.Runs) != 1 {
		t.Fatalf("want 1 run, got %d", len(r.Runs))
	}
	rn := r.Runs[0]
	if rn.Direction != LTR {
		t.Fatalf("want LTR run, got %v", rn.Direction)
	}
	if rn.Text != in {
		t.Fatalf("want unchanged text %q, got %q", in, rn.Text)
	}
	if rn.LogicalStart != 0 {
		t.Fatalf("want LogicalStart 0, got %d", rn.LogicalStart)
	}
	if rn.Length != len([]rune(in)) {
		t.Fatalf("want Length %d, got %d", len([]rune(in)), rn.Length)
	}
	if r.BaseDirection != LTR {
		t.Fatalf("want base LTR, got %v", r.BaseDirection)
	}
	for i, want := range []int{0, 1, 2, 3} {
		if r.LogicalToVisual[i] != want || r.VisualToLogical[want] != i {
			t.Fatalf("identity map broken at index %d", i)
		}
	}
}

func TestReorderPureRTL(t *testing.T) {
	in := "شهادة فحص"
	r, err := Reorder(in)
	if err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}
	if r.BaseDirection != RTL {
		t.Fatalf("want base RTL, got %v", r.BaseDirection)
	}
	if len(r.Runs) != 1 {
		t.Fatalf("want 1 run, got %d", len(r.Runs))
	}
	rn := r.Runs[0]
	if rn.Direction != RTL {
		t.Fatalf("want RTL run, got %v", rn.Direction)
	}
	if got := concatRuns(r); got != "صحف ةداهش" {
		t.Fatalf("want %q, got %q", "صحف ةداهش", got)
	}
}

func TestReorderMixedCertHeader(t *testing.T) {
	in := "Certificate شهادة 2026"
	r, err := Reorder(in)
	if err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}
	// The neutral between the Arabic word and its (weak-R) number context
	// resolves into the RTL run (rule N1), so the flat concatenation shows a
	// seam space moved to the run's front; the PDF engine repositions it by
	// placing each run with its own direction. The run structure below is
	// the contract.
	if len(r.Runs) != 3 {
		t.Fatalf("want 3 runs, got %d", len(r.Runs))
	}
	if r.Runs[0].Direction != LTR || r.Runs[0].Text != "Certificate " ||
		r.Runs[0].LogicalStart != 0 {
		t.Fatalf("run 0 wrong: %+v", r.Runs[0])
	}
	if r.Runs[1].Direction != RTL || r.Runs[1].Text != " ةداهش" ||
		r.Runs[1].LogicalStart != 12 || r.Runs[1].Length != 6 {
		t.Fatalf("run 1 wrong: %+v", r.Runs[1])
	}
	if r.Runs[2].Direction != LTR || r.Runs[2].Text != "2026" ||
		r.Runs[2].LogicalStart != 18 || r.Runs[2].Length != 4 {
		t.Fatalf("run 2 wrong: %+v", r.Runs[2])
	}
	// Logical ش -> leftmost slot of the RTL run; inverse agrees.
	if r.LogicalToVisual[12] != 17 {
		t.Fatalf("want LogicalToVisual[12]=17, got %d", r.LogicalToVisual[12])
	}
	if r.VisualToLogical[17] != 12 {
		t.Fatalf("want VisualToLogical[17]=12, got %d", r.VisualToLogical[17])
	}
	// Controls absent, so every logical index has a visual slot.
	for i, v := range r.LogicalToVisual {
		if v < 0 {
			t.Fatalf("unexpected control-like removal at logical index %d", i)
		}
	}
}

func TestReorderNumbersStayLTRInsideRTL(t *testing.T) {
	in := "شهادة 2026"
	r, err := Reorder(in)
	if err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}
	want := " ةداهش2026"
	if got := concatRuns(r); got != want {
		t.Fatalf("want visual %q, got %q", want, got)
	}
	if len(r.Runs) != 2 {
		t.Fatalf("want 2 runs, got %d", len(r.Runs))
	}
	if r.Runs[0].Direction != RTL || r.Runs[0].Text != " ةداهش" {
		t.Fatalf("run 0 wrong: %+v", r.Runs[0])
	}
	if r.Runs[1].Direction != LTR || r.Runs[1].Text != "2026" {
		t.Fatalf("run 1 wrong: %+v", r.Runs[1])
	}
	// Digits keep forward order: each digit maps to itself visually.
	for d := 0; d < 4; d++ {
		vis := r.LogicalToVisual[6+d] // logical index of digit '2','0','2','6'
		if got := r.VisualToLogical[vis]; got != 6+d {
			t.Fatalf("digit %d misplaced: logical %d -> visual %d -> logical %d", d, 6+d, vis, got)
		}
	}
}

func TestReorderEmbeddingControls(t *testing.T) {
	in := "a\u202Bشب\u202Cd" // a, RLE, ش, ب, PDF, d
	r, err := Reorder(in)
	if err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}
	if got := concatRuns(r); got != "aبشd" {
		t.Fatalf("want visual %q, got %q", "aبشd", got)
	}
	if len(r.Runs) != 3 {
		t.Fatalf("want 3 runs, got %d", len(r.Runs))
	}
	if r.Runs[1].Direction != RTL || r.Runs[1].Text != "بش" {
		t.Fatalf("embedded run wrong: %+v", r.Runs[1])
	}
	// Formatting controls are removed from the output maps.
	if r.LogicalToVisual[1] != -1 || r.LogicalToVisual[4] != -1 {
		t.Fatalf("controls must map to -1, got %v", r.LogicalToVisual)
	}
	if len(r.VisualToLogical) != 4 {
		t.Fatalf("want 4 displayed runes, got %d", len(r.VisualToLogical))
	}
}

func TestReorderIsolateRLI(t *testing.T) {
	in := "a\u2067شب\u2069c" // a, RLI, ش, ب, PDI, c
	r, err := Reorder(in)
	if err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}
	if got := concatRuns(r); got != "aبشc" {
		t.Fatalf("want visual %q, got %q", "aبشc", got)
	}
	if len(r.Runs) != 3 || r.Runs[1].Direction != RTL || r.Runs[1].Text != "بش" {
		t.Fatalf("unexpected runs: %+v", r.Runs)
	}
	// Isolate contents do not leak into the surrounding LTR paragraph.
	if r.LogicalToVisual[2] != 2 || r.VisualToLogical[1] != 3 {
		t.Fatalf("index maps wrong for isolate content: %v / %v",
			r.LogicalToVisual, r.VisualToLogical)
	}
}

func TestReorderIsolateFSIFromFirstStrong(t *testing.T) {
	in := "a\u2068\u05D0\u05D1\u2069c" // a, FSI, alef (R), bet (R), PDI, c
	r, err := Reorder(in)
	if err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}
	if got := concatRuns(r); got != "aבאc" {
		t.Fatalf("want visual %q, got %q", "aבאc", got)
	}
	if len(r.Runs) != 3 || r.Runs[1].Direction != RTL || r.Runs[1].Text != "בא" {
		t.Fatalf("unexpected runs: %+v", r.Runs)
	}
}

func TestReorderUnmatchedPDFFails(t *testing.T) {
	cases := []string{
		"ab\u202C",
		"\u202C",
		"a\u202Bb\u202C\u202C",
		"ab\u2067c\u202C", // PDF inside an isolate is invalid: PDI terminates it
	}
	for _, in := range cases {
		if _, err := Reorder(in); err != ErrUnmatchedPDF {
			t.Errorf("Reorder(%q): want ErrUnmatchedPDF, got %v", in, err)
		}
	}
}

func TestReorderUnmatchedPDIFails(t *testing.T) {
	for _, in := range []string{"\u2069", "ab\u2069"} {
		if _, err := Reorder(in); err != ErrUnmatchedPDI {
			t.Errorf("Reorder(%q): want ErrUnmatchedPDI, got %v", in, err)
		}
	}
}

func TestReorderInvalidUTF8Fails(t *testing.T) {
	for _, in := range []string{"\xff", "ab\xfe\xfc", "\xc3"} {
		if _, err := Reorder(in); err != ErrInvalidUTF8 {
			t.Errorf("Reorder(%q): want ErrInvalidUTF8, got %v", in, err)
		}
	}
}

func TestReorderMirroredParentheses(t *testing.T) {
	in := "(شهادة)"
	r, err := Reorder(in)
	if err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}
	if len(r.Runs) != 1 {
		t.Fatalf("want 1 run, got %d", len(r.Runs))
	}
	if r.Runs[0].Direction != RTL {
		t.Fatalf("want RTL run, got %v", r.Runs[0].Direction)
	}
	if got := r.Runs[0].Text; got != "(ةداهش)" {
		t.Fatalf("want mirrored visual %q, got %q", "(ةداهش)", got)
	}
}

func TestReorderLTREmbedInsideRTL(t *testing.T) {
	in := "ش\u202Aabc\u202Cة"
	r, err := Reorder(in)
	if err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}
	want := "ةabcش"
	if got := concatRuns(r); got != want {
		t.Fatalf("want visual %q, got %q", want, got)
	}
	if len(r.Runs) != 3 {
		t.Fatalf("want 3 runs, got %d", len(r.Runs))
	}
	if r.Runs[0].Direction != RTL || r.Runs[0].Text != "ة" {
		t.Fatalf("run 0 wrong: %+v", r.Runs[0])
	}
	if r.Runs[1].Direction != LTR || r.Runs[1].Text != "abc" {
		t.Fatalf("run 1 wrong: %+v", r.Runs[1])
	}
	if r.Runs[2].Direction != RTL || r.Runs[2].Text != "ش" {
		t.Fatalf("run 2 wrong: %+v", r.Runs[2])
	}
}

func TestReorderEmptyInput(t *testing.T) {
	r, err := Reorder("")
	if err != nil {
		t.Fatalf("Reorder(\"\"): unexpected error: %v", err)
	}
	if len(r.Runs) != 0 || len(r.VisualToLogical) != 0 || len(r.LogicalToVisual) != 0 {
		t.Fatalf("empty input must produce empty maps and runs, got %+v", r)
	}
	if r.BaseDirection != LTR {
		t.Fatalf("want base LTR for empty input, got %v", r.BaseDirection)
	}
}

func TestReorderWithBaseInvalid(t *testing.T) {
	if _, err := ReorderWithBase("a", Direction(42)); err != ErrInvalidBase {
		t.Fatalf("want ErrInvalidBase, got %v", err)
	}
}

func TestReorderWithBaseForces(t *testing.T) {
	// Pure Latin under a forced RTL base stays LTR (rule I2 drops L to the
	// nearest even level); the base direction governs paragraph placement.
	r, err := ReorderWithBase("abc", RTL)
	if err != nil {
		t.Fatalf("ReorderWithBase: unexpected error: %v", err)
	}
	if r.BaseDirection != RTL || r.BaseLevel != 1 {
		t.Fatalf("want RTL base, got %v/%d", r.BaseDirection, r.BaseLevel)
	}
	if got := concatRuns(r); got != "abc" {
		t.Fatalf("want visual %q, got %q", "abc", got)
	}
}

func TestIsRTLRune(t *testing.T) {
	cases := []struct {
		r    rune
		want bool
	}{
		{'A', false},
		{'1', false},
		{' ', false},
		{0x05D0, true},  // Hebrew alef
		{0x0627, true},  // Arabic alef
		{0x0660, true},  // Arabic-Indic zero
		{0x06F0, true},  // extended Arabic-Indic zero
		{0x0750, true},  // Arabic Supplement
		{0x08A0, true},  // Arabic Extended-A
		{0xFB50, true},  // Arabic Presentation Forms-A
		{0xFE70, true},  // Arabic Presentation Forms-B
		{0x0900, false}, // Devanagari
		{0xFEFF, true},  // BOM falls in Forms-B; acceptable block-wide bound
	}
	for _, c := range cases {
		if got := IsRTLRune(c.r); got != c.want {
			t.Errorf("IsRTLRune(%U): want %v, got %v", c.r, c.want, got)
		}
	}
}

func TestHasRTL(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Certificate", false},
		{"Certificate 2026", false},
		{"شهادة", true},
		{"Certificate شهادة 2026", true},
		{"", false},
	}
	for _, c := range cases {
		if got := HasRTL(c.in); got != c.want {
			t.Errorf("HasRTL(%q): want %v, got %v", c.in, c.want, got)
		}
	}
}

func TestMirrorRune(t *testing.T) {
	pairs := []struct{ a, b rune }{
		{'(', ')'}, {'[', ']'}, {'{', '}'}, {'<', '>'}, {'«', '»'},
	}
	for _, p := range pairs {
		if MirrorRune(p.a) != p.b {
			t.Errorf("MirrorRune(%q): want %q, got %q", p.a, p.b, MirrorRune(p.a))
		}
		if MirrorRune(p.b) != p.a {
			t.Errorf("MirrorRune(%q): want %q, got %q", p.b, p.a, MirrorRune(p.b))
		}
	}
	if MirrorRune('A') != 'A' {
		t.Errorf("MirrorRune('A'): want 'A', got %q", MirrorRune('A'))
	}
	if MirrorRune(0x0627) != 0x0627 {
		t.Errorf("MirrorRune(alef): want alef, got %q", MirrorRune(0x0627))
	}
}
