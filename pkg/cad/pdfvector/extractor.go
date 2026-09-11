package pdfvector

import (
	"strconv"
	"strings"

	"integin/pkg/cad/dxf"
)

// Matrix represents a 2D affine transformation matrix [a, b, c, d, e, f].
// [x', y', 1] = [x, y, 1] * [ a  b  0 ]
//                           [ c  d  0 ]
//                           [ e  f  1 ]
type Matrix struct {
	A, B, C, D, E, F float64
}

// IdentityMatrix returns the identity 2D transformation matrix.
func IdentityMatrix() Matrix {
	return Matrix{A: 1, B: 0, C: 0, D: 1, E: 0, F: 0}
}

// TransformPoint applies the affine matrix to a 2D/3D point.
func (m Matrix) TransformPoint(p dxf.Point3D) dxf.Point3D {
	return dxf.Point3D{
		X: m.A*p.X + m.C*p.Y + m.E,
		Y: m.B*p.X + m.D*p.Y + m.F,
		Z: p.Z,
	}
}

// Multiply concatenates two transformation matrices: m * o.
func (m Matrix) Multiply(o Matrix) Matrix {
	return Matrix{
		A: m.A*o.A + m.B*o.C,
		B: m.A*o.B + m.B*o.D,
		C: m.C*o.A + m.D*o.C,
		D: m.C*o.B + m.D*o.D,
		E: m.E*o.A + m.F*o.C + o.E,
		F: m.E*o.B + m.F*o.D + o.F,
	}
}

// GraphicsState represents the current PDF drawing context.
type GraphicsState struct {
	CTM            Matrix // Current Transformation Matrix
	CurrentPoint   dxf.Point3D
	StartSubpath   dxf.Point3D
	CurrentLayer   string
	ChordTolerance float64
}

// Extractor decodes raw PDF content streams into CAD entities.
type Extractor struct {
	stateStack []GraphicsState
	state      GraphicsState
	subpath    []dxf.Point3D
	isClosed   bool
	entities   []dxf.Entity
}

// NewExtractor initializes a PDF vector extractor.
func NewExtractor() *Extractor {
	initialState := GraphicsState{
		CTM:            IdentityMatrix(),
		CurrentLayer:   "PDF_GEOMETRY",
		ChordTolerance: 0.05,
	}
	return &Extractor{
		stateStack: make([]GraphicsState, 0),
		state:      initialState,
		subpath:    make([]dxf.Point3D, 0),
		entities:   make([]dxf.Entity, 0),
	}
}

