package workorder

import (
	"context"
	"errors"
	"testing"
)

type invalidAssignStateRepo struct {
	serviceTestRepo
}

func (invalidAssignStateRepo) GetWorkOrder(context.Context, ActorContext, string) (WorkOrder, error) {
	order := validWorkOrder()
	order.ExecutionState = ExecutionCompleted
	return order, nil
}

func TestApplicationServiceAssignScopeRejectsInvalidExecutionState(t *testing.T) {
	repo := &invalidAssignStateRepo{}
	service, err := NewService(ServiceDependencies{
		Repository:   repo,
		Transactions: serviceTestTx{repo: repo},
		Authorizer:   serviceTestAuth{},
	})
	if err != nil {
		t.Fatal(err)
	}

	command := AssignScopeCommand{
		Actor:       validActor(),
		Operation:   validOperation(),
		WorkOrderID: "wo-1",
		Assignment: Assignment{
			ID:             "assignment-1",
			TenantID:       "tenant-a",
			OrganizationID: "org-a",
			WorkOrderID:    "wo-1",
			InspectorID:    "inspector-1",
			ScopeItemIDs:   []string{"scope-1"},
			State:          AssignmentActive,
			Revision:       1,
			EffectiveFrom:  testTime(),
		},
	}

	_, err = service.AssignScope(context.Background(), command)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("assigning a completed work order returned %v, want %v", err, ErrInvalidTransition)
	}
}
