package onboarding

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// Simulator handles end-to-end device cryptographic enrollment and authority gating.
type EnrollmentSimulator struct {
	serverPubKey  ed25519.PublicKey
	serverPrivKey ed25519.PrivateKey
	deviceStore   map[string]DeviceTrustRecord
	challenges    map[string]DeviceEnrollmentChallenge
	manifests     map[string]WorkPackageManifest
}

func NewEnrollmentSimulator() (*EnrollmentSimulator, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &EnrollmentSimulator{
		serverPubKey:  pub,
		serverPrivKey: priv,
		deviceStore:   make(map[string]DeviceTrustRecord),
		challenges:    make(map[string]DeviceEnrollmentChallenge),
		manifests:     make(map[string]WorkPackageManifest),
	}, nil
}

// 1. CreateEnrollmentChallenge creates a short-lived nonce for QR pairing.
func (s *EnrollmentSimulator) CreateEnrollmentChallenge(tenantID, inspectorID string) (*DeviceEnrollmentChallenge, error) {
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, err
	}
	nonceHex := hex.EncodeToString(nonceBytes)
	challengeID := fmt.Sprintf("chal_%s", nonceHex[:8])

	challenge := DeviceEnrollmentChallenge{
		ChallengeID:   challengeID,
		TenantID:      tenantID,
		InspectorID:   inspectorID,
		Nonce:         nonceHex,
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		QRCodePayload: fmt.Sprintf("integin://enroll?chal=%s&tenant=%s&nonce=%s", challengeID, tenantID, nonceHex),
	}
	s.challenges[challengeID] = challenge
	return &challenge, nil
}

// 2. ProcessDeviceEnrollment verifies device signature and binds public key to inspector.
func (s *EnrollmentSimulator) ProcessDeviceEnrollment(sub DeviceEnrollmentSubmission) (*DeviceTrustRecord, error) {
	chal, exists := s.challenges[sub.ChallengeID]
	if !exists {
		return nil, errors.New("invalid or unknown challenge ID")
	}
	if time.Now().After(chal.ExpiresAt) {
		return nil, errors.New("challenge expired")
	}

	devPubBytes, err := hex.DecodeString(sub.DevicePublicKey)
	if err != nil || len(devPubBytes) != ed25519.PublicKeySize {
		return nil, errors.New("invalid device public key format")
	}

	sigBytes, err := hex.DecodeString(sub.SignedNonce)
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}

	// Cryptographic verification: verify signed nonce with device public key
	if !ed25519.Verify(devPubBytes, []byte(chal.Nonce), sigBytes) {
		return nil, errors.New("cryptographic verification failed: device signature does not match nonce")
	}

	// Attestation claim: when present (non-zero KeyOrigin), validate it and
	// enforce the default permissive policy. Strict policies are opt-in via
	// VerifyClaim for callers that require hardware/biometric guarantees.
	// When the claim is an Android STRONGBOX origin AND the device submitted
	// a certificate chain, the chain must verify offline against the
	// operator-configured roots or enrollment fails closed (no record is
	// stored). Non-Android origins ignore ChainPEM entirely.
	var attestationVerified bool
	if sub.Attestation.KeyOrigin != "" {
		if err := VerifyClaim(sub.Attestation, AttestationPolicy{}); err != nil {
			return nil, fmt.Errorf("attestation claim rejected: %w", err)
		}
		if sub.Attestation.KeyOrigin == KeyOriginStrongBox && len(sub.Attestation.ChainPEM) > 0 {
			if _, err := verifyPEMChain(sub.Attestation.ChainPEM, chainVerifyRoots, []byte(chal.Nonce)); err != nil {
				return nil, fmt.Errorf("attestation chain rejected: %w", err)
			}
			attestationVerified = true
		}
	}

	// Register trusted device
	deviceID := fmt.Sprintf("dev_%s", hex.EncodeToString(devPubBytes[:6]))
	record := DeviceTrustRecord{
		DeviceID:                  deviceID,
		TenantID:                  chal.TenantID,
		InspectorID:               sub.InspectorID,
		DevicePublicKey:           sub.DevicePublicKey,
		DeviceModel:               sub.DeviceModel,
		IsActive:                  true,
		EnrolledAt:                time.Now(),
		LastSyncedAt:              time.Now(),
		AttestationOrigin:         string(sub.Attestation.KeyOrigin),
		AttestationBiometricBound: sub.Attestation.BiometricBound,
		AttestationVerified:       attestationVerified,
	}

	s.deviceStore[deviceID] = record
	delete(s.challenges, sub.ChallengeID) // single-use challenge consumption
	return &record, nil
}

