package sync

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
	"integin/internal/security"
)

func TestVerifyDeviceProofDerivesScopeFromRegisteredDevice(t *testing.T) {
	now := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	processor, device, authority, privateKey := proofFixture(t, now, []string{"work_package.read"})
	proof := signedDeviceProof(now, device, authority, privateKey)

	verified, err := processor.VerifyDeviceProof(context.Background(), proof, authority, now)
	if err != nil {
		t.Fatal(err)
	}
	if verified.TenantID != device.TenantID() || verified.OrganizationID != device.OrganizationID() || verified.UserID != device.UserID() || verified.DeviceID != device.ID() {
		t.Fatalf("verified scope was not derived from registered device: %#v", verified)
	}
}

func TestVerifyDeviceProofRejectsTamperingBeforeScopeDerivation(t *testing.T) {
	now := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	processor, device, authority, privateKey := proofFixture(t, now, []string{"work_package.read"})
	proof := signedDeviceProof(now, device, authority, privateKey)
	proof.InspectionID = "inspection-tampered"

	if _, err := processor.VerifyDeviceProof(context.Background(), proof, authority, now); err == nil {
		t.Fatal("expected tampered proof rejection")
	}
}

func TestVerifyDeviceProofRequiresManifestReadCapability(t *testing.T) {
	now := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	processor, device, authority, privateKey := proofFixture(t, now, []string{"inspection.perform"})
	proof := signedDeviceProof(now, device, authority, privateKey)

	if _, err := processor.VerifyDeviceProof(context.Background(), proof, authority, now); err == nil {
		t.Fatal("expected missing manifest-read capability rejection")
	}
}

func proofFixture(t *testing.T, now time.Time, scopes []string) (*Processor, device_trust.Device, device_trust.AuthorityPackage, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := security.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	device, err := device_trust.NewDevice("device-proof-1", "tenant-proof-1", "org-proof-1", "user-proof-1", base64.StdEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Trust(); err != nil {
		t.Fatal(err)
	}
	authority, err := device_trust.IssueAuthorityPackage(device, "authority-proof-1", "proof-secret", scopes, now.Add(-time.Minute), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewProcessor("proof-secret")
	if err != nil {
		t.Fatal(err)
	}
	processor.RegisterDevice(device)
	return processor, device, authority, privateKey
}

func signedDeviceProof(now time.Time, device device_trust.Device, authority device_trust.AuthorityPackage, privateKey ed25519.PrivateKey) DeviceProof {
	proof := DeviceProof{
		ProtocolVersion:    DeviceProofProtocolVersion,
		Purpose:            WorkPackageManifestReadPurpose,
		RequestID:          "request-proof-1",
		DeviceID:           device.ID(),
		AuthorityID:        authority.ID,
		AuthorityEpoch:     authority.Epoch,
		InspectionID:       "inspection-proof-1",
		IssuedAt:           now.Add(-time.Second),
		ExpiresAt:          now.Add(time.Minute),
		SignatureAlgorithm: "Ed25519",
		KeyID:              security.DeviceKeyID(privateKey.Public().(ed25519.PublicKey)),
	}
	proof.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(canonicalDeviceProof(proof))))
	return proof
}
