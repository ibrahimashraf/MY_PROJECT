package onboarding

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testAppleAppID = "TEAM1234.com.integin.fieldapp"

func TestProvisionAttestationFromEnvInjectsRoots(t *testing.T) {
	defer resetAttestationGlobals()
	dir := t.TempDir()
	t.Setenv(envGoogleRootsFile, writeRootPEM(t, dir, "google"))
	t.Setenv(envAppleRootsFile, writeRootPEM(t, dir, "apple"))
	t.Setenv(envAppleAppID, testAppleAppID)

	if err := ProvisionAttestationFromEnv(); err != nil {
		t.Fatalf("provision: %v", err)
	}
	if chainVerifyRoots == nil {
		t.Fatal("Google roots not injected")
	}
	if appleAttestRoots == nil {
		t.Fatal("Apple roots not injected")
	}
	if enrollmentExpectedAppID != testAppleAppID {
		t.Fatalf("expectedAppID = %q, want %q", enrollmentExpectedAppID, testAppleAppID)
	}
}

func TestProvisionAttestationFromEnvMissingFileFailsClosed(t *testing.T) {
	defer resetAttestationGlobals()
	t.Setenv(envGoogleRootsFile, filepath.Join(t.TempDir(), "no-such-file.pem"))
	err := ProvisionAttestationFromEnv()
	if err == nil {
		t.Fatal("missing Google roots file accepted")
	}
	if !strings.Contains(err.Error(), envGoogleRootsFile) {
		t.Fatalf("error does not name the variable: %v", err)
	}
}

func TestProvisionAttestationFromEnvInvalidPEMFailsClosed(t *testing.T) {
	defer resetAttestationGlobals()
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(bad, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envAppleRootsFile, bad)
	t.Setenv(envAppleAppID, testAppleAppID)
	err := ProvisionAttestationFromEnv()
	if err == nil {
		t.Fatal("invalid Apple roots file accepted")
	}
	if !strings.Contains(err.Error(), envAppleRootsFile) {
		t.Fatalf("error does not name the variable: %v", err)
	}
	if appleAttestRoots != nil {
		t.Fatal("invalid roots partially injected")
	}
}

func TestProvisionAttestationFromEnvApplePairRequired(t *testing.T) {
	defer resetAttestationGlobals()
	t.Setenv(envAppleAppID, testAppleAppID)
	if err := ProvisionAttestationFromEnv(); err == nil {
		t.Fatal("appID without roots file accepted")
	} else if !strings.Contains(err.Error(), envAppleRootsFile) {
		t.Fatalf("error does not name the pair: %v", err)
	}

	resetAttestationGlobals()
	t.Setenv(envAppleRootsFile, writeRootPEM(t, t.TempDir(), "apple"))
	t.Setenv(envAppleAppID, "")
	t.Setenv(envGoogleRootsFile, "")
	if err := ProvisionAttestationFromEnv(); err == nil {
		t.Fatal("roots file without appID accepted")
	} else if !strings.Contains(err.Error(), envAppleAppID) {
		t.Fatalf("error does not name the pair: %v", err)
	}
}

func TestProvisionAttestationFromEnvAllUnsetIsNoop(t *testing.T) {
	defer resetAttestationGlobals()
	t.Setenv(envGoogleRootsFile, "")
	t.Setenv(envAppleRootsFile, "")
	t.Setenv(envAppleAppID, "")
	if err := ProvisionAttestationFromEnv(); err != nil {
		t.Fatalf("unset provisioning must not fail: %v", err)
	}
	if chainVerifyRoots != nil || appleAttestRoots != nil || enrollmentExpectedAppID != "" {
		t.Fatal("unset provisioning changed the package posture")
	}
}

func resetAttestationGlobals() {
	chainVerifyRoots = nil
	appleAttestRoots = nil
	enrollmentExpectedAppID = ""
}

func TestVerifyRootBundleValidPEMs(t *testing.T) {
	for _, name := range []string{"google", "apple"} {
		pemPath := writeRootPEM(t, t.TempDir(), name)
		data, err := os.ReadFile(pemPath)
		if err != nil {
			t.Fatal(err)
		}
		pool, subjects, err := VerifyRootBundle(name, data, time.Now())
		if err != nil {
			t.Fatalf("VerifyRootBundle(%s): %v", name, err)
		}
		if pool == nil {
			t.Fatalf("VerifyRootBundle(%s) returned nil pool", name)
		}
		if len(subjects) != 1 {
			t.Fatalf("VerifyRootBundle(%s) subjects = %d, want 1", name, len(subjects))
		}
		if subjects[0].CommonName != name+" root" || !subjects[0].IsCA {
			t.Fatalf("VerifyRootBundle(%s) subject = %+v", name, subjects[0])
		}
	}
}

func TestVerifyRootBundleRejectsMalformed(t *testing.T) {
	_, _, err := VerifyRootBundle("google", []byte("not a certificate"), time.Now())
	if err == nil {
		t.Fatal("malformed root PEM accepted")
	}
	if !strings.Contains(err.Error(), "google") {
		t.Fatalf("error does not name the bundle: %v", err)
	}
}

func TestValidateProvisionedRootsRejectsExpired(t *testing.T) {
	pemPath := writeExpiredRootPEM(t, t.TempDir(), "expired-ca")
	data, err := os.ReadFile(pemPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := VerifyRootBundle("apple", data, time.Now()); err == nil {
		t.Fatal("expired root accepted")
	} else if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("error does not mention expiry: %v", err)
	}

	empty := x509.NewCertPool()
	if err := ValidateProvisionedRoots(empty, "google", time.Now()); err == nil {
		t.Fatal("empty pool accepted")
	}
	if err := ValidateProvisionedRoots(nil, "google", time.Now()); err == nil {
		t.Fatal("nil pool accepted")
	}
}

func TestValidateProvisionedAppleAttestationAllOrNothing(t *testing.T) {
	roots, err := LoadAttestationRootPEM(validRootPEM(t, "apple"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateProvisionedAppleAttestation(nil, testAppleAppID, time.Now()); err == nil {
		t.Fatal("appID without roots accepted")
	}
	if err := ValidateProvisionedAppleAttestation(roots, "", time.Now()); err == nil {
		t.Fatal("roots without appID accepted")
	}
	if err := ValidateProvisionedAppleAttestation(nil, "", time.Now()); err != nil {
		t.Fatalf("all-unset must not fail: %v", err)
	}
	if err := ValidateProvisionedAppleAttestation(roots, testAppleAppID, time.Now()); err != nil {
		t.Fatalf("valid pair rejected: %v", err)
	}
	if err := ValidateProvisionedAppleAttestation(roots, "NOBUNDLE", time.Now()); err == nil {
		t.Fatal("appID without a '.' separator accepted")
	}
}

// validRootPEM returns the PEM bytes of a fresh self-signed CA.
func validRootPEM(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(writeRootPEM(t, t.TempDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// writeExpiredRootPEM creates a self-signed CA already past its NotAfter.
func writeExpiredRootPEM(t *testing.T, dir, name string) string {
	t.Helper()
	return writeRootPEMWithValidity(t, dir, name, time.Now().Add(-2*time.Hour), time.Now().Add(-time.Hour))
}

// writeRootPEM creates a fresh self-signed CA and returns its PEM file path.
func writeRootPEM(t *testing.T, dir, name string) string {
	t.Helper()
	return writeRootPEMWithValidity(t, dir, name, time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
}

func writeRootPEMWithValidity(t *testing.T, dir, name string, notBefore, notAfter time.Time) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name + " root"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name+".pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
