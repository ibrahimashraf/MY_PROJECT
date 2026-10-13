package onboarding

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"integin/internal/domain/device_trust"
)

const defaultChallengeTTL = 10 * time.Minute

// EnrollmentSubmissionResult reports a successfully submitted enrollment that
// is now pending tenant-admin approval.
type EnrollmentSubmissionResult struct {
	RequestID      string    `json:"request_id"`
	TenantID       string    `json:"tenant_id"`
	OrganizationID string    `json:"organization_id"`
	UserID         string    `json:"user_id"`
	DeviceID       string    `json:"device_id"`
	Status         string    `json:"status"`
	SubmittedAt    time.Time `json:"submitted_at"`

	// Request is the created pending enrollment request, exposed so the
	// tenant-admin approval flow can act on it directly. It is never
	// serialized to the wire.
	Request *device_trust.EnrollmentRequest `json:"-"`
}

// ProductionEnrollmentService is the production device enrollment engine. It
// replaces the loopback EnrollmentSimulator assumptions: every submission is
// bound to an authenticated OIDC subject, challenges are expiring and
// single-use, attestation claims are policy-validated, and the outcome is a
// PENDING enrollment request that only a tenant admin can approve.
type ProductionEnrollmentService struct {
	mu             sync.Mutex
	now            func() time.Time
	challengeTTL   time.Duration
	policy         AttestationPolicy
	challenges     map[string]*DeviceEnrollmentChallenge
	orgByChallenge map[string]string
	pending        map[string]*device_trust.EnrollmentRequest
}

// NewProductionEnrollmentService creates the production enrollment engine
// enforcing the given attestation policy; the zero policy is permissive.
func NewProductionEnrollmentService(policy AttestationPolicy) *ProductionEnrollmentService {
	return &ProductionEnrollmentService{
		now:            time.Now,
		challengeTTL:   defaultChallengeTTL,
		policy:         policy,
		challenges:     make(map[string]*DeviceEnrollmentChallenge),
		orgByChallenge: make(map[string]string),
		pending:        make(map[string]*device_trust.EnrollmentRequest),
	}
}

// CreateChallenge issues a short-lived, single-use enrollment challenge bound
// to a tenant, an OIDC subject, and an organization.
func (s *ProductionEnrollmentService) CreateChallenge(tenantID, userID, orgID string) (*DeviceEnrollmentChallenge, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(orgID) == "" {
		return nil, errors.New("tenant_id, user_id, and organization_id are required")
	}
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, err
	}
	nonceHex := hex.EncodeToString(nonceBytes)
	challengeID := fmt.Sprintf("chal_%s", nonceHex[:8])

	s.mu.Lock()
	defer s.mu.Unlock()
	challenge := &DeviceEnrollmentChallenge{
		ChallengeID:   challengeID,
		TenantID:      tenantID,
		InspectorID:   userID,
		Nonce:         nonceHex,
		ExpiresAt:     s.now().Add(s.challengeTTL),
		QRCodePayload: fmt.Sprintf("integin://enroll?chal=%s&tenant=%s&nonce=%s", challengeID, tenantID, nonceHex),
	}
	s.challenges[challengeID] = challenge
	s.orgByChallenge[challengeID] = orgID
	return challenge, nil
}

// SubmitEnrollment consumes a challenge with a signed proof-of-possession
// submission. It enforces OIDC identity binding (the authenticated subject
// must match the submission's inspector), challenge validity, expiration, and
// single-use consumption, validates the attestation claim against the service
// policy, and produces a PENDING enrollment request awaiting admin approval.
func (s *ProductionEnrollmentService) SubmitEnrollment(sub DeviceEnrollmentSubmission, oidcSubject string) (*EnrollmentSubmissionResult, error) {
	if strings.TrimSpace(sub.ChallengeID) == "" {
		return nil, errors.New("challenge_id is required")
	}
	if strings.TrimSpace(oidcSubject) == "" {
		return nil, errors.New("oidc subject is required")
	}
	if sub.InspectorID != oidcSubject {
		return nil, errors.New("oidc identity mismatch: submission inspector does not match authenticated subject")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.challenges[sub.ChallengeID]
	if !ok {
		return nil, errors.New("invalid or already consumed challenge")
	}
	orgID := s.orgByChallenge[sub.ChallengeID]
	if s.now().After(challenge.ExpiresAt) {
		return nil, errors.New("challenge expired")
	}

	if sub.Attestation.KeyOrigin != "" {
		if err := VerifyClaim(sub.Attestation, s.policy); err != nil {
			return nil, fmt.Errorf("attestation claim rejected: %w", err)
		}
	}

	deviceID, err := deriveDeviceID(sub.DevicePublicKey)
	if err != nil {
		return nil, err
	}
	request, err := device_trust.CreateEnrollmentRequest(
		fmt.Sprintf("enr_%s", challenge.Nonce[:8]),
		challenge.TenantID,
		orgID,
		oidcSubject,
		deviceID,
		sub.DevicePublicKey,
		challenge.Nonce,
		sub.SignedNonce,
		toEnrollmentAttestation(sub.Attestation),
		s.now().UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("enrollment request rejected: %w", err)
	}

	delete(s.challenges, sub.ChallengeID)
	delete(s.orgByChallenge, sub.ChallengeID)
	s.pending[request.RequestID] = request
	return &EnrollmentSubmissionResult{
		RequestID: request.RequestID, TenantID: request.TenantID,
		OrganizationID: request.OrganizationID, UserID: request.UserID,
		DeviceID: request.DeviceID, Status: string(request.Status),
		SubmittedAt: request.RequestedAt, Request: request,
	}, nil
}

// deriveDeviceID deterministically binds the device identity to its signing
// key so a spoofed device id cannot be substituted for another key.
func deriveDeviceID(publicKey string) (string, error) {
	pub, err := device_trust.DecodePublicKey(publicKey)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("dev_%s", hex.EncodeToString(pub[:6])), nil
}

// toEnrollmentAttestation copies the onboarding attestation claim into the
// domain enrollment record.
func toEnrollmentAttestation(claim AttestationClaim) device_trust.EnrollmentAttestation {
	return device_trust.EnrollmentAttestation{
		KeyOrigin:      string(claim.KeyOrigin),
		BiometricBound: claim.BiometricBound,
		OSVersion:      claim.OSVersion,
	}
}
