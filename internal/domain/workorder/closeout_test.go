package workorder

import (
	"testing"
	"time"
)

func TestCloseInspectorScope_MultiInspectorSharedOrder(t *testing.T) {
	wo := WorkOrder{
		ID:               "wo-offshore-100",
		JobNumber:        "JOB-2026-001",
		TenantID:         "tenant-aramco",
		OrganizationID:   "org-rig-7",
		ClientID:         "client-aramco",
		RequestState:     RequestAccepted,
		ExecutionState:   ExecutionAssigned,
		CommercialState:  CommercialNotReady,
		CertificateState: CertificateNotStarted,
		Revision:         1,
	}

	assignAlice := Assignment{
		ID:             "assign-alice-01",
		TenantID:       wo.TenantID,
		OrganizationID: wo.OrganizationID,
		WorkOrderID:    wo.ID,
		InspectorID:    "inspector-alice",
		ScopeItemIDs:   []string{"scope-hook-1", "scope-hook-2"},
		State:          AssignmentActive,
		Revision:       1,
		EffectiveFrom:  time.Now().UTC(),
	}

	assignBob := Assignment{
		ID:             "assign-bob-02",
		TenantID:       wo.TenantID,
		OrganizationID: wo.OrganizationID,
		WorkOrderID:    wo.ID,
		InspectorID:    "inspector-bob",
		ScopeItemIDs:   []string{"scope-shackle-1", "scope-shackle-2", "scope-shackle-3"},
		State:          AssignmentActive,
		Revision:       1,
		EffectiveFrom:  time.Now().UTC(),
	}

	allAssignments := []Assignment{assignAlice, assignBob}

	// 1. Inspector Alice closes her assigned scope
	summaryAlice := InspectorScopeSummary{
		SummaryID:      "sum-alice-01",
		AssignmentID:   assignAlice.ID,
		WorkOrderID:    wo.ID,
		InspectorID:    "inspector-alice",
		ScopeItemIDs:   assignAlice.ScopeItemIDs,
		CompletedCount: 2,
		BlockedCount:   0,
		TimesheetHours: 4.5,
		TravelHours:    1.0,
		Notes:          "Hooks inspected and proof load checked",
		ClosedAt:       time.Now().UTC(),
	}

	woAfterAlice, closedAlice, err := CloseInspectorScope(wo, assignAlice, summaryAlice, allAssignments)
	if err != nil {
		t.Fatalf("CloseInspectorScope Alice: %v", err)
	}

	// Alice's assignment is completed
	if closedAlice.State != AssignmentCompleted {
		t.Errorf("Alice assignment expected completed, got %s", closedAlice.State)
	}

	// SHARED WORK ORDER MUST NOT BE CLOSED: Bob is still working!
	if woAfterAlice.ExecutionState == ExecutionCompleted {
		t.Fatalf("CRITICAL GOVERNANCE FAILURE: Shared work order closed prematurely while Bob is active!")
	}
	if woAfterAlice.ExecutionState != ExecutionInProgress {
		t.Errorf("expected execution state in_progress, got %s", woAfterAlice.ExecutionState)
	}
	if woAfterAlice.CommercialState != CommercialNotReady {
		t.Errorf("expected commercial state not_ready, got %s", woAfterAlice.CommercialState)
	}

	// 2. Inspector Bob closes his assigned scope
	allAssignmentsAfterAlice := []Assignment{closedAlice, assignBob}
	summaryBob := InspectorScopeSummary{
		SummaryID:      "sum-bob-02",
		AssignmentID:   assignBob.ID,
		WorkOrderID:    wo.ID,
		InspectorID:    "inspector-bob",
		ScopeItemIDs:   assignBob.ScopeItemIDs,
		CompletedCount: 3,
		BlockedCount:   0,
		TimesheetHours: 5.0,
		TravelHours:    1.0,
		Notes:          "Shackles magnaflux tested",
		ClosedAt:       time.Now().UTC(),
	}

	woAfterBob, closedBob, err := CloseInspectorScope(woAfterAlice, assignBob, summaryBob, allAssignmentsAfterAlice)
	if err != nil {
		t.Fatalf("CloseInspectorScope Bob: %v", err)
	}

	// Bob's assignment is completed
	if closedBob.State != AssignmentCompleted {
		t.Errorf("Bob assignment expected completed, got %s", closedBob.State)
	}

	// ALL ASSIGNMENTS COMPLETED: Work order transitions to ExecutionCompleted & CommercialReadyForOfficeReview!
	if woAfterBob.ExecutionState != ExecutionCompleted {
		t.Errorf("expected execution state completed, got %s", woAfterBob.ExecutionState)
	}
	if woAfterBob.CommercialState != CommercialReadyForOfficeReview {
		t.Errorf("expected commercial state ready_for_office_confirmation, got %s", woAfterBob.CommercialState)
	}

	// 3. Compile Combined Completion Report
	combinedReport := CompileCombinedReport(woAfterBob, []InspectorScopeSummary{summaryAlice, summaryBob})
	if !combinedReport.AllAssignmentsCompleted {
		t.Error("expected AllAssignmentsCompleted to be true")
	}
	if combinedReport.TotalScopeCount != 5 {
		t.Errorf("expected total scope count 5, got %d", combinedReport.TotalScopeCount)
	}
	if combinedReport.TotalBillableHours != 11.5 { // (4.5+1.0) + (5.0+1.0) = 11.5
		t.Errorf("expected 11.5 billable hours, got %f", combinedReport.TotalBillableHours)
	}

	// 4. Commercial Release for Invoicing
	woReleased, invoice, err := ReleaseCommercialForInvoice(woAfterBob, combinedReport, "billing-officer-sarah")
	if err != nil {
		t.Fatalf("ReleaseCommercialForInvoice: %v", err)
	}

	if woReleased.CommercialState != CommercialReleasedForInvoice {
		t.Errorf("expected commercial state released_for_invoice, got %s", woReleased.CommercialState)
	}
	if invoice.TotalBillableHours != 11.5 {
		t.Errorf("expected invoice total billable hours 11.5, got %f", invoice.TotalBillableHours)
	}
	if invoice.ReleasedBy != "billing-officer-sarah" {
		t.Errorf("expected released by billing-officer-sarah, got %s", invoice.ReleasedBy)
	}

	// TECHNICAL ISOLATION CHECK: Certificate state remains untampered
	if woReleased.CertificateState != CertificateNotStarted {
		t.Errorf("CRITICAL LEAK: commercial release mutated certificate state: %s", woReleased.CertificateState)
	}
}

