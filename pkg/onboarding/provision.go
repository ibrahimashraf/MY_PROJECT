package onboarding

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
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
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("%s: %w", envGoogleRootsFile, err)
		}
		roots, _, err := VerifyRootBundle("google", data, time.Now())
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
	data, err := os.ReadFile(appleRootsFile)
	if err != nil {
		return fmt.Errorf("%s: %w", envAppleRootsFile, err)
	}
	roots, _, err := VerifyRootBundle("apple", data, time.Now())
	if err != nil {
		return fmt.Errorf("%s: %w", envAppleRootsFile, err)
	}
	if err := ValidateProvisionedAppleAttestation(roots, appleAppID, time.Now()); err != nil {
		return err
	}
	SetAppleAttestRoots(roots)
	SetEnrollmentExpectedAppID(appleAppID)
	return nil
}

// RootSubject is the operator-facing identity summary for one provisioned
// attestation root certificate.
type RootSubject struct {
	CommonName string
	NotBefore  time.Time
	NotAfter   time.Time
	IsCA       bool
}

// LoadAttestationRootPEM parses a PEM-encoded root bundle into an x509.CertPool
// (the in-memory sibling of LoadAttestationRoots, which reads a file). A bundle
// that yields no certificates fails closed.
func LoadAttestationRootPEM(pemData []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemData) {
		return nil, errors.New("no certificates found in attestation roots PEM")
	}
	return pool, nil
}

// VerifyRootBundle validates a PEM root bundle at boot: every PEM block must
// parse to a currently-valid root CA, and at least one certificate must be
// present. now anchors the validity window so boot checks and tests are
// deterministic. It returns the resulting pool plus one root-subject summary
// per certificate.
func VerifyRootBundle(name string, pemData []byte, now time.Time) (*x509.CertPool, []RootSubject, error) {
	pool := x509.NewCertPool()
	var subjects []RootSubject
	rest := pemData
	for len(rest) > 0 {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("%s attestation roots contain an unparseable certificate: %w", name, err)
		}
		if !cert.IsCA {
			return nil, nil, fmt.Errorf("%s attestation root %q is not a CA", name, cert.Subject.CommonName)
		}
		if now.After(cert.NotAfter) {
			return nil, nil, fmt.Errorf("%s attestation root %q expired at %s", name, cert.Subject.CommonName, cert.NotAfter.Format(time.RFC3339))
		}
		pool.AddCert(cert)
		subjects = append(subjects, RootSubject{
			CommonName: cert.Subject.CommonName,
			NotBefore:  cert.NotBefore,
			NotAfter:   cert.NotAfter,
			IsCA:       cert.IsCA,
		})
	}
	if len(subjects) == 0 {
		return nil, nil, fmt.Errorf("%s: no certificates found in attestation roots PEM", name)
	}
	return pool, subjects, nil
}

// ValidateProvisionedRoots checks a loaded root pool: it must not be nil and
// must contain at least one certificate (x509.CertPool exposes no certificate
// enumeration, so per-certificate CA/expiry checks live in VerifyRootBundle).
// name labels the pool in errors (e.g. "google", "apple").
func ValidateProvisionedRoots(pool *x509.CertPool, name string, now time.Time) error {
	if pool == nil {
		return fmt.Errorf("%s attestation roots are not configured", name)
	}
	if len(pool.Subjects()) == 0 {
		return fmt.Errorf("%s attestation roots pool contains no certificates", name)
	}
	return nil
}

// ValidateProvisionedAppleAttestation enforces the Apple all-or-nothing pair at
// boot: roots configured require an App ID and an App ID requires roots. When
// configured, the roots must form a non-empty pool and the App ID must be a
// non-empty "<teamID>.<bundleID>" (RFC 8485 style) value.
func ValidateProvisionedAppleAttestation(roots *x509.CertPool, appID string, now time.Time) error {
	if (roots == nil) != (strings.TrimSpace(appID) == "") {
		return fmt.Errorf("%s and %s must be set together (Apple App Attest provisioning is all-or-nothing)", envAppleRootsFile, envAppleAppID)
	}
	if roots == nil {
		return nil
	}
	if err := ValidateProvisionedRoots(roots, "apple", now); err != nil {
		return err
	}
	if !strings.Contains(appID, ".") {
		return fmt.Errorf("%s: expected \"<teamID>.<bundleID>\" App ID, got %q", envAppleAppID, appID)
	}
	return nil
}
