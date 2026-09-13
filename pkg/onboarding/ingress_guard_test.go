package onboarding

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"integin/pkg/timeguard"
)

func strongBoxDevice(verified bool) DeviceTrustRecord {
	return DeviceTrustRecord{
		DeviceID:            "dev_strongbox",
		TenantID:            "ten_test",
		InspectorID:         "insp_test",
		DevicePublicKey:     hex.EncodeToString(make([]byte, ed25519.PublicKeySize)),
		IsActive:            true,
		AttestationOrigin:   string(KeyOriginStrongBox),
		AttestationVerified: verified,
	}
}

func clockTick(t time.Time, mono int64) IngressClockSample {
	return IngressClockSample{WallTime: t, MonotonicNano: mono, Source: timeguard.GPS_ATTESTED}
}

func TestValidateIngressAcceptsVerifiedHardwareStream(t *testing.T) {
	now := time.Now().UTC()
	guard := NewIngressGuardWithNow(AttestationPolicy{RequireHardware: true}, 10*time.Second, func() time.Time { return now })

	payload := FieldIngressPayload{
		Device:    strongBoxDevice(true),
		Timestamp: clockTick(now.Add(-time.Second), 100),
	}
	if err := guard.ValidateIngress(payload); err != nil {
		t.Fatalf("valid hardware ingress rejected: %v", err)
	}

	// Second sample with monotonic progress and forward clock passes.
	payload.Timestamp = clockTick(now, 200)
	if err := guard.ValidateIngress(payload); err != nil {
		t.Fatalf("valid ordered sample rejected: %v", err)
	}
}

func TestValidateIngressRejectsSoftwareWhenPolicyRequiresHardware(t *testing.T) {
	now := time.Now().UTC()
	guard := NewIngressGuard(AttestationPolicy{RequireHardware: true}, 10*time.Second)

	soft := strongBoxDevice(true)
	soft.AttestationOrigin = string(KeyOriginSoftware)
	payload := FieldIngressPayload{Device: soft, Timestamp: clockTick(now, 1)}
	if err := guard.ValidateIngress(payload); !errors.Is(err, ErrSoftwareOriginRejected) {
		t.Fatalf("software key accepted under RequireHardware: %v", err)
	}
}

func TestValidateIngressRejectsUnverifiedHardwareClaimUnderStrictPolicy(t *testing.T) {
	now := time.Now().UTC()
	guard := NewIngressGuard(AttestationPolicy{RequireHardware: true}, 10*time.Second)
	payload := FieldIngressPayload{Device: strongBoxDevice(false), Timestamp: clockTick(now, 1)}
	if err := guard.ValidateIngress(payload); err == nil {
		t.Fatal("STRONGBOX claim never verified offline accepted under RequireHardware")
	}
}

func TestValidateIngressRejectsInactiveDevice(t *testing.T) {
	now := time.Now().UTC()
	guard := NewIngressGuard(AttestationPolicy{}, 10*time.Second)
	dev := strongBoxDevice(true)
	dev.IsActive = false
	payload := FieldIngressPayload{Device: dev, Timestamp: clockTick(now, 1)}
	if err := guard.ValidateIngress(payload); !errors.Is(err, ErrIngressDeviceInactive) {
		t.Fatalf("inactive device accepted: %v", err)
	}
}

func TestValidateIngressRejectsMonotonicRegression(t *testing.T) {
	now := time.Now().UTC()
	guard := NewIngressGuard(AttestationPolicy{}, 10*time.Second)

	if err := guard.ValidateIngress(FieldIngressPayload{Device: strongBoxDevice(true), Timestamp: clockTick(now.Add(-time.Second), 500)}); err != nil {
		t.Fatalf("anchor sample rejected: %v", err)
	}
	// Clock moved forward but the monotonic tick moved backwards: rollback.
	if err := guard.ValidateIngress(FieldIngressPayload{Device: strongBoxDevice(true), Timestamp: clockTick(now, 400)}); !errors.Is(err, timeguard.ErrMonotonicSequenceBroken) {
		t.Fatalf("monotonic regression not rejected: %v", err)
	}
}

