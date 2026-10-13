package inspection

import (
	"testing"
	"time"

	"integin/internal/shared/events"
	"integin/internal/shared/types"
)

func newTestInspection(t *testing.T) Inspection {
	t.Helper()
	result, err := New("inspection-1", "tenant-1", "org-1", types.EnvironmentTesting, "asset-1", "annual", time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func testFinding(severity types.Severity) Finding {
	return Finding{ID: "finding-1", AssetID: "asset-1", SectionID: "structure", ItemID: "item-1", ItemPrompt: "Inspect structure", Response: "FAIL", Severity: severity, Notes: "recorded in field", EvidenceRefs: []string{"photo-1"}, RecordedBy: "inspector-1"}
}

func TestInspectionFullLifecycle(t *testing.T) {
	inspection := newTestInspection(t)
	if err := inspection.Assign("inspector-1"); err != nil {
		t.Fatal(err)
	}
	if err := inspection.Start(); err != nil {
		t.Fatal(err)
	}
	finding := testFinding(types.SeverityCritical)
	if err := inspection.RecordFinding(finding); err != nil {
		t.Fatal(err)
	}
	if err := inspection.Complete("critical defect found"); err != nil {
		t.Fatal(err)
	}
	if inspection.OverallResult() != types.VerdictCritical {
		t.Fatalf("expected critical verdict, got %s", inspection.OverallResult())
	}
	if err := inspection.SubmitForReview(); err != nil {
		t.Fatal(err)
	}
	if err := inspection.Approve("reviewer-1"); err != nil {
		t.Fatal(err)
	}
	if err := inspection.Close(); err != nil {
		t.Fatal(err)
	}
	if inspection.Status() != types.InspectionClosed {
		t.Fatalf("expected closed, got %s", inspection.Status())
	}
	emitted := inspection.Events()
	if len(emitted) != 7 {
		t.Fatalf("expected 7 lifecycle events, got %d", len(emitted))
	}
	for _, event := range emitted {
		if event.TenantID() != "tenant-1" || event.OrganizationID() != "org-1" {
			t.Fatalf("event lost tenant context: %s", event.EventType())
		}
	}
	if emitted[1].EventType() != events.InspectionStarted || emitted[4].EventType() != events.InspectionSubmitted {
		t.Fatalf("unexpected event sequence: %s, %s", emitted[1].EventType(), emitted[4].EventType())
	}
}

func TestInspectionRejectsInvalidTransitions(t *testing.T) {
	inspection := newTestInspection(t)
	if err := inspection.Start(); err == nil {
		t.Fatal("expected start before assignment to fail")
	}
	if err := inspection.Assign("inspector-1"); err != nil {
		t.Fatal(err)
	}
	if err := inspection.Approve("reviewer-1"); err == nil {
		t.Fatal("expected approval before review to fail")
	}
}

func TestInspectionEnforcesSeparationOfDuties(t *testing.T) {
	inspection := newTestInspection(t)
	if err := inspection.Assign("inspector-1"); err != nil {
		t.Fatal(err)
	}
	if err := inspection.Start(); err != nil {
		t.Fatal(err)
	}
	if err := inspection.Complete(""); err != nil {
		t.Fatal(err)
	}
	if err := inspection.SubmitForReview(); err != nil {
		t.Fatal(err)
	}
	if err := inspection.Approve("inspector-1"); err == nil {
		t.Fatal("inspector should not approve own inspection")
	}
	if inspection.Status() != types.InspectionPendingReview {
		t.Fatal("failed approval must not mutate status")
	}
}

func TestVerdictMonoidUsesStrongestSeverity(t *testing.T) {
	findings := []Finding{testFinding(types.SeverityAdvisory), testFinding(types.SeverityMinor), testFinding(types.SeverityMajor)}
	if got := ComputeVerdict(findings); got != types.VerdictMajor {
		t.Fatalf("expected major, got %s", got)
	}
	findings = append(findings, testFinding(types.SeverityCritical))
	if got := ComputeVerdict(findings); got != types.VerdictCritical {
		t.Fatalf("expected critical, got %s", got)
	}
	if got := ComputeVerdict(nil); got != types.VerdictPass {
		t.Fatalf("expected pass for no findings, got %s", got)
	}
}

func TestRejectedInspectionCreatesImmutablePriorRevision(t *testing.T) {
	inspection := newTestInspection(t)
	if err := inspection.Assign("inspector-1"); err != nil {
		t.Fatal(err)
	}
	if err := inspection.Start(); err != nil {
		t.Fatal(err)
	}
	if err := inspection.RecordFinding(testFinding(types.SeverityMajor)); err != nil {
		t.Fatal(err)
	}
	if err := inspection.Complete("needs correction"); err != nil {
		t.Fatal(err)
	}
	if err := inspection.SubmitForReview(); err != nil {
		t.Fatal(err)
	}
	if err := inspection.ReturnForRevision("reviewer-1", "photo evidence is insufficient"); err != nil {
		t.Fatal(err)
	}
	if inspection.Revision() != 2 || inspection.Status() != types.InspectionInProgress {
		t.Fatalf("expected revision 2 in progress, got revision %d status %s", inspection.Revision(), inspection.Status())
	}
	history := inspection.RevisionHistory()
	if len(history) != 1 || history[0].Number != 1 || history[0].Status != types.InspectionRejected {
		t.Fatalf("unexpected revision history: %#v", history)
	}
	if len(history[0].Findings) != 1 || history[0].Findings[0].Severity != types.SeverityMajor {
		t.Fatalf("prior finding not preserved: %#v", history[0].Findings)
	}
	if len(inspection.Findings()) != 0 {
		t.Fatal("new revision should start with no findings")
	}
	history[0].Findings[0].Notes = "mutated outside aggregate"
	if inspection.RevisionHistory()[0].Findings[0].Notes == "mutated outside aggregate" {
		t.Fatal("revision history was mutable through accessor")
	}
}

func TestFindingDefaultsInspectionAndDefensivelyCopiesEvidence(t *testing.T) {
	inspection := newTestInspection(t)
	if err := inspection.Assign("inspector-1"); err != nil {
		t.Fatal(err)
	}
	if err := inspection.Start(); err != nil {
		t.Fatal(err)
	}
	finding := testFinding(types.SeverityMinor)
	if err := inspection.RecordFinding(finding); err != nil {
		t.Fatal(err)
	}
	returned := inspection.Findings()
	if returned[0].InspectionID != inspection.ID() {
		t.Fatal("inspection id was not defaulted")
	}
	returned[0].EvidenceRefs[0] = "changed"
	if inspection.Findings()[0].EvidenceRefs[0] == "changed" {
		t.Fatal("evidence references were mutable through accessor")
	}
}
