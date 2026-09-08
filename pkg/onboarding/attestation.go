package onboarding

import "errors"

// Sentinel validation errors for attestation claims.
var (
	ErrUnknownKeyOrigin        = errors.New("unknown attestation key origin")
	ErrNoneWithEvidence        = errors.New("NONE origin must not carry an attestation blob or biometric binding")
	ErrHardwareMissingBlob     = errors.New("hardware attestation origin requires a non-empty attestation blob")
	ErrHardwareMissingOS       = errors.New("hardware attestation origin requires an OS version")
	ErrSoftwareOriginRejected  = errors.New("policy requires hardware attestation")
	ErrBiometricBindingMissing = errors.New("policy requires biometric binding")
)

func (k KeyOrigin) Known() bool {
	switch k {
	case KeyOriginSecureEnclave, KeyOriginStrongBox, KeyOriginSoftware, KeyOriginNone:
		return true
	default:
		return false
	}
}

func (k KeyOrigin) Hardware() bool {
	return k == KeyOriginSecureEnclave || k == KeyOriginStrongBox
}

// Validate checks the structural soundness of a claim. The blob is treated as
// opaque evidence; only presence/origin/binding semantics are enforced.
func (c AttestationClaim) Validate() error {
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
// The zero value is permissive: it accepts any structurally valid claim
// (including SOFTWARE/NONE), preserving pre-attestation enrollment behavior.
type AttestationPolicy struct {
	RequireHardware         bool
	RequireBiometricBinding bool
}

// VerifyClaim validates a claim and enforces the policy. Unknown future
// origins are rejected (fail closed) regardless of policy.
func VerifyClaim(claim AttestationClaim, policy AttestationPolicy) error {
	if err := claim.Validate(); err != nil {
		return err
	}
	if policy.RequireHardware && !claim.KeyOrigin.Hardware() {
		return ErrSoftwareOriginRejected
	}
	if policy.RequireBiometricBinding && !claim.BiometricBound {
		return ErrBiometricBindingMissing
	}
	return nil
}
