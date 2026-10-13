package sync

import "errors"

// DeviceProofFailureReason is the closed source-owned classification used by
// the unmounted manifest boundary. It contains no raw proof, identity, key, or
// authority values.
type DeviceProofFailureReason string

const (
	DeviceProofFailureExpired           DeviceProofFailureReason = "expired"
	DeviceProofFailureAuthorityMismatch DeviceProofFailureReason = "authority_mismatch"
	DeviceProofFailureKeyUnknown        DeviceProofFailureReason = "key_unknown"
	DeviceProofFailureSignatureInvalid  DeviceProofFailureReason = "signature_invalid"
)

// DeviceProofFailure preserves a closed reason for source-owned observation
// while deliberately keeping its public Error string non-diagnostic.
type DeviceProofFailure struct {
	Reason DeviceProofFailureReason
	cause  error
}

func (e *DeviceProofFailure) Error() string { return "device proof rejected" }
func (e *DeviceProofFailure) Unwrap() error { return e.cause }

// NewDeviceProofFailure is provided for deterministic boundary tests and
// adapters. Production verification creates these values internally.
func NewDeviceProofFailure(reason DeviceProofFailureReason) error {
	return &DeviceProofFailure{Reason: reason}
}

// ProofFailureReasonOf exposes only a known closed reason. Unknown failures
// remain diagnostic and cannot become public receipt cases.
func ProofFailureReasonOf(err error) (DeviceProofFailureReason, bool) {
	var failure *DeviceProofFailure
	if !errors.As(err, &failure) {
		return "", false
	}
	switch failure.Reason {
	case DeviceProofFailureExpired, DeviceProofFailureAuthorityMismatch, DeviceProofFailureKeyUnknown, DeviceProofFailureSignatureInvalid:
		return failure.Reason, true
	default:
		return "", false
	}
}

func proofFailure(reason DeviceProofFailureReason, cause error) error {
	return &DeviceProofFailure{Reason: reason, cause: cause}
}