func TestCloseoutValidationFailures(t *testing.T) {
	wo := WorkOrder{
		ID:               "wo-101",
		JobNumber:        "JOB-101",
		TenantID:         "t1",
		OrganizationID:   "org1",
		ClientID:         "c1",
		ExecutionState:   ExecutionInProgress,
		CommercialState:  CommercialNotReady,
		CertificateState: CertificateNotStarted,
		Revision:         1,
	}

	assign := Assignment{
		ID:             "a-1",
		TenantID:       wo.TenantID,
		OrganizationID: wo.OrganizationID,
		WorkOrderID:    wo.ID,
		InspectorID:    "inspector-1",
		ScopeItemIDs:   []string{"s-1"},
		State:          AssignmentActive,
		Revision:       1,
		EffectiveFrom:  time.Now().UTC(),
	}

	// 1. Mismatched inspector in summary
	badSummary := InspectorScopeSummary{
		InspectorID: "different-inspector",
	}
	_, _, err := CloseInspectorScope(wo, assign, badSummary, []Assignment{assign})
	if err != ErrInspectorMismatch {
		t.Errorf("expected ErrInspectorMismatch, got: %v", err)
	}

	// 2. Already completed assignment cannot be closed again
	assignCompleted := assign
	assignCompleted.State = AssignmentCompleted
	_, _, err = CloseInspectorScope(wo, assignCompleted, InspectorScopeSummary{InspectorID: "inspector-1"}, []Assignment{assignCompleted})
	if err != ErrAssignmentNotActive {
		t.Errorf("expected ErrAssignmentNotActive, got: %v", err)
	}

	// 3. Premature commercial release
	rep := CombinedCompletionReport{WorkOrderID: wo.ID}
	_, _, err = ReleaseCommercialForInvoice(wo, rep, "billing-user")
	if err != ErrWorkOrderNotCompleted {
		t.Errorf("expected ErrWorkOrderNotCompleted, got: %v", err)
	}
}
