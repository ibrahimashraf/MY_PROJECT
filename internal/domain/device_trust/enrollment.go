package device_trust

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// EventEnrollmentApproved is the audit event emitted when a tenant admin
// approves a pending enrollment request.
const EventEnrollmentApproved = "EnrollmentApproved"

// EventDeviceKeyRotated is the audit event emitted when a trusted device
// rotates its signing key and bumps its authority epoch.
const EventDeviceKeyRotated = "DeviceKeyRotated"

// EnrollmentStatus is the lifecycle state of a device enrollment request.
type EnrollmentStatus string

const (
	EnrollmentPending  EnrollmentStatus = "PENDING"
	EnrollmentApproved EnrollmentStatus = "APPROVED"
	EnrollmentRejected EnrollmentStatus = "REJECTED"
)

// EnrollmentAttestation is the hardware-attestation evidence recorded on an
// enrollment request. The attestation blob itself is opaque evidence validated
// at the onboarding boundary; the domain only carries the posture facts.
type EnrollmentAttestation struct {
	KeyOrigin      string `json:"key_origin"`
	BiometricBound bool   `json:"biometric_bound"`
	OSVersion      string `json:"os_version,omitempty"`
}

// EnrollmentRequest is a tenant-bound device enrollment awaiting admin
// approval. UserID is the OIDC subject that authenticated the submission.
type EnrollmentRequest struct {
	RequestID      string                `json:"request_id"`
	TenantID       string                `json:"tenant_id"`
	OrganizationID string                `json:"organization_id"`
	UserID         string                `json:"user_id"`
	DeviceID       string                `json:"device_id"`
	PublicKey      string                `json:"public_key"`
	Nonce          string                `json:"nonce"`
	Signature      string                `json:"signature"`
	Attestation    EnrollmentAttestation `json:"attestation,omitempty"`
	RequestedAt    time.Time             `json:"requested_at"`
	Status         EnrollmentStatus      `json:"status"`
}

// DecodePublicKey decodes an Ed25519 public key provided as hex or base64.
func DecodePublicKey(s string) (ed25519.PublicKey, error) {
	b, err := decodeKeyMaterial(s, ed25519.PublicKeySize)
	if err != nil {
		return nil, fmt.Errorf("invalid ed25519 public key: %w", err)
	}
	return ed25519.PublicKey(b), nil
}

// decodeKeyMaterial decodes exactly size bytes of hex or base64 key material.
// It fails closed on any ambiguity: the decoded length must match exactly.
func decodeKeyMaterial(s string, size int) ([]byte, error) {
	if strings.TrimSpace(s) == "" {
		return nil, errors.New("empty key material")
	}
	if b, err := hex.DecodeString(s); err == nil && len(b) == size {
		return b, nil
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil && len(b) == size {
			return b, nil
		}
	}
	return nil, fmt.Errorf("expected %d bytes encoded as hex or base64", size)
}

// CreateEnrollmentRequest validates the required identity fields and
// cryptographically verifies the device proof of possession: the signature
// over the server-issued nonce, checked against the claimed public key.
// It fails closed on invalid signatures, malformed keys, or missing fields,
// and returns a PENDING request ready for tenant-admin approval.
func CreateEnrollmentRequest(requestID, tenantID, organizationID, userID, deviceID, publicKey, nonce, signature string, attestation EnrollmentAttestation, requestedAt time.Time) (*EnrollmentRequest, error) {
	for field, value := range map[string]string{
		"request_id": requestID, "tenant_id": tenantID, "organization_id": organizationID,
		"user_id": userID, "device_id": deviceID, "public_key": publicKey,
		"nonce": nonce, "signature": signature,
	} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("%s is required", field)
		}
	}
	if requestedAt.IsZero() {
		return nil, errors.New("requested_at is required")
	}
	pub, err := DecodePublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	sig, err := decodeKeyMaterial(signature, ed25519.SignatureSize)
	if err != nil {
		return nil, fmt.Errorf("invalid ed25519 signature: %w", err)
	}
	if !ed25519.Verify(pub, []byte(nonce), sig) {
		return nil, errors.New("proof of possession failed: signature does not verify against public key and nonce")
	}
	return &EnrollmentRequest{
		RequestID: requestID, TenantID: tenantID, OrganizationID: organizationID,
		UserID: userID, DeviceID: deviceID, PublicKey: publicKey,
		Nonce: nonce, Signature: signature, Attestation: attestation,
		RequestedAt: requestedAt.UTC(), Status: EnrollmentPending,
	}, nil
}

// ApproveEnrollment is the tenant-admin gate: it promotes a PENDING request
// into a trusted device, emits the device-trust event plus an approval audit
// event, and returns the trusted device. Already-approved or rejected
// requests cannot be re-approved.
func ApproveEnrollment(req *EnrollmentRequest, approvedBy string) (*Device, error) {
	if req == nil {
		return nil, errors.New("enrollment request is required")
	}
	if strings.TrimSpace(approvedBy) == "" {
		return nil, errors.New("approving tenant admin is required")
	}
	if req.Status != EnrollmentPending {
		return nil, fmt.Errorf("only pending enrollments can be approved, got %s", req.Status)
	}
	device, err := NewDevice(req.DeviceID, req.TenantID, req.OrganizationID, req.UserID, req.PublicKey)
	if err != nil {
		return nil, err
	}
	if err := device.Trust(); err != nil {
		return nil, err
	}
	if err := device.appendEvent(EventEnrollmentApproved, map[string]string{
		"request_id": req.RequestID, "device_id": device.id, "approved_by": approvedBy,
	}); err != nil {
		return nil, err
	}
	req.Status = EnrollmentApproved
	return &device, nil
}

// RejectEnrollment is the tenant-admin gate for refusing a PENDING request.
// The approval flow is permanently closed for a rejected request.
func RejectEnrollment(req *EnrollmentRequest, rejectedBy, reason string) error {
	if req == nil {
		return errors.New("enrollment request is required")
	}
	if strings.TrimSpace(rejectedBy) == "" {
		return errors.New("rejecting tenant admin is required")
	}
	if strings.TrimSpace(reason) == "" {
		return errors.New("rejection reason is required")
	}
	if req.Status != EnrollmentPending {
		return fmt.Errorf("only pending enrollments can be rejected, got %s", req.Status)
	}
	req.Status = EnrollmentRejected
	return nil
}

// RotateDeviceKey verifies proof of possession of a new key, then bumps the
// device authority epoch (invalidating every previously issued authority
// package) and swaps the live public key. The device keeps its trust state.
func RotateDeviceKey(device *Device, newPublicKey, newNonce, newSig string) error {
	if device == nil {
		return errors.New("device is required")
	}
	newKey, err := DecodePublicKey(newPublicKey)
	if err != nil {
		return err
	}
	sig, err := decodeKeyMaterial(newSig, ed25519.SignatureSize)
	if err != nil {
		return fmt.Errorf("invalid ed25519 signature: %w", err)
	}
	if !ed25519.Verify(newKey, []byte(newNonce), sig) {
		return errors.New("key rotation proof of possession failed: signature does not verify against new public key")
	}
	device.epoch++
	if err := device.appendEvent(EventDeviceKeyRotated, map[string]any{"epoch": device.epoch, "public_key": newPublicKey}); err != nil {
		device.epoch--
		return err
	}
	device.publicKey = newPublicKey
	return nil
}

// appendEvent records an audit event on the device's emitted trail.
func (d *Device) appendEvent(eventType string, payload any) error {
	event, err := d.makeEvent(eventType, payload)
	if err != nil {
		return err
	}
	d.emitted = append(d.emitted, event)
	return nil
}
