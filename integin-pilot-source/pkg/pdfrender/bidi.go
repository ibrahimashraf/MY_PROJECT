// Package pdfrender provides text shaping support for certificate PDF
// generation, starting with a pragmatic Unicode bidirectional (BiDi)
// reordering engine for mixed Arabic/Latin (RTL/LTR) content.
//
// bidi.go implements a subset of UAX #9 (Unicode Bidirectional Algorithm)
// sufficient for the bilingual Saudi (SASO/ZATCA) and UAE (ADNOC) certificate
// cells: explicit embedding controls (LRE/RLE/PDF, LRO/RLO), isolates
// (LRI/RLI/FSI/PDI), RTL run detection across the Arabic and Hebrew blocks
// including Arabic Presentation Forms, weak number handling (European and
// Arabic-Indic digits), neutral resolution between strong runs, and bracket
// mirroring inside RTL runs. It is deterministic, single-threaded, pure Go,
// with no I/O.
//
// Output is a list of directional runs in visual order plus logical<->visual
// index maps that a PDF text placement engine consumes directly.
//
// Deliberate simplifications (ponytail: single paragraph per call, no
// paragraph-separator splitting; European digits adjacent to Arabic letters
// are NOT promoted to Arabic-Indic numbers per weak rule W2 since the
// requirement "numbers stay LTR inside RTL" takes precedence; deprecated
// format aliases U+206A..U+206F are treated as neutrals; a closed isolate
// must be terminated by PDI, a stray PDF inside one fails closed).
package pdfrender

import (
	"errors"
	"unicode"
)

// Sentinel errors. BiDi processing fails closed: malformed or ambiguous
// directional input is an error, never silently "best-effort" text.
var (
	ErrInvalidUTF8    = errors.New("pdfrender: invalid UTF-8 input")
	ErrUnmatchedPDF   = errors.New("pdfrender: unmatched PDF (pop directional formatting)")
	ErrUnmatchedPDI   = errors.New("pdfrender: unmatched PDI (pop directional isolate)")
	ErrNestingTooDeep = errors.New("pdfrender: directional embedding nesting exceeds 125 levels")
	ErrInvalidBase    = errors.New("pdfrender: invalid base direction")
)

// Direction is a resolved text run direction.
type Direction int

const (
	// LTR is left-to-right (even embedding level).
	LTR Direction = iota
	// RTL is right-to-left (odd embedding level).
	RTL
)

// Run is a contiguous span of the logical input rendered as a single
// direction. Runs are returned in visual order.
//
// LogicalStart and Length are rune indices into the input string, not byte
// offsets. Text holds the visual-order glyphs for the span: reversed for RTL
// runs and with mirrored bracket glyphs applied (L4).
type Run struct {
	LogicalStart int
	Length       int
	Direction    Direction
	Text         string
}

// ReorderResult is the result of resolving one logical string into visual
// order.
//
// LogicalToVisual maps each logical rune index to its visual slot number;
// explicit bidirectional control runes (LRE/RLE/PDF/LRO/RLO/LRI/RLI/FSI/PDI)
// are removed from the output and map to -1. VisualToLogical is the inverse,
// indexed from 0 and covering only displayed runes.
type ReorderResult struct {
	Runs            []Run
	LogicalToVisual []int
	VisualToLogical []int
	BaseDirection   Direction
	BaseLevel       int
}

// Control characters carrying explicit directional formatting or isolation.
const (
	runLRE = '\u202A'
	runRLE = '\u202B'
	runPDF = '\u202C'
	runLRO = '\u202D'
	runRLO = '\u202E'
	runLRI = '\u2066'
	runRLI = '\u2067'
	runFSI = '\u2068'
	runPDI = '\u2069'
)

// Maximum explicit embedding + isolate nesting depth (UAX #9 rule X9).
const maxEmbeddingDepth = 125

const overrideNone = -1

// Run-time directional classes. Strong R subsumes AL (Arabic letters are
// typed directly as R per rule W3).
type class uint8

const (
	clsL class = iota
	clsR
	clsEN // European number
	clsAN // Arabic-Indic number
	clsNSM
	clsNeutral
	// Explicit directional controls:
	clsLRE
	clsRLE
	clsPDF
	clsLRO
	clsRLO
	clsLRI
	clsRLI
	clsFSI
	clsPDI
)

// IsRTLRune reports whether r belongs to a right-to-left block: Hebrew,
// Arabic, Arabic Supplement, Arabic Extended-A, or the Arabic Presentation
// Forms (A and B).
func IsRTLRune(r rune) bool {
	return (r >= 0x0590 && r <= 0x05FF) || // Hebrew
		(r >= 0x0600 && r <= 0x06FF) || // Arabic (incl. Arabic-Indic digits)
		(r >= 0x0750 && r <= 0x077F) || // Arabic Supplement
		(r >= 0x08A0 && r <= 0x08FF) || // Arabic Extended-A
		(r >= 0xFB50 && r <= 0xFDFF) || // Arabic Presentation Forms-A
		(r >= 0xFE70 && r <= 0xFEFF) // Arabic Presentation Forms-B
}

// HasRTL reports whether s contains any right-to-left code point.
func HasRTL(s string) bool {
	for _, r := range s {
		if IsRTLRune(r) {
			return true
		}
	}
	return false
}

// bidiMirrorPairs holds rule L4 bracket pairs applied inside RTL runs.
var bidiMirrorPairs = map[rune]rune{
	'(': ')', ')': '(',
	'[': ']', ']': '[',
	'{': '}', '}': '{',
	'<': '>', '>': '<',
	'«': '»', '»': '«',
}

// MirrorRune returns the mirrored glyph for a bracket code point, or r itself
// when r has no mirroring pair.
func MirrorRune(r rune) rune {
	if m, ok := bidiMirrorPairs[r]; ok {
		return m
	}
	return r
}

// Reorder resolves s into visual order using the first strong rune to pick
// the paragraph base direction (UAX #9 rules P2/P3).
func Reorder(s string) (ReorderResult, error) {
	return reorder(s, -1, LTR)
}

// ReorderWithBase resolves s into visual order with an explicit paragraph
// base direction.
func ReorderWithBase(s string, base Direction) (ReorderResult, error) {
	if base != LTR && base != RTL {
		return ReorderResult{}, ErrInvalidBase
	}
	return reorder(s, int(base), base)
}

func reorder(s string, baseOverride int, base Direction) (ReorderResult, error) {
	runes, kinds, ws, err := decode(s)
	if err != nil {
		return ReorderResult{}, err
	}
	n := len(runes)
	if n == 0 {
		r := ReorderResult{
			LogicalToVisual: []int{},
			VisualToLogical: []int{},
			BaseDirection:   base,
		}
		return r, nil
	}

	applyNSM(kinds) // W1: combining marks take the previous character's class

	if baseOverride < 0 {
		base = detectBaseDirection(kinds)
	}
	paraLevel := 0
	if base == RTL {
		paraLevel = 1
	}

	levels := make([]int, n)
	overrides := make([]int, n)
	for i := range overrides {
		overrides[i] = overrideNone
	}
	if err := resolveLevels(runes, kinds, levels, overrides, paraLevel, base); err != nil {
		return ReorderResult{}, err
	}

	resolveNeutrals(kinds, levels) // N1/N2: neutrals follow surrounding strongs

	finalLevels := make([]int, n)
	for i := range runes {
		switch {
		case overrides[i] != overrideNone: // LRO/RLO force the display direction
			if overrides[i] == int(RTL) {
				finalLevels[i] = levels[i] | 1
			} else {
				finalLevels[i] = levels[i] &^ 1
			}
		case kinds[i] == clsR || kinds[i] == clsAN: // I1: R/AL/AN -> odd
			finalLevels[i] = levels[i] | 1
		default: // I2: L/EN -> even
			finalLevels[i] = levels[i] &^ 1
		}
	}
	// L1: trailing whitespace at paragraph end hangs at the paragraph level.
	for i := n - 1; i >= 0; i-- {
		if kind := kinds[i]; kind == clsLRE || kind == clsRLE || kind == clsPDF ||
			kind == clsLRO || kind == clsRLO || kind == clsLRI ||
			kind == clsRLI || kind == clsFSI || kind == clsPDI {
			continue
		}
		if !ws[i] {
			break
		}
		finalLevels[i] = paraLevel
	}

	display := make([]int, 0, n)
	for i := range runes {
		if !isControlClass(kinds[i]) {
			display = append(display, i)
		}
	}
	reorderLevels(display, finalLevels)

	r := ReorderResult{
		LogicalToVisual: make([]int, n),
		VisualToLogical: make([]int, len(display)),
		BaseDirection:   base,
		BaseLevel:       paraLevel,
	}
	for i := range r.LogicalToVisual {
		r.LogicalToVisual[i] = -1
	}
	for v, log := range display {
		r.LogicalToVisual[log] = v
		r.VisualToLogical[v] = log
	}

	r.Runs = buildRuns(runes, kinds, display, finalLevels)
	return r, nil
}

