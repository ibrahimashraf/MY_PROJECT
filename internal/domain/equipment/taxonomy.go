package equipment

import (
	"errors"
	"fmt"
	"time"
)

// TaxonomyLevel is the typed ISO 14224 9-tier hierarchy level of an equipment
// node.
type TaxonomyLevel uint8

const (
	LevelIndustry         TaxonomyLevel = 1 // Industry
	LevelBusinessCategory TaxonomyLevel = 2 // Business Category
	LevelInstallation     TaxonomyLevel = 3 // Installation
	LevelPlantUnit        TaxonomyLevel = 4 // Plant / Unit
	LevelSectionSystem    TaxonomyLevel = 5 // Section / System
	LevelEquipmentUnit    TaxonomyLevel = 6 // Equipment Unit (e.g. Crane)
	LevelSubUnit          TaxonomyLevel = 7 // Sub-Unit (e.g. Winch, Boom, Slewing Mechanism)
	LevelComponentItem    TaxonomyLevel = 8 // Component Item (e.g. Wire Rope, Hook Block, Shackle, Hydraulic Ram)
	LevelPart             TaxonomyLevel = 9 // Part
)

const TaxonomyDepth uint8 = 9

var taxonomyLevelNames = [TaxonomyDepth + 1]string{
	"",
	"Industry",
	"BusinessCategory",
	"Installation",
	"PlantUnit",
	"SectionSystem",
	"EquipmentUnit",
	"SubUnit",
	"ComponentItem",
	"Part",
}

func (l TaxonomyLevel) String() string {
	if l < LevelIndustry || l > LevelPart {
		return fmt.Sprintf("TaxonomyLevel(%d)", uint8(l))
	}
	return taxonomyLevelNames[l]
}

// TaxonomyNodeKind is the typed kind of an equipment node.
type TaxonomyNodeKind string

const (
	KindFacilityData  TaxonomyNodeKind = "FACILITY_DATA"
	KindEquipmentUnit TaxonomyNodeKind = "EQUIPMENT_UNIT"
	KindSubUnit       TaxonomyNodeKind = "SUB_UNIT"
	KindComponentItem TaxonomyNodeKind = "COMPONENT_ITEM"
	KindPart          TaxonomyNodeKind = "PART"
)

// ComponentClass is the typed ISO 14224 component class for lifting units.
type ComponentClass string

const (
	ComponentWireRope     ComponentClass = "WIRE_ROPE"
	ComponentHookBlock    ComponentClass = "HOOK_BLOCK"
	ComponentShackle      ComponentClass = "SHACKLE"
	ComponentHydraulicRam ComponentClass = "HYDRAULIC_RAM"
	ComponentGearbox      ComponentClass = "GEARBOX"
	ComponentMotor        ComponentClass = "MOTOR"
	ComponentSlewingDrive ComponentClass = "SLEWING_DRIVE"
	ComponentBoomSection  ComponentClass = "BOOM_SECTION"
	ComponentGeneric      ComponentClass = "GENERIC"
)

// DiscardStatus is the independent discard status of a component item.
type DiscardStatus string

const (
	DiscardActive   DiscardStatus = "ACTIVE"
	DiscardPending  DiscardStatus = "PENDING_DISCARD"
	DiscardRejected DiscardStatus = "REJECTED_DISCARD"
)

var (
	ErrInvalidNodeIdentity  = errors.New("equipment node identity is incomplete")
	ErrInvalidParentID      = errors.New("parent equipment node does not exist")
	ErrInvalidLevelSequence = errors.New("child level must be exactly one deeper than its parent")
	ErrInvalidNodeKind      = errors.New("node kind is not valid for its taxonomy level")
	ErrDuplicateNodeID      = errors.New("equipment node id already exists in the taxonomy")
	ErrInvalidRoot          = errors.New("taxonomy root must be a LevelIndustry facility node")
	ErrComponentIdentity    = errors.New("component item identity is incomplete")
)

