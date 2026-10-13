package sync

import (
	"context"
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
	return SignTransaction(NewTransaction(id, "tenant-1", "device-1", "user-1", sequence, "FindingRecorded", payload), "secret", "default")
}

func TestSyncAppliesAndDeduplicatesTransactions(t *testing.T) {
	processor, err := NewProcessor(map[string]string{"default": "secret"})
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
	stale := processor.Submit(signedTransaction("tx-stale", 1, []byte(`{"finding":"stale"}`)), authority, at)
	if stale.Outcome != Conflict {
		t.Fatalf("expected stale sequence conflict, got %#v", stale)
	}
}

func TestSyncHoldsSequenceGapsAndAppliesNextSequence(t *testing.T) {
	processor, err := NewProcessor(map[string]string{"default": "secret"})
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
	resumed := processor.Submit(signedTransaction("tx-2", 2, []byte("second")), authority, at)
	if resumed.Outcome != Applied {
		t.Fatalf("expected held transaction to resume as applied, got %#v", resumed)
	}
	if len(processor.HeldTransactions()) != 0 {
		t.Fatal("applied held transaction was not removed")
	}
	if duplicate := processor.Submit(signedTransaction("tx-2", 2, []byte("second")), authority, at); duplicate.Outcome != Duplicate {
		t.Fatalf("expected resumed transaction replay to be duplicate, got %#v", duplicate)
	}
}

func TestSyncDrainHeldCascadesConsecutiveSequences(t *testing.T) {
	processor, err := NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	device, authority := trustedDevice(t)
	processor.RegisterDevice(device)
	at := time.Date(2026, 8, 13, 12, 30, 0, 0, time.UTC)

	// Submit out of order: tx-3 (seq 3) and tx-2 (seq 2)
	gap3 := processor.Submit(signedTransaction("tx-3", 3, []byte("third")), authority, at)
	if gap3.Outcome != Held || gap3.ExpectedSequence != 1 {
		t.Fatalf("expected tx-3 held, got %#v", gap3)
	}
	gap2 := processor.Submit(signedTransaction("tx-2", 2, []byte("second")), authority, at)
	if gap2.Outcome != Held || gap2.ExpectedSequence != 1 {
		t.Fatalf("expected tx-2 held, got %#v", gap2)
	}
	if len(processor.HeldTransactions()) != 2 {
		t.Fatalf("expected 2 held transactions, got %d", len(processor.HeldTransactions()))
	}

	// Submit tx-1 (seq 1)
	first := processor.Submit(signedTransaction("tx-1", 1, []byte("first")), authority, at)
	if first.Outcome != Applied {
		t.Fatalf("expected tx-1 applied, got %#v", first)
	}

	// Now DrainHeld should cascade drain tx-2 (seq 2) then tx-3 (seq 3)
	drained := processor.DrainHeld(context.Background(), "tenant-1", "device-1", at)
	if len(drained) != 2 {
		t.Fatalf("expected 2 transactions drained, got %d (%#v)", len(drained), drained)
	}
	if drained[0].TransactionID != "tx-2" || drained[0].Outcome != Applied || drained[0].ExpectedSequence != 2 {
		t.Fatalf("expected tx-2 drained as seq 2 applied, got %#v", drained[0])
	}
	if drained[1].TransactionID != "tx-3" || drained[1].Outcome != Applied || drained[1].ExpectedSequence != 3 {
		t.Fatalf("expected tx-3 drained as seq 3 applied, got %#v", drained[1])
	}
	if len(processor.HeldTransactions()) != 0 {
		t.Fatalf("expected 0 held transactions remaining, got %d", len(processor.HeldTransactions()))
	}

	// Further replay should be duplicate
	if dup := processor.Submit(signedTransaction("tx-3", 3, []byte("third")), authority, at); dup.Outcome != Duplicate {
		t.Fatalf("expected tx-3 duplicate after drain, got %#v", dup)
	}
}

