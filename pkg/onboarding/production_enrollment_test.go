package onboarding

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/device_trust"
)

func newProdService(policy AttestationPolicy) *ProductionEnrollmentService {
	svc := NewProductionEnrollmentService(policy)
	svc.now = func() time.Time { return time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC) }
	return svc
}

// signedSubmission generates a fresh keypair and signs the challenge nonce.
func signedSubmission(t *testing.T, challenge *DeviceEnrollmentChallenge, claim AttestationClaim) DeviceEnrollmentSubmission {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return DeviceEnrollmentSubmission{
		ChallengeID:     challenge.ChallengeID,
		InspectorID:     challenge.InspectorID,
		DevicePublicKey: hex.EncodeToString(pub),
		DeviceModel:     "Test Tablet",
		SignedNonce:     hex.EncodeToString(ed25519.Sign(priv, []byte(challenge.Nonce))),
		Attestation:     claim,
	}
}

func TestProductionEnrollmentHappyPathPendingApprovalAndAuthorityPackage(t *testing.T) {
	svc := newProdService(AttestationPolicy{})
	challenge, err := svc.CreateChallenge("tenant-1", "user-oidc-1", "org-1")
	if err != nil {
		t.Fatal(err)
	}
	sub := signedSubmission(t, challenge, AttestationClaim{})

	result, err := svc.SubmitEnrollment(sub, "user-oidc-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Request == nil || result.Request.Status != device_trust.EnrollmentPending {
		t.Fatalf("submission should create a pending request, got %+v", result)
	}
	if result.UserID != "user-oidc-1" || result.OrganizationID != "org-1" {
		t.Fatalf("request must be bound to the oidc subject and org")
	}
	if _, err := svc.SubmitEnrollment(sub, "user-oidc-1"); err == nil {
		t.Fatal("consumed challenge must not be reusable")
	}

	device, err := device_trust.ApproveEnrollment(result.Request, "admin-1")
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 9, 14, 9, 0, 0, 0, time.UTC)
	pkg, err := device_trust.IssueAuthorityPackage(*device, "authority-1", "secret", []string{"inspect:read"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := device_trust.ValidateAuthorityPackage(pkg, *device, "secret", issuedAt.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
}

func TestProductionEnrollmentRejectsOIDCIdentityMismatch(t *testing.T) {
	svc := newProdService(AttestationPolicy{})
	challenge, err := svc.CreateChallenge("tenant-1", "user-oidc-1", "org-1")
	if err != nil {
		t.Fatal(err)
	}
	sub := signedSubmission(t, challenge, AttestationClaim{})
	if _, err := svc.SubmitEnrollment(sub, "attacker-subject"); err == nil {
		t.Fatal("submission under a different oidc subject must fail")
	}
	if _, err := svc.SubmitEnrollment(sub, ""); err == nil {
		t.Fatal("empty oidc subject must fail")
	}
}

func TestProductionEnrollmentRejectsExpiredChallenge(t *testing.T) {
	svc := newProdService(AttestationPolicy{})
	challenge, err := svc.CreateChallenge("tenant-1", "user-oidc-1", "org-1")
	if err != nil {
		t.Fatal(err)
	}
	sub := signedSubmission(t, challenge, AttestationClaim{})
	svc.now = func() time.Time { return time.Date(2026, 9, 14, 13, 0, 0, 0, time.UTC) }
	if _, err := svc.SubmitEnrollment(sub, "user-oidc-1"); err == nil {
		t.Fatal("expired challenge must fail")
	} else if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductionEnrollmentEnforcesAttestationPolicy(t *testing.T) {
	svc := newProdService(AttestationPolicy{RequireHardware: true})
	challenge, err := svc.CreateChallenge("tenant-1", "user-oidc-1", "org-1")
	if err != nil {
		t.Fatal(err)
	}
	sub := signedSubmission(t, challenge, AttestationClaim{KeyOrigin: KeyOriginSoftware})
	if _, err := svc.SubmitEnrollment(sub, "user-oidc-1"); !errors.Is(err, ErrSoftwareOriginRejected) {
		t.Fatalf("SOFTWARE claim under RequireHardware must fail, got %v", err)
	}

	hwSvc := newProdService(AttestationPolicy{RequireHardware: true})
	challenge, err = hwSvc.CreateChallenge("tenant-1", "user-oidc-1", "org-1")
	if err != nil {
		t.Fatal(err)
	}
	hw := AttestationClaim{KeyOrigin: KeyOriginStrongBox, OSVersion: "Android 14", AttestationBlob: "deadbeef"}
	result, err := hwSvc.SubmitEnrollment(signedSubmission(t, challenge, hw), "user-oidc-1")
	if err != nil {
		t.Fatalf("hardware claim should pass RequireHardware: %v", err)
	}
	if result.Request.Attestation.KeyOrigin != string(KeyOriginStrongBox) {
		t.Fatalf("attestation posture not carried onto the request")
	}
}

func TestProductionEnrollmentRejectsForgedDeviceSignature(t *testing.T) {
	svc := newProdService(AttestationPolicy{})
	challenge, err := svc.CreateChallenge("tenant-1", "user-oidc-1", "org-1")
	if err != nil {
		t.Fatal(err)
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, attackerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	forged := DeviceEnrollmentSubmission{
		ChallengeID:     challenge.ChallengeID,
		InspectorID:     "user-oidc-1",
		DevicePublicKey: hex.EncodeToString(pub),
		SignedNonce:     hex.EncodeToString(ed25519.Sign(attackerPriv, []byte(challenge.Nonce))),
	}
	if _, err := svc.SubmitEnrollment(forged, "user-oidc-1"); err == nil {
		t.Fatal("signature bound to a different key must fail")
	}
}
