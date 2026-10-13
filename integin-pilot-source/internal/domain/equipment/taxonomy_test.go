package equipment

import (
	"errors"
	"testing"
)

func mustAdd(t *testing.T, tax *EquipmentTaxonomy, parentID string, n *EquipmentNode) {
	t.Helper()
	if err := tax.Add(parentID, n); err != nil {
		t.Fatalf("Add(%q, %q): %v", parentID, n.ID, err)
	}
}

func facilityNode(id, name string, level TaxonomyLevel) *EquipmentNode {
	return &EquipmentNode{ID: id, Level: level, Kind: KindFacilityData, Name: name}
}

func TestTaxonomyLevelNames(t *testing.T) {
	want := [TaxonomyDepth + 1]string{0: "", 1: "Industry", 2: "BusinessCategory", 3: "Installation", 4: "PlantUnit", 5: "SectionSystem", 6: "EquipmentUnit", 7: "SubUnit", 8: "ComponentItem", 9: "Part"}
	for l := uint8(1); l <= uint8(LevelPart); l++ {
		level := TaxonomyLevel(l)
		if got := level.String(); got != want[l] {
			t.Fatalf("level %d String() = %q, want %q", l, got, want[l])
		}
	}
}

func TestNineTierTaxonomyPath(t *testing.T) {
	tax, err := NewEquipmentTaxonomy("ind-oilgas", "Oil & Gas Industry")
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	mustAdd(t, tax, "ind-oilgas", facilityNode("biz-upstream", "Upstream", LevelBusinessCategory))
	mustAdd(t, tax, "biz-upstream", facilityNode("inst-rig01", "Rig 01", LevelInstallation))
	mustAdd(t, tax, "inst-rig01", facilityNode("plant-rig01", "Rig 01 Plant", LevelPlantUnit))
	mustAdd(t, tax, "plant-rig01", facilityNode("sec-crane", "Crane Section", LevelSectionSystem))
	mustAdd(t, tax, "sec-crane", &EquipmentNode{ID: "eou-crane", Level: LevelEquipmentUnit, Kind: KindEquipmentUnit, Name: "Crane", ISO14224Code: "A8"})
	mustAdd(t, tax, "eou-crane", &EquipmentNode{ID: "sub-winch", Level: LevelSubUnit, Kind: KindSubUnit, Name: "Winch"})
	mustAdd(t, tax, "sub-winch", nodeWithComponent(t, "cp-main-rope", "WIRE-ROPE-SER-1", ComponentWireRope, "Main hoist wire rope"))
	mustAdd(t, tax, "cp-main-rope", &EquipmentNode{ID: "part-socket", Level: LevelPart, Kind: KindPart, Name: "Rope socket terminal"})

	path, err := tax.Path("part-socket")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if len(path) != int(TaxonomyDepth) {
		t.Fatalf("path len = %d, want %d", len(path), TaxonomyDepth)
	}
	for i := uint8(1); i <= uint8(LevelPart); i++ {
		node := path[i-1]
		if node.Level != TaxonomyLevel(i) {
			t.Fatalf("path[%d] level = %s, want %s", i-1, node.Level, TaxonomyLevel(i))
		}
	}
	if path[0].ID != "ind-oilgas" || path[len(path)-1].ID != "part-socket" {
		t.Fatalf("path endpoints = %q..%q", path[0].ID, path[len(path)-1].ID)
	}
}

func nodeWithComponent(t *testing.T, id, serial string, class ComponentClass, name string) *EquipmentNode {
	t.Helper()
	comp, err := NewComponentItem(id+"-comp", serial, class)
	if err != nil {
		t.Fatalf("NewComponentItem: %v", err)
	}
	return &EquipmentNode{ID: id, Level: LevelComponentItem, Kind: KindComponentItem, Name: name, Component: comp}
}

