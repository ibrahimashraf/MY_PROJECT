package reinspection

import (
	"testing"
	"time"

	"integin/internal/shared/types"
)

func TestIssueCAR_SeverityGate(t *testing.T) {
	now := time.Now().UTC()

	// 1. Critical defect -> Eligible
	carCrit, err := IssueCAR("car-01", "t1", "org1", "asset-crane-1", "insp-orig-101", "f-hook-crack", types.SeverityCritical, "Hook throat crack detected", now)
	if err != nil {
		t.Fatalf("IssueCAR Critical: %v", err)
	}
	if carCrit.Status != CAROpen {
		t.Errorf("expected status OPEN, got %s", carCrit.Status)
	}

	// 2. Major defect -> Eligible
	carMajor, err := IssueCAR("car-02", "t1", "org1", "asset-crane-1", "insp-orig-101", "f-wire-corrosion", types.SeverityMajor, "Severe wire rope localized pitting", now)
	if err != nil {
		t.Fatalf("IssueCAR Major: %v", err)
	}
	if carMajor.Status != CAROpen {
		t.Errorf("expected status OPEN, got %s", carMajor.Status)
	}

	// 3. Minor defect -> Ineligible for CAR (must be resolved via standard advisory)
	_, err = IssueCAR("car-03", "t1", "org1", "asset-crane-1", "insp-orig-101", "f-paint-scratch", types.SeverityMinor, "Paint scratch on outrigger", now)
	if err == nil {
		t.Fatal("expected error for Minor severity CAR, got nil")
	}

	// 4. Incomplete identity
	_, err = IssueCAR("", "t1", "org1", "asset-crane-1", "insp-orig-101", "f-hook", types.SeverityCritical, "desc", now)
	if err == nil {
		t.Fatal("expected error for empty CAR ID, got nil")
	}
}

func TestCARLifecycle_RemediationAndReinspectionPass(t *testing.T) {
	now := time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC)
	car, err := IssueCAR("car-hook-100", "tenant-sa", "org-dammam", "crane-liebherr-01", "insp-orig-500", "finding-throat-deform", types.SeverityCritical, "Throat opening deformed by 15%", now)
	if err != nil {
		t.Fatalf("IssueCAR: %v", err)
	}

	// 1. Client submits remediation with replacement certificate evidence
	remedDate := now.Add(24 * time.Hour)
	err = car.SubmitRemediation(
		"Replaced hook assembly with certified OEM spare parts; verified torque settings",
		[]string{"ev-mill-cert-hook-888.pdf", "ev-torque-calibration-receipt.pdf"},
		"contractor-engineer-ahmed",
		remedDate,
	)
	if err != nil {
		t.Fatalf("SubmitRemediation: %v", err)
	}
	if car.Status != CARRemediated {
		t.Errorf("expected status REMEDIATED, got %s", car.Status)
	}
	if len(car.RemediationEvidenceRefs) != 2 {
		t.Errorf("expected 2 evidence refs, got %d", len(car.RemediationEvidenceRefs))
	}

	// 2. Office schedules targeted re-inspection
	err = car.ScheduleReinspection("insp-recheck-501")
	if err != nil {
		t.Fatalf("ScheduleReinspection: %v", err)
	}
	if car.Status != CARReinspectionScheduled {
		t.Errorf("expected status REINSPECTION_SCHEDULED, got %s", car.Status)
	}

	// 3. Technical authority conducts re-inspection and passes
	recheckDate := remedDate.Add(4 * time.Hour)
	binding, err := car.VerifyOutcome(
		"insp-recheck-501",
		types.VerdictPass,
		"inspector-senior-khalid",
		[]string{"item-hook-throat-dimension", "item-hook-safety-latch"},
		recheckDate,
	)
	if err != nil {
		t.Fatalf("VerifyOutcome: %v", err)
	}

	// CAR is now verified closed
	if car.Status != CARVerifiedClosed {
		t.Errorf("expected status VERIFIED_CLOSED, got %s", car.Status)
	}
	if car.ClosedAt == nil || *car.ClosedAt != recheckDate {
		t.Errorf("expected closed_at %v, got %v", recheckDate, car.ClosedAt)
	}

	// Verify forward traceability binding
	if binding.OriginalInspectionID != "insp-orig-500" {
		t.Errorf("binding original inspection mismatch: %s", binding.OriginalInspectionID)
	}
	if binding.CARID != "car-hook-100" {
		t.Errorf("binding CAR ID mismatch: %s", binding.CARID)
	}
	if binding.ReinspectionID != "insp-recheck-501" {
		t.Errorf("binding reinspection ID mismatch: %s", binding.ReinspectionID)
	}
	if binding.Outcome != types.VerdictPass {
		t.Errorf("binding outcome mismatch: %s", binding.Outcome)
	}
}

func TestCARLifecycle_ReinspectionFailAndReject(t *testing.T) {
	now := time.Now().UTC()
	car, err := IssueCAR("car-rope-200", "t1", "org1", "crane-02", "insp-orig-600", "finding-broken-wires", types.SeverityCritical, "Valley wire breaks in 6d length", now)
	if err != nil {
		t.Fatalf("IssueCAR: %v", err)
	}

	// Client submits remediation claiming surface lubricated
	err = car.SubmitRemediation("Cleaned and greased rope", []string{"photo-greased-rope.jpg"}, "maintenance-lead", now)
	if err != nil {
		t.Fatalf("SubmitRemediation: %v", err)
	}

	// Inspector re-inspects: wire breaks still present, fails again with CRITICAL
	_, err = car.VerifyOutcome("insp-recheck-601", types.VerdictCritical, "inspector-bob", []string{"item-wire-rope-strands"}, now)
	if err != ErrReinspectionFailed {
		t.Fatalf("expected ErrReinspectionFailed, got: %v", err)
	}

	// CAR status is REJECTED
	if car.Status != CARRejected {
		t.Errorf("expected status REJECTED, got %s", car.Status)
	}

	// Client must re-submit genuine remediation (e.g. rope replaced)
	err = car.SubmitRemediation("Completely replaced wire rope reel with certified spool", []string{"reel-test-cert.pdf"}, "contractor-supervisor", now)
	if err != nil {
		t.Fatalf("SubmitRemediation from REJECTED: %v", err)
	}
	if car.Status != CARRemediated {
		t.Errorf("expected status REMEDIATED after re-submission, got %s", car.Status)
	}
}
