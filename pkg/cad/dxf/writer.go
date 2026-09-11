package dxf

import (
	"fmt"
	"io"
	"sort"
)

// Writer serializes Drawing objects into standard AutoCAD ASCII DXF format.
type Writer struct {
	w io.Writer
}

// NewWriter creates a new DXF Writer targeting the given io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) writeCode(code int, val any) error {
	switch v := val.(type) {
	case float64:
		_, err := fmt.Fprintf(w.w, "%3d\n%.6f\n", code, v)
		return err
	case int, int16, int32, int64:
		_, err := fmt.Fprintf(w.w, "%3d\n%d\n", code, v)
		return err
	case string:
		_, err := fmt.Fprintf(w.w, "%3d\n%s\n", code, v)
		return err
	default:
		_, err := fmt.Fprintf(w.w, "%3d\n%v\n", code, v)
		return err
	}
}

// Write serializes the drawing to ASCII DXF.
func (w *Writer) Write(d *Drawing) error {
	// 1. HEADER SECTION
	if err := w.writeHeader(); err != nil {
		return err
	}

	// 2. TABLES SECTION (Layers)
	if err := w.writeTables(d); err != nil {
		return err
	}

	// 3. BLOCKS SECTION
	if err := w.writeBlocks(d); err != nil {
		return err
	}

	// 4. ENTITIES SECTION
	if err := w.writeEntities(d.Entities); err != nil {
		return err
	}

	// 5. EOF
	return w.writeCode(0, "EOF")
}

func (w *Writer) writeHeader() error {
	if err := w.writeCode(0, "SECTION"); err != nil {
		return err
	}
	if err := w.writeCode(2, "HEADER"); err != nil {
		return err
	}
	// AutoCAD Version AC1015 (AutoCAD 2000 standard compatibility)
	if err := w.writeCode(9, "$ACADVER"); err != nil {
		return err
	}
	if err := w.writeCode(1, "AC1015"); err != nil {
		return err
	}
	// ADV-05: Statutory immutable audit comment & non-certified disclaimer
	if err := w.writeCode(999, "INTEGIN_AUDIT_STAMP: NOT CERTIFIED FOR RIGGING/LIFTING WITHOUT LICENSED PE STAMP (ASME B30.5 / OSHA 1926)"); err != nil {
		return err
	}
	return w.writeCode(0, "ENDSEC")
}

func (w *Writer) writeTables(d *Drawing) error {
	if err := w.writeCode(0, "SECTION"); err != nil {
		return err
	}
	if err := w.writeCode(2, "TABLES"); err != nil {
		return err
	}

	// Layer Table
	if err := w.writeCode(0, "TABLE"); err != nil {
		return err
	}
	if err := w.writeCode(2, "LAYER"); err != nil {
		return err
	}
	if err := w.writeCode(70, len(d.Layers)); err != nil {
		return err
	}

	// Sort layer names for deterministic output
	names := make([]string, 0, len(d.Layers))
	for name := range d.Layers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		layer := d.Layers[name]
		if err := w.writeCode(0, "LAYER"); err != nil {
			return err
		}
		if err := w.writeCode(100, "AcDbSymbolTableRecord"); err != nil {
			return err
		}
		if err := w.writeCode(100, "AcDbLayerTableRecord"); err != nil {
			return err
		}
		if err := w.writeCode(2, layer.Name); err != nil {
			return err
		}
		flags := 0
		if !layer.IsVisible {
			flags |= 1
		}
		if layer.IsLocked {
			flags |= 4
		}
		if err := w.writeCode(70, flags); err != nil {
			return err
		}
		color := layer.Color
		if color == 0 {
			color = 7 // Default White/Black
		}
		if err := w.writeCode(62, color); err != nil {
			return err
		}
		lineType := layer.LineType
		if lineType == "" {
			lineType = "CONTINUOUS"
		}
		if err := w.writeCode(6, lineType); err != nil {
			return err
		}
	}

	if err := w.writeCode(0, "ENDTAB"); err != nil {
		return err
	}
	return w.writeCode(0, "ENDSEC")
}

