package onboarding

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

// enroll signs a fresh challenge with a fresh key and submits the claim.
func enroll(t *testing.T, sim *EnrollmentSimulator, claim AttestationClaim) (*DeviceTrustRecord, error) {
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
	return sim.ProcessDeviceEnrollment(sub)
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
		record, err := enroll(t, sim, AttestationClaim{KeyOrigin: origin})
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
	record, err := enroll(t, sim, claim)
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
		_, err = enroll(t, sim, tc.claim)
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
	record, err := enroll(t, sim, AttestationClaim{})
	if err != nil {
		t.Fatalf("unattested enrollment failed: %v", err)
	}
	if record.AttestationOrigin != "" || record.AttestationBiometricBound {
		t.Fatalf("unattested record has non-zero posture: origin=%q bound=%v", record.AttestationOrigin, record.AttestationBiometricBound)
	}
}