func TestCraneLiftingUnitAssembly(t *testing.T) {
	tax, err := NewEquipmentTaxonomy("ind-cranes", "Crane Industry")
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	mustAdd(t, tax, "ind-cranes", facilityNode("biz-mobile", "Mobile Cranes", LevelBusinessCategory))
	mustAdd(t, tax, "biz-mobile", facilityNode("inst-crane", "Crane Installation", LevelInstallation))
	mustAdd(t, tax, "inst-crane", facilityNode("plant-1", "Plant", LevelPlantUnit))
	mustAdd(t, tax, "plant-1", facilityNode("sec-hoist", "Hoisting Section", LevelSectionSystem))
	mustAdd(t, tax, "sec-hoist", &EquipmentNode{ID: "u-crane", Level: LevelEquipmentUnit, Kind: KindEquipmentUnit, Name: "Crane LTM1500", ISO14224Code: "A8.1"})

	for _, sub := range []struct{ id, name string }{
		{"su-winch", "Winch"},
		{"su-boom", "Boom"},
		{"su-slew", "Slewing Mechanism"},
	} {
		mustAdd(t, tax, "u-crane", &EquipmentNode{ID: sub.id, Level: LevelSubUnit, Kind: KindSubUnit, Name: sub.name})
	}
	mustAdd(t, tax, "su-winch", nodeWithComponent(t, "cp-rope", "ROPE-SER-77", ComponentWireRope, "Wire Rope"))
	mustAdd(t, tax, "su-boom", nodeWithComponent(t, "cp-hook", "HOOK-SER-3", ComponentHookBlock, "Hook Block"))
	mustAdd(t, tax, "su-boom", nodeWithComponent(t, "cp-ram", "RAM-SER-9", ComponentHydraulicRam, "Boom luffing ram"))
	mustAdd(t, tax, "su-slew", nodeWithComponent(t, "cp-shackle", "SHACKLE-SER-11", ComponentShackle, "Shackle"))

	if len(tax.Components()) != 4 {
		t.Fatalf("Components() = %d, want 4", len(tax.Components()))
	}
	winch, ok := tax.Find("su-winch")
	if !ok {
		t.Fatal("winch node not found")
	}
	if len(winch.Children()) != 1 || winch.Children()[0].ID != "cp-rope" {
		t.Fatalf("winch children = %v", winch.Children())
	}
	removedCheck := []string{"ROPE-SER-77", "HOOK-SER-3", "RAM-SER-9", "SHACKLE-SER-11"}
	for i, comp := range tax.Components() {
		if comp.SerialNumber != removedCheck[i] {
			t.Fatalf("component %d serial = %q, want %q", i, comp.SerialNumber, removedCheck[i])
		}
	}
}

func TestTaxonomyLevelLadderEnforced(t *testing.T) {
	tax, err := NewEquipmentTaxonomy("ind", "Industry")
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	mustAdd(t, tax, "ind", facilityNode("biz", "Biz", LevelBusinessCategory))

	t.Run("skipping a level rejected", func(t *testing.T) {
		err := tax.Add("ind", facilityNode("inst", "Inst", LevelInstallation)) // goes 1 -> 3
		if !errors.Is(err, ErrInvalidLevelSequence) {
			t.Fatalf("err = %v, want %v", err, ErrInvalidLevelSequence)
		}
	})

	t.Run("equipment unit cannot be planted directly under industry", func(t *testing.T) {
		err := tax.Add("ind", &EquipmentNode{ID: "u", Level: LevelEquipmentUnit, Kind: KindEquipmentUnit, Name: "Crane"})
		if !errors.Is(err, ErrInvalidLevelSequence) {
			t.Fatalf("err = %v, want %v", err, ErrInvalidLevelSequence)
		}
	})

	t.Run("engineered crane hierarchy from scratch", func(t *testing.T) {
		tax2, err := NewEquipmentTaxonomy("ind2", "Industry 2")
		if err != nil {
			t.Fatalf("root: %v", err)
		}
		if err := tax2.Add("ind2", &EquipmentNode{ID: "u2", Level: LevelEquipmentUnit, Kind: KindEquipmentUnit, Name: "Crane"}); !errors.Is(err, ErrInvalidLevelSequence) {
			t.Fatalf("err = %v, want %v", err, ErrInvalidLevelSequence)
		}
	})
}

func TestNodeKindLevelCompatibility(t *testing.T) {
	tax, err := NewEquipmentTaxonomy("ind", "Industry")
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	mustAdd(t, tax, "ind", facilityNode("biz", "Biz", LevelBusinessCategory))
	mustAdd(t, tax, "biz", facilityNode("inst", "Inst", LevelInstallation))
	mustAdd(t, tax, "inst", facilityNode("plant", "Plant", LevelPlantUnit))
	mustAdd(t, tax, "plant", facilityNode("sec", "Sec", LevelSectionSystem))
	mustAdd(t, tax, "sec", &EquipmentNode{ID: "u", Level: LevelEquipmentUnit, Kind: KindEquipmentUnit, Name: "Crane"})
	mustAdd(t, tax, "u", &EquipmentNode{ID: "sub", Level: LevelSubUnit, Kind: KindSubUnit, Name: "Winch"})

	t.Run("wrong kind for its own level rejected", func(t *testing.T) {
		// Level 7 under the crane (valid sequence), but KindPart is not a
		// valid kind at the SubUnit level.
		err := tax.Add("u", &EquipmentNode{ID: "bad", Level: LevelSubUnit, Kind: KindPart, Name: "bad part"})
		if !errors.Is(err, ErrInvalidNodeKind) {
			t.Fatalf("err = %v, want %v", err, ErrInvalidNodeKind)
		}
	})

	t.Run("sub-unit at industry level rejected", func(t *testing.T) {
		err := tax.Add("ind", &EquipmentNode{ID: "sx", Level: LevelSubUnit, Kind: KindSubUnit, Name: "Winch"})
		if !errors.Is(err, ErrInvalidLevelSequence) && !errors.Is(err, ErrInvalidNodeKind) {
			t.Fatalf("err = %v, want sequence or kind error", err)
		}
	})
}

