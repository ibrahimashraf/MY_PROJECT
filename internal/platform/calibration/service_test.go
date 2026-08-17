package calibration

import (
	"testing"
	"time"

	sharedcalibration "integin/internal/shared/calibration"
	"integin/internal/shared/types"
)

func TestCalibrationExpiryBlocksSubmissionAndTriggersHook(t *testing.T) {
	var expiredID string
	service := New(func(record sharedcalibration.Record) { expiredID = record.ID })
	due := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	record := sharedcalibration.Record{ID: "cal-1", TenantID: "tenant-1", EquipmentID: "meter-1", StandardReference: "ISO-1", CalibrationDate: due.Add(-time.Hour), NextDueDate: due, TechnicianID: "tech-1", Result: "PASS", Status: types.CalibrationActive}
	if err := service.Create(record); err != nil {
		t.Fatal(err)
	}
	if err := service.SubmissionAllowed("tenant-1", "meter-1", due.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	expired := service.ExpireDue(due)
	if len(expired) != 1 || expiredID != "cal-1" {
		t.Fatalf("unexpected expiry: %#v %s", expired, expiredID)
	}
	if err := service.SubmissionAllowed("tenant-1", "meter-1", due.Add(time.Minute)); err == nil {
		t.Fatal("expired calibration should block submission")
	}
}

func TestCalibrationSupersession(t *testing.T) {
	service := New(nil)
	record := sharedcalibration.Record{ID: "cal-1", TenantID: "tenant-1", EquipmentID: "meter-1", StandardReference: "ISO-1", CalibrationDate: time.Now(), NextDueDate: time.Now().Add(time.Hour), TechnicianID: "tech-1", Result: "PASS", Status: types.CalibrationActive}
	if err := service.Create(record); err != nil {
		t.Fatal(err)
	}
	if err := service.Supersede("tenant-1", "cal-1"); err != nil {
		t.Fatal(err)
	}
	loaded, err := service.Get("tenant-1", "cal-1")
	if err != nil || loaded.Status != types.CalibrationSuperseded {
		t.Fatalf("unexpected supersession: %#v %v", loaded, err)
	}
}