// decode validates the UTF-8 input and classifies every rune. Malformed
// bytes return ErrInvalidUTF8 instead of leaking U+FFFD replacement chars
// into the output.
func decode(s string) ([]rune, []class, []bool, error) {
	runes := make([]rune, 0, len(s))
	kinds := make([]class, 0, len(s))
	ws := make([]bool, 0, len(s))
	for i := 0; i < len(s); {
		r, size := rune(0), 0
		b := s[i]
		switch {
		case b < 0x80:
			r, size = rune(b), 1
		case b < 0xC0:
			return nil, nil, nil, ErrInvalidUTF8
		case b < 0xE0:
			r, size = rune(b&0x1F)<<6, 2
		case b < 0xF0:
			r, size = rune(b&0x0F)<<12, 3
		case b < 0xF8:
			r, size = rune(b&0x07)<<18, 4
		default:
			return nil, nil, nil, ErrInvalidUTF8
		}
		if i+size > len(s) {
			return nil, nil, nil, ErrInvalidUTF8
		}
		for j := 1; j < size; j++ {
			c := s[i+j]
			if c < 0x80 || c >= 0xC0 {
				return nil, nil, nil, ErrInvalidUTF8
			}
			r |= rune(c&0x3F) << uint(6*(size-j-1))
		}
		if r > 0x10FFFF || (r >= 0xD800 && r <= 0xDFFF) {
			return nil, nil, nil, ErrInvalidUTF8
		}
		switch size {
		case 2:
			if r < 0x80 {
				return nil, nil, nil, ErrInvalidUTF8 // overlong
			}
		case 3:
			if r < 0x800 {
				return nil, nil, nil, ErrInvalidUTF8
			}
		case 4:
			if r < 0x10000 {
				return nil, nil, nil, ErrInvalidUTF8
			}
		}
		runes = append(runes, r)
		kinds = append(kinds, classify(r))
		ws = append(ws, isWhitespace(r))
		i += size
	}
	return runes, kinds, ws, nil
}

func classify(r rune) class {
	switch r {
	case runLRE:
		return clsLRE
	case runRLE:
		return clsRLE
	case runPDF:
		return clsPDF
	case runLRO:
		return clsLRO
	case runRLO:
		return clsRLO
	case runLRI:
		return clsLRI
	case runRLI:
		return clsRLI
	case runFSI:
		return clsFSI
	case runPDI:
		return clsPDI
	case '\u200E': // LRM: strong L
		return clsL
	case '\u200F', '\u061C': // RLM / ALM: strong R
		return clsR
	case '\uFEFF': // BOM / ZWNBSP: neutral
		return clsNeutral
	}
	switch {
	case r >= '0' && r <= '9':
		return clsEN
	case (r >= 0x0660 && r <= 0x0669) || (r >= 0x06F0 && r <= 0x06F9):
		return clsAN
	case IsRTLRune(r):
		return clsR
	case unicode.Is(unicode.Mn, r):
		return clsNSM
	case unicode.IsLetter(r):
		return clsL
	default:
		return clsNeutral
	}
}

func isWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', '\u00A0', '\u1680':
		return true
	}
	return (r >= '\u2000' && r <= '\u200A') ||
		r == '\u202F' || r == '\u205F' || r == '\u3000'
}

func isControlClass(k class) bool {
	return k >= clsLRE && k <= clsPDI
}

// applyNSM implements rule W1 for the pragmatic subset: each nonspacing mark
// inherits the directional class of the preceding displayed rune.
func applyNSM(kinds []class) {
	for i := range kinds {
		if kinds[i] != clsNSM {
			continue
		}
		kinds[i] = clsNeutral
		for j := i - 1; j >= 0; j-- {
			if isControlClass(kinds[j]) {
				continue
			}
			kinds[i] = kinds[j]
			break
		}
	}
}

type embLevel struct {
	level    int
	override int // overrideNone, LTR, or RTL
}

// resolveLevels walks the explicit formatting controls, computing the
// embedding level and active override for every rune. Unmatched PDF/PDI and
// over-deep nesting fail closed.
func resolveLevels(runes []rune, kinds []class, levels, overrides []int, paraLevel int, base Direction) error {
	stack := make([]embLevel, 0, 8)
	isolateBarriers := make([]int, 0, 4)
	cur := paraLevel

	for i, k := range kinds {
		switch k {
		case clsLRE, clsRLE, clsLRO, clsRLO:
			if len(stack) >= maxEmbeddingDepth {
				return ErrNestingTooDeep
			}
			levels[i] = cur
			wantEven := k == clsLRE || k == clsLRO
			next := cur + 1
			if (next%2 == 0) != wantEven {
				next++
			}
			ov := overrideNone
			if k == clsLRO {
				ov = int(LTR)
			} else if k == clsRLO {
				ov = int(RTL)
			}
			stack = append(stack, embLevel{level: next, override: ov})
			cur = next

		case clsPDF:
			if len(stack) == 0 {
				return ErrUnmatchedPDF
			}
			if len(isolateBarriers) > 0 && len(stack) <= isolateBarriers[len(isolateBarriers)-1]+1 {
				// Popping would remove the isolate's own embedding: the
				// isolate must instead be closed with PDI.
				return ErrUnmatchedPDF
			}
			levels[i] = cur
			stack = stack[:len(stack)-1]
			if len(stack) > 0 {
				cur = stack[len(stack)-1].level
			} else {
				cur = paraLevel
			}

		case clsLRI, clsRLI, clsFSI:
			if len(stack) >= maxEmbeddingDepth {
				return ErrNestingTooDeep
			}
			levels[i] = cur
			d := LTR
			switch k {
			case clsRLI:
				d = RTL
			case clsFSI:
				d = fsiDirection(runes, kinds, i+1, base)
			}
			wantEven := d == LTR
			next := cur + 1
			if (next%2 == 0) != wantEven {
				next++
			}
			isolateBarriers = append(isolateBarriers, len(stack))
			stack = append(stack, embLevel{level: next, override: overrideNone})
			cur = next

		case clsPDI:
			if len(isolateBarriers) == 0 {
				return ErrUnmatchedPDI
			}
			levels[i] = cur
			barrier := isolateBarriers[len(isolateBarriers)-1]
			isolateBarriers = isolateBarriers[:len(isolateBarriers)-1]
			for len(stack) > barrier {
				stack = stack[:len(stack)-1]
			}
			if len(stack) > 0 {
				cur = stack[len(stack)-1].level
			} else {
				cur = paraLevel
			}

		default:
			levels[i] = cur
			if len(stack) > 0 {
				overrides[i] = stack[len(stack)-1].override
			}
		}
	}
	return nil
}

