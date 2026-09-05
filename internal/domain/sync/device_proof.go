package sync

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"integin/internal/domain/device_trust"
	"integin/internal/domain/workpackage"
	"integin/internal/security"
)

const (
	// DeviceProofProtocolVersion is the only accepted proof envelope version.
	DeviceProofProtocolVersion = workpackage.ManifestProofProtocolVersion
	// WorkPackageManifestReadPurpose scopes a proof to package-manifest retrieval.
	WorkPackageManifestReadPurpose = workpackage.ManifestProofPurpose
	proofMaximumLifetime           = 5 * time.Minute
	proofFutureSkew                = 30 * time.Second
)

// DeviceProof is a signed, short-lived request proof. Its device ID is a
// lookup key only; verified server-side state supplies the trusted scope.
type DeviceProof struct {
	ProtocolVersion    string
	Purpose            string
	RequestID          string
	DeviceID           string
	AuthorityID        string
	AuthorityEpoch     uint64
	InspectionID       string
	IssuedAt           time.Time
	ExpiresAt          time.Time
	SignatureAlgorithm string
	KeyID              string
	Signature          string
}

// VerifiedDeviceContext contains only scope derived from server-held device and
// authority state after a proof has passed cryptographic verification.
type VerifiedDeviceContext struct {
	TenantID        string
	OrganizationID  string
	UserID          string
	DeviceID        string
	AuthorityID     string
	AuthorityEpoch  uint64
	ProofExpiresAt  time.Time
	AuthorityExpiry time.Time
}

// VerifyDeviceProof validates a short-lived manifest-read proof against
// registered server-held device and authority state. It deliberately performs
// no replay-store mutation; a mounted handler must consume RequestID through a
// durable replay boundary after this verification succeeds.
func (p *Processor) VerifyDeviceProof(
	ctx context.Context,
	proof DeviceProof,
	authority device_trust.AuthorityPackage,
	at time.Time,
) (VerifiedDeviceContext, error) {
	if p == nil {
		return VerifiedDeviceContext{}, errors.New("sync processor is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return VerifiedDeviceContext{}, fmt.Errorf("device proof context: %w", err)
	}
	if at.IsZero() {
		return VerifiedDeviceContext{}, errors.New("device proof verification time is required")
	}
	if proof.ProtocolVersion != DeviceProofProtocolVersion {
		return VerifiedDeviceContext{}, errors.New("unsupported device proof protocol version")
	}
	if proof.Purpose != WorkPackageManifestReadPurpose {
		return VerifiedDeviceContext{}, errors.New("unsupported device proof purpose")
	}
	if strings.TrimSpace(proof.RequestID) == "" || strings.TrimSpace(proof.DeviceID) == "" || strings.TrimSpace(proof.InspectionID) == "" {
		return VerifiedDeviceContext{}, errors.New("device proof request, device, and inspection are required")
	}
	if strings.Contains(proof.RequestID, "|") || strings.Contains(proof.DeviceID, "|") || strings.Contains(proof.AuthorityID, "|") || strings.Contains(proof.InspectionID, "|") || strings.Contains(proof.KeyID, "|") {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureSignatureInvalid, errors.New("device proof fields cannot contain pipe delimiter '|'"))
	}
	if proof.IssuedAt.IsZero() || proof.ExpiresAt.IsZero() {
		return VerifiedDeviceContext{}, errors.New("device proof issuance and expiry are required")
	}
	issuedAt := proof.IssuedAt.UTC()
	expiresAt := proof.ExpiresAt.UTC()
	now := at.UTC()
	if issuedAt.After(now.Add(proofFutureSkew)) || !expiresAt.After(now) || !expiresAt.After(issuedAt) || expiresAt.Sub(issuedAt) > proofMaximumLifetime {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureExpired, errors.New("device proof is outside its permitted lifetime"))
	}
	if proof.SignatureAlgorithm != workpackage.ManifestProofSignatureAlgorithm || strings.TrimSpace(proof.KeyID) == "" || strings.TrimSpace(proof.Signature) == "" {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureSignatureInvalid, errors.New("device proof requires Ed25519 signature metadata"))
	}

	p.mu.RLock()
	device, found := p.devices[proof.DeviceID]
	var secret string
	for _, v := range p.secrets {
		secret = v
		break
	}
	p.mu.RUnlock()
	if !found {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureKeyUnknown, errors.New("device is not registered"))
	}
	if authority.ID != proof.AuthorityID || authority.Epoch != proof.AuthorityEpoch {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureAuthorityMismatch, errors.New("authority reference does not match server-held authority"))
	}
	if now.Before(authority.IssuedAt.UTC()) || !now.Before(authority.ExpiresAt.UTC()) {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureExpired, errors.New("authority package is outside its permitted lifetime"))
	}
	if err := device_trust.ValidateAuthorityPackage(authority, device, secret, now); err != nil {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureAuthorityMismatch, fmt.Errorf("validate device authority: %w", err))
	}
	if !contains(authority.Scopes, "work_package.read") {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureAuthorityMismatch, errors.New("authority does not permit work package manifest read"))
	}

	encodedKey := device.PublicKey()
	publicKey, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureKeyUnknown, errors.New("registered device public key is invalid"))
	}
	if proof.KeyID != security.DeviceKeyID(ed25519.PublicKey(publicKey)) {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureKeyUnknown, errors.New("device proof key identifier does not match registered device"))
	}
	signature, err := base64.StdEncoding.DecodeString(proof.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureSignatureInvalid, errors.New("device proof signature is invalid"))
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), []byte(canonicalDeviceProof(proof)), signature) {
		return VerifiedDeviceContext{}, proofFailure(DeviceProofFailureSignatureInvalid, errors.New("device proof signature verification failed"))
	}

	return VerifiedDeviceContext{
		TenantID:        device.TenantID(),
		OrganizationID:  device.OrganizationID(),
		UserID:          device.UserID(),
		DeviceID:        device.ID(),
		AuthorityID:     authority.ID,
		AuthorityEpoch:  authority.Epoch,
		ProofExpiresAt:  expiresAt,
		AuthorityExpiry: authority.ExpiresAt.UTC(),
	}, nil
}

func canonicalDeviceProof(proof DeviceProof) string {
	return workpackage.CanonicalManifestReadProof(
		proof.RequestID,
		proof.DeviceID,
		proof.AuthorityID,
		proof.AuthorityEpoch,
		proof.InspectionID,
		proof.IssuedAt,
		proof.ExpiresAt,
		proof.KeyID,
	)
}