// InspectionRecord is one entry in a component item's independent inspection
// history.
type InspectionRecord struct {
	InspectionID string    `json:"inspection_id"`
	Date         time.Time `json:"date"`
	Outcome      string    `json:"outcome"`
	Notes        string    `json:"notes,omitempty"`
	Discarded    bool      `json:"discarded"`
}

// ComponentItem is an ISO 14224 component item (e.g. wire rope, hook block,
// shackle, hydraulic ram). It maintains its own operational lifecycle,
// inspection history, and discard status independently of the parent
// equipment unit or sub-unit.
type ComponentItem struct {
	ID                string             `json:"id"`
	SerialNumber      string             `json:"serial_number"`
	Class             ComponentClass     `json:"class"`
	Status            EquipmentStatus    `json:"status"`
	DiscardStatus     DiscardStatus      `json:"discard_status"`
	DiscardedAt       time.Time          `json:"discarded_at,omitempty"`
	DiscardReason     string             `json:"discard_reason,omitempty"`
	InspectionHistory []InspectionRecord `json:"inspection_history,omitempty"`
	Fatigue           *FatigueState      `json:"fatigue,omitempty"`
}

func NewComponentItem(id, serialNumber string, class ComponentClass) (*ComponentItem, error) {
	if id == "" || serialNumber == "" {
		return nil, ErrComponentIdentity
	}
	return &ComponentItem{
		ID:            id,
		SerialNumber:  serialNumber,
		Class:         class,
		Status:        EquipmentActive,
		DiscardStatus: DiscardActive,
	}, nil
}

// RecordInspection appends an inspection to the component's independent
// history. A Discarded inspection transitions the component to a rejected
// (discarded) state.
func (c *ComponentItem) RecordInspection(rec InspectionRecord) {
	if rec.Date.IsZero() {
		rec.Date = time.Now().UTC()
	}
	c.InspectionHistory = append(c.InspectionHistory, rec)
	if rec.Discarded {
		c.DiscardStatus = DiscardRejected
		c.DiscardedAt = rec.Date
		if rec.Notes != "" {
			c.DiscardReason = rec.Notes
		}
	}
}

// RecordDiscard rejects the component once it fails its discard criterion.
func (c *ComponentItem) RecordDiscard(reason string) {
	c.DiscardStatus = DiscardRejected
	c.DiscardedAt = time.Now().UTC()
	c.DiscardReason = reason
}

// IsDiscarded reports whether the component has been rejected for discard.
func (c ComponentItem) IsDiscarded() bool {
	return c.DiscardStatus == DiscardRejected
}

// SyncFatigueLockout marks the component pending discard when its attached
// ISO 13374 fatigue accumulator has crossed the safe damage threshold. It
// returns true when a lockout was applied and never touches the parent node.
func (c *ComponentItem) SyncFatigueLockout() bool {
	if c.Fatigue == nil || !c.Fatigue.LockedOut {
		return false
	}
	if c.DiscardStatus == DiscardActive {
		c.DiscardStatus = DiscardPending
		c.DiscardReason = "cumulative fatigue damage reached the safe lockout threshold (ISO 13374 Block 5)"
	}
	return true
}

// EquipmentNode is a typed node in the ISO 14224 9-tier hierarchy.
type EquipmentNode struct {
	ID           string           `json:"id"`
	ParentID     string           `json:"parent_id,omitempty"`
	Level        TaxonomyLevel    `json:"level"`
	Kind         TaxonomyNodeKind `json:"kind"`
	ISO14224Code string           `json:"iso_14224_code,omitempty"`
	Name         string           `json:"name"`
	Status       EquipmentStatus  `json:"status,omitempty"`
	Component    *ComponentItem   `json:"component,omitempty"`
	children     []*EquipmentNode `json:"-"`
	parent       *EquipmentNode   `json:"-"`
}

// Children returns the direct children of the node.
func (n *EquipmentNode) Children() []*EquipmentNode {
	if n == nil {
		return nil
	}
	return append([]*EquipmentNode(nil), n.children...)
}

// beget appends child to this node.
func (n *EquipmentNode) beget(child *EquipmentNode) {
	child.parent = n
	n.children = append(n.children, child)
}