// 3. IssueWorkPackageManifest signs a work package bundle for the paired device.
func (s *EnrollmentSimulator) IssueWorkPackageManifest(tenantID, workOrderID, deviceID, inspectorID string, checklistJSON string) (*WorkPackageManifest, error) {
	dev, exists := s.deviceStore[deviceID]
	if !exists || !dev.IsActive {
		return nil, errors.New("device not enrolled or inactive")
	}
	if dev.InspectorID != inspectorID {
		return nil, errors.New("authority mismatch: device not bound to inspector")
	}

	// Package digest
	hash := sha256.Sum256([]byte(checklistJSON))
	packageHash := hex.EncodeToString(hash[:])

	manifestID := fmt.Sprintf("man_%s", packageHash[:8])
	now := time.Now()
	validUntil := now.Add(24 * time.Hour)

	// Server authority signs manifest
	signPayload := fmt.Sprintf("%s|%s|%s|%s|%s", manifestID, tenantID, workOrderID, deviceID, packageHash)
	sig := ed25519.Sign(s.serverPrivKey, []byte(signPayload))

	manifest := WorkPackageManifest{
		ManifestID:         manifestID,
		TenantID:           tenantID,
		WorkOrderID:        workOrderID,
		AssignedDeviceID:   deviceID,
		InspectorID:        inspectorID,
		IssuedAt:           now,
		ValidUntil:         validUntil,
		PackageHash:        packageHash,
		AuthoritySignature: hex.EncodeToString(sig),
	}
	s.manifests[manifestID] = manifest
	return &manifest, nil
}

// 4. VerifyOfflineReceipt verifies the sealed receipt submitted by the tablet.
func (s *EnrollmentSimulator) VerifyOfflineReceipt(receipt SignedInspectionReceipt) (bool, error) {
	manifest, exists := s.manifests[receipt.ManifestID]
	if !exists {
		return false, errors.New("unknown manifest ID")
	}

	dev, exists := s.deviceStore[manifest.AssignedDeviceID]
	if !exists || !dev.IsActive {
		return false, errors.New("device revoked or not active")
	}

	devPubBytes, err := hex.DecodeString(dev.DevicePublicKey)
	if err != nil {
		return false, errors.New("corrupted device key in store")
	}

	sigBytes, err := hex.DecodeString(receipt.DeviceSignature)
	if err != nil {
		return false, errors.New("invalid receipt signature encoding")
	}

	// Receipt signature payload: ReceiptID|ManifestID|AssetID|OverallResult|PayloadDigest
	receiptSignPayload := fmt.Sprintf("%s|%s|%s|%s|%s", receipt.ReceiptID, receipt.ManifestID, receipt.AssetID, receipt.OverallResult, receipt.PayloadDigest)
	
	if !ed25519.Verify(devPubBytes, []byte(receiptSignPayload), sigBytes) {
		return false, errors.New("tamper alert: receipt signature invalid")
	}

	return true, nil
}

// VerifyOfflineReceiptWithPolicy verifies a receipt exactly like
// VerifyOfflineReceipt and then enforces the device attestation posture
// recorded at enrollment against policy. The zero policy is permissive and
// behaves identically to VerifyOfflineReceipt. On policy failure it returns
// (false, one of the attestation sentinel errors).
func (s *EnrollmentSimulator) VerifyOfflineReceiptWithPolicy(receipt SignedInspectionReceipt, policy AttestationPolicy) (bool, error) {
	valid, err := s.VerifyOfflineReceipt(receipt)
	if err != nil || !valid {
		return false, err
	}
	if policy == (AttestationPolicy{}) {
		return true, nil
	}
	manifest, exists := s.manifests[receipt.ManifestID]
	if !exists {
		return false, errors.New("unknown manifest ID")
	}
	dev, exists := s.deviceStore[manifest.AssignedDeviceID]
	if !exists || !dev.IsActive {
		return false, errors.New("device revoked or not active")
	}
	if err := checkPosture(dev.AttestationOrigin, dev.AttestationBiometricBound, policy); err != nil {
		return false, err
	}
	return true, nil
}
