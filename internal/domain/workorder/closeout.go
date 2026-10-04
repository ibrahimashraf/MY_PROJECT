package workorder

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrAssignmentNotActive     = errors.New("cannot close an assignment that is not active")
	ErrAssignmentMismatch      = errors.New("assignment does not belong to target work order")
	ErrInspectorMismatch       = errors.New("summary inspector does not match assigned inspector")
	ErrWorkOrderNotCompleted   = errors.New("cannot release commercial invoice: execution scope is not fully completed")
	ErrCommercialNotReady      = errors.New("work order is not in ready_for_office_confirmation commercial state")
	ErrUnauthorizedBillingRole = errors.New("commercial release requires authorized billing officer identity")
)

// InspectorScopeSummary records the individual work and timesheet summary
// submitted by an inspector upon completing their assigned scope.
type InspectorScopeSummary struct {
	SummaryID       string    `json:"summary_id"`
	AssignmentID    string    `json:"assignment_id"`
	WorkOrderID     string    `json:"work_order_id"`
	InspectorID     string    `json:"inspector_id"`
	ScopeItemIDs    []string  `json:"scope_item_ids"`
	CompletedCount  int       `json:"completed_count"`
	BlockedCount    int       `json:"blocked_count"`
	TimesheetHours  float64   `json:"timesheet_hours"`
	TravelHours     float64   `json:"travel_hours"`
	Notes           string    `json:"notes,omitempty"`
	ClosedAt        time.Time `json:"closed_at"`
}

// CombinedCompletionReport aggregates all individual inspector scope summaries
// across a shared work order once all operational assignments are resolved.
type CombinedCompletionReport struct {
	WorkOrderID             string                  `json:"work_order_id"`
	JobNumber               string                  `json:"job_number"`
	AllAssignmentsCompleted bool                    `json:"all_assignments_completed"`
	TotalScopeCount         int                     `json:"total_scope_count"`
	CompletedScopeCount     int                     `json:"completed_scope_count"`
	BlockedScopeCount       int                     `json:"blocked_scope_count"`
	TotalBillableHours      float64                 `json:"total_billable_hours"`
	Summaries               []InspectorScopeSummary `json:"summaries"`
	CompiledAt              time.Time               `json:"compiled_at"`
}

// InvoiceDraft represents an authorized commercial billing draft derived from
// approved service lines, strictly isolated from technical authority and certificates.
type InvoiceDraft struct {
	InvoiceID          string    `json:"invoice_id"`
	WorkOrderID        string    `json:"work_order_id"`
	ClientID           string    `json:"client_id"`
	JobNumber          string    `json:"job_number"`
	TotalBillableHours float64   `json:"total_billable_hours"`
	Status             string    `json:"status"`
	ReleasedBy         string    `json:"released_by"`
	ReleasedAt         time.Time `json:"released_at"`
}

// CloseInspectorScope completes a specific inspector's assigned scope.
// It generates their personal timesheet/work summary without prematurely
// closing the shared work order if other inspectors still have active assignments.
func CloseInspectorScope(
	wo WorkOrder,
	targetAssignment Assignment,
	summary InspectorScopeSummary,
	allAssignments []Assignment,
) (WorkOrder, Assignment, error) {
	if err := wo.ValidateIdentity(); err != nil {
		return WorkOrder{}, Assignment{}, err
	}
	if targetAssignment.WorkOrderID != wo.ID || targetAssignment.TenantID != wo.TenantID {
		return WorkOrder{}, Assignment{}, ErrAssignmentMismatch
	}
	if targetAssignment.State != AssignmentActive {
		return WorkOrder{}, Assignment{}, ErrAssignmentNotActive
	}
	if summary.InspectorID != targetAssignment.InspectorID {
		return WorkOrder{}, Assignment{}, ErrInspectorMismatch
	}

	// 1. Mark target assignment as completed
	closedAssignment := targetAssignment
	closedAssignment.State = AssignmentCompleted
	closedAssignment.Revision++
	now := time.Now().UTC()
	closedAssignment.EffectiveUntil = &now

	// 2. Evaluate remaining assignments on the shared work order
	allResolved := true
	for _, a := range allAssignments {
		if a.ID == targetAssignment.ID {
			continue // This one is now closed
		}
		if a.State == AssignmentActive {
			allResolved = false
			break
		}
	}

	updatedWO := wo
	updatedWO.Revision++

	if allResolved {
		// All inspectors have closed their scope: order is ready for technical review / office commercial review
		updatedWO.ExecutionState = ExecutionCompleted
		updatedWO.CommercialState = CommercialReadyForOfficeReview
	} else {
		// Other inspectors still working: order remains in progress or partially submitted
		if updatedWO.ExecutionState == ExecutionReady || updatedWO.ExecutionState == ExecutionAssigned {
			updatedWO.ExecutionState = ExecutionInProgress
		} else {
			updatedWO.ExecutionState = ExecutionPartiallySubmitted
		}
	}

	return updatedWO, closedAssignment, nil
}