func TestSyncRejectsTamperingConflictsAndSecurityFailures(t *testing.T) {
	processor, err := NewProcessor(map[string]string{"default": "secret"})
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
	forgedReplay := transaction
	forgedReplay.Signature = "invalid"
	if result := processor.Submit(forgedReplay, authority, at); result.Outcome != SecurityFailure {
		t.Fatalf("expected invalid replay to fail authentication rather than deduplicate, got %#v", result)
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
	processor, err := NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	device, authority := trustedDevice(t)
	processor.RegisterDevice(device)
	at := time.Date(2026, 8, 13, 12, 30, 0, 0, time.UTC)
	wrongTenant := signedTransaction("tx-1", 1, []byte("payload"))
	wrongTenant.TenantID = "tenant-2"
	wrongTenant.Signature = SignTransaction(wrongTenant, "secret", "default").Signature
	if result := processor.Submit(wrongTenant, authority, at); result.Code != types.ErrTenantMismatch {
		t.Fatalf("expected tenant mismatch, got %#v", result)
	}
	invalidSignature := signedTransaction("tx-2", 1, []byte("payload"))
	invalidSignature.Signature = "bad"
	if result := processor.Submit(invalidSignature, authority, at); result.Outcome != SecurityFailure {
		t.Fatalf("expected signature security failure, got %#v", result)
	}
	unknownAlgorithm := signedTransaction("tx-3", 1, []byte("payload"))
	unknownAlgorithm.SignatureAlgorithm = "unknown"
	unknownAlgorithm.Signature = signTransaction(unknownAlgorithm, "secret")
	if result := processor.Submit(unknownAlgorithm, authority, at); result.Outcome != SecurityFailure {
		t.Fatalf("expected unknown algorithm security failure, got %#v", result)
	}
}

func TestSyncAcceptsVersionedEd25519EnvelopeAndEnforcesCapability(t *testing.T) {
	publicKey, privateKey, err := security.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	device, err := device_trust.NewDevice("device-ed", "tenant-1", "org-1", "user-1", base64.StdEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	authority, err := device_trust.IssueAuthorityPackage(device, "authority-ed", "secret", []string{"inspection.perform"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	processor.RegisterDevice(device)
	transaction := NewTransaction("tx-ed", "tenant-1", "device-ed", "user-1", 1, "InspectionSubmitted", []byte(`{"inspection_id":"inspection-1"}`))
	transaction.OrganizationID = "org-1"
	transaction.EntityID = "inspection-1"
	transaction.AuthorityID = authority.ID
	transaction.AuthorityEpoch = authority.Epoch
	transaction.CapturedAt = issuedAt.Add(10 * time.Minute)
	transaction, err = SignTransactionEd25519(transaction, privateKey, security.DeviceKeyID(publicKey))
	if err != nil {
		t.Fatal(err)
	}
	if result := processor.Submit(transaction, authority, issuedAt.Add(20*time.Minute)); result.Outcome != Applied {
		t.Fatalf("expected Ed25519 transaction to apply, got %#v", result)
	}

	unauthorized := NewTransaction("tx-ed-2", "tenant-1", "device-ed", "user-1", 2, "InspectionEvidenceUploaded", []byte(`{"evidence_id":"evidence-1"}`))
	unauthorized.OrganizationID = "org-1"
	unauthorized.EntityID = "inspection-1"
	unauthorized.AuthorityID = authority.ID
	unauthorized.AuthorityEpoch = authority.Epoch
	unauthorized.CapturedAt = issuedAt.Add(10 * time.Minute)
	unauthorized, err = SignTransactionEd25519(unauthorized, privateKey, security.DeviceKeyID(publicKey))
	if err != nil {
		t.Fatal(err)
	}
	if result := processor.Submit(unauthorized, authority, issuedAt.Add(20*time.Minute)); result.Outcome != SecurityFailure {
		t.Fatalf("expected capability failure, got %#v", result)
	}

	wrongOrganization := NewTransaction("tx-ed-org", "tenant-1", "device-ed", "user-1", 2, "InspectionSubmitted", []byte(`{"inspection_id":"inspection-1"}`))
	wrongOrganization.OrganizationID = "org-2"
	wrongOrganization.EntityID = "inspection-1"
	wrongOrganization.AuthorityID = authority.ID
	wrongOrganization.AuthorityEpoch = authority.Epoch
	wrongOrganization.CapturedAt = issuedAt.Add(10 * time.Minute)
	wrongOrganization, err = SignTransactionEd25519(wrongOrganization, privateKey, security.DeviceKeyID(publicKey))
	if err != nil {
		t.Fatal(err)
	}
	if result := processor.Submit(wrongOrganization, authority, issuedAt.Add(20*time.Minute)); result.Outcome != SecurityFailure {
		t.Fatalf("expected organization binding failure, got %#v", result)
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
	authority, err := device_trust.IssueAuthorityPackage(device, "authority-keyed", "secret", []string{"inspection.perform"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	processor.RegisterDevice(device)
	tx := NewTransaction("tx-keyed", "tenant-1", "device-keyed", "user-1", 1, "InspectionSubmitted", []byte(`{"inspection_id":"inspection-1"}`))
	tx.OrganizationID = "org-1"
	tx.EntityID = "inspection-1"
	tx.AuthorityID = authority.ID
	tx.AuthorityEpoch = authority.Epoch
	tx.CapturedAt = issuedAt.Add(10 * time.Minute)
	tx, err = SignTransactionEd25519(tx, privateKey, security.DeviceKeyID(publicKey))
	if err != nil {
		t.Fatal(err)
	}
	result := processor.Submit(tx, authority, issuedAt.Add(20*time.Minute))
	if result.Outcome != Applied {
		t.Fatalf("expected device-bound transaction to apply, got %#v", result)
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
	authority, err := device_trust.IssueAuthorityPackage(device, "authority-forge", "secret", []string{"inspection.perform"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	processor.RegisterDevice(device)
	tx := NewTransaction("tx-forge", "tenant-1", "device-forge", "user-1", 1, "InspectionSubmitted", []byte(`{"inspection_id":"inspection-1"}`))
	tx.OrganizationID = "org-1"
	tx.EntityID = "inspection-1"
	tx.AuthorityID = authority.ID
	tx.AuthorityEpoch = authority.Epoch
	tx.CapturedAt = issuedAt.Add(10 * time.Minute)
	// Sign with attacker's key but claim the device's key ID.
	tx.KeyID = security.DeviceKeyID(publicKey)
	tx.SignatureAlgorithm = "Ed25519"
	tx.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(attackerKey, []byte(canonicalTransaction(tx))))
	if result := processor.Submit(tx, authority, issuedAt.Add(20*time.Minute)); result.Outcome != SecurityFailure {
		t.Fatalf("expected forged key rejection, got %#v", result)
	}
}

func TestSyncRejectsUnknownDevice(t *testing.T) {
	processor, err := NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	untrustedAuthorityDevice, err := device_trust.NewDevice("authority-only", "tenant-1", "org-1", "user-1", "public-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := untrustedAuthorityDevice.Trust(); err != nil {
		t.Fatal(err)
	}
	authority, err := device_trust.IssueAuthorityPackage(untrustedAuthorityDevice, "authority-unreg", "secret", []string{"inspection-1"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	// Transaction from a device that is not registered with the processor.
	tx := SignTransaction(NewTransaction("tx-unreg", "tenant-1", "device-unknown", "user-1", 1, "FindingRecorded", []byte("payload")), "secret", "default")
	result := processor.Submit(tx, authority, issuedAt.Add(time.Minute))
	if result.Code != types.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized for unknown device, got code=%s outcome=%s", result.Code, result.Outcome)
	}
}

func TestSyncLegacyHMACFallbackAcceptedWithCounter(t *testing.T) {
	processor, err := NewProcessor(map[string]string{"default": "secret"})
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
	processor, err := NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	processor.RegisterDevice(device)
	tx := SignTransaction(NewTransaction("tx-hmac-keyed", "tenant-1", "device-keyed-hmac", "user-1", 1, "FindingRecorded", []byte("payload")), "secret", "default")
	result := processor.Submit(tx, authority, issuedAt.Add(time.Minute))
	if result.Outcome != SecurityFailure {
		t.Fatalf("expected HMAC rejected for keyed device, got %#v", result)
	}
	if processor.HMACFallbackCount.Load() != 0 {
		t.Fatalf("HMAC fallback counter must not increment for rejected keyed device, got %d", processor.HMACFallbackCount.Load())
	}
}

func TestV1CanonicalEnvelopeTestVector(t *testing.T) {
	transaction := Transaction{
		ProtocolVersion:    "v1",
		TransactionID:      "tx-ed",
		TenantID:           "tenant-1",
		OrganizationID:     "org-1",
		Environment:        "LIVE",
		DeviceID:           "device-1",
		UserID:             "user-1",
		SequenceNumber:     1,
		Operation:          "InspectionSubmitted",
		EntityID:           "inspection-1",
		PayloadHash:        "hash",
		AuthorityID:        "authority-1",
		AuthorityEpoch:     4,
		CapturedAt:         time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC),
		SignatureAlgorithm: "Ed25519",
		KeyID:              "key-1",
	}
	want := "v1|tx-ed|tenant-1|org-1|LIVE|device-1|user-1|1|InspectionSubmitted|inspection-1|hash|authority-1|4|2026-08-13T12:00:00Z|Ed25519|key-1"
	if got := canonicalTransaction(transaction); got != want {
		t.Fatalf("canonical envelope mismatch: got %q want %q", got, want)
	}
}
