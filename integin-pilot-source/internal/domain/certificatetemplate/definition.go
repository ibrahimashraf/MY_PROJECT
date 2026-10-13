package certificatetemplate

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type BindingKey string

const (
	InspectionID                BindingKey = "inspection.id"
	InspectionWorkOrderID       BindingKey = "inspection.work_order_id"
	InspectionAssetID           BindingKey = "inspection.asset_id"
	InspectionInspectorID       BindingKey = "inspection.inspector_id"
	InspectionLifecycleState    BindingKey = "inspection.lifecycle_state"
	InspectionRevision          BindingKey = "inspection.revision"
	InspectionFinalizationState BindingKey = "inspection.finalization_state"
	InspectionCreatedAt         BindingKey = "inspection.created_at"
	InspectionUpdatedAt         BindingKey = "inspection.updated_at"
)

type ValueKind string

const (
	ValueText      ValueKind = "TEXT"
	ValueInteger   ValueKind = "INTEGER"
	ValueDateTime  ValueKind = "DATETIME"
	ValueEnum      ValueKind = "ENUM"
	ValueRepeating ValueKind = "REPEATING"
)

type BindingSpec struct {
	Key              BindingKey
	ValueKind        ValueKind
	Available        bool
	ConditionAllowed bool
}

type Catalog struct {
	Version int
	Specs   map[BindingKey]BindingSpec
}

func DefaultCatalog() Catalog {
	return Catalog{Version: 1, Specs: map[BindingKey]BindingSpec{
		InspectionID:                {Key: InspectionID, ValueKind: ValueText, Available: true},
		InspectionWorkOrderID:       {Key: InspectionWorkOrderID, ValueKind: ValueText, Available: true},
		InspectionAssetID:           {Key: InspectionAssetID, ValueKind: ValueText, Available: true},
		InspectionInspectorID:       {Key: InspectionInspectorID, ValueKind: ValueText, Available: true},
		InspectionLifecycleState:    {Key: InspectionLifecycleState, ValueKind: ValueEnum, Available: true, ConditionAllowed: true},
		InspectionRevision:          {Key: InspectionRevision, ValueKind: ValueInteger, Available: true},
		InspectionFinalizationState: {Key: InspectionFinalizationState, ValueKind: ValueEnum, Available: true, ConditionAllowed: true},
		InspectionCreatedAt:         {Key: InspectionCreatedAt, ValueKind: ValueDateTime, Available: true},
		InspectionUpdatedAt:         {Key: InspectionUpdatedAt, ValueKind: ValueDateTime, Available: true},
	}}
}

func (c Catalog) Spec(key BindingKey) (BindingSpec, bool) {
	spec, ok := c.Specs[key]
	return spec, ok
}

type Status string

const (
	Draft    Status = "DRAFT"
	Approved Status = "APPROVED"
	Retired  Status = "RETIRED"
)

type PresentationKind string

const (
	TextCell        PresentationKind = "TEXT"
	CheckboxCell    PresentationKind = "CHECKBOX"
	DateCell        PresentationKind = "DATE"
	StaticTextCell  PresentationKind = "STATIC_TEXT"
	RepeatingRegion PresentationKind = "REPEATING_REGION"
)

type FitPolicy string

const (
	WrapRequired       FitPolicy = "WRAP_REQUIRED"
	SingleLineRequired FitPolicy = "SINGLE_LINE_REQUIRED"
	CheckboxMap        FitPolicy = "CHECKBOX_MAP"
	RepeatRequired     FitPolicy = "REPEAT_REQUIRED"
)

type ConditionOperator string

const (
	Equals    ConditionOperator = "EQUALS"
	NotEquals ConditionOperator = "NOT_EQUALS"
)

type Applicability struct {
	BindingKey BindingKey
	Operator   ConditionOperator
	Literal    string
}

type Rectangle struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

func (r Rectangle) Validate(pageWidth, pageHeight float64) error {
	if r.X < 0 || r.Y < 0 || r.Width <= 0 || r.Height <= 0 {
		return errors.New("cell rectangle requires non-negative origin and positive dimensions")
	}
	if r.X+r.Width > pageWidth || r.Y+r.Height > pageHeight {
		return errors.New("cell rectangle exceeds page bounds")
	}
	return nil
}

func (r Rectangle) Overlaps(other Rectangle) bool {
	return r.X < other.X+other.Width && other.X < r.X+r.Width && r.Y < other.Y+other.Height && other.Y < r.Y+r.Height
}

type Cell struct {
	ID             string
	PageNumber     int
	Rectangle      Rectangle
	Kind           PresentationKind
	BindingKey     BindingKey
	StaticText     string
	Label          string
	FitPolicy      FitPolicy
	MaxLines       int
	Required       bool
	CheckboxValues map[string]string
	Applicability  *Applicability
	RepeatSource   BindingKey
	MaxItems       int
}

