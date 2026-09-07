package calibration

import (
	"testing"
	"time"

	"integin/internal/shared/types"
)

func TestRecordValidationAndExpiry(t *testing.T) {
	calibrationDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	dueDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	record := Record{ID: "cal-1", TenantID: "tenant-1", EquipmentID: "asset-1", StandardReference: "ISO-1", CalibrationDate: calibrationDate, NextDueDate: dueDate, TechnicianID: "tech-1", Result: "PASS", Status: types.CalibrationActive}
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	if record.IsExpired(dueDate) {
		t.Fatal("record should not expire on its due date")
	}
	if !record.IsExpired(dueDate.Add(time.Nanosecond)) {
		t.Fatal("record should expire after its due date")
	}
}

func TestRecordRejectsBackwardsDueDate(t *testing.T) {
	record := Record{ID: "cal-1", TenantID: "tenant-1", EquipmentID: "asset-1", StandardReference: "ISO-1", CalibrationDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), NextDueDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TechnicianID: "tech-1", Status: types.CalibrationActive}
	if err := record.Validate(); err == nil {
		t.Fatal("expected invalid date ordering")
	}
}
