package onboarding

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"integin/internal/domain/device_trust"
)

// ClientEnrollmentCoordinator is the field-tablet-side model of the production
// enrollment flow. It owns a fresh Ed25519 keypair, signs the server-issued
// challenge nonce as proof of possession, constructs the EnrollmentRequest
// payload, and tracks its tenant-admin approval status. It pairs with
// ProductionEnrollmentService on the server side.
type ClientEnrollmentCoordinator struct {
	mu             sync.RWMutex
	tenantID       string
	organizationID string
	userID         string
	deviceModel    string
	attestation    AttestationClaim
	publicKey      ed25519.PublicKey
	privateKey     ed25519.PrivateKey
	request        *device_trust.EnrollmentRequest
}

// NewClientEnrollmentCoordinator creates a coordinator bound to the given OIDC
// subject and organization, generating a fresh device keypair from the
// platform RNG.
func NewClientEnrollmentCoordinator(tenantID, organizationID, userID, deviceModel string, attestation AttestationClaim) (*ClientEnrollmentCoordinator, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(organizationID) == "" || strings.TrimSpace(userID) == "" {
		return nil, errors.New("tenant_id, organization_id, and user_id are required")
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &ClientEnrollmentCoordinator{
		tenantID: tenantID, organizationID: organizationID, userID: userID,
		deviceModel: deviceModel, attestation: attestation, publicKey: pub, privateKey: priv,
	}, nil
}

// PublicKey returns the device public key hex-encoded, in the wire format the
// server decodes.
func (c *ClientEnrollmentCoordinator) PublicKey() string {
	return hex.EncodeToString(c.publicKey)
}

// DeviceID is the deterministic device identity derived from the signing key,
// so a spoofed device id can never be substituted for another key.
func (c *ClientEnrollmentCoordinator) DeviceID() (string, error) {
	return deriveDeviceID(c.PublicKey())
}

// SignChallenge validates the server challenge and signs its nonce, producing
// the enrollment submission payload. The challenge must be unexpired and bound
// to this coordinator's OIDC subject, otherwise the proof of possession is
// refused before it ever leaves the device.
func (c *ClientEnrollmentCoordinator) SignChallenge(challenge *DeviceEnrollmentChallenge) (*DeviceEnrollmentSubmission, error) {
	if challenge == nil {
		return nil, errors.New("challenge is required")
	}
	if strings.TrimSpace(challenge.ChallengeID) == "" {
		return nil, errors.New("challenge id is required")
	}
	if len(challenge.Nonce) < 8 {
		return nil, errors.New("challenge nonce is too short")
	}
	if challenge.InspectorID != c.userID {
		return nil, errors.New("challenge is bound to a different user")
	}
	if time.Now().After(challenge.ExpiresAt) {
		return nil, errors.New("challenge expired")
	}
	signature := ed25519.Sign(c.privateKey, []byte(challenge.Nonce))
	return &DeviceEnrollmentSubmission{
		ChallengeID:     challenge.ChallengeID,
		InspectorID:     c.userID,
		DevicePublicKey: c.PublicKey(),
		DeviceModel:     c.deviceModel,
		SignedNonce:     hex.EncodeToString(signature),
		Attestation:     c.attestation,
	}, nil
}

// BuildEnrollmentRequest constructs the EnrollmentRequest payload for the
// server's tenant-admin approval flow. It signs the challenge nonce and reuses
// the domain builder, which re-verifies proof of possession and fails closed
// on any invalid field. The resulting request is retained for approval-status
// tracking.
func (c *ClientEnrollmentCoordinator) BuildEnrollmentRequest(challenge *DeviceEnrollmentChallenge, requestedAt time.Time) (*device_trust.EnrollmentRequest, error) {
	sub, err := c.SignChallenge(challenge)
	if err != nil {
		return nil, err
	}
	deviceID, err := deriveDeviceID(sub.DevicePublicKey)
	if err != nil {
		return nil, err
	}
	request, err := device_trust.CreateEnrollmentRequest(
		fmt.Sprintf("enr_%s", challenge.Nonce[:8]),
		c.tenantID, c.organizationID, c.userID, deviceID,
		sub.DevicePublicKey, challenge.Nonce, sub.SignedNonce,
		toEnrollmentAttestation(c.attestation), requestedAt,
	)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.request = request
	c.mu.Unlock()
	return request, nil
}

// ApprovalStatus reports the lifecycle state of the last constructed
// enrollment request. Callers that see PENDING must wait for the tenant admin
// to ApproveEnrollment (which flips the shared request to APPROVED).
func (c *ClientEnrollmentCoordinator) ApprovalStatus() (device_trust.EnrollmentStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.request == nil {
		return "", errors.New("no enrollment request has been constructed")
	}
	return c.request.Status, nil
}