func TestValidateIngressRejectsClockDriftBeyondTolerance(t *testing.T) {
	now := time.Now().UTC()
	guard := NewIngressGuard(AttestationPolicy{}, 10*time.Second)

	if err := guard.ValidateIngress(FieldIngressPayload{Device: strongBoxDevice(true), Timestamp: clockTick(now.Add(-time.Second), 100)}); err != nil {
		t.Fatalf("anchor sample rejected: %v", err)
	}

	// 30s backwards beyond the 10s tolerance.
	if err := guard.ValidateIngress(FieldIngressPayload{Device: strongBoxDevice(true), Timestamp: clockTick(now.Add(-31*time.Second), 200)}); !errors.Is(err, timeguard.ErrClockBackwardsDrift) {
		t.Fatalf("backwards drift not rejected: %v", err)
	}

	guard2 := NewIngressGuard(AttestationPolicy{}, 10*time.Second)
	if err := guard2.ValidateIngress(FieldIngressPayload{Device: strongBoxDevice(true), Timestamp: clockTick(now.Add(-time.Second), 100)}); err != nil {
		t.Fatalf("anchor sample rejected: %v", err)
	}
	// 30s forward jump beyond tolerance.
	if err := guard2.ValidateIngress(FieldIngressPayload{Device: strongBoxDevice(true), Timestamp: clockTick(now.Add(29*time.Second), 200)}); !errors.Is(err, timeguard.ErrClockForwardJump) {
		t.Fatalf("forward jump not rejected: %v", err)
	}
}

func TestValidateIngressRejectsStaleFabricatedTimestamp(t *testing.T) {
	now := time.Now().UTC()
	guard := NewIngressGuardWithNow(AttestationPolicy{}, 10*time.Second, func() time.Time { return now })

	// One lone sample fabricated days ago cannot anchor as monotonic truth.
	payload := FieldIngressPayload{Device: strongBoxDevice(true), Timestamp: clockTick(now.Add(-48*time.Hour), 1)}
	if err := guard.ValidateIngress(payload); !errors.Is(err, ErrIngressClockNotRecent) {
		t.Fatalf("stale fabricated timestamp not rejected: %v", err)
	}
}

func TestValidateReceiptAllAxesSimultaneously(t *testing.T) {
	now := time.Now().UTC()
	guard := NewIngressGuardWithNow(AttestationPolicy{RequireHardware: true}, 10*time.Second, func() time.Time { return now })

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	dev := strongBoxDevice(true)
	dev.DevicePublicKey = hex.EncodeToString(pub)
	dev.DeviceID = "dev_verified"

	receipt := SignedInspectionReceipt{
		ReceiptID:     "rcpt_safe",
		ManifestID:    "man_safe",
		WorkOrderID:   "wo_test",
		AssetID:       "asset_test",
		OverallResult: "PASSED",
		PayloadDigest: "aa",
		CompletedAt:   now.Add(-time.Second),
	}
	receipt.DeviceSignature = hex.EncodeToString(ed25519.Sign(priv, []byte(receiptSigningPayload(receipt))))

	validation := IngressReceiptValidation{
		Receipt:   receipt,
		Device:    dev,
		Timestamp: clockTick(now.Add(-time.Second), 100),
	}
	if err := guard.ValidateReceipt(validation); err != nil {
		t.Fatalf("valid receipt rejected: %v", err)
	}

	// (a) Tampered payload fails the signature axis alone.
	tampered := validation
	tampered.Receipt.OverallResult = "FAILED_CRITICAL"
	if err := guard.ValidateReceipt(tampered); err == nil {
		t.Fatal("tampered receipt accepted")
	}

	// (b) Monotonic regression combined with a still-fresh clock fails the
	// clock axis while signature and posture stay valid.
	replay := validation
	replay.Timestamp = clockTick(now.Add(-time.Second), 50)
	if err := guard.ValidateReceipt(replay); !errors.Is(err, timeguard.ErrMonotonicSequenceBroken) {
		t.Fatalf("clock rollback on receipt not rejected: %v", err)
	}

	// (c) Software posture fails even with a perfect signature and clock.
	soft := validation
	soft.Device = dev
	soft.Device.AttestationOrigin = string(KeyOriginSoftware)
	if err := guard.ValidateReceipt(soft); !errors.Is(err, ErrSoftwareOriginRejected) {
		t.Fatalf("software posture receipt accepted under RequireHardware: %v", err)
	}
}
