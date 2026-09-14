package licensing

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	// ErrLicenseRevoked reports a license that appears on a revocation list.
	ErrLicenseRevoked = errors.New("license has been revoked")
	// ErrRevocationCacheExpired reports that the revocation cache entry for a
	// license can no longer be trusted. Fail-closed: the license is treated as
	// revoked until the cache is refreshed, never as not-revoked.
	ErrRevocationCacheExpired = errors.New("revocation cache entry expired, fresh CRL/OCSP response required")
	// ErrInvalidRevocationEntry reports a malformed revocation list entry.
	ErrInvalidRevocationEntry = errors.New("revocation cache entry is invalid")
)

// RevocationEntry is one short-lived revocation record, mirroring a CRL
// revoked-certificate entry or an OCSP "revoked" single response.
type RevocationEntry struct {
	LicenseID string    `json:"license_id"`
	Reason    string    `json:"reason"`
	RevokedAt time.Time `json:"revoked_at"`
	// ExpiresAt bounds the freshness of this record. After it, the cache
	// fails closed (ErrRevocationCacheExpired) until reloaded.
	ExpiresAt time.Time `json:"expires_at"`
}

// RevokedLicenseError carries revocation details for audit and UI surfacing.
type RevokedLicenseError struct {
	LicenseID string
	Reason    string
	RevokedAt time.Time
}

func (e *RevokedLicenseError) Error() string {
	return fmt.Sprintf("license %s revoked on %s: %s", e.LicenseID, e.RevokedAt.Format(time.RFC3339), e.Reason)
}

func (e *RevokedLicenseError) Is(target error) bool { return target == ErrLicenseRevoked }

// RevocationCache is an in-memory, short-lived CRL/OCSP-style revocation
// cache. It holds revocation entries keyed by license ID and answers
// fail-closed: an expired cache entry is denied, never silently trusted.
type RevocationCache struct {
	mu      sync.RWMutex
	entries map[string]RevocationEntry
}

// NewRevocationCache returns an empty, ready-to-load revocation cache.
func NewRevocationCache() *RevocationCache {
	return &RevocationCache{entries: make(map[string]RevocationEntry)}
}

// Load replaces the cache contents with a fresh CRL/OCSP batch. Any invalid
// entry rejects the whole load so a truncated or corrupted list can never
// leave a partially trusted cache behind.
func (c *RevocationCache) Load(entries []RevocationEntry) error {
	validated := make(map[string]RevocationEntry, len(entries))
	for _, e := range entries {
		if err := validateRevocationEntry(e); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidRevocationEntry, err)
		}
		key := strings.TrimSpace(e.LicenseID)
		if _, dup := validated[key]; dup {
			return fmt.Errorf("%w: duplicate license_id %q", ErrInvalidRevocationEntry, key)
		}
		validated[key] = e
	}
	c.mu.Lock()
	c.entries = validated
	c.mu.Unlock()
	return nil
}

// Snapshot returns a copy of the current revocation entries.
func (c *RevocationCache) Snapshot() []RevocationEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]RevocationEntry, 0, len(c.entries))
	for _, e := range c.entries {
		out = append(out, e)
	}
	return out
}

// CheckLicense answers whether licenseID is revoked at time `at`. It is
// fail-closed: revoked licenses are denied, and license records whose
// freshness window has passed are denied with ErrRevocationCacheExpired.
func (c *RevocationCache) CheckLicense(licenseID string, at time.Time) error {
	c.mu.RLock()
	entry, ok := c.entries[strings.TrimSpace(licenseID)]
	c.mu.RUnlock()
	if !ok {
		return nil
	}
	if at.After(entry.ExpiresAt) {
		return ErrRevocationCacheExpired
	}
	return &RevokedLicenseError{
		LicenseID: entry.LicenseID,
		Reason:    entry.Reason,
		RevokedAt: entry.RevokedAt,
	}
}

func validateRevocationEntry(e RevocationEntry) error {
	if strings.TrimSpace(e.LicenseID) == "" {
		return errors.New("license_id is required")
	}
	if e.RevokedAt.IsZero() {
		return errors.New("revoked_at is required")
	}
	if e.ExpiresAt.IsZero() {
		return errors.New("expires_at is required")
	}
	if !e.ExpiresAt.After(e.RevokedAt) {
		return errors.New("expires_at must be after revoked_at")
	}
	return nil
}

// ValidateWithRevocation runs the standard single-key validation and then the
// fail-closed revocation check. A nil cache is itself a fail-closed failure:
// a validator without a revocation feed must not silently trust any token.
func (v *UniversalLicenseValidator) ValidateWithRevocation(token *SignedLicenseToken, currentHardwareID string, cache *RevocationCache) error {
	if cache == nil {
		return ErrRevocationCacheExpired
	}
	if err := v.ValidateToken(token, currentHardwareID); err != nil {
		return err
	}
	return cache.CheckLicense(token.Payload.LicenseID, v.timeSource.Now())
}
