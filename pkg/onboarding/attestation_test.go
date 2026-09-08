package onboarding

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// enroll signs a fresh challenge with a fresh key and submits the claim.
// The private key is returned so callers can sign offline receipts.
func enroll(t *testing.T, sim *EnrollmentSimulator, claim AttestationClaim) (*DeviceTrustRecord, ed25519.PrivateKey, error) {
	t.Helper()
	chal, err := sim.CreateEnrollmentChallenge("ten_test", "insp_test")
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	sub := DeviceEnrollmentSubmission{
		ChallengeID:     chal.ChallengeID,
		InspectorID:     "insp_test",
		DevicePublicKey: hex.EncodeToString(pub),
		DeviceModel:     "Test Tablet",
		SignedNonce:     hex.EncodeToString(ed25519.Sign(priv, []byte(chal.Nonce))),
		Attestation:     claim,
	}
	record, err := sim.ProcessDeviceEnrollment(sub)
	if err != nil {
		return nil, nil, err
	}
	return record, priv, nil
}

func TestAttestationValidate(t *testing.T) {
	valid := []AttestationClaim{
		{KeyOrigin: KeyOriginSecureEnclave, BiometricBound: true, OSVersion: "iOS 17.2", AttestationBlob: "0102abcd", KeyAlias: "com.integin.enclave"},
		{KeyOrigin: KeyOriginStrongBox, OSVersion: "Android 14", AttestationBlob: "deadbeef"},
		{KeyOrigin: KeyOriginSoftware},
		{KeyOrigin: KeyOriginNone},
	}
	for _, claim := range valid {
		if err := claim.Validate(); err != nil {
			t.Fatalf("valid claim rejected (%s): %v", claim.KeyOrigin, err)
		}
	}

	invalid := []AttestationClaim{
		{KeyOrigin: "TPM"}, // unknown origin
		{},                 // empty origin
		{KeyOrigin: KeyOriginStrongBox, OSVersion: "Android 14"},     // hardware without blob
		{KeyOrigin: KeyOriginSecureEnclave, AttestationBlob: "abcd"}, // hardware without OS
		{KeyOrigin: KeyOriginNone, AttestationBlob: "abcd"},          // spoof: NONE with blob
		{KeyOrigin: KeyOriginNone, BiometricBound: true},             // spoof: NONE claiming biometric
	}
	for _, claim := range invalid {
		if err := claim.Validate(); err == nil {
			t.Fatalf("invalid claim accepted (%+v)", claim)
		}
	}
}

func TestAttestationPolicy(t *testing.T) {
	software := AttestationClaim{KeyOrigin: KeyOriginSoftware}
	none := AttestationClaim{KeyOrigin: KeyOriginNone}
	hwBound := AttestationClaim{KeyOrigin: KeyOriginSecureEnclave, BiometricBound: true, OSVersion: "iOS 17", AttestationBlob: "abcd"}
	hwUnbound := AttestationClaim{KeyOrigin: KeyOriginStrongBox, OSVersion: "Android 14", AttestationBlob: "abcd"}

	// (a) default zero policy is permissive.
	for _, claim := range []AttestationClaim{software, none, hwBound, hwUnbound} {
		if err := VerifyClaim(claim, AttestationPolicy{}); err != nil {
			t.Fatalf("permissive policy rejected %s: %v", claim.KeyOrigin, err)
		}
	}

	// (e) RequireHardware rejects SOFTWARE/NONE.
	strict := AttestationPolicy{RequireHardware: true}
	for _, claim := range []AttestationClaim{software, none} {
		if err := VerifyClaim(claim, strict); !errors.Is(err, ErrSoftwareOriginRejected) {
			t.Fatalf("RequireHardware accepted %s (err=%v)", claim.KeyOrigin, err)
		}
	}
	if err := VerifyClaim(hwBound, strict); err != nil {
		t.Fatalf("RequireHardware rejected hardware claim: %v", err)
	}

	// (f) RequireBiometricBinding rejects unbound.
	bio := AttestationPolicy{RequireBiometricBinding: true}
	if err := VerifyClaim(hwUnbound, bio); !errors.Is(err, ErrBiometricBindingMissing) {
		t.Fatalf("RequireBiometricBinding accepted unbound claim: %v", err)
	}
	if err := VerifyClaim(hwBound, bio); err != nil {
		t.Fatalf("RequireBiometricBinding rejected bound claim: %v", err)
	}

	// Unknown future origins fail closed under any policy.
	if err := VerifyClaim(AttestationClaim{KeyOrigin: "TPM"}, AttestationPolicy{}); !errors.Is(err, ErrUnknownKeyOrigin) {
		t.Fatalf("unknown origin not fail-closed: %v", err)
	}
}

