package dxf

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Reader parses AutoCAD ASCII DXF streams into Drawing objects.
type Reader struct {
	scanner *bufio.Scanner
	lineNum int
	peeked  *pair
}

// NewReader initializes a DXF Reader from an io.Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
	}
}

type pair struct {
	code int
	val  string
}

func (r *Reader) pushback(p *pair) {
	r.peeked = p
}

func (r *Reader) nextPair() (*pair, error) {
	if r.peeked != nil {
		p := r.peeked
		r.peeked = nil
		return p, nil
	}

	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	r.lineNum++
	codeStr := strings.TrimSpace(r.scanner.Text())
	if codeStr == "" {
		return r.nextPair()
	}

	code, err := strconv.Atoi(codeStr)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid DXF group code %q: %w", r.lineNum, codeStr, err)
	}

	if !r.scanner.Scan() {
		return nil, fmt.Errorf("line %d: unexpected EOF reading DXF value for code %d", r.lineNum, code)
	}
	r.lineNum++
	val := strings.TrimRight(r.scanner.Text(), "\r\n")

	return &pair{code: code, val: val}, nil
}

// Read parses the DXF stream and returns the populated Drawing.
func (r *Reader) Read() (*Drawing, error) {
	d := NewDrawing()

	for {
		p, err := r.nextPair()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if p.code == 0 && p.val == "SECTION" {
			sec, err := r.nextPair()
			if err != nil {
				return nil, err
			}
			if sec.code == 2 {
				switch sec.val {
				case "TABLES":
					if err := r.parseTables(d); err != nil {
						return nil, err
					}
				case "BLOCKS":
					if err := r.parseBlocks(d); err != nil {
						return nil, err
					}
				case "ENTITIES":
					if err := r.parseEntities(d, false); err != nil {
						return nil, err
					}
				}
			}
		} else if p.code == 0 && p.val == "EOF" {
			break
		}
	}

	return d, nil
}