// ParseStream processes an uncompressed PDF content stream string into CAD entities.
func (e *Extractor) ParseStream(stream string) ([]dxf.Entity, error) {
	tokens := tokenizePDF(stream)
	var operandStack []float64

	for _, tok := range tokens {
		if val, err := strconv.ParseFloat(tok, 64); err == nil {
			operandStack = append(operandStack, val)
			continue
		}

		// Operator encountered
		switch tok {
		case "q": // Save graphics state
			e.stateStack = append(e.stateStack, e.state)
		case "Q": // Restore graphics state
			if len(e.stateStack) > 0 {
				e.state = e.stateStack[len(e.stateStack)-1]
				e.stateStack = e.stateStack[:len(e.stateStack)-1]
			}
		case "cm": // Concat matrix (a b c d e f cm)
			if len(operandStack) >= 6 {
				n := len(operandStack)
				m := Matrix{
					A: operandStack[n-6],
					B: operandStack[n-5],
					C: operandStack[n-4],
					D: operandStack[n-3],
					E: operandStack[n-2],
					F: operandStack[n-1],
				}
				e.state.CTM = m.Multiply(e.state.CTM)
			}
		case "m": // Move to (x y m)
			if len(operandStack) >= 2 {
				e.flushSubpath()
				n := len(operandStack)
				pt := e.state.CTM.TransformPoint(dxf.Point3D{X: operandStack[n-2], Y: operandStack[n-1]})
				e.state.CurrentPoint = pt
				e.state.StartSubpath = pt
				e.subpath = []dxf.Point3D{pt}
			}
		case "l": // Line to (x y l)
			if len(operandStack) >= 2 {
				n := len(operandStack)
				pt := e.state.CTM.TransformPoint(dxf.Point3D{X: operandStack[n-2], Y: operandStack[n-1]})
				e.subpath = append(e.subpath, pt)
				e.state.CurrentPoint = pt
			}
		case "c": // Cubic Bézier (x1 y1 x2 y2 x3 y3 c)
			if len(operandStack) >= 6 {
				n := len(operandStack)
				p1 := e.state.CTM.TransformPoint(dxf.Point3D{X: operandStack[n-6], Y: operandStack[n-5]})
				p2 := e.state.CTM.TransformPoint(dxf.Point3D{X: operandStack[n-4], Y: operandStack[n-3]})
				p3 := e.state.CTM.TransformPoint(dxf.Point3D{X: operandStack[n-2], Y: operandStack[n-1]})

				curvePts := LinearizeCubicBezier(e.state.CurrentPoint, p1, p2, p3, e.state.ChordTolerance)
				if len(curvePts) > 1 {
					e.subpath = append(e.subpath, curvePts[1:]...)
				}
				e.state.CurrentPoint = p3
			}
		case "re": // Rectangle (x y w h re)
			if len(operandStack) >= 4 {
				e.flushSubpath()
				n := len(operandStack)
				x, y := operandStack[n-4], operandStack[n-3]
				w, h := operandStack[n-2], operandStack[n-1]

				p0 := e.state.CTM.TransformPoint(dxf.Point3D{X: x, Y: y})
				p1 := e.state.CTM.TransformPoint(dxf.Point3D{X: x + w, Y: y})
				p2 := e.state.CTM.TransformPoint(dxf.Point3D{X: x + w, Y: y + h})
				p3 := e.state.CTM.TransformPoint(dxf.Point3D{X: x, Y: y + h})

				rect := &dxf.Polyline{
					BaseEntity: dxf.BaseEntity{LayerName: e.state.CurrentLayer},
					IsClosed:   true,
					Vertices: []dxf.Vertex{
						{Point: p0},
						{Point: p1},
						{Point: p2},
						{Point: p3},
					},
				}
				e.entities = append(e.entities, rect)
			}
		case "h": // Close subpath
			e.isClosed = true
			if len(e.subpath) > 0 {
				e.state.CurrentPoint = e.state.StartSubpath
			}
		case "S", "s", "f", "F", "B", "b": // Stroke or Fill path
			if tok == "s" || tok == "b" {
				e.isClosed = true
			}
			e.flushSubpath()
		}

		operandStack = operandStack[:0]
	}

	e.flushSubpath()
	return e.entities, nil
}

func (e *Extractor) flushSubpath() {
	if len(e.subpath) < 2 {
		e.subpath = e.subpath[:0]
		e.isClosed = false
		return
	}

	if len(e.subpath) == 2 && !e.isClosed {
		// Emit simple Line
		e.entities = append(e.entities, &dxf.Line{
			BaseEntity: dxf.BaseEntity{LayerName: e.state.CurrentLayer},
			Start:      e.subpath[0],
			End:        e.subpath[1],
		})
	} else {
		// Emit Polyline
		vertices := make([]dxf.Vertex, len(e.subpath))
		for i, pt := range e.subpath {
			vertices[i] = dxf.Vertex{Point: pt}
		}
		e.entities = append(e.entities, &dxf.Polyline{
			BaseEntity: dxf.BaseEntity{LayerName: e.state.CurrentLayer},
			IsClosed:   e.isClosed,
			Vertices:   vertices,
		})
	}

	e.subpath = e.subpath[:0]
	e.isClosed = false
}

func tokenizePDF(s string) []string {
	var tokens []string
	var cur strings.Builder

	inComment := false
	for i := 0; i < len(s); i++ {
		b := s[i]
		if inComment {
			if b == '\n' || b == '\r' {
				inComment = false
			}
			continue
		}
		if b == '%' {
			inComment = true
			continue
		}

		if isPDFWhitespace(b) || isPDFDelimiter(b) {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			continue
		}

		cur.WriteByte(b)
	}

	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func isPDFWhitespace(b byte) bool {
	return b == 0 || b == 9 || b == 10 || b == 12 || b == 13 || b == 32
}

func isPDFDelimiter(b byte) bool {
	return b == '(' || b == ')' || b == '<' || b == '>' || b == '[' || b == ']' || b == '{' || b == '}' || b == '/' || b == '%'
}