// EquipmentTaxonomy is the typed ISO 14224 9-tier relational tree.
type EquipmentTaxonomy struct {
	root *EquipmentNode
	byID map[string]*EquipmentNode
}

// NewEquipmentTaxonomy seeds a taxonomy with the LevelIndustry root node.
func NewEquipmentTaxonomy(industryID, industryName string) (*EquipmentTaxonomy, error) {
	if industryID == "" {
		return nil, ErrInvalidRoot
	}
	root := &EquipmentNode{
		ID:    industryID,
		Level: LevelIndustry,
		Kind:  KindFacilityData,
		Name:  industryName,
	}
	return &EquipmentTaxonomy{
		root: root,
		byID: map[string]*EquipmentNode{root.ID: root},
	}, nil
}

// Root returns the taxonomy root node.
func (t *EquipmentTaxonomy) Root() *EquipmentNode {
	if t == nil {
		return nil
	}
	return t.root
}

// Add attaches a child node under an existing parent, enforcing the typed
// level ladder and node-kind/level compatibility.
func (t *EquipmentTaxonomy) Add(parentID string, node *EquipmentNode) error {
	if node == nil || node.ID == "" || node.Name == "" {
		return ErrInvalidNodeIdentity
	}
	if _, exists := t.byID[node.ID]; exists {
		return ErrDuplicateNodeID
	}
	parent, ok := t.byID[parentID]
	if !ok || parent == nil {
		return ErrInvalidParentID
	}
	if node.Level != parent.Level+1 {
		return fmt.Errorf("%w: parent %s level %s, child %s level %s", ErrInvalidLevelSequence, parentID, parent.Level, node.ID, node.Level)
	}
	if !kindAllowedAtLevel(node.Kind, node.Level) {
		return fmt.Errorf("%w: kind %s at level %s", ErrInvalidNodeKind, node.Kind, node.Level)
	}
	node.ParentID = parentID
	if node.Status == "" {
		node.Status = EquipmentActive
	}
	parent.beget(node)
	t.byID[node.ID] = node
	return nil
}

// Find returns the node with the given id.
func (t *EquipmentTaxonomy) Find(id string) (*EquipmentNode, bool) {
	node, ok := t.byID[id]
	return node, ok
}

// Path returns the ordered node chain from the root down to the given id.
func (t *EquipmentTaxonomy) Path(id string) ([]*EquipmentNode, error) {
	node, ok := t.byID[id]
	if !ok {
		return nil, ErrInvalidParentID
	}
	var trail []*EquipmentNode
	for cur := node; cur != nil; cur = cur.parent {
		trail = append(trail, cur)
	}
	for i, j := 0, len(trail)-1; i < j; i, j = i+1, j-1 {
		trail[i], trail[j] = trail[j], trail[i]
	}
	return trail, nil
}

// Components returns every component item in the taxonomy, in traversal
// order, regardless of depth.
func (t *EquipmentTaxonomy) Components() []*ComponentItem {
	var found []*ComponentItem
	t.walk(t.root, func(n *EquipmentNode) {
		if n.Component != nil {
			found = append(found, n.Component)
		}
	})
	return found
}

// walk traverses the tree depth-first.
func (t *EquipmentTaxonomy) walk(start *EquipmentNode, visit func(*EquipmentNode)) {
	if start == nil {
		return
	}
	visit(start)
	for _, c := range start.children {
		t.walk(c, visit)
	}
}

// kindAllowedAtLevel validates the typed node-kind / level compatibility.
func kindAllowedAtLevel(kind TaxonomyNodeKind, level TaxonomyLevel) bool {
	switch kind {
	case KindFacilityData:
		return level >= LevelIndustry && level <= LevelSectionSystem
	case KindEquipmentUnit:
		return level == LevelEquipmentUnit
	case KindSubUnit:
		return level == LevelSubUnit
	case KindComponentItem:
		return level == LevelComponentItem
	case KindPart:
		return level == LevelPart
	}
	return false
}
