package workorder

import (
	"context"
	"errors"
	"testing"
)

type mismatchServiceRepo struct {
	serviceTestRepo
}

func (r *mismatchServiceRepo) CreateRequest(context.Context, CreateRequestCommand) (MutationReceipt, error) {
	return MutationReceipt{}, errors.New("idempotency payload mismatch")
}

func TestApplicationServiceLetsRepositoryRejectSameKeyWithDifferentPayload(t *testing.T) {
	repo := &mismatchServiceRepo{serviceTestRepo: serviceTestRepo{found: true, receipt: MutationReceipt{OperationID: "old-op", IdempotencyKey: "idem-1", TenantID: "tenant-a", WorkOrderID: "wo-1", Revision: 1, Status: ReceiptAccepted}}}
	service, err := NewService(ServiceDependencies{Repository: repo, Transactions: serviceTestTx{repo: repo}, Authorizer: serviceTestAuth{}})
	if err != nil {
		t.Fatal(err)
	}
	command := CreateRequestCommand{Actor: validActor(), Operation: validOperation(), WorkOrder: validWorkOrder()}
	_, err = service.CreateRequest(context.Background(), command)
	if err == nil {
		t.Fatal("expected repository idempotency mismatch to be propagated")
	}
}
