package licensing

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrInvalidThreshold reports an invalid m-of-n configuration.
	ErrInvalidThreshold = errors.New("m-of-n threshold is invalid")
	// ErrThresholdExceedsRing reports M greater than the authorized key ring size.
	ErrThresholdExceedsRing = errors.New("m-of-n threshold exceeds the key ring size")
	// ErrQuorumNotMet reports fewer than M distinct authorized vendor key
	// signatures over the token payload.
	ErrQuorumNotMet = errors.New("m-of-n quorum not met: fewer authorized signatures than threshold")
)

// ringKey is one authorized vendor public key in the signing ring.
type ringKey struct {
	hexID string
	pub   ed25519.PublicKey
}

// MOfNValidator replaces a single embedded vendor public key with a ring of N
// authorized vendor public keys. A token is trusted only when at least M
// distinct ring keys have each produced a valid signature (the primary
// Signature plus its Signoffs) over the same canonical payload digest.
type MOfNValidator struct {
	ring       []ringKey
	m          int
	timeSource timeSource
}

// NewMOfNValidator builds an m-of-n validator from N hex-encoded Ed25519
// public keys. M must be between 1 and N inclusive, keys must be valid, and
// duplicate keys are rejected so one vendor can never be double-counted.
func NewMOfNValidator(hexKeys []string, m int) (*MOfNValidator, error) {
	if len(hexKeys) < 1 || m < 1 {
		return nil, ErrInvalidThreshold
	}
	if m > len(hexKeys) {
		return nil, ErrThresholdExceedsRing
	}
	ring := make([]ringKey, 0, len(hexKeys))
	seen := make(map[string]struct{}, len(hexKeys))
	for _, key := range hexKeys {
		clean := strings.TrimSpace(key)
		pubBytes, err := hex.DecodeString(clean)
		if err != nil {
			return nil, fmt.Errorf("%w: key %q: %v", ErrInvalidThreshold, clean, err)
		}
		if len(pubBytes) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("%w: key %q has length %d, want %d", ErrInvalidThreshold, clean, len(pubBytes), ed25519.PublicKeySize)
		}
		id := strings.ToLower(clean)
		if _, dup := seen[id]; dup {
			return nil, fmt.Errorf("%w: duplicate authorized key %q", ErrInvalidThreshold, clean)
		}
		seen[id] = struct{}{}
		ring = append(ring, ringKey{hexID: id, pub: ed25519.PublicKey(pubBytes)})
	}
	sort.Slice(ring, func(i, j int) bool { return ring[i].hexID < ring[j].hexID })
	return &MOfNValidator{ring: ring, m: m, timeSource: realTimeSource{}}, nil
}

// RingSize reports N (the number of authorized keys).
func (v *MOfNValidator) RingSize() int { return len(v.ring) }

// Threshold reports M.
func (v *MOfNValidator) Threshold() int { return v.m }

// ValidateToken verifies cryptographic quorum, validity window, and hardware
// constraints. It returns nil only when at least M distinct ring keys verify
// one of the token's signatures (Signature plus any Signoffs) over the same
// payload digest.
func (v *MOfNValidator) ValidateToken(token *SignedLicenseToken, currentHardwareID string) error {
	if token == nil {
		return errors.New("license token cannot be nil")
	}
	payloadBytes, err := json.Marshal(token.Payload)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(payloadBytes)

	count, err := v.countDistinctSigners(digest[:], token.Signatures())
	if err != nil {
		return err
	}
	if count < v.m {
		return ErrQuorumNotMet
	}

	return checkLicenseWindow(token.Payload, currentHardwareID, v.timeSource.Now())
}

// countDistinctSigners returns how many distinct ring keys verified at least
// one of the supplied signatures over the digest.
func (v *MOfNValidator) countDistinctSigners(digest []byte, signatures [][]byte) (int, error) {
	verified := make(map[string]struct{})
	for _, sigBytes := range signatures {
		if len(sigBytes) != ed25519.SignatureSize {
			return 0, ErrInvalidSignature
		}
		for _, key := range v.ring {
			if _, ok := verified[key.hexID]; ok {
				continue
			}
			if ed25519.Verify(key.pub, digest, sigBytes) {
				verified[key.hexID] = struct{}{}
			}
		}
	}
	return len(verified), nil
}

// Signatures returns the token's primary signature plus any signoffs, in
// order, deduplicated by byte equality.
func (t *SignedLicenseToken) Signatures() [][]byte {
	out := make([][]byte, 0, 1+len(t.Signoffs))
	seen := make(map[string]struct{})
	for _, s := range append([]string{t.Signature}, t.Signoffs...) {
		if s == "" {
			continue
		}
		clean := strings.TrimSpace(s)
		if _, dup := seen[clean]; dup {
			continue
		}
		raw, err := hex.DecodeString(clean)
		if err != nil {
			// Keep the raw bytes; the verifier rejects bad lengths.
			out = append(out, []byte(clean))
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, raw)
	}
	return out
}

// ValidateWithRevocation runs the m-of-n quorum validation followed by the
// fail-closed revocation check.
func (v *MOfNValidator) ValidateWithRevocation(token *SignedLicenseToken, currentHardwareID string, cache *RevocationCache) error {
	if cache == nil {
		return ErrRevocationCacheExpired
	}
	if err := v.ValidateToken(token, currentHardwareID); err != nil {
		return err
	}
	return cache.CheckLicense(token.Payload.LicenseID, v.timeSource.Now())
}
