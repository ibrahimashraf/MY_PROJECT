package workorder

import (
	"errors"
	"testing"
)

func TestSubmitPartialCommandRejectsCompletedExecution(t *testing.T) {
	order := validWorkOrder()
	order.ExecutionState = ExecutionCompleted
	command := SubmitPartialCommand{Actor: validActor(), Operation: validOperation(), WorkOrderID: order.ID, AssignmentID: "assignment-1", InspectionIDs: []string{"inspection-1"}}

	if err := ValidateSubmitPartialCommand(command, order); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected completed order rejection, got %v", err)
	}
}
