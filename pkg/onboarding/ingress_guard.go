package onboarding

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"integin/pkg/timeguard"
)

// Sentinel errors for ingress guarding. The zero-value stance is fail closed:
// every rejected payload returns one of these so callers never silently drop
// attestation or clock evidence (Hazard 32).
var (
	ErrIngressDeviceInactive = errors.New("ingress: field device is revoked or not active")
	ErrIngressClockNotRecent = errors.New("ingress: field clock is outside the allowed skew of server time")
	ErrIngressReceiptSigning = errors.New("ingress: device signature payload")
)

// IngressClockSample is one wall-clock reading submitted with a field payload,
// coupled to an independent monotonic tick (Hazard 33: monotonic must never
// regress across a live payload stream).
type IngressClockSample struct {
	WallTime      time.Time
	MonotonicNano int64
	Source        timeguard.TimeSource
}

// FieldIngressPayload is a live payload arriving from an enrolled field
// terminal: the trust record of the reporting device plus the timestamp it
// claims. Both axes are guarded before acceptance.
type FieldIngressPayload struct {
	Device    DeviceTrustRecord
	Timestamp IngressClockSample
}

// IngressReceiptValidation couples a sealed inspection receipt with the
// reporting device's trust posture and its claimed clock. A receipt is only
// accepted when all three verify simultaneously.
type IngressReceiptValidation struct {
	Receipt   SignedInspectionReceipt
	Device    DeviceTrustRecord
	Timestamp IngressClockSample
}

// IngressGuard is the server-side ingress choke point for field payloads. It
// combines hardware-attestation posture enforcement (checkPosture against an
// AttestationPolicy) with timeguard monotonic/drift validation on the device
// clock. It never trusts a raw client timestamp and never accepts a software
// key when the policy requires hardware.
type IngressGuard struct {
	clock   *timeguard.ClockGuard
	maxSkew time.Duration
	policy  AttestationPolicy
	now     func() time.Time
}

// NewIngressGuard returns a guard enforcing policy and tolerating up to
// maxAllowedClockSkew of wall-clock deviation across the live stream. Without
// an injected Now the guard still rejects monotonic regression and stream
// drift, but cannot reject a single forged timestamp; use
// NewIngressGuardWithNow where freshness matters.
func NewIngressGuard(policy AttestationPolicy, maxAllowedClockSkew time.Duration) *IngressGuard {
	return NewIngressGuardWithNow(policy, maxAllowedClockSkew, nil)
}

// NewIngressGuardWithNow additionally cross-checks every incoming wall-clock
// sample against server time within the same skew bound, so a lone receipt
// cannot anchor a fabricated timestamp as monotonic truth. now must return
// the current UTC time; passing nil disables the freshness cross-check.
func NewIngressGuardWithNow(policy AttestationPolicy, maxAllowedClockSkew time.Duration, now func() time.Time) *IngressGuard {
	return &IngressGuard{
		clock:   timeguard.NewClockGuard(maxAllowedClockSkew),
		maxSkew: maxAllowedClockSkew,
		policy:  policy,
		now:     now,
	}
}

// Policy returns the attestation policy in force for every guarded payload.
func (g *IngressGuard) Policy() AttestationPolicy {
	return g.policy
}

// ValidateIngress guards a live field payload: the reporting device must be
// active, its recorded posture must satisfy the policy (software keys are
// rejected when the policy requires hardware), and its clock must pass the
// monotonic/drift guard.
func (g *IngressGuard) ValidateIngress(payload FieldIngressPayload) error {
	if err := g.validateDevice(payload.Device); err != nil {
		return err
	}
	return g.validateClock(payload.Timestamp)
}

