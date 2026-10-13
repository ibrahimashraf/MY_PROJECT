package onboarding

// Fixture note: this test is LIVE — every run mints a fresh Ed25519 keypair
// inside Dart and enrolls it here. No checked-in vectors exist and nothing
// needs regenerating; contract drift on either side breaks this test.

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDartSimulatorRoundTripEnrollsClaimed(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode: skipping live Dart round-trip")
	}
	dartBin, err := exec.LookPath("dart")
	if err != nil {
		t.Skip("dart not on PATH: skipping live round-trip")
	}
	if err := exec.Command(dartBin, "--version").Run(); err != nil {
		t.Skipf("dart not functional (%v): skipping live round-trip", err)
	}

	const tenantID = "ten_roundtrip"
	const inspectorID = "insp_roundtrip"

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed to resolve test file path")
	}
	fieldApp := filepath.Join(filepath.Dir(thisFile), "..", "..", "field_app")
	helper := filepath.Join(fieldApp, "tool", "attestation_roundtrip.dart")

	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("NewEnrollmentSimulator: %v", err)
	}

	runDart := func(challengeID, nonce string) DeviceEnrollmentSubmission {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, dartBin, "run", helper,
			"--challenge-id", challengeID,
			"--inspector-id", inspectorID,
			"--nonce", nonce,
			"--device-model", "Simulated Rugged Tablet",
		)
		cmd.Dir = fieldApp
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("dart helper failed: %v\nstderr:\n%s\nstdout:\n%s", err, stderr.String(), stdout.String())
		}
		var sub DeviceEnrollmentSubmission
		if err := json.Unmarshal(stdout.Bytes(), &sub); err != nil {
			t.Fatalf("parse dart submission JSON (stdout=%q): %v", stdout.String(), err)
		}
		return sub
	}

	chal, err := sim.CreateEnrollmentChallenge(tenantID, inspectorID)
	if err != nil {
		t.Fatalf("CreateEnrollmentChallenge: %v", err)
	}
	sub := runDart(chal.ChallengeID, chal.Nonce)

	record, err := sim.ProcessDeviceEnrollment(sub)
	if err != nil {
		t.Fatalf("ProcessDeviceEnrollment: %v", err)
	}
	if record.AttestationOrigin != string(KeyOriginSoftware) {
		t.Errorf("AttestationOrigin = %q, want %q", record.AttestationOrigin, KeyOriginSoftware)
	}
	if record.AttestationVerified {
		t.Errorf("simulated device must enroll CLAIMED, never VERIFIED")
	}
	if record.AttestationBiometricBound {
		t.Errorf("simulated device must not be biometric-bound")
	}
	if record.InspectorID != inspectorID {
		t.Errorf("InspectorID = %q, want %q", record.InspectorID, inspectorID)
	}
	if record.DevicePublicKey != sub.DevicePublicKey {
		t.Errorf("DevicePublicKey = %q, want Dart-sent %q", record.DevicePublicKey, sub.DevicePublicKey)
	}

	// Negative: re-enroll the same submission against a fresh challenge with
	// the last hex char of signed_nonce flipped. It must fail at ed25519
	// verification — a vacuous pass (verifier not exercised) would accept it.
	tamperChal, err := sim.CreateEnrollmentChallenge(tenantID, inspectorID)
	if err != nil {
		t.Fatalf("CreateEnrollmentChallenge (tamper): %v", err)
	}
	tampered := sub
	tampered.ChallengeID = tamperChal.ChallengeID
	replacement := byte('0')
	if tampered.SignedNonce[len(tampered.SignedNonce)-1] == '0' {
		replacement = '1'
	}
	tampered.SignedNonce = tampered.SignedNonce[:len(tampered.SignedNonce)-1] + string(replacement)
	if _, err := sim.ProcessDeviceEnrollment(tampered); err == nil || !strings.Contains(err.Error(), "cryptographic verification failed") {
		t.Fatalf("tampered signed_nonce not rejected by signature verification, err=%v", err)
	}
}
