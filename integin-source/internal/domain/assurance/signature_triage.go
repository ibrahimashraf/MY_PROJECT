package assurance

import (
	"crypto/ed25519"
	"errors"
	"time"
)

var (
	ErrSignatureTampered = errors.New("cryptographic signature is invalid against all known keys (tampering detected)")
	ErrKeyRevoked        = errors.New("signature verified, but key was strictly revoked prior to evaluation")
	ErrQuarantineRequired = errors.New("signature verified, but falls within 24h revocation grace period (quarantine required)")
)

// PublicKeyRecord represents a historical or current Ed25519 public key for a signer.
type PublicKeyRecord struct {
	Key       ed25519.PublicKey
	ValidFrom time.Time
	ValidTo   *time.Time
	RevokedAt *time.Time
}

// KeyRegistry defines the port for fetching a signer's cryptographic identity history.
type KeyRegistry interface {
	GetSignerKeys(signerID string) ([]PublicKeyRecord, error)
}

// SignatureTriage executes the deterministic recovery tree for signature verification.
type SignatureTriage struct {
	registry KeyRegistry
}

func NewSignatureTriage(registry KeyRegistry) *SignatureTriage {
	return &SignatureTriage{registry: registry}
}

// VerifyRecord executes the corrected Phase 4 protocol:
// 1. Mathematical verification against all keys.
// 2. Temporal validation of the key's lifecycle against the record's EffectiveAt.
func (t *SignatureTriage) VerifyRecord(payload []byte, signature []byte, signerID string, effectiveAt time.Time) error {
	keys, err := t.registry.GetSignerKeys(signerID)
	if err != nil || len(keys) == 0 {
		return ErrSignatureTampered // Fail closed if no keys exist
	}

	var validKey *PublicKeyRecord

	// Phase 1: Mathematical Verification (MUST precede temporal trust)
	for _, k := range keys {
		if ed25519.Verify(k.Key, payload, signature) {
			validKey = &k
			break
		}
	}

	if validKey == nil {
		// The payload does not mathematically match ANY known key.
		// The payload or timestamp cannot be trusted. Immediate reject.
		return ErrSignatureTampered
	}

	// Phase 2: Temporal & Revocation Evaluation
	// We now mathematically trust that the signer generated this payload.
	// We can now safely evaluate the effective timestamp against the key lifecycle.

	// Did the inspection happen before the key was valid?
	if effectiveAt.Before(validKey.ValidFrom) {
		return ErrSignatureTampered // Time-travel payload
	}

	// Was the key revoked?
	if validKey.RevokedAt != nil {
		// If revoked BEFORE the inspection started
		if effectiveAt.After(*validKey.RevokedAt) {
			gracePeriodEnd := validKey.RevokedAt.Add(24 * time.Hour)
			
			if effectiveAt.Before(gracePeriodEnd) {
				// Within 24h grace window -> Quarantine (Provisionally Non-Compliant)
				return ErrQuarantineRequired
			}
			// Past the 24h grace window -> Hard Reject
			return ErrKeyRevoked
		}
	}

	// Key is valid and was active at the time of evaluation.
	return nil
}