// CompileCombinedReport aggregates individual inspector scope summaries into
// a combined work-order completion report for office governance.
func CompileCombinedReport(wo WorkOrder, summaries []InspectorScopeSummary) CombinedCompletionReport {
	totalScope := 0
	completedScope := 0
	blockedScope := 0
	totalHours := 0.0

	for _, s := range summaries {
		totalScope += len(s.ScopeItemIDs)
		completedScope += s.CompletedCount
		blockedScope += s.BlockedCount
		totalHours += (s.TimesheetHours + s.TravelHours)
	}

	return CombinedCompletionReport{
		WorkOrderID:             wo.ID,
		JobNumber:               wo.JobNumber,
		AllAssignmentsCompleted: wo.ExecutionState == ExecutionCompleted,
		TotalScopeCount:         totalScope,
		CompletedScopeCount:     completedScope,
		BlockedScopeCount:       blockedScope,
		TotalBillableHours:      totalHours,
		Summaries:               append([]InspectorScopeSummary(nil), summaries...),
		CompiledAt:              time.Now().UTC(),
	}
}

// ReleaseCommercialForInvoice authorizes commercial billing release for a completed
// work order. In accordance with the INTEGIN operating model, the resulting invoice
// references the work order and combined summary, but CANNOT alter or override
// any technical findings, evidence, reviews, or certificate states.
func ReleaseCommercialForInvoice(
	wo WorkOrder,
	report CombinedCompletionReport,
	authorizedActorID string,
) (WorkOrder, InvoiceDraft, error) {
	if strings.TrimSpace(authorizedActorID) == "" {
		return WorkOrder{}, InvoiceDraft{}, ErrUnauthorizedBillingRole
	}
	if wo.ExecutionState != ExecutionCompleted {
		return WorkOrder{}, InvoiceDraft{}, ErrWorkOrderNotCompleted
	}
	if wo.CommercialState != CommercialReadyForOfficeReview {
		return WorkOrder{}, InvoiceDraft{}, ErrCommercialNotReady
	}
	if report.WorkOrderID != wo.ID {
		return WorkOrder{}, InvoiceDraft{}, fmt.Errorf("%w: report work order mismatch", ErrAssignmentMismatch)
	}

	updatedWO := wo
	updatedWO.CommercialState = CommercialReleasedForInvoice
	updatedWO.Revision++

	now := time.Now().UTC()
	invoice := InvoiceDraft{
		InvoiceID:          fmt.Sprintf("INV-%s-%d", wo.JobNumber, now.Unix()),
		WorkOrderID:        wo.ID,
		ClientID:           wo.ClientID,
		JobNumber:          wo.JobNumber,
		TotalBillableHours: report.TotalBillableHours,
		Status:             "DRAFT",
		ReleasedBy:         authorizedActorID,
		ReleasedAt:         now,
	}

	return updatedWO, invoice, nil
}
