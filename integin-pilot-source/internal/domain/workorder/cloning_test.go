package workorder

import (
	"testing"
	"time"

	"integin/internal/domain/inspection"
	"integin/internal/shared/types"
)

func TestGroupByLocation(t *testing.T) {
	items := []ScopeItem{
		{ID: "scope-3", LocationID: "LOC-B", AssetID: "ASSET-3", AssetType: "hook", TenantID: "t1", OrganizationID: "org1", ClientID: "c1"},
		{ID: "scope-1", LocationID: "LOC-A", AssetID: "ASSET-1", AssetType: "wire_rope", TenantID: "t1", OrganizationID: "org1", ClientID: "c1"},
		{ID: "scope-2", LocationID: "LOC-A", AssetID: "ASSET-2", AssetType: "shackle", TenantID: "t1", OrganizationID: "org1", ClientID: "c1"},
		{ID: "scope-4", LocationID: "", AssetID: "ASSET-4", AssetType: "sling", TenantID: "t1", OrganizationID: "org1", ClientID: "c1"},
	}

	sections := GroupByLocation(items)
	if len(sections) != 3 {
		t.Fatalf("expected 3 location sections, got %d", len(sections))
	}

	// Verify deterministic sorting: DEFAULT_UNASSIGNED_LOCATION, LOC-A, LOC-B
	if sections[0].LocationID != "DEFAULT_UNASSIGNED_LOCATION" {
		t.Errorf("section 0 expected DEFAULT_UNASSIGNED_LOCATION, got %s", sections[0].LocationID)
	}
	if sections[1].LocationID != "LOC-A" {
		t.Errorf("section 1 expected LOC-A, got %s", sections[1].LocationID)
	}
	if sections[2].LocationID != "LOC-B" {
		t.Errorf("section 2 expected LOC-B, got %s", sections[2].LocationID)
	}

	// Verify items within LOC-A are sorted by ID: scope-1, scope-2
	if len(sections[1].ScopeItems) != 2 || sections[1].ScopeItems[0].ID != "scope-1" || sections[1].ScopeItems[1].ID != "scope-2" {
		t.Errorf("LOC-A scope items not sorted correctly: %+v", sections[1].ScopeItems)
	}
}

func TestCloneInspectionDraft_ProofIsolation(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	src, err := inspection.New("insp-source-01", "tenant-alpha", "org-alpha", types.EnvironmentTesting, "asset-crane-101", "crane_annual", now)
	if err != nil {
		t.Fatalf("inspection.New: %v", err)
	}

	if err := src.Assign("inspector-bob"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if err := src.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	measVal := 48.5
	err = src.RecordFinding(inspection.Finding{
		ID:            "f-1",
		InspectionID:  src.ID(),
		AssetID:       src.RootAssetID(),
		SectionID:     "sec-hook",
		ItemID:        "item-throat",
		ItemPrompt:    "Hook Throat Opening (mm)",
		Response:      "MEASURED",
		MeasuredValue: &measVal,
		MeasuredUnit:  "mm",
		Severity:      types.SeverityCritical,
		Notes:         "Excessive deformation beyond 10% limit",
		EvidenceRefs:  []string{"evidence-photo-sha256-hash-001", "sensor-scan-002"},
		RecordedBy:    "inspector-bob",
		RecordedAt:    now,
	})
	if err != nil {
		t.Fatalf("RecordFinding: %v", err)
	}

	// Clone to asset-crane-102
	cloneDate := now.Add(2 * time.Hour)
	clone, meta, err := CloneInspectionDraft(src, "insp-target-02", "asset-crane-102", "LOC-DECK-01", "inspector-alice", cloneDate)
	if err != nil {
		t.Fatalf("CloneInspectionDraft: %v", err)
	}

	// 1. Identity & State
	if clone.ID() != "insp-target-02" {
		t.Errorf("expected clone ID insp-target-02, got %s", clone.ID())
	}
	if clone.RootAssetID() != "asset-crane-102" {
		t.Errorf("expected clone RootAssetID asset-crane-102, got %s", clone.RootAssetID())
	}
	if clone.Status() != types.InspectionScheduled {
		t.Errorf("expected clone status %s, got %s", types.InspectionScheduled, clone.Status())
	}

	// 2. Metadata Audit Record
	if !meta.IsClonedDraft {
		t.Error("expected IsClonedDraft to be true")
	}
	if meta.SourceInspectionID != "insp-source-01" {
		t.Errorf("expected source ID insp-source-01, got %s", meta.SourceInspectionID)
	}
	if len(meta.CarriedFindings) != 1 {
		t.Fatalf("expected 1 carried finding, got %d", len(meta.CarriedFindings))
	}

	cf := meta.CarriedFindings[0]
	if cf.InspectionID != "insp-target-02" || cf.AssetID != "asset-crane-102" {
		t.Errorf("carried finding identity mismatch: %+v", cf)
	}
	if cf.ItemPrompt != "Hook Throat Opening (mm)" || cf.Response != "MEASURED" {
		t.Errorf("carried finding prompt/response mismatch: %+v", cf)
	}
	if cf.MeasuredValue == nil || *cf.MeasuredValue != 48.5 || cf.MeasuredUnit != "mm" {
		t.Errorf("carried finding measurement mismatch: %+v", cf)
	}
	if cf.Severity != types.SeverityCritical || cf.Notes != "Excessive deformation beyond 10% limit" {
		t.Errorf("carried finding defect parameters mismatch: %+v", cf)
	}

	// 3. ZERO EVIDENCE BLEED ASSERTION (ISO 17020 §6.2)
	if len(cf.EvidenceRefs) != 0 {
		t.Fatalf("CRITICAL SECURITY FAILURE: EvidenceRefs leaked into clone! Got: %v", cf.EvidenceRefs)
	}
}

func TestCloneInspectionDraft_ValidationFailures(t *testing.T) {
	now := time.Now().UTC()
	src, err := inspection.New("insp-valid-01", "t1", "org1", types.EnvironmentTesting, "asset-1", "visual", now)
	if err != nil {
		t.Fatalf("inspection.New: %v", err)
	}

	// 1. Same ID conflict
	_, _, err = CloneInspectionDraft(src, "insp-valid-01", "asset-2", "loc-1", "actor", now)
	if err != ErrCloneIdentityConflict {
		t.Errorf("expected ErrCloneIdentityConflict, got: %v", err)
	}

	// 2. Empty target ID
	_, _, err = CloneInspectionDraft(src, "", "asset-2", "loc-1", "actor", now)
	if err != ErrInvalidCloneTarget {
		t.Errorf("expected ErrInvalidCloneTarget for empty ID, got: %v", err)
	}

	// 3. Empty target asset
	_, _, err = CloneInspectionDraft(src, "insp-valid-02", "", "loc-1", "actor", now)
	if err != ErrInvalidCloneTarget {
		t.Errorf("expected ErrInvalidCloneTarget for empty asset, got: %v", err)
	}

	// 4. Empty source
	_, _, err = CloneInspectionDraft(inspection.Inspection{}, "insp-valid-02", "asset-2", "loc-1", "actor", now)
	if err != ErrSourceInspectionNil {
		t.Errorf("expected ErrSourceInspectionNil, got: %v", err)
	}
}
