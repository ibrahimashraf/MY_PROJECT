package certificatetemplate

import (
	"errors"
	"testing"
	"time"
)

func approvedRecord() DefinitionRecord {
	definition := validDefinition()
	definition.Status = Approved
	return DefinitionRecord{Definition: definition, CreatedBy: "admin-1", CreatedAt: time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC), ApprovedBy: "admin-2", ApprovedAt: time.Date(2026, time.August, 21, 1, 0, 0, 0, time.UTC)}
}

func TestResolveUsesOnlyCanonicalBoundValues(t *testing.T) {
	resolved, err := Resolve(approvedRecord(), InspectionValues("inspection-1", "work-order-1", "asset-1", "inspector-1", "APPROVED", 2, "FINALIZED", time.Now(), time.Now()))
	if err != nil || len(resolved.Cells) != 2 || resolved.Cells[0].RenderedText != "asset-1" && resolved.Cells[1].RenderedText != "asset-1" {
		t.Fatalf("resolve canonical cells: %#v err=%v", resolved, err)
	}
}

func TestResolveRejectsMissingRequiredAndUnapprovedTemplate(t *testing.T) {
	record := approvedRecord()
	if _, err := Resolve(record, CanonicalValues{InspectionID: "inspection-1"}); !errors.Is(err, ErrRequiredValueMissing) {
		t.Fatalf("expected required canonical value error, got %v", err)
	}
	record.Status = Draft
	if _, err := Resolve(record, CanonicalValues{}); !errors.Is(err, ErrTemplateNotApproved) {
		t.Fatalf("expected unapproved template error, got %v", err)
	}
}

func TestResolveRejectsCheckboxAndNewlineOverflow(t *testing.T) {
	record := approvedRecord()
	record.Cells[0].FitPolicy = SingleLineRequired
	record.Cells[0].BindingKey = InspectionAssetID
	if _, err := Resolve(record, InspectionValues("inspection-1", "work-order-1", "asset\n1", "inspector-1", "APPROVED", 2, "FINALIZED", time.Now(), time.Now())); !errors.Is(err, ErrValueDoesNotFit) {
		t.Fatalf("expected single-line overflow error, got %v", err)
	}
}