func (w *Writer) writeBlocks(d *Drawing) error {
	if err := w.writeCode(0, "SECTION"); err != nil {
		return err
	}
	if err := w.writeCode(2, "BLOCKS"); err != nil {
		return err
	}

	blockNames := make([]string, 0, len(d.Blocks))
	for name := range d.Blocks {
		blockNames = append(blockNames, name)
	}
	sort.Strings(blockNames)

	for _, name := range blockNames {
		b := d.Blocks[name]
		if err := w.writeCode(0, "BLOCK"); err != nil {
			return err
		}
		if err := w.writeCode(100, "AcDbEntity"); err != nil {
			return err
		}
		layer := b.LayerName
		if layer == "" {
			layer = "0"
		}
		if err := w.writeCode(8, layer); err != nil {
			return err
		}
		if err := w.writeCode(100, "AcDbBlockBegin"); err != nil {
			return err
		}
		if err := w.writeCode(2, b.Name); err != nil {
			return err
		}
		if err := w.writeCode(70, 0); err != nil {
			return err
		}
		if err := w.writeCode(10, b.BasePoint.X); err != nil {
			return err
		}
		if err := w.writeCode(20, b.BasePoint.Y); err != nil {
			return err
		}
		if err := w.writeCode(30, b.BasePoint.Z); err != nil {
			return err
		}
		if err := w.writeCode(3, b.Name); err != nil {
			return err
		}
		if err := w.writeCode(1, ""); err != nil {
			return err
		}

		for _, entity := range b.Entities {
			if err := w.writeSingleEntity(entity); err != nil {
				return err
			}
		}

		if err := w.writeCode(0, "ENDBLK"); err != nil {
			return err
		}
		if err := w.writeCode(100, "AcDbBlockEnd"); err != nil {
			return err
		}
	}

	return w.writeCode(0, "ENDSEC")
}

func (w *Writer) writeEntities(entities []Entity) error {
	if err := w.writeCode(0, "SECTION"); err != nil {
		return err
	}
	if err := w.writeCode(2, "ENTITIES"); err != nil {
		return err
	}

	for _, e := range entities {
		if err := w.writeSingleEntity(e); err != nil {
			return err
		}
	}

	return w.writeCode(0, "ENDSEC")
}

func (w *Writer) writeSingleEntity(e Entity) error {
	switch v := e.(type) {
	case *Line:
		return w.writeLine(v)
	case *Polyline:
		return w.writePolyline(v)
	case *Circle:
		return w.writeCircle(v)
	case *Arc:
		return w.writeArc(v)
	case *Text:
		return w.writeText(v)
	case *Insert:
		return w.writeInsert(v)
	default:
		return fmt.Errorf("unsupported DXF entity type: %T", e)
	}
}

func (w *Writer) writeLine(l *Line) error {
	if err := w.writeCode(0, "LINE"); err != nil {
		return err
	}
	if err := w.writeCode(100, "AcDbEntity"); err != nil {
		return err
	}
	if err := w.writeCode(8, l.Layer()); err != nil {
		return err
	}
	if l.Color != 0 {
		if err := w.writeCode(62, l.Color); err != nil {
			return err
		}
	}
	if err := w.writeCode(100, "AcDbLine"); err != nil {
		return err
	}
	if err := w.writeCode(10, l.Start.X); err != nil {
		return err
	}
	if err := w.writeCode(20, l.Start.Y); err != nil {
		return err
	}
	if err := w.writeCode(30, l.Start.Z); err != nil {
		return err
	}
	if err := w.writeCode(11, l.End.X); err != nil {
		return err
	}
	if err := w.writeCode(21, l.End.Y); err != nil {
		return err
	}
	return w.writeCode(31, l.End.Z)
}