// ValidateReceipt verifies a sealed inspection receipt on all three axes at
// once: the device signature over the receipt payload, the device trust
// posture against policy, and the receipt's CompletedAt clock sample against
// timeguard plus freshness. Any single failing axis rejects the whole receipt.
func (g *IngressGuard) ValidateReceipt(v IngressReceiptValidation) error {
	if err := g.validateDevice(v.Device); err != nil {
		return err
	}
	if err := verifyReceiptSignature(v.Receipt, v.Device.DevicePublicKey); err != nil {
		return err
	}
	sample := IngressClockSample{
		WallTime:      v.Receipt.CompletedAt,
		MonotonicNano: v.Timestamp.MonotonicNano,
		Source:        v.Timestamp.Source,
	}
	if v.Receipt.CompletedAt.IsZero() {
		return fmt.Errorf("%w: completed_at is required", ErrIngressClockNotRecent)
	}
	return g.validateClock(sample)
}

// validateDevice enforces activeness and, when the policy requires hardware,
// also demands that the recorded attestation was actually verified offline at
// enrollment (AttestationVerified). checkPosture remains the enforcement axis
// for origin/biometric posture; the verified flag closes the loop so a
// claimed STRONGBOX origin that never supplied a verifiable chain is not
// treated as hardware trust.
func (g *IngressGuard) validateDevice(device DeviceTrustRecord) error {
	if !device.IsActive {
		return ErrIngressDeviceInactive
	}
	if err := checkPosture(device.AttestationOrigin, device.AttestationBiometricBound, g.policy); err != nil {
		return err
	}
	if g.policy.RequireHardware && !device.AttestationVerified {
		return errors.New("ingress: policy requires hardware attestation and the device's claim was never verified offline")
	}
	return nil
}

// validateClock feeds the sample into the timeguard ClockGuard (monotonic
// regression and drift rejection) and, when a Now provider is configured,
// rejects timestamps that are not fresh relative to server time.
func (g *IngressGuard) validateClock(sample IngressClockSample) error {
	src := sample.Source
	if src == "" {
		src = timeguard.MONOTONIC_LOCAL
	}
	if err := g.clock.ValidateSample(timeguard.MonotonicSample{
		WallTime:      sample.WallTime,
		MonotonicNano: sample.MonotonicNano,
		Source:        src,
	}); err != nil {
		return err
	}
	if g.now != nil {
		if err := g.freshness(sample); err != nil {
			return err
		}
	}
	return nil
}

// freshness rejects wall-clock samples that deviate from server time by more
// than the configured skew, preventing a lone payload from fabricating an
// anchored timestamp.
func (g *IngressGuard) freshness(sample IngressClockSample) error {
	skew := g.now().UTC().Sub(sample.WallTime)
	if skew < -g.maxSkew || skew > g.maxSkew {
		return fmt.Errorf("%w: sample %s deviates from server time by %s", ErrIngressClockNotRecent, sample.WallTime.UTC().Format(time.RFC3339Nano), skew)
	}
	return nil
}

// receiptSigningPayload is the canonical string the enrolled device signs when
// sealing a receipt: ReceiptID|ManifestID|AssetID|OverallResult|PayloadDigest.
// The same payload must be reconstructed by every verifier — keeping the one
// format here means ingress and enrollment receipt verification cannot drift
// apart.
func receiptSigningPayload(r SignedInspectionReceipt) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s", r.ReceiptID, r.ManifestID, r.AssetID, r.OverallResult, r.PayloadDigest)
}

// verifyReceiptSignature checks the device's Ed25519 signature over the
// canonical receipt signing payload using the device's enrolled public key.
func verifyReceiptSignature(r SignedInspectionReceipt, devicePublicKeyHex string) error {
	devPubBytes, err := hex.DecodeString(devicePublicKeyHex)
	if err != nil {
		return fmt.Errorf("ingress: corrupted device key in store: %w", err)
	}
	if len(devPubBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("ingress: corrupted device key in store: %d bytes", len(devPubBytes))
	}
	sigBytes, err := hex.DecodeString(r.DeviceSignature)
	if err != nil {
		return fmt.Errorf("%w: invalid receipt signature encoding", ErrIngressReceiptSigning)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("%w: invalid receipt signature length", ErrIngressReceiptSigning)
	}
	if !ed25519.Verify(devPubBytes, []byte(receiptSigningPayload(r)), sigBytes) {
		return errors.New("tamper alert: receipt signature invalid")
	}
	return nil
}
