package onboarding

import (
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
)

func TestClientEnrollmentSignChallengeIsBoundToOwnKey(t *testing.T) {
	coord, err := NewClientEnrollmentCoordinator("tenant-1", "org-1", "user-oidc-1", "Test Tablet", AttestationClaim{})
	if err != nil {
		t.Fatal(err)
	}
	challenge := &DeviceEnrollmentChallenge{
		ChallengeID: "chal-1", TenantID: "tenant-1", InspectorID: "user-oidc-1",
		Nonce: "0123456789abcdef0123456789abcdef", ExpiresAt: time.Now().Add(time.Hour),
	}
	sub, err := coord.SignChallenge(challenge)
	if err != nil {
		t.Fatal(err)
	}
	if sub.InspectorID != "user-oidc-1" {
		t.Fatal("submission must carry the coordinator's own oidc subject")
	}
	pub, err := device_trust.DecodePublicKey(sub.DevicePublicKey)
	if err != nil {
		t.Fatal(err)
	}
	sigBytes, err := hex.DecodeString(sub.SignedNonce)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(pub, []byte(challenge.Nonce), sigBytes) {
		t.Fatal("signed nonce must verify against the submitted public key")
	}
	if got, err := coord.DeviceID(); err != nil || got == "" {
		t.Fatalf("device id must derive from the signing key, got %q err %v", got, err)
	}
}

func TestClientEnrollmentSignChallengeRejectsExpiredAndForeignChallenges(t *testing.T) {
	coord, err := NewClientEnrollmentCoordinator("tenant-1", "org-1", "user-oidc-1", "Test Tablet", AttestationClaim{})
	if err != nil {
		t.Fatal(err)
	}
	expired := &DeviceEnrollmentChallenge{
		ChallengeID: "chal-1", TenantID: "tenant-1", InspectorID: "user-oidc-1",
		Nonce: "0123456789abcdef0123456789abcdef", ExpiresAt: time.Now().Add(-time.Minute),
	}
	if _, err := coord.SignChallenge(expired); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired challenge must be refused, got %v", err)
	}
	foreign := &DeviceEnrollmentChallenge{
		ChallengeID: "chal-1", TenantID: "tenant-1", InspectorID: "attacker",
		Nonce: "0123456789abcdef0123456789abcdef", ExpiresAt: time.Now().Add(time.Hour),
	}
	if _, err := coord.SignChallenge(foreign); err == nil || !strings.Contains(err.Error(), "different user") {
		t.Fatalf("challenge for another user must be refused, got %v", err)
	}
	if _, err := coord.SignChallenge(nil); err == nil {
		t.Fatal("nil challenge must be refused")
	}
}

func TestClientEnrollmentSubmissionFlowsThroughProductionService(t *testing.T) {
	svc := NewProductionEnrollmentService(AttestationPolicy{})
	challenge, err := svc.CreateChallenge("tenant-1", "user-oidc-1", "org-1")
	if err != nil {
		t.Fatal(err)
	}
	coord, err := NewClientEnrollmentCoordinator("tenant-1", "org-1", "user-oidc-1", "Test Tablet", AttestationClaim{})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := coord.SignChallenge(challenge)
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.SubmitEnrollment(*sub, "user-oidc-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != string(device_trust.EnrollmentPending) {
		t.Fatalf("client submission must create a pending request, got %q", result.Status)
	}
	wantDeviceID, err := coord.DeviceID()
	if err != nil {
		t.Fatal(err)
	}
	if result.DeviceID != wantDeviceID {
		t.Fatalf("server must bind the request to the client-derived device id, got %q want %q", result.DeviceID, wantDeviceID)
	}
}

func TestClientEnrollmentBuildRequestAndApprovalStatus(t *testing.T) {
	coord, err := NewClientEnrollmentCoordinator("tenant-1", "org-1", "user-oidc-1", "Test Tablet", AttestationClaim{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coord.ApprovalStatus(); err == nil {
		t.Fatal("approval status must fail before any request is constructed")
	}

	challenge := &DeviceEnrollmentChallenge{
		ChallengeID: "chal-1", TenantID: "tenant-1", InspectorID: "user-oidc-1",
		Nonce: "0123456789abcdef0123456789abcdef", ExpiresAt: time.Now().Add(time.Hour),
	}
	request, err := coord.BuildEnrollmentRequest(challenge, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if request.Status != device_trust.EnrollmentPending {
		t.Fatalf("constructed request must start pending, got %q", request.Status)
	}
	status, err := coord.ApprovalStatus()
	if err != nil || status != device_trust.EnrollmentPending {
		t.Fatalf("approval status after construction must be pending, got %q err %v", status, err)
	}

	if _, err := device_trust.ApproveEnrollment(request, "admin-1"); err != nil {
		t.Fatal(err)
	}
	status, err = coord.ApprovalStatus()
	if err != nil || status != device_trust.EnrollmentApproved {
		t.Fatalf("approval status after admin approval must be APPROVED, got %q err %v", status, err)
	}
}