// (a) SOFTWARE/NONE claims enroll under the default permissive policy.
func TestEnrollmentAcceptsSoftwareAndNoneClaims(t *testing.T) {
	for _, origin := range []KeyOrigin{KeyOriginSoftware, KeyOriginNone} {
		sim, err := NewEnrollmentSimulator()
		if err != nil {
			t.Fatalf("sim init: %v", err)
		}
		record, _, err := enroll(t, sim, AttestationClaim{KeyOrigin: origin})
		if err != nil {
			t.Fatalf("enrollment with %s rejected under permissive policy: %v", origin, err)
		}
		if record.AttestationOrigin != string(origin) {
			t.Fatalf("posture origin not recorded: got %q want %q", record.AttestationOrigin, origin)
		}
	}
}

// (b) hardware claim with blob enrolls and posture is recorded.
func TestEnrollmentAcceptsHardwareClaimAndRecordsPosture(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim init: %v", err)
	}
	claim := AttestationClaim{KeyOrigin: KeyOriginSecureEnclave, BiometricBound: true, OSVersion: "iOS 17.2", AttestationBlob: "0102abcd", KeyAlias: "k1"}
	record, _, err := enroll(t, sim, claim)
	if err != nil {
		t.Fatalf("hardware claim enrollment failed: %v", err)
	}
	if record.AttestationOrigin != string(KeyOriginSecureEnclave) {
		t.Fatalf("posture origin not recorded: got %q", record.AttestationOrigin)
	}
	if !record.AttestationBiometricBound {
		t.Fatalf("biometric binding not recorded")
	}
}

// (c)(d) hardware-without-blob, spoofed NONE, and unknown origins are rejected.
func TestEnrollmentRejectsInvalidAttestation(t *testing.T) {
	cases := []struct {
		name  string
		claim AttestationClaim
	}{
		{"hardware without blob", AttestationClaim{KeyOrigin: KeyOriginSecureEnclave, OSVersion: "iOS 17"}},
		{"none with blob (spoof)", AttestationClaim{KeyOrigin: KeyOriginNone, AttestationBlob: "abcd"}},
		{"none with biometric (spoof)", AttestationClaim{KeyOrigin: KeyOriginNone, BiometricBound: true}},
		{"unknown origin", AttestationClaim{KeyOrigin: "TPM"}},
	}
	for _, tc := range cases {
		sim, err := NewEnrollmentSimulator()
		if err != nil {
			t.Fatalf("sim init: %v", err)
		}
		_, _, err = enroll(t, sim, tc.claim)
		if err == nil {
			t.Fatalf("%s: enrollment accepted invalid claim", tc.name)
		}
		if !strings.Contains(err.Error(), "attestation claim rejected") {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
	}
}

// (g) Unattested submissions (zero-value claim) keep the legacy happy path.
func TestEnrollmentWithoutAttestationKeepsLegacyBehavior(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim init: %v", err)
	}
	record, _, err := enroll(t, sim, AttestationClaim{})
	if err != nil {
		t.Fatalf("unattested enrollment failed: %v", err)
	}
	if record.AttestationOrigin != "" || record.AttestationBiometricBound {
		t.Fatalf("unattested record has non-zero posture: origin=%q bound=%v", record.AttestationOrigin, record.AttestationBiometricBound)
	}
}

// signedOfflineReceipt issues a manifest for the enrolled device and seals a
// receipt signed by the device's private key.
func signedOfflineReceipt(t *testing.T, sim *EnrollmentSimulator, record *DeviceTrustRecord, priv ed25519.PrivateKey) SignedInspectionReceipt {
	t.Helper()
	manifest, err := sim.IssueWorkPackageManifest("ten_test", "wo_test", record.DeviceID, "insp_test", `{"ok":true}`)
	if err != nil {
		t.Fatalf("issue manifest: %v", err)
	}
	digest := sha256.Sum256([]byte("payload"))
	payloadDigest := hex.EncodeToString(digest[:])
	receiptID := "rcpt_test"
	signPayload := fmt.Sprintf("%s|%s|%s|%s|%s", receiptID, manifest.ManifestID, "asset_test", "PASSED", payloadDigest)
	return SignedInspectionReceipt{
		ReceiptID:       receiptID,
		ManifestID:      manifest.ManifestID,
		AssetID:         "asset_test",
		OverallResult:   "PASSED",
		PayloadDigest:   payloadDigest,
		DeviceSignature: hex.EncodeToString(ed25519.Sign(priv, []byte(signPayload))),
	}
}

// (a)(d) Zero policy equals VerifyOfflineReceipt: accepts a receipt from an
// unattested device and still rejects a tampered signature.
func TestVerifyOfflineReceiptPermissivePolicy(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim init: %v", err)
	}
	record, priv, err := enroll(t, sim, AttestationClaim{})
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	receipt := signedOfflineReceipt(t, sim, record, priv)

	valid, err := sim.VerifyOfflineReceiptWithPolicy(receipt, AttestationPolicy{})
	if err != nil || !valid {
		t.Fatalf("permissive policy vetoed valid unattested receipt: %v", err)
	}

	tampered := receipt
	tampered.OverallResult = "FAILED_CRITICAL"
	valid, err = sim.VerifyOfflineReceiptWithPolicy(tampered, AttestationPolicy{})
	if valid || err == nil {
		t.Fatalf("tampered receipt accepted under permissive policy (err=%v)", err)
	}
}

