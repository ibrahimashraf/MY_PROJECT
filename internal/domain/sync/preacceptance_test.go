package sync

import (
	"context"
	"errors"
	"testing"
	"time"
)

type rejectingPreAcceptancePolicy struct {
	calls int
}

func (p *rejectingPreAcceptancePolicy) ValidatePreAcceptance(context.Context, Transaction) error {
	p.calls++
	return errors.New("work package binding rejected")
}

func TestPreAcceptancePolicyRunsAfterVerificationAndBeforeAcceptance(t *testing.T) {
	processor, err := NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	device, authority := trustedDevice(t)
	processor.RegisterDevice(device)
	policy := &rejectingPreAcceptancePolicy{}
	processor.SetPreAcceptancePolicy(policy)
	at := time.Date(2026, 8, 13, 12, 30, 0, 0, time.UTC)

	tampered := signedTransaction("policy-tampered", 1, []byte(`{"finding":"pass"}`))
	tampered.Signature = "invalid"
	if result := processor.Submit(tampered, authority, at); result.Outcome != SecurityFailure {
		t.Fatalf("tampered outcome = %#v, want security failure", result)
	}
	if policy.calls != 0 {
		t.Fatalf("policy ran before signature verification: calls = %d", policy.calls)
	}

	transaction := signedTransaction("policy-rejected", 1, []byte(`{"finding":"pass"}`))
	if result := processor.Submit(transaction, authority, at); result.Outcome != Rejected {
		t.Fatalf("policy rejection outcome = %#v, want rejected", result)
	}
	if policy.calls != 1 {
		t.Fatalf("policy calls = %d, want 1", policy.calls)
	}

	processor.SetPreAcceptancePolicy(nil)
	if result := processor.Submit(transaction, authority, at); result.Outcome != Applied {
		t.Fatalf("rejected transaction was retained as accepted: %#v", result)
	}
}
