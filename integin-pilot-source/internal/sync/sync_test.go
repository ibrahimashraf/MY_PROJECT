package sync

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
	"integin/internal/security"
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

func TestSyncDeviceBoundVerifyAcceptsAndAttributes(t *testing.T) {
	publicKey, privateKey, err := security.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	device, err := device_trust.NewDevice("device-keyed", "tenant-1", "org-1", "user-1", base64.StdEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	authority, err := device_trust.IssueAuthorityPackage(device, "authority-keyed", "secret", []string{"inspection-1"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewProcessor("secret")
	if err != nil {
		t.Fatal(err)
	}
	processor.RegisterDevice(device)
	tx := NewTransaction("tx-keyed", "tenant-1", "device-keyed", "user-1", 1, "FindingRecorded", []byte("payload"))
	tx, err = SignTransactionEd25519(tx, privateKey, security.DeviceKeyID(publicKey))
	if err != nil {
		t.Fatal(err)
	}
	result := processor.Submit(tx, authority, issuedAt.Add(20*time.Minute))
	if result.Outcome != Applied {
		t.Fatalf("expected device-bound transaction to apply, got %#v", result)
	}
	registered, ok := processor.devices["device-keyed"]
	if !ok || registered.UserID() != "user-1" || registered.TenantID() != "tenant-1" {
		t.Fatalf("device/inspector attribution not recoverable from verified artifact: %#v", registered)
	}
	if processor.HMACFallbackCount.Load() != 0 {
		t.Fatalf("device-bound verify must not use HMAC fallback, count=%d", processor.HMACFallbackCount.Load())
	}
}

func TestSyncRejectsForgedDeviceKey(t *testing.T) {
	publicKey, _, err := security.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	_, attackerKey, err := security.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	device, err := device_trust.NewDevice("device-forge", "tenant-1", "org-1", "user-1", base64.StdEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	authority, err := device_trust.IssueAuthorityPackage(device, "authority-forge", "secret", []string{"inspection-1"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewProcessor("secret")
	if err != nil {
		t.Fatal(err)
	}
	processor.RegisterDevice(device)
	tx := NewTransaction("tx-forge", "tenant-1", "device-forge", "user-1", 1, "FindingRecorded", []byte("payload"))
	tx.KeyID = security.DeviceKeyID(publicKey)
	tx.SignatureAlgorithm = "Ed25519"
	tx.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(attackerKey, []byte(canonicalTransaction(tx))))
	if result := processor.Submit(tx, authority, issuedAt.Add(20*time.Minute)); result.Outcome != SecurityFailure {
		t.Fatalf("expected forged key rejection, got %#v", result)
	}
}

func TestSyncRejectsUnknownDevice(t *testing.T) {
	processor, err := NewProcessor("secret")
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	authorityDevice, err := device_trust.NewDevice("authority-only", "tenant-1", "org-1", "user-1", "public-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := authorityDevice.Trust(); err != nil {
		t.Fatal(err)
	}
	authority, err := device_trust.IssueAuthorityPackage(authorityDevice, "authority-unreg", "secret", []string{"inspection-1"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tx := signedTransaction("tx-unreg", 1, []byte(`{"finding":"unknown-device"}`))
	result := processor.Submit(tx, authority, issuedAt.Add(time.Minute))
	if result.Code != types.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized for unknown device, got code=%s outcome=%s", result.Code, result.Outcome)
	}
}

func TestSyncLegacyHMACFallbackAcceptedWithCounter(t *testing.T) {
	processor, err := NewProcessor("secret")
	if err != nil {
		t.Fatal(err)
	}
	device, authority := trustedDevice(t)
	processor.RegisterDevice(device)
	at := time.Date(2026, 8, 13, 12, 30, 0, 0, time.UTC)
	tx := signedTransaction("tx-hmac", 1, []byte(`{"finding":"hmac-legacy"}`))
	result := processor.Submit(tx, authority, at)
	if result.Outcome != Applied {
		t.Fatalf("expected legacy HMAC transaction to apply, got %#v", result)
	}
	if processor.HMACFallbackCount.Load() != 1 {
		t.Fatalf("expected HMAC fallback counter to be 1, got %d", processor.HMACFallbackCount.Load())
	}
}

func TestSyncRejectsHMACWhenDeviceHasKeyMaterial(t *testing.T) {
	publicKey, _, err := security.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	device, err := device_trust.NewDevice("device-keyed-hmac", "tenant-1", "org-1", "user-1", base64.StdEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	authority, err := device_trust.IssueAuthorityPackage(device, "authority-keyed-hmac", "secret", []string{"inspection-1"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewProcessor("secret")
	if err != nil {
		t.Fatal(err)
	}
	processor.RegisterDevice(device)
	tx := signedTransaction("tx-hmac-keyed", 1, []byte("payload"))
	tx.DeviceID = "device-keyed-hmac"
	tx.Signature = signTransaction(tx, "secret")
	result := processor.Submit(tx, authority, issuedAt.Add(time.Minute))
	if result.Outcome != SecurityFailure {
		t.Fatalf("expected HMAC rejected for keyed device, got %#v", result)
	}
	if processor.HMACFallbackCount.Load() != 0 {
		t.Fatalf("HMAC fallback counter must not increment for rejected keyed device, got %d", processor.HMACFallbackCount.Load())
	}
}