// fsiDirection resolves the direction of an FSI isolate from the first strong
// rune inside it, falling back to the paragraph direction when there is none.
func fsiDirection(runes []rune, kinds []class, start int, base Direction) Direction {
	depth := 0
	for i := start; i < len(runes); i++ {
		switch kinds[i] {
		case clsLRI, clsRLI, clsFSI:
			depth++
		case clsPDI:
			depth--
			if depth < 0 {
				return base
			}
		case clsL:
			return LTR
		case clsR:
			return RTL
		}
	}
	return base
}

func detectBaseDirection(kinds []class) Direction {
	for _, k := range kinds {
		switch k {
		case clsL:
			return LTR
		case clsR:
			return RTL
		}
	}
	return LTR
}

// contextStrong scans outward from an index for the nearest strong context.
// Rule N1 treats EN and AN as R-direction context; L is L-direction.
func contextStrong(kinds []class, start, step int) (class, bool) {
	for i := start; i >= 0 && i < len(kinds); i += step {
		switch kinds[i] {
		case clsL:
			return clsL, true
		case clsR, clsEN, clsAN:
			return clsR, true
		}
	}
	return clsNeutral, false
}

// resolveNeutrals implements rules N1/N2: a maximal run of neutral characters
// takes the direction shared by both contexts, or the embedding level's
// direction when the contexts disagree.
func resolveNeutrals(kinds []class, levels []int) {
	for i := 0; i < len(kinds); i++ {
		if kinds[i] != clsNeutral {
			continue
		}
		j := i
		for j < len(kinds) && kinds[j] == clsNeutral {
			j++
		}
		left, hasLeft := contextStrong(kinds, i-1, -1)
		right, hasRight := contextStrong(kinds, j, 1)
		target := clsR
		if hasLeft && hasRight && left == right {
			target = left
		} else if levels[i]%2 == 0 {
			target = clsL
		}
		for k := i; k < j; k++ {
			kinds[k] = target
		}
		i = j - 1
	}
}

// reorderLevels is UAX #9 rule L2: reverse each maximal run whose level is at
// or above the current threshold, walking thresholds from highest to lowest.
func reorderLevels(display []int, levels []int) {
	max := 0
	for _, log := range display {
		if levels[log] > max {
			max = levels[log]
		}
	}
	for lev := max; lev >= 1; lev-- {
		i := 0
		for i < len(display) {
			if levels[display[i]] >= lev {
				j := i + 1
				for j < len(display) && levels[display[j]] >= lev {
					j++
				}
				reverse(display[i:j])
				i = j
			} else {
				i++
			}
		}
	}
}

func reverse(a []int) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}

// buildRuns groups the visual-order indices into maximal directional runs,
// emitting visual glyphs with brackets mirrored inside RTL runs (rule L4).
func buildRuns(runes []rune, kinds []class, display []int, finalLevels []int) []Run {
	var runs []Run
	for v := 0; v < len(display); {
		first := display[v]
		dir := LTR
		if finalLevels[first]%2 == 1 {
			dir = RTL
		}
		j := v + 1
		for j < len(display) {
			if (finalLevels[display[j]]%2 == 1) != (dir == RTL) {
				break
			}
			j++
		}
		start, end := first, display[v]
		if display[j-1] < start {
			start = display[j-1]
		}
		if display[j-1] > end {
			end = display[j-1]
		}
		var sb []rune
		for _, log := range display[v:j] {
			g := runes[log]
			if dir == RTL {
				g = MirrorRune(g)
			}
			sb = append(sb, g)
		}
		runs = append(runs, Run{
			LogicalStart: start,
			Length:       end - start + 1,
			Direction:    dir,
			Text:         string(sb),
		})
		v = j
	}
	return runs
}
