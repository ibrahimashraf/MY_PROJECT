package device_trust

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"integin/internal/shared/types"
)

func newEnrollmentKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string, string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	nonce := "nonce-enrollment-test-0001"
	return pub, priv, nonce, hex.EncodeToString(ed25519.Sign(priv, []byte(nonce)))
}

func newPendingRequest(t *testing.T) *EnrollmentRequest {
	t.Helper()
	pub, _, nonce, signed := newEnrollmentKey(t)
	req, err := CreateEnrollmentRequest(
		"req-1", "tenant-1", "org-1", "user-oidc-1", "device-1",
		hex.EncodeToString(pub), nonce, signed,
		EnrollmentAttestation{KeyOrigin: "STRONGBOX", OSVersion: "Android 14"},
		time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func TestEnrollmentHappyPathChallengeToTrustedDeviceWithAuthorityPackage(t *testing.T) {
	req := newPendingRequest(t)
	if req.Status != EnrollmentPending || req.RequestedAt.IsZero() {
		t.Fatal("request should start pending with a recorded timestamp")
	}

	device, err := ApproveEnrollment(req, "admin-tenant-1")
	if err != nil {
		t.Fatal(err)
	}
	if device.State() != types.DeviceTrusted {
		t.Fatalf("approved device should be trusted, got %s", device.State())
	}
	if req.Status != EnrollmentApproved {
		t.Fatalf("request status should be APPROVED, got %s", req.Status)
	}
	if len(device.Events()) != 2 {
		t.Fatalf("expected trust + approval events, got %d", len(device.Events()))
	}
	if device.Events()[1].EventType() != EventEnrollmentApproved {
		t.Fatalf("approval audit event missing, got %s", device.Events()[1].EventType())
	}

	issuedAt := time.Date(2026, 9, 14, 9, 0, 0, 0, time.UTC)
	pkg, err := IssueAuthorityPackage(*device, "authority-1", "secret", []string{"inspect:read"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAuthorityPackage(pkg, *device, "secret", issuedAt.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
}

func TestEnrollmentFailsClosedOnInvalidSignatureAndMalformedKey(t *testing.T) {
	pub, priv, nonce, _ := newEnrollmentKey(t)
	attackerPub, _, _, _ := newEnrollmentKey(t)

	forged := hex.EncodeToString(ed25519.Sign(priv, []byte("forged-message")))
	if _, err := CreateEnrollmentRequest("req-1", "tenant-1", "org-1", "user-1", "device-1", hex.EncodeToString(pub), nonce, forged, EnrollmentAttestation{}, time.Now()); err == nil {
		t.Fatal("signature over a different message should fail")
	}
	if _, err := CreateEnrollmentRequest("req-1", "tenant-1", "org-1", "user-1", "device-1", hex.EncodeToString(attackerPub), nonce, forged, EnrollmentAttestation{}, time.Now()); err == nil {
		t.Fatal("signature bound to another key should fail")
	}

	goodSigned := hex.EncodeToString(ed25519.Sign(priv, []byte(nonce)))
	for _, tc := range []struct {
		name, publicKey, signature string
	}{
		{"bad public key encoding", "!!not-a-key!!", goodSigned},
		{"short public key", hex.EncodeToString([]byte{1, 2, 3}), goodSigned},
		{"bad signature encoding", hex.EncodeToString(pub), "zzzz-not-a-signature"},
		{"short signature", hex.EncodeToString(pub), strings.Repeat("00", 8)},
	} {
		if _, err := CreateEnrollmentRequest("req-1", "tenant-1", "org-1", "user-1", "device-1", tc.publicKey, nonce, tc.signature, EnrollmentAttestation{}, time.Now()); err == nil {
			t.Fatalf("%s: malformed material should fail", tc.name)
		}
	}
}

func TestEnrollmentAcceptsBase64KeyMaterialAndValidatesRequiredFields(t *testing.T) {
	pub, priv, nonce, _ := newEnrollmentKey(t)
	sigRaw := ed25519.Sign(priv, []byte(nonce))
	req, err := CreateEnrollmentRequest(
		"req-1", "tenant-1", "org-1", "user-1", "device-1",
		base64.StdEncoding.EncodeToString(pub), nonce,
		base64.RawStdEncoding.EncodeToString(sigRaw),
		EnrollmentAttestation{}, time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if req.PublicKey == "" {
		t.Fatal("request should retain the submitted public key")
	}

	goodSigned := hex.EncodeToString(sigRaw)
	pubHex := hex.EncodeToString(pub)
	for _, tc := range []struct {
		name           string
		requestID, org string
	}{
		{"empty request id", "", "org-1"},
		{"empty tenant", "req-1", "org-1"},
		{"empty organization", "req-1", ""},
	} {
		tenant := "tenant-1"
		if tc.name == "empty tenant" {
			tenant = ""
		}
		if _, err := CreateEnrollmentRequest(tc.requestID, tenant, tc.org, "user-1", "device-1", pubHex, nonce, goodSigned, EnrollmentAttestation{}, time.Now()); err == nil {
			t.Fatalf("%s should fail", tc.name)
		}
	}
	if _, err := CreateEnrollmentRequest("req-1", "tenant-1", "org-1", "user-1", "device-1", pubHex, nonce, goodSigned, EnrollmentAttestation{}, time.Time{}); err == nil {
		t.Fatal("zero requested_at should fail")
	}
}

func TestEnrollmentRejectWorkflow(t *testing.T) {
	req := newPendingRequest(t)
	if err := RejectEnrollment(req, "admin-1", "untrusted device model"); err != nil {
		t.Fatal(err)
	}
	if req.Status != EnrollmentRejected {
		t.Fatalf("request should be REJECTED, got %s", req.Status)
	}
	if _, err := ApproveEnrollment(req, "admin-1"); err == nil {
		t.Fatal("rejected request must not be approvable")
	}

	if err := RejectEnrollment(newPendingRequest(t), "admin-1", ""); err == nil {
		t.Fatal("rejection without a reason should fail")
	}
	approved := newPendingRequest(t)
	if _, err := ApproveEnrollment(approved, "admin-1"); err != nil {
		t.Fatal(err)
	}
	if err := RejectEnrollment(approved, "admin-1", "changed my mind"); err == nil {
		t.Fatal("approved request must not be rejectable")
	}
}

func TestEnrollmentKeyRotationInvalidatesEarlierAuthorityPackages(t *testing.T) {
	req := newPendingRequest(t)
	device, err := ApproveEnrollment(req, "admin-1")
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Date(2026, 9, 14, 9, 0, 0, 0, time.UTC)
	oldPkg, err := IssueAuthorityPackage(*device, "authority-1", "secret", []string{"inspect:read"}, issuedAt, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAuthorityPackage(oldPkg, *device, "secret", issuedAt.Add(30*time.Minute)); err != nil {
		t.Fatalf("package should validate before rotation: %v", err)
	}

	newPub, newPriv, newNonce, _ := newEnrollmentKey(t)
	if err := RotateDeviceKey(device, hex.EncodeToString(newPub), newNonce, hex.EncodeToString(ed25519.Sign(newPriv, []byte(newNonce)))); err != nil {
		t.Fatal(err)
	}
	if device.Epoch() != 2 || device.PublicKey() != hex.EncodeToString(newPub) {
		t.Fatalf("rotation should bump epoch to 2 and swap key, epoch=%d", device.Epoch())
	}
	if device.State() != types.DeviceTrusted {
		t.Fatalf("rotation should preserve trust, got %s", device.State())
	}
	if err := ValidateAuthorityPackage(oldPkg, *device, "secret", issuedAt.Add(30*time.Minute)); err == nil {
		t.Fatal("pre-rotation authority package should be invalid after rotation")
	}

	_, attackerPriv, _, _ := newEnrollmentKey(t)
	if err := RotateDeviceKey(device, hex.EncodeToString(newPub), newNonce, hex.EncodeToString(ed25519.Sign(attackerPriv, []byte(newNonce)))); err == nil {
		t.Fatal("rotation signed by the wrong key should fail possession check")
	}
	if device.Epoch() != 2 {
		t.Fatal("failed rotation must not bump the epoch")
	}
}