type Definition struct {
	ID             string
	TemplateCode   string
	Version        int
	Title          string
	AssetType      string
	Status         Status
	CatalogVersion int
	PageCount      int
	PageWidth      float64
	PageHeight     float64
	Cells          []Cell
}

var safeIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func (d Definition) Validate(catalog Catalog) error {
	if strings.TrimSpace(d.ID) == "" || !safeIdentifier.MatchString(d.TemplateCode) || strings.TrimSpace(d.Title) == "" || strings.TrimSpace(d.AssetType) == "" {
		return errors.New("template id, safe template code, title, and asset type are required")
	}
	if d.Version <= 0 || d.CatalogVersion != catalog.Version || d.PageCount <= 0 || d.PageWidth <= 0 || d.PageHeight <= 0 {
		return errors.New("positive version, matching catalog version, page count, and page dimensions are required")
	}
	if d.Status != Draft && d.Status != Approved && d.Status != Retired {
		return errors.New("invalid template status")
	}
	if len(d.Cells) == 0 {
		return errors.New("at least one certificate cell is required")
	}
	byPage := map[int][]Cell{}
	cellIDs := map[string]bool{}
	for _, cell := range d.Cells {
		if !safeIdentifier.MatchString(cell.ID) || cellIDs[cell.ID] {
			return errors.New("certificate cell ids must be unique safe identifiers")
		}
		cellIDs[cell.ID] = true
		if cell.PageNumber <= 0 || cell.PageNumber > d.PageCount {
			return errors.New("certificate cell page number is outside the template")
		}
		if err := cell.Rectangle.Validate(d.PageWidth, d.PageHeight); err != nil {
			return fmt.Errorf("cell %s: %w", cell.ID, err)
		}
		if err := cell.validate(catalog); err != nil {
			return fmt.Errorf("cell %s: %w", cell.ID, err)
		}
		for _, existing := range byPage[cell.PageNumber] {
			if cell.Rectangle.Overlaps(existing.Rectangle) {
				return fmt.Errorf("cell %s overlaps cell %s", cell.ID, existing.ID)
			}
		}
		byPage[cell.PageNumber] = append(byPage[cell.PageNumber], cell)
	}
	return nil
}

func (c Cell) validate(catalog Catalog) error {
	switch c.Kind {
	case StaticTextCell:
		if c.BindingKey != "" || strings.TrimSpace(c.StaticText) == "" || c.FitPolicy != WrapRequired || c.MaxLines <= 0 {
			return errors.New("static text requires text, wrap policy, and positive maximum lines without a binding key")
		}
	case TextCell, DateCell, CheckboxCell:
		spec, ok := catalog.Spec(c.BindingKey)
		if !ok || !spec.Available {
			return errors.New("cell binding key is unknown or unavailable")
		}
		if c.Kind == DateCell && spec.ValueKind != ValueDateTime {
			return errors.New("date cell requires a datetime binding key")
		}
		if c.Kind == CheckboxCell {
			if c.FitPolicy != CheckboxMap || len(c.CheckboxValues) == 0 {
				return errors.New("checkbox cell requires an approved checkbox map")
			}
		} else if c.FitPolicy != WrapRequired && c.FitPolicy != SingleLineRequired {
			return errors.New("text and date cells require a bounded text fit policy")
		} else if c.MaxLines <= 0 || (c.FitPolicy == SingleLineRequired && c.MaxLines != 1) {
			return errors.New("text cell maximum line count is invalid")
		}
	case RepeatingRegion:
		spec, ok := catalog.Spec(c.RepeatSource)
		if !ok || !spec.Available || spec.ValueKind != ValueRepeating || c.FitPolicy != RepeatRequired || c.MaxItems <= 0 {
			return errors.New("repeating region requires an available approved repeating source and positive maximum items")
		}
	default:
		return errors.New("unsupported certificate cell kind")
	}
	if c.Applicability != nil {
		spec, ok := catalog.Spec(c.Applicability.BindingKey)
		if !ok || !spec.ConditionAllowed || (c.Applicability.Operator != Equals && c.Applicability.Operator != NotEquals) || strings.TrimSpace(c.Applicability.Literal) == "" {
			return errors.New("certificate cell applicability is not an approved declarative condition")
		}
	}
	return nil
}

func (d Definition) SortedCells() []Cell {
	cells := append([]Cell(nil), d.Cells...)
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].PageNumber != cells[j].PageNumber {
			return cells[i].PageNumber < cells[j].PageNumber
		}
		return cells[i].ID < cells[j].ID
	})
	return cells
}
