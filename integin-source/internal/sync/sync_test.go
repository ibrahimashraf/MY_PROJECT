package sync

import (
	"testing"
	"time"

	"integin/internal/domain/device_trust"
	"integin/internal/shared/types"
)

func trustedDevice(t *testing.T) (device_trust.Device, device_trust.AuthorityPackage) {
	t.Helper()
	device, err := device_trust.NewDevice("device-1", "tenant-1", "org-1", "user-1", "public-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	authority, err := device_trust.IssueAuthorityPackage(device, "authority-1", "secret", []string{"inspection-1"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return device, authority
}

func signedTransaction(id string, sequence uint64, payload []byte) Transaction {
	return SignTransaction(NewTransaction(id, "tenant-1", "device-1", "user-1", sequence, "FindingRecorded", payload), "secret")
}

func TestSyncAppliesAndDeduplicatesTransactions(t *testing.T) {
	processor, err := NewProcessor("secret")
	if err != nil {
		t.Fatal(err)
	}
	device, authority := trustedDevice(t)
	processor.RegisterDevice(device)
	at := time.Date(2026, 8, 13, 12, 30, 0, 0, time.UTC)
	transaction := signedTransaction("tx-1", 1, []byte(`{"finding":"pass"}`))
	applied := processor.Submit(transaction, authority, at)
	if applied.Outcome != Applied {
		t.Fatalf("expected applied, got %#v", applied)
	}
	duplicate := processor.Submit(transaction, authority, at)
	if duplicate.Outcome != Duplicate {
		t.Fatalf("expected duplicate, got %#v", duplicate)
	}
}

func TestSyncHoldsSequenceGapsAndAppliesNextSequence(t *testing.T) {
	processor, err := NewProcessor("secret")
	if err != nil {
		t.Fatal(err)
	}
	device, authority := trustedDevice(t)
	processor.RegisterDevice(device)
	at := time.Date(2026, 8, 13, 12, 30, 0, 0, time.UTC)
	gap := processor.Submit(signedTransaction("tx-2", 2, []byte("second")), authority, at)
	if gap.Outcome != Held || gap.ExpectedSequence != 1 {
		t.Fatalf("expected held sequence gap, got %#v", gap)
	}
	if len(processor.HeldTransactions()) != 1 {
		t.Fatal("held transaction was not retained")
	}
	first := processor.Submit(signedTransaction("tx-1", 1, []byte("first")), authority, at)
	if first.Outcome != Applied {
		t.Fatalf("expected first transaction applied, got %#v", first)
	}
}

func TestSyncRejectsTamperingConflictsAndSecurityFailures(t *testing.T) {
	processor, err := NewProcessor("secret")
	if err != nil {
		t.Fatal(err)
	}
	device, authority := trustedDevice(t)
	processor.RegisterDevice(device)
	at := time.Date(2026, 8, 13, 12, 30, 0, 0, time.UTC)
	transaction := signedTransaction("tx-1", 1, []byte("payload"))
	tampered := transaction
	tampered.Payload = []byte("changed")
	if result := processor.Submit(tampered, authority, at); result.Outcome != SecurityFailure {
		t.Fatalf("expected tamper security failure, got %#v", result)
	}
	if result := processor.Submit(transaction, authority, at); result.Outcome != Applied {
		t.Fatalf("expected original to apply, got %#v", result)
	}
	reused := signedTransaction("tx-1", 1, []byte("different"))
	if result := processor.Submit(reused, authority, at); result.Outcome != Conflict {
		t.Fatalf("expected conflict, got %#v", result)
	}
	device.Revoke("lost device")
	processor.RegisterDevice(device)
	if result := processor.Submit(signedTransaction("tx-2", 2, []byte("after revoke")), authority, at); result.Outcome != SecurityFailure {
		t.Fatalf("expected revoked-device security failure, got %#v", result)
	}
}

func TestSyncRejectsTenantAndSignatureMismatches(t *testing.T) {
	processor, err := NewProcessor("secret")
	if err != nil {
		t.Fatal(err)
	}
	device, authority := trustedDevice(t)
	processor.RegisterDevice(device)
	at := time.Date(2026, 8, 13, 12, 30, 0, 0, time.UTC)
	wrongTenant := signedTransaction("tx-1", 1, []byte("payload"))
	wrongTenant.TenantID = "tenant-2"
	wrongTenant.Signature = signTransaction(wrongTenant, "secret")
	if result := processor.Submit(wrongTenant, authority, at); result.Code != types.ErrTenantMismatch {
		t.Fatalf("expected tenant mismatch, got %#v", result)
	}
	invalidSignature := signedTransaction("tx-2", 1, []byte("payload"))
	invalidSignature.Signature = "bad"
	if result := processor.Submit(invalidSignature, authority, at); result.Outcome != SecurityFailure {
		t.Fatalf("expected signature security failure, got %#v", result)
	}
}
