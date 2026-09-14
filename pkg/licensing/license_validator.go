package licensing

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidSignature   = errors.New("license signature verification failed: token is forged or corrupted")
	ErrLicenseExpired     = errors.New("license has expired")
	ErrLicenseNotYetValid = errors.New("license is not yet valid (notBefore check failed)")
	ErrHardwareMismatch   = errors.New("hardware lock ID does not match running appliance")
	ErrJurisdictionDenied = errors.New("operating jurisdiction is not permitted by license covenants")
)

// UniversalLicenseValidator validates license tokens locally with zero cloud phone-home requirement.
type UniversalLicenseValidator struct {
	trustedRootPublicKey ed25519.PublicKey
	timeSource           timeSource
}

type timeSource interface {
	Now() time.Time
}

type realTimeSource struct{}

func (realTimeSource) Now() time.Time { return time.Now().UTC() }

// NewUniversalLicenseValidator initializes a validator with the vendor's pinned Ed25519 public key.
func NewUniversalLicenseValidator(trustedPublicKeyHex string) (*UniversalLicenseValidator, error) {
	pubBytes, err := hex.DecodeString(strings.TrimSpace(trustedPublicKeyHex))
	if err != nil {
		return nil, err
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, errors.New("invalid Ed25519 public key size")
	}

	return &UniversalLicenseValidator{
		trustedRootPublicKey: ed25519.PublicKey(pubBytes),
		timeSource:           realTimeSource{},
	}, nil
}

// ValidateToken verifies cryptographic integrity, validity window, and hardware constraints.
func (v *UniversalLicenseValidator) ValidateToken(token *SignedLicenseToken, currentHardwareID string) error {
	if token == nil {
		return errors.New("license token cannot be nil")
	}

	sigBytes, err := hex.DecodeString(token.Signature)
	if err != nil {
		return ErrInvalidSignature
	}

	payloadBytes, err := json.Marshal(token.Payload)
	if err != nil {
		return err
	}

	digest := sha256.Sum256(payloadBytes)
	if !ed25519.Verify(v.trustedRootPublicKey, digest[:], sigBytes) {
		return ErrInvalidSignature
	}

	return checkLicenseWindow(token.Payload, currentHardwareID, v.timeSource.Now())
}

// checkLicenseWindow enforces the validity window and hardware lock constraints
// shared by every validator shape (single key and m-of-n quorum).
func checkLicenseWindow(payload LicensePayload, currentHardwareID string, now time.Time) error {
	if !payload.NotBefore.IsZero() && now.Before(payload.NotBefore) {
		return ErrLicenseNotYetValid
	}
	if !payload.ExpiresAt.IsZero() && now.After(payload.ExpiresAt) {
		return ErrLicenseExpired
	}

	// Hardware Lock Constraint (for Air-Gapped Dedicated Appliances)
	if payload.HardwareLockID != "" {
		if currentHardwareID == "" || strings.TrimSpace(payload.HardwareLockID) != strings.TrimSpace(currentHardwareID) {
			return ErrHardwareMismatch
		}
	}

	return nil
}

// CheckJurisdictionAllowed verifies if the target country is within the licensed scope.
func (v *UniversalLicenseValidator) CheckJurisdictionAllowed(token *SignedLicenseToken, isoCode string) error {
	if token == nil {
		return errors.New("license token cannot be nil")
	}
	cleanCode := strings.ToUpper(strings.TrimSpace(isoCode))
	for _, allowed := range token.Payload.Covenants.AllowedJurisdictions {
		if allowed == "GLOBAL" || strings.ToUpper(strings.TrimSpace(allowed)) == cleanCode {
			return nil
		}
	}
	return ErrJurisdictionDenied
}