func (r *Reader) parseTables(d *Drawing) error {
	for {
		p, err := r.nextPair()
		if err != nil {
			return err
		}
		if p.code == 0 && p.val == "ENDSEC" {
			break
		}
		if p.code == 0 && p.val == "TABLE" {
			tbl, err := r.nextPair()
			if err != nil {
				return err
			}
			if tbl.code == 2 && tbl.val == "LAYER" {
				if err := r.parseLayerTable(d); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *Reader) parseLayerTable(d *Drawing) error {
	var currentLayer *Layer

	for {
		p, err := r.nextPair()
		if err != nil {
			return err
		}
		if p.code == 0 {
			if currentLayer != nil && currentLayer.Name != "" {
				d.Layers[currentLayer.Name] = currentLayer
				currentLayer = nil
			}
			if p.val == "ENDTAB" {
				break
			}
			if p.val == "LAYER" {
				currentLayer = &Layer{IsVisible: true, LineType: "CONTINUOUS", Color: 7}
			}
		} else if currentLayer != nil {
			switch p.code {
			case 2:
				currentLayer.Name = p.val
			case 62:
				color, _ := strconv.Atoi(p.val)
				if color < 0 {
					currentLayer.IsVisible = false
					color = -color
				}
				currentLayer.Color = int16(color)
			case 6:
				currentLayer.LineType = p.val
			case 70:
				flags, _ := strconv.Atoi(p.val)
				if flags&1 != 0 {
					currentLayer.IsVisible = false
				}
				if flags&4 != 0 {
					currentLayer.IsLocked = true
				}
			}
		}
	}
	return nil
}

func (r *Reader) parseBlocks(d *Drawing) error {
	var currentBlock *Block

	for {
		p, err := r.nextPair()
		if err != nil {
			return err
		}
		if p.code == 0 {
			if p.val == "ENDSEC" {
				break
			}
			if p.val == "BLOCK" {
				currentBlock = &Block{Entities: make([]Entity, 0)}
			} else if p.val == "ENDBLK" {
				if currentBlock != nil && currentBlock.Name != "" {
					d.AddBlock(currentBlock)
					currentBlock = nil
				}
			} else if currentBlock != nil {
				// Entity inside block
				entity, err := r.parseEntityByType(p.val)
				if err != nil {
					return err
				}
				if entity != nil {
					currentBlock.Entities = append(currentBlock.Entities, entity)
				}
			}
		} else if currentBlock != nil {
			switch p.code {
			case 2:
				currentBlock.Name = p.val
			case 8:
				currentBlock.LayerName = p.val
			case 10:
				currentBlock.BasePoint.X, _ = strconv.ParseFloat(p.val, 64)
			case 20:
				currentBlock.BasePoint.Y, _ = strconv.ParseFloat(p.val, 64)
			case 30:
				currentBlock.BasePoint.Z, _ = strconv.ParseFloat(p.val, 64)
			}
		}
	}
	return nil
}

func (r *Reader) parseEntities(d *Drawing, stopAtEndBlk bool) error {
	for {
		p, err := r.nextPair()
		if err != nil {
			return err
		}
		if p.code == 0 {
			if p.val == "ENDSEC" || (stopAtEndBlk && p.val == "ENDBLK") {
				break
			}
			entity, err := r.parseEntityByType(p.val)
			if err != nil {
				return err
			}
			if entity != nil {
				d.AddEntity(entity)
			}
		}
	}
	return nil
}

func (r *Reader) parseEntityByType(eType string) (Entity, error) {
	switch eType {
	case "LINE":
		return r.parseLine()
	case "LWPOLYLINE":
		return r.parsePolyline()
	case "CIRCLE":
		return r.parseCircle()
	case "ARC":
		return r.parseArc()
	case "TEXT":
		return r.parseText()
	case "INSERT":
		return r.parseInsert()
	default:
		// Skip unknown entity until next group code 0
		for {
			p, err := r.nextPair()
			if err != nil {
				return nil, err
			}
			if p.code == 0 {
				r.pushback(p)
				return nil, nil
			}
		}
	}
}

func (r *Reader) parseLine() (Entity, error) {
	l := &Line{}
	for {
		p, err := r.nextPair()
		if err != nil {
			return nil, err
		}
		if p.code == 0 {
			r.pushback(p)
			return l, nil
		}
		switch p.code {
		case 8:
			l.LayerName = p.val
		case 62:
			c, _ := strconv.Atoi(p.val)
			l.Color = int16(c)
		case 10:
			l.Start.X, _ = strconv.ParseFloat(p.val, 64)
		case 20:
			l.Start.Y, _ = strconv.ParseFloat(p.val, 64)
		case 30:
			l.Start.Z, _ = strconv.ParseFloat(p.val, 64)
		case 11:
			l.End.X, _ = strconv.ParseFloat(p.val, 64)
		case 21:
			l.End.Y, _ = strconv.ParseFloat(p.val, 64)
		case 31:
			l.End.Z, _ = strconv.ParseFloat(p.val, 64)
		}
	}
}

func (r *Reader) parsePolyline() (Entity, error) {
	pLine := &Polyline{Vertices: make([]Vertex, 0)}
	var curX, curY float64
	hasCur := false

	for {
		p, err := r.nextPair()
		if err != nil {
			return nil, err
		}
		if p.code == 0 {
			if hasCur {
				pLine.Vertices = append(pLine.Vertices, Vertex{Point: Point3D{X: curX, Y: curY}})
			}
			r.pushback(p)
			return pLine, nil
		}
		switch p.code {
		case 8:
			pLine.LayerName = p.val
		case 62:
			c, _ := strconv.Atoi(p.val)
			pLine.Color = int16(c)
		case 70:
			flags, _ := strconv.Atoi(p.val)
			pLine.IsClosed = (flags & 1) != 0
		case 10:
			if hasCur {
				pLine.Vertices = append(pLine.Vertices, Vertex{Point: Point3D{X: curX, Y: curY}})
			}
			curX, _ = strconv.ParseFloat(p.val, 64)
			curY = 0
			hasCur = true
		case 20:
			curY, _ = strconv.ParseFloat(p.val, 64)
		case 42:
			if len(pLine.Vertices) > 0 {
				bulge, _ := strconv.ParseFloat(p.val, 64)
				pLine.Vertices[len(pLine.Vertices)-1].Bulge = bulge
			}
		}
	}
}

func (r *Reader) parseCircle() (Entity, error) {
	c := &Circle{}
	for {
		p, err := r.nextPair()
		if err != nil {
			return nil, err
		}
		if p.code == 0 {
			r.pushback(p)
			return c, nil
		}
		switch p.code {
		case 8:
			c.LayerName = p.val
		case 62:
			col, _ := strconv.Atoi(p.val)
			c.Color = int16(col)
		case 10:
			c.Center.X, _ = strconv.ParseFloat(p.val, 64)
		case 20:
			c.Center.Y, _ = strconv.ParseFloat(p.val, 64)
		case 30:
			c.Center.Z, _ = strconv.ParseFloat(p.val, 64)
		case 40:
			c.Radius, _ = strconv.ParseFloat(p.val, 64)
		}
	}
}

func (r *Reader) parseArc() (Entity, error) {
	a := &Arc{}
	for {
		p, err := r.nextPair()
		if err != nil {
			return nil, err
		}
		if p.code == 0 {
			r.pushback(p)
			return a, nil
		}
		switch p.code {
		case 8:
			a.LayerName = p.val
		case 62:
			col, _ := strconv.Atoi(p.val)
			a.Color = int16(col)
		case 10:
			a.Center.X, _ = strconv.ParseFloat(p.val, 64)
		case 20:
			a.Center.Y, _ = strconv.ParseFloat(p.val, 64)
		case 30:
			a.Center.Z, _ = strconv.ParseFloat(p.val, 64)
		case 40:
			a.Radius, _ = strconv.ParseFloat(p.val, 64)
		case 50:
			a.StartAngle, _ = strconv.ParseFloat(p.val, 64)
		case 51:
			a.EndAngle, _ = strconv.ParseFloat(p.val, 64)
		}
	}
}

func (r *Reader) parseText() (Entity, error) {
	t := &Text{}
	for {
		p, err := r.nextPair()
		if err != nil {
			return nil, err
		}
		if p.code == 0 {
			r.pushback(p)
			return t, nil
		}
		switch p.code {
		case 8:
			t.LayerName = p.val
		case 62:
			col, _ := strconv.Atoi(p.val)
			t.Color = int16(col)
		case 10:
			t.Insertion.X, _ = strconv.ParseFloat(p.val, 64)
		case 20:
			t.Insertion.Y, _ = strconv.ParseFloat(p.val, 64)
		case 30:
			t.Insertion.Z, _ = strconv.ParseFloat(p.val, 64)
		case 40:
			t.Height, _ = strconv.ParseFloat(p.val, 64)
		case 1:
			t.Value = p.val
		case 50:
			t.Rotation, _ = strconv.ParseFloat(p.val, 64)
		}
	}
}

func (r *Reader) parseInsert() (Entity, error) {
	i := &Insert{ScaleX: 1.0, ScaleY: 1.0, ScaleZ: 1.0}
	for {
		p, err := r.nextPair()
		if err != nil {
			return nil, err
		}
		if p.code == 0 {
			r.pushback(p)
			return i, nil
		}
		switch p.code {
		case 8:
			i.LayerName = p.val
		case 2:
			i.BlockName = p.val
		case 10:
			i.Insertion.X, _ = strconv.ParseFloat(p.val, 64)
		case 20:
			i.Insertion.Y, _ = strconv.ParseFloat(p.val, 64)
		case 30:
			i.Insertion.Z, _ = strconv.ParseFloat(p.val, 64)
		case 41:
			i.ScaleX, _ = strconv.ParseFloat(p.val, 64)
		case 42:
			i.ScaleY, _ = strconv.ParseFloat(p.val, 64)
		case 43:
			i.ScaleZ, _ = strconv.ParseFloat(p.val, 64)
		case 50:
			i.RotationDeg, _ = strconv.ParseFloat(p.val, 64)
		}
	}
}
