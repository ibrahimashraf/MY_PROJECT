package workorder

import (
	"errors"
	"fmt"
	"strings"
)

// HandoverState models the D7-5 handover lifecycle.
// Planned persistence: migrations/0012_work_order_handover.sql with CHECK
// handover_state IN ('requested','approved','transferred').
// FKs from_assignment_id / to_assignment_id reference work_order_assignment
// (tenant_id, organization_id, id) ON DELETE RESTRICT — matching
// migrations/0009_work_order_persistence.sql:117 inspection_record_assignment_fk
// semantics. Assignments cannot be deleted while a handover references them.
type HandoverState string

const (
	HandoverRequested   HandoverState = "requested"
	HandoverApproved    HandoverState = "approved"
	HandoverTransferred HandoverState = "transferred"
)

var ErrInvalidHandover = errors.New("work-order handover is incomplete")

// Handover captures a controlled responsibility transfer between two
// work_order_assignment rows. Tenant and organization are server-derived
// and must match the owning WorkOrder and both assignments.
type Handover struct {
	ID               string        `json:"handover_id"`
	TenantID         string        `json:"tenant_id"`
	OrganizationID   string        `json:"organization_id"`
	WorkOrderID      string        `json:"work_order_id"`
	FromAssignmentID string        `json:"from_assignment_id"`
	ToAssignmentID   string        `json:"to_assignment_id"`
	State            HandoverState `json:"state"`
	Revision         int64         `json:"revision"`
}

// HandoverCommand is the application boundary for handover mutations.
// Actor and Operation are server-derived; WorkOrderID is the aggregate
// under handover; Handover carries the requested handover identity and
// from/to assignment FKs.
type HandoverCommand struct {
	Actor       ActorContext
	Operation   OperationMeta
	WorkOrderID string
	Handover    Handover
}

func isValidHandoverState(s HandoverState) bool {
	switch s {
	case HandoverRequested, HandoverApproved, HandoverTransferred:
		return true
	default:
		return false
	}
}

// CanTransitionHandover enforces the ordered handover lifecycle:
// requested -> approved -> transferred. Same-state is idempotent.
func CanTransitionHandover(from, to HandoverState) bool {
	if from == to {
		return true
	}
	allowed := map[HandoverState]map[HandoverState]bool{
		HandoverRequested: {HandoverApproved: true},
		HandoverApproved:  {HandoverTransferred: true},
	}
	return allowed[from][to]
}

// ValidateFor checks handover identity, tenant/org binding, FK presence,
// state, revision, and that both assignments belong to the same work order.
// It does not check actor authority — that belongs to the Authorizer.
func (h Handover) ValidateFor(order WorkOrder, fromAssignment, toAssignment Assignment) error {
	if strings.TrimSpace(h.ID) == "" || strings.TrimSpace(h.TenantID) == "" || strings.TrimSpace(h.OrganizationID) == "" || strings.TrimSpace(h.WorkOrderID) == "" {
		return fmt.Errorf("%w: handover identity", ErrInvalidHandover)
	}
	if strings.TrimSpace(h.FromAssignmentID) == "" || strings.TrimSpace(h.ToAssignmentID) == "" {
		return fmt.Errorf("%w: from/to assignment is required", ErrInvalidHandover)
	}
	if h.FromAssignmentID == h.ToAssignmentID {
		return fmt.Errorf("%w: from and to assignment must differ", ErrInvalidHandover)
	}
	if h.TenantID != order.TenantID || h.OrganizationID != order.OrganizationID || h.WorkOrderID != order.ID {
		return fmt.Errorf("%w: tenant or work-order mismatch", ErrInvalidHandover)
	}
	if !isValidHandoverState(h.State) {
		return fmt.Errorf("%w: invalid handover state", ErrInvalidHandover)
	}
	if h.Revision < 1 {
		return fmt.Errorf("%w: revision must be positive", ErrInvalidHandover)
	}
	// FK integrity: handover FK columns must match the supplied assignments
	// and those assignments must themselves be valid for the work order.
	// Persistence must enforce ON DELETE RESTRICT on both FKs
	// (tenant_id, organization_id, from_assignment_id) and (tenant_id, organization_id, to_assignment_id)
	// referencing work_order_assignment — mirroring 0009:117.
	if h.FromAssignmentID != fromAssignment.ID {
		return fmt.Errorf("%w: from assignment mismatch", ErrInvalidHandover)
	}
	if h.ToAssignmentID != toAssignment.ID {
		return fmt.Errorf("%w: to assignment mismatch", ErrInvalidHandover)
	}
	if err := fromAssignment.ValidateFor(order); err != nil {
		return fmt.Errorf("%w: from assignment invalid: %v", ErrInvalidHandover, err)
	}
	if err := toAssignment.ValidateFor(order); err != nil {
		return fmt.Errorf("%w: to assignment invalid: %v", ErrInvalidHandover, err)
	}
	if fromAssignment.TenantID != h.TenantID || fromAssignment.OrganizationID != h.OrganizationID {
		return fmt.Errorf("%w: from assignment tenant mismatch", ErrInvalidHandover)
	}
	if toAssignment.TenantID != h.TenantID || toAssignment.OrganizationID != h.OrganizationID {
		return fmt.Errorf("%w: to assignment tenant mismatch", ErrInvalidHandover)
	}
	return nil
}

// ValidateHandoverCommand validates actor, operation, work-order binding,
// and the embedded handover including FK to work_order_assignment with
// ON DELETE RESTRICT semantics.
func ValidateHandoverCommand(command HandoverCommand, order WorkOrder, fromAssignment, toAssignment Assignment) error {
	if err := validateActorOperation(command.Actor, command.Operation); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidHandover, err)
	}
	if command.Actor.TenantID != order.TenantID || command.Actor.OrganizationID != order.OrganizationID {
		return fmt.Errorf("%w: actor tenant or organization mismatch", ErrInvalidHandover)
	}
	if strings.TrimSpace(command.WorkOrderID) == "" || command.WorkOrderID != order.ID {
		return fmt.Errorf("%w: work-order mismatch", ErrInvalidHandover)
	}
	if strings.TrimSpace(command.WorkOrderID) != strings.TrimSpace(command.Handover.WorkOrderID) {
		return fmt.Errorf("%w: handover work-order mismatch", ErrInvalidHandover)
	}
	if command.Actor.TenantID != command.Handover.TenantID || command.Actor.OrganizationID != command.Handover.OrganizationID {
		return fmt.Errorf("%w: actor/handover tenant mismatch", ErrInvalidHandover)
	}
	return command.Handover.ValidateFor(order, fromAssignment, toAssignment)
}
