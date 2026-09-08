package workorder

import (
	"context"
	"errors"
	"testing"
	"time"
)

type submitTestRepo struct {
	serviceTestRepo
	submitCalls int
}

func (r *submitTestRepo) GetWorkOrder(context.Context, ActorContext, string) (WorkOrder, error) {
	order := validWorkOrder()
	order.ExecutionState = ExecutionInProgress
	return order, nil
}

func (r *submitTestRepo) SubmitPartial(context.Context, SubmitPartialCommand) (MutationReceipt, error) {
	r.submitCalls++
	return MutationReceipt{OperationID: "op-1", IdempotencyKey: "idem-1", TenantID: "tenant-a", WorkOrderID: "wo-1", Revision: 2, Status: ReceiptAccepted}, nil
}

type fakeGate struct {
	blocked         map[string]bool
	err             error
	checks          []string
	checkedTenantID string
	checkedOrgID    string
}

func (g *fakeGate) SubmissionBlocked(_ context.Context, tenantID, orgID, equipmentID string, _ time.Time) (bool, error) {
	g.checks = append(g.checks, equipmentID)
	g.checkedTenantID = tenantID
	g.checkedOrgID = orgID
	if g.err != nil {
		return false, g.err
	}
	return g.blocked[equipmentID], nil
}

func newSubmitService(t *testing.T, repo *submitTestRepo, gate CalibrationGate) Service {
	t.Helper()
	service, err := NewService(ServiceDependencies{Repository: repo, Transactions: serviceTestTx{repo: repo}, Authorizer: serviceTestAuth{}, CalibrationGate: gate})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func submitCommand(equipmentIDs []string) SubmitPartialCommand {
	return SubmitPartialCommand{
		Actor:         validActor(),
		Operation:     validOperation(),
		WorkOrderID:   "wo-1",
		AssignmentID:  "assignment-1",
		InspectionIDs: []string{"inspection-1"},
		EquipmentIDs:  equipmentIDs,
	}
}

func TestSubmitPartialBlocksEquipmentWithoutUnexpiredCalibration(t *testing.T) {
	repo := &submitTestRepo{}
	gate := &fakeGate{blocked: map[string]bool{"equip-1": true}}
	service := newSubmitService(t, repo, gate)

	_, err := service.SubmitPartial(context.Background(), submitCommand([]string{"equip-1"}))
	if !errors.Is(err, ErrCalibrationExpired) {
		t.Fatalf("expected ErrCalibrationExpired, got %v", err)
	}
	if repo.submitCalls != 0 {
		t.Fatalf("submit must never run for blocked equipment, got %d calls", repo.submitCalls)
	}
}

func TestSubmitPartialAllowsEquipmentWithUnexpiredCalibration(t *testing.T) {
	repo := &submitTestRepo{}
	gate := &fakeGate{blocked: map[string]bool{"equip-1": false, "equip-2": false}}
	service := newSubmitService(t, repo, gate)

	receipt, err := service.SubmitPartial(context.Background(), submitCommand([]string{"equip-1", "equip-2"}))
	if err != nil {
		t.Fatalf("unexpected submit rejection: %v", err)
	}
	if repo.submitCalls != 1 {
		t.Fatalf("expected exactly one submit, got %d", repo.submitCalls)
	}
	if receipt.Status != ReceiptAccepted {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	if gate.checkedTenantID != "tenant-a" || gate.checkedOrgID != "org-a" {
		t.Fatalf("gate must be evaluated under order scope, got tenant=%q org=%q", gate.checkedTenantID, gate.checkedOrgID)
	}
}

func TestSubmitPartialWithoutEquipmentOrGateSubmitsAsBefore(t *testing.T) {
	repo := &submitTestRepo{}
	service := newSubmitService(t, repo, nil)

	if _, err := service.SubmitPartial(context.Background(), submitCommand(nil)); err != nil {
		t.Fatalf("empty EquipmentIDs with nil gate must submit, got %v", err)
	}
	if repo.submitCalls != 1 {
		t.Fatalf("expected one submit, got %d", repo.submitCalls)
	}

	repo.submitCalls = 0
	if _, err := service.SubmitPartial(context.Background(), submitCommand([]string{"equip-1"})); err != nil {
		t.Fatalf("non-empty EquipmentIDs with nil gate must submit, got %v", err)
	}
	if repo.submitCalls != 1 {
		t.Fatalf("expected one submit, got %d", repo.submitCalls)
	}
}

func TestSubmitPartialFailsClosedOnGateError(t *testing.T) {
	repo := &submitTestRepo{}
	gateErr := errors.New("calibration store unreachable")
	gate := &fakeGate{err: gateErr}
	service := newSubmitService(t, repo, gate)

	_, err := service.SubmitPartial(context.Background(), submitCommand([]string{"equip-1"}))
	if !errors.Is(err, gateErr) {
		t.Fatalf("expected gate error to propagate, got %v", err)
	}
	if repo.submitCalls != 0 {
		t.Fatalf("submit must never run on gate failure, got %d calls", repo.submitCalls)
	}
}

func TestSubmitPartialFirstBlockedEquipmentShortCircuits(t *testing.T) {
	repo := &submitTestRepo{}
	gate := &fakeGate{blocked: map[string]bool{"equip-1": true, "equip-2": false}}
	service := newSubmitService(t, repo, gate)

	_, err := service.SubmitPartial(context.Background(), submitCommand([]string{"equip-1", "equip-2"}))
	if !errors.Is(err, ErrCalibrationExpired) {
		t.Fatalf("expected ErrCalibrationExpired, got %v", err)
	}
	if len(gate.checks) != 1 || gate.checks[0] != "equip-1" {
		t.Fatalf("expected short-circuit after first blocked equipment, checks=%v", gate.checks)
	}
}