package onboarding

import (
	"errors"
	"fmt"
)

// Sentinel validation errors for attestation claims.
var (
	ErrUnknownKeyOrigin        = errors.New("attestation: unrecognized key origin")
	ErrNoneWithEvidence        = errors.New("attestation: NONE origin cannot carry hardware evidence")
	ErrHardwareMissingBlob     = errors.New("attestation: hardware origin missing attestation certificate chain")
	ErrHardwareMissingOS       = errors.New("attestation: hardware origin missing OS version")
	ErrSoftwareOriginRejected  = errors.New("attestation: policy requires hardware attestation (enclave, strongbox, or TEE)")
	ErrBiometricBindingMissing = errors.New("attestation: biometric binding required but not asserted")
	ErrStrongBoxMandated       = errors.New("attestation: site policy strictly mandates discrete StrongBox hardware")
)

func (k KeyOrigin) Known() bool {
	switch k {
	case KeyOriginSecureEnclave, KeyOriginStrongBox, KeyOriginTEE, KeyOriginSoftware, KeyOriginNone:
		return true
	default:
		return false
	}
}

func (k KeyOrigin) Hardware() bool {
	return k == KeyOriginSecureEnclave || k == KeyOriginStrongBox || k == KeyOriginTEE
}

// Validate checks the structural soundness of a claim. The blob is treated as
// opaque evidence; only presence/origin/binding semantics are enforced.
func (c AttestationClaim) Validate() error {
	// New enrollment submissions must explicitly declare a known KeyOrigin.
	// Empty string ("") is intentionally rejected here; legacy database record
	// grandfathering to KeyOriginNone is handled downstream exclusively in
	// CheckPosture(). Callers holding legacy zero-value claims must skip
	// Validate() entirely (see ProcessDeviceEnrollment's KeyOrigin != "" guard).
	if !c.KeyOrigin.Known() {
		return ErrUnknownKeyOrigin
	}
	if c.KeyOrigin == KeyOriginNone {
		if c.AttestationBlob != "" || c.BiometricBound {
			return ErrNoneWithEvidence
		}
		return nil
	}
	if c.KeyOrigin.Hardware() {
		if c.AttestationBlob == "" {
			return ErrHardwareMissingBlob
		}
		if c.OSVersion == "" {
			return ErrHardwareMissingOS
		}
	}
	return nil
}

// AttestationPolicy expresses site trust requirements for enrolled devices.
// The zero value is permissive across all KNOWN origins (including SOFTWARE
// and NONE), preserving pre-attestation enrollment behavior. It remains
// strictly fail-closed against unrecognized origin strings.
//
// Zero-Value Behavior:
// An unconfigured (zero-value) policy accepts any structurally valid claim
// but rejects unparseable or unrecognized origins.
type AttestationPolicy struct {
	RequireHardware         bool `json:"require_hardware"`
	RequireStrongBox        bool `json:"require_strongbox"` // Strict floor for life-safety components
	RequireBiometricBinding bool `json:"require_biometric_binding"`
}

// VerifyClaim validates a claim and enforces the policy. Unknown future
// origins are rejected (fail closed) regardless of policy.
func VerifyClaim(claim AttestationClaim, policy AttestationPolicy) error {
	if err := claim.Validate(); err != nil {
		return err
	}
	return CheckPosture(string(claim.KeyOrigin), claim.BiometricBound, policy)
}

// CheckPosture enforces a policy against a recorded device posture. It
// accepts string origins to verify persisted database receipts (e.g.,
// DeviceTrustRecord).
//
// Backward Compatibility:
// Empty string origins ("") are mapped to KeyOriginNone to preserve legacy
// pre-attestation records under zero-value policies while remaining
// fail-closed against unknown junk. The structural claim checks (blob/OS
// presence) live in AttestationClaim.Validate and are not repeated here, so
// this helper is also safe for enrollment-derived records whose evidence was
// already validated.
func CheckPosture(origin string, biometricBound bool, policy AttestationPolicy) error {
	keyOrigin := KeyOrigin(origin)
	if origin == "" {
		keyOrigin = KeyOriginNone
	}
	if !keyOrigin.Known() {
		return fmt.Errorf("%w: %q", ErrUnknownKeyOrigin, origin)
	}
	if policy.RequireStrongBox && keyOrigin != KeyOriginStrongBox {
		return fmt.Errorf("%w: recorded origin is %s", ErrStrongBoxMandated, origin)
	}
	if policy.RequireHardware && !keyOrigin.Hardware() {
		return fmt.Errorf("%w: recorded origin is %s", ErrSoftwareOriginRejected, origin)
	}
	if policy.RequireBiometricBinding && !biometricBound {
		return ErrBiometricBindingMissing
	}
	return nil
}