func (w *Writer) writePolyline(p *Polyline) error {
	if err := w.writeCode(0, "LWPOLYLINE"); err != nil {
		return err
	}
	if err := w.writeCode(100, "AcDbEntity"); err != nil {
		return err
	}
	if err := w.writeCode(8, p.Layer()); err != nil {
		return err
	}
	if p.Color != 0 {
		if err := w.writeCode(62, p.Color); err != nil {
			return err
		}
	}
	if err := w.writeCode(100, "AcDbPolyline"); err != nil {
		return err
	}
	if err := w.writeCode(90, len(p.Vertices)); err != nil {
		return err
	}
	flags := 0
	if p.IsClosed {
		flags = 1
	}
	if err := w.writeCode(70, flags); err != nil {
		return err
	}

	for _, v := range p.Vertices {
		if err := w.writeCode(10, v.Point.X); err != nil {
			return err
		}
		if err := w.writeCode(20, v.Point.Y); err != nil {
			return err
		}
		if v.Bulge != 0.0 {
			if err := w.writeCode(42, v.Bulge); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Writer) writeCircle(c *Circle) error {
	if err := w.writeCode(0, "CIRCLE"); err != nil {
		return err
	}
	if err := w.writeCode(100, "AcDbEntity"); err != nil {
		return err
	}
	if err := w.writeCode(8, c.Layer()); err != nil {
		return err
	}
	if c.Color != 0 {
		if err := w.writeCode(62, c.Color); err != nil {
			return err
		}
	}
	if err := w.writeCode(100, "AcDbCircle"); err != nil {
		return err
	}
	if err := w.writeCode(10, c.Center.X); err != nil {
		return err
	}
	if err := w.writeCode(20, c.Center.Y); err != nil {
		return err
	}
	if err := w.writeCode(30, c.Center.Z); err != nil {
		return err
	}
	return w.writeCode(40, c.Radius)
}

func (w *Writer) writeArc(a *Arc) error {
	if err := w.writeCode(0, "ARC"); err != nil {
		return err
	}
	if err := w.writeCode(100, "AcDbEntity"); err != nil {
		return err
	}
	if err := w.writeCode(8, a.Layer()); err != nil {
		return err
	}
	if a.Color != 0 {
		if err := w.writeCode(62, a.Color); err != nil {
			return err
		}
	}
	if err := w.writeCode(100, "AcDbCircle"); err != nil {
		return err
	}
	if err := w.writeCode(10, a.Center.X); err != nil {
		return err
	}
	if err := w.writeCode(20, a.Center.Y); err != nil {
		return err
	}
	if err := w.writeCode(30, a.Center.Z); err != nil {
		return err
	}
	if err := w.writeCode(40, a.Radius); err != nil {
		return err
	}
	if err := w.writeCode(100, "AcDbArc"); err != nil {
		return err
	}
	if err := w.writeCode(50, a.StartAngle); err != nil {
		return err
	}
	return w.writeCode(51, a.EndAngle)
}

func (w *Writer) writeText(t *Text) error {
	if err := w.writeCode(0, "TEXT"); err != nil {
		return err
	}
	if err := w.writeCode(100, "AcDbEntity"); err != nil {
		return err
	}
	if err := w.writeCode(8, t.Layer()); err != nil {
		return err
	}
	if t.Color != 0 {
		if err := w.writeCode(62, t.Color); err != nil {
			return err
		}
	}
	if err := w.writeCode(100, "AcDbText"); err != nil {
		return err
	}
	if err := w.writeCode(10, t.Insertion.X); err != nil {
		return err
	}
	if err := w.writeCode(20, t.Insertion.Y); err != nil {
		return err
	}
	if err := w.writeCode(30, t.Insertion.Z); err != nil {
		return err
	}
	if err := w.writeCode(40, t.Height); err != nil {
		return err
	}
	if err := w.writeCode(1, t.Value); err != nil {
		return err
	}
	if t.Rotation != 0.0 {
		if err := w.writeCode(50, t.Rotation); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) writeInsert(i *Insert) error {
	if err := w.writeCode(0, "INSERT"); err != nil {
		return err
	}
	if err := w.writeCode(100, "AcDbEntity"); err != nil {
		return err
	}
	if err := w.writeCode(8, i.Layer()); err != nil {
		return err
	}
	if err := w.writeCode(100, "AcDbBlockReference"); err != nil {
		return err
	}
	if err := w.writeCode(2, i.BlockName); err != nil {
		return err
	}
	if err := w.writeCode(10, i.Insertion.X); err != nil {
		return err
	}
	if err := w.writeCode(20, i.Insertion.Y); err != nil {
		return err
	}
	if err := w.writeCode(30, i.Insertion.Z); err != nil {
		return err
	}
	scaleX := i.ScaleX
	if scaleX == 0 {
		scaleX = 1.0
	}
	scaleY := i.ScaleY
	if scaleY == 0 {
		scaleY = 1.0
	}
	scaleZ := i.ScaleZ
	if scaleZ == 0 {
		scaleZ = 1.0
	}
	if err := w.writeCode(41, scaleX); err != nil {
		return err
	}
	if err := w.writeCode(42, scaleY); err != nil {
		return err
	}
	if err := w.writeCode(43, scaleZ); err != nil {
		return err
	}
	if i.RotationDeg != 0 {
		if err := w.writeCode(50, i.RotationDeg); err != nil {
			return err
		}
	}
	return nil
}
