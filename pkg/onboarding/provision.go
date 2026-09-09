package onboarding

import (
	"fmt"
	"os"
	"strings"
)

// Attestation roots and the expected App ID are operator-provisioned at boot
// via environment (documented in INTEGIN_LOCAL_ENV.example). Nothing
// attestation-critical is embedded from memory — a misconfigured root pipeline
// must fail at boot, not silently at the first enrollment.

const (
	envGoogleRootsFile = "INTEGIN_ATTEST_GOOGLE_ROOTS_FILE"
	envAppleRootsFile  = "INTEGIN_ATTEST_APPLE_ROOTS_FILE"
	envAppleAppID      = "INTEGIN_APPLE_APP_ID"
)

// ProvisionAttestationFromEnv is the boot-time loader for the offline
// attestation roots. It reads:
//
//	INTEGIN_ATTEST_GOOGLE_ROOTS_FILE  PEM file of Google's Android Hardware
//	    Attestation roots -> SetAttestationRoots.
//	INTEGIN_ATTEST_APPLE_ROOTS_FILE   PEM file of Apple's App Attest Root CA
//	    -> SetAppleAttestRoots.
//	INTEGIN_APPLE_APP_ID              expected "<teamID>.<bundleID>"
//	    -> SetEnrollmentExpectedAppID.
//
// A set-but-invalid variable fails closed with an error naming the variable,
// so a missing or unparseable PEM aborts boot instead of disabling
// verification. The Apple roots and App ID are all-or-nothing: pinning only
// one would arm a verifier that can never succeed, or silently skip Apple
// verification. When all three are unset the package keeps its existing
// unconfigured posture — chain/CBOR claims then fail closed at enrollment
// with the roots-unconfigured sentinels.
func ProvisionAttestationFromEnv() error {
	if file := strings.TrimSpace(os.Getenv(envGoogleRootsFile)); file != "" {
		roots, err := LoadAttestationRoots(file)
		if err != nil {
			return fmt.Errorf("%s: %w", envGoogleRootsFile, err)
		}
		SetAttestationRoots(roots)
	}

	appleRootsFile := strings.TrimSpace(os.Getenv(envAppleRootsFile))
	appleAppID := strings.TrimSpace(os.Getenv(envAppleAppID))
	if (appleRootsFile == "") != (appleAppID == "") {
		return fmt.Errorf("%s and %s must be set together (Apple App Attest provisioning is all-or-nothing)", envAppleRootsFile, envAppleAppID)
	}
	if appleRootsFile == "" {
		return nil
	}
	roots, err := LoadAppleAttestRoots(appleRootsFile)
	if err != nil {
		return fmt.Errorf("%s: %w", envAppleRootsFile, err)
	}
	SetAppleAttestRoots(roots)
	SetEnrollmentExpectedAppID(appleAppID)
	return nil
}
