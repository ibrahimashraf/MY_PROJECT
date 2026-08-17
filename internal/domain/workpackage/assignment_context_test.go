package workpackage

import (
	"testing"
	"time"
)

func TestAssignmentContextValidate(t *testing.T) {
	valid := AssignmentContext{
		RootAssetID:      "asset-root-1",
		InspectionType:   "thorough-inspection",
		ProcedureVersion: "v1",
		ScheduledAt:      time.Date(2026, time.August, 17, 9, 0, 0, 0, time.UTC),
		FieldAssetIDs:    map[string]string{"field-1": "asset-root-1"},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("validate valid context: %v", err)
	}

	invalid := valid
	invalid.FieldAssetIDs = map[string]string{"field-1": ""}
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected invalid field asset mapping rejection")
	}
}
