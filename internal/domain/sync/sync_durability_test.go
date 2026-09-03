package sync

import (
	"context"
	"testing"
	"time"

	"integin/internal/syncstate"
)

func TestSyncInvalidEnvelopeDoesNotProbeDurableReceipt(t *testing.T) {
	state := &receiptLookupSpy{}
	processor, err := NewProcessorWithState(map[string]string{"default": "secret"}, state)
	if err != nil {
		t.Fatal(err)
	}
	device, authority := trustedDevice(t)
	processor.RegisterDevice(device)
	transaction := signedTransaction("tx-state-auth-order", 1, []byte("payload"))
	transaction.Signature = "invalid"
	if result := processor.Submit(transaction, authority, time.Date(2026, 8, 13, 12, 30, 0, 0, time.UTC)); result.Outcome != SecurityFailure {
		t.Fatalf("expected security failure, got %#v", result)
	}
	if state.receiptLookups != 0 {
		t.Fatalf("invalid envelope performed %d durable receipt lookups", state.receiptLookups)
	}
}

type receiptLookupSpy struct {
	receiptLookups int
}

func (s *receiptLookupSpy) GetLastAcceptedSequence(context.Context, string, string) (uint64, error) {
	return 0, syncstate.ErrNotFound
}

func (s *receiptLookupSpy) SaveReceipt(context.Context, syncstate.Receipt) error {
	return nil
}

func (s *receiptLookupSpy) GetReceipt(context.Context, string, string) (syncstate.Receipt, error) {
	s.receiptLookups++
	return syncstate.Receipt{}, syncstate.ErrNotFound
}

func (s *receiptLookupSpy) SaveHeld(context.Context, syncstate.HeldTransaction) error {
	return nil
}

func (s *receiptLookupSpy) ListHeld(context.Context, string, string) ([]syncstate.HeldTransaction, error) {
	return nil, nil
}