func TestFindMissingAndDuplicateRejected(t *testing.T) {
	tax, err := NewEquipmentTaxonomy("ind", "Industry")
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	mustAdd(t, tax, "ind", facilityNode("biz", "Biz", LevelBusinessCategory))

	if _, ok := tax.Find("nope"); ok {
		t.Fatal("missing node reported as found")
	}
	mustAdd(t, tax, "biz", facilityNode("inst", "Inst", LevelInstallation))
	err = tax.Add("ind", facilityNode("inst", "Inst2", LevelInstallation))
	if !errors.Is(err, ErrDuplicateNodeID) {
		t.Fatalf("err = %v, want %v", err, ErrDuplicateNodeID)
	}
	if err := tax.Add("missing-parent", facilityNode("x", "X", LevelInstallation)); !errors.Is(err, ErrInvalidParentID) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidParentID)
	}
}

func TestNewComponentItemRequiresIdentity(t *testing.T) {
	if _, err := NewComponentItem("", "SER-1", ComponentWireRope); !errors.Is(err, ErrComponentIdentity) {
		t.Fatalf("err = %v, want %v", err, ErrComponentIdentity)
	}
	if _, err := NewComponentItem("cp", "", ComponentWireRope); !errors.Is(err, ErrComponentIdentity) {
		t.Fatalf("err = %v, want %v", err, ErrComponentIdentity)
	}
	if _, err := NewComponentItem("cp", "SER-1", ComponentWireRope); err != nil {
		t.Fatalf("valid component rejected: %v", err)
	}
}

func TestComponentIndependentDiscardIsolation(t *testing.T) {
	tax, err := NewEquipmentTaxonomy("ind", "Industry")
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	mustAdd(t, tax, "ind", facilityNode("biz", "Biz", LevelBusinessCategory))
	mustAdd(t, tax, "biz", facilityNode("inst", "Inst", LevelInstallation))
	mustAdd(t, tax, "inst", facilityNode("plant", "Plant", LevelPlantUnit))
	mustAdd(t, tax, "plant", facilityNode("sec", "Sec", LevelSectionSystem))
	mustAdd(t, tax, "sec", &EquipmentNode{ID: "u-crane", Level: LevelEquipmentUnit, Kind: KindEquipmentUnit, Name: "Crane"})
	mustAdd(t, tax, "u-crane", &EquipmentNode{ID: "su-winch", Level: LevelSubUnit, Kind: KindSubUnit, Name: "Winch"})
	mustAdd(t, tax, "su-winch", nodeWithComponent(t, "cp-rope", "ROPE-SER-77", ComponentWireRope, "Wire Rope"))
	mustAdd(t, tax, "su-winch", nodeWithComponent(t, "cp-hook", "HOOK-SER-3", ComponentHookBlock, "Hook Block"))

	rope, _ := tax.Find("cp-rope")
	hook, _ := tax.Find("cp-hook")
	crane, _ := tax.Find("u-crane")

	if crane.Status != EquipmentActive {
		t.Fatalf("crane status = %q, want ACTIVE", crane.Status)
	}
	rope.Component.RecordDiscard("exceeds ISO 4309 broken wire count 6d")
	if !rope.Component.IsDiscarded() {
		t.Fatal("wire rope should be discarded")
	}
	if rope.Component.DiscardStatus != DiscardRejected {
		t.Fatalf("rope discard status = %q, want %q", rope.Component.DiscardStatus, DiscardRejected)
	}
	if crane.Status != EquipmentActive {
		t.Fatalf("crane status changed to %q after component discard; must stay ACTIVE", crane.Status)
	}
	if hook.Component.IsDiscarded() {
		t.Fatal("sibling hook block must remain active when wire rope is discarded")
	}
}

func TestComponentIndependentInspectionLifecycle(t *testing.T) {
	comp, err := NewComponentItem("cp-1", "ROPE-SER-77", ComponentWireRope)
	if err != nil {
		t.Fatalf("NewComponentItem: %v", err)
	}
	comp.RecordInspection(InspectionRecord{InspectionID: "INSP-01", Outcome: "PASS", Notes: "visual ok"})
	comp.RecordInspection(InspectionRecord{InspectionID: "INSP-02", Outcome: "FAIL", Notes: "broken wire cluster", Discarded: true})
	if len(comp.InspectionHistory) != 2 {
		t.Fatalf("inspection history = %d, want 2", len(comp.InspectionHistory))
	}
	if !comp.IsDiscarded() {
		t.Fatal("component should be discarded after failing inspection")
	}
	if comp.DiscardReason != "broken wire cluster" {
		t.Fatalf("discard reason = %q", comp.DiscardReason)
	}
	if comp.DiscardedAt.IsZero() {
		t.Fatal("discarded_at must be recorded")
	}
	// A passing inspection must never auto-discard a component.
	comp2, err := NewComponentItem("cp-2", "HOOK-SER-3", ComponentHookBlock)
	if err != nil {
		t.Fatalf("NewComponentItem: %v", err)
	}
	comp2.RecordInspection(InspectionRecord{InspectionID: "INSP-03", Outcome: "PASS"})
	if comp2.IsDiscarded() {
		t.Fatal("passing inspection must not discard the component")
	}
	if comp2.DiscardStatus != DiscardActive {
		t.Fatalf("discard status = %q, want ACTIVE", comp2.DiscardStatus)
	}
}