// (b) RequireHardware accepts STRONGBOX-enrolled receipts and rejects
// SOFTWARE and legacy unattested devices.
func TestVerifyOfflineReceiptRequireHardware(t *testing.T) {
	policy := AttestationPolicy{RequireHardware: true}

	strongBox, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim init: %v", err)
	}
	strongRec, strongPriv, err := enroll(t, strongBox, AttestationClaim{KeyOrigin: KeyOriginStrongBox, OSVersion: "Android 14", AttestationBlob: "deadbeef"})
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	receipt := signedOfflineReceipt(t, strongBox, strongRec, strongPriv)
	valid, err := strongBox.VerifyOfflineReceiptWithPolicy(receipt, policy)
	if err != nil || !valid {
		t.Fatalf("RequireHardware rejected StrongBox receipt: %v", err)
	}

	software, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim init: %v", err)
	}
	softRec, softPriv, err := enroll(t, software, AttestationClaim{KeyOrigin: KeyOriginSoftware})
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	receipt = signedOfflineReceipt(t, software, softRec, softPriv)
	valid, err = software.VerifyOfflineReceiptWithPolicy(receipt, policy)
	if valid || !errors.Is(err, ErrSoftwareOriginRejected) {
		t.Fatalf("RequireHardware accepted SOFTWARE receipt (err=%v)", err)
	}

	unattested, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim init: %v", err)
	}
	legacyRec, legacyPriv, err := enroll(t, unattested, AttestationClaim{})
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	receipt = signedOfflineReceipt(t, unattested, legacyRec, legacyPriv)
	valid, err = unattested.VerifyOfflineReceiptWithPolicy(receipt, policy)
	if valid || !errors.Is(err, ErrSoftwareOriginRejected) {
		t.Fatalf("RequireHardware accepted unattested receipt (err=%v)", err)
	}

	// Unknown origin (not one of the 4 + "") fails closed under a strict policy.
	strongRec.AttestationOrigin = "TPM"
	strongBox.deviceStore[strongRec.DeviceID] = *strongRec
	receipt = signedOfflineReceipt(t, strongBox, strongRec, strongPriv)
	valid, err = strongBox.VerifyOfflineReceiptWithPolicy(receipt, policy)
	if valid || !errors.Is(err, ErrUnknownKeyOrigin) {
		t.Fatalf("RequireHardware accepted unknown origin (err=%v)", err)
	}
}

// (c) RequireBiometricBinding rejects a bound-originless hardware device and
// accepts one with biometric binding recorded.
func TestVerifyOfflineReceiptRequireBiometricBinding(t *testing.T) {
	policy := AttestationPolicy{RequireBiometricBinding: true}

	unbound, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim init: %v", err)
	}
	record, priv, err := enroll(t, unbound, AttestationClaim{KeyOrigin: KeyOriginSecureEnclave, OSVersion: "iOS 17", AttestationBlob: "abcd"})
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	receipt := signedOfflineReceipt(t, unbound, record, priv)
	valid, err := unbound.VerifyOfflineReceiptWithPolicy(receipt, policy)
	if valid || !errors.Is(err, ErrBiometricBindingMissing) {
		t.Fatalf("RequireBiometricBinding accepted unbound receipt (err=%v)", err)
	}

	bound, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim init: %v", err)
	}
	record, priv, err = enroll(t, bound, AttestationClaim{KeyOrigin: KeyOriginSecureEnclave, BiometricBound: true, OSVersion: "iOS 17", AttestationBlob: "abcd"})
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	receipt = signedOfflineReceipt(t, bound, record, priv)
	valid, err = bound.VerifyOfflineReceiptWithPolicy(receipt, policy)
	if err != nil || !valid {
		t.Fatalf("RequireBiometricBinding rejected bound receipt: %v", err)
	}
}

// (e) Unknown manifest still errors, regardless of policy.
func TestVerifyOfflineReceiptUnknownManifestWithPolicy(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim init: %v", err)
	}
	receipt := SignedInspectionReceipt{ManifestID: "man_unknown", DeviceSignature: "00"}
	valid, err := sim.VerifyOfflineReceiptWithPolicy(receipt, AttestationPolicy{RequireHardware: true})
	if valid || err == nil {
		t.Fatalf("unknown manifest accepted (err=%v)", err)
	}
	if !strings.Contains(err.Error(), "unknown manifest ID") {
		t.Fatalf("unexpected error: %v", err)
	}
}
