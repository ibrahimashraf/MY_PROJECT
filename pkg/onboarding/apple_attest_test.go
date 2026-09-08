package onboarding

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"
	"time"
)

// --- Hermetic CBOR builders (test-only minimal definite-length encoder) ---

// cborHead encodes a definite-length CBOR head for the given major type.
func cborHead(major byte, n int) []byte {
	h := []byte{major << 5}
	if n < 24 {
		h[0] |= byte(n)
	} else if n < 256 {
		h[0] |= 24
		h = append(h, byte(n))
	} else if n < 65536 {
		h[0] |= 25
		h = append(h, byte(n>>8), byte(n))
	} else {
		h[0] |= 26
		h = append(h, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
	return h
}

func cborBytes(b []byte) []byte {
	return append(cborHead(2, len(b)), b...)
}
func cborText(s string) []byte {
	return append(cborHead(3, len(s)), s...)
}
func cborArray(items ...[]byte) []byte {
	out := cborHead(4, len(items))
	for _, it := range items {
		out = append(out, it...)
	}
	return out
}
func cborMap(pairs ...[]byte) []byte {
	out := cborHead(5, len(pairs)/2)
	for _, it := range pairs {
		out = append(out, it...)
	}
	return out
}

// appleAuthData builds a minimal authenticator data block whose rpIdHash is
// SHA256(rpID). The attested-credential fields are dummy bytes: the verifier
// only consumes rpIdHash and covers the rest via the nonce.
func appleAuthData(rpID string) []byte {
	rpHash := sha256.Sum256([]byte(rpID))
	out := rpHash[:]
	out = append(out, 0x41)       // flags: UP | AT
	out = append(out, 0, 0, 0, 0) // signCount
	out = append(out, make([]byte, 16)...)
	out = append(out, 0, 3)
	out = append(out, 0xaa, 0xbb, 0xcc)
	out = append(out, 0xa5) // partial COSE public key (unparsed by verifier)
	return out
}

// buildAppleAttObj encodes an Apple App Attest attestation object in CBOR.
func buildAppleAttObj(format string, authData []byte, x5c [][]byte) []byte {
	certs := make([][]byte, 0, len(x5c))
	for _, c := range x5c {
		certs = append(certs, cborBytes(c))
	}
	return cborMap(
		cborText("fmt"), cborText(format),
		cborText("authData"), cborBytes(authData),
		cborText("attStmt"), cborMap(
			cborText("x5c"), cborArray(certs...),
			cborText("receipt"), cborBytes([]byte("opaque-receipt")),
		),
	)
}

// newAppleAttestLeaf mints an ECDSA CA + leaf signed by it with the given
// nonce embedded in the App Attest 8.2 extension. The extension value is
// stored as a DER OCTET STRING wrapping the nonce (how Apple emits the
// field), exercising the unwrap path in leafNonce. Returns the DER bytes and
// the parsed CA for use as the injected root.
func newAppleAttestLeaf(t *testing.T, nonce []byte) (leafDER, caDER []byte, ca *x509.Certificate) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gen ca key: %v", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(11),
		Subject:               pkix.Name{CommonName: "Test App Attest Root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	caDER, err = x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create ca: %v", err)
	}
	ca, err = x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse ca: %v", err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gen leaf key: %v", err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(22),
		Subject:      pkix.Name{CommonName: "Test Attest Device"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		ExtraExtensions: []pkix.Extension{
			{Id: appAttestNonceOID, Critical: false, Value: derTLV(0x04, nonce)},
		},
	}
	leafDER, err = x509.CreateCertificate(rand.Reader, leafTmpl, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}
	return leafDER, caDER, ca
}

// TestVerifyAppleAttestation (a)-(g): hermetic offline App Attest verification.
func TestVerifyAppleAttestation(t *testing.T) {
	challenge := []byte("enrollment-nonce-abc")
	appID := "TEAM1234.com.integin.fieldapp"

	// (a) valid object with a nonce bound to the challenge and an rpIdHash
	// matching the appID verifies with every flag set.
	authData := appleAuthData(appID)
	nonce := appleNonce(authData, challenge)
	leafDER, caDER, ca := newAppleAttestLeaf(t, nonce)
	attObj := buildAppleAttObj("apple-appattest", authData, [][]byte{leafDER, caDER})
	info, err := VerifyAppleAttestation(attObj, caPool(ca), challenge, appID)
	if err != nil {
		t.Fatalf("(a) valid object rejected: %v", err)
	}
	if !info.VerifiedBoot || !info.ChallengeMatched || !info.AppIDMatched {
		t.Fatalf("(a) flags not all set: %+v", info)
	}

	// (b) wrong challenge (different clientData) rejected.
	wrongAuth := appleAuthData(appID)
	wrongNonce := appleNonce(wrongAuth, []byte("different-clientdata"))
	leafDER, caDER, ca = newAppleAttestLeaf(t, wrongNonce)
	attObj = buildAppleAttObj("apple-appattest", wrongAuth, [][]byte{leafDER, caDER})
	if _, err := VerifyAppleAttestation(attObj, caPool(ca), challenge, appID); !errors.Is(err, ErrAppleAttestChallengeMismatch) {
		t.Fatalf("(b) wrong challenge err=%v", err)
	}

	// (c) authData bound to a different appID rejected.
	foreignAuth := appleAuthData("TEAM0000.com.other.app")
	foreignNonce := appleNonce(foreignAuth, challenge)
	leafDER, caDER, ca = newAppleAttestLeaf(t, foreignNonce)
	attObj = buildAppleAttObj("apple-appattest", foreignAuth, [][]byte{leafDER, caDER})
	if _, err := VerifyAppleAttestation(attObj, caPool(ca), challenge, appID); !errors.Is(err, ErrAppleAttestAppIDMismatch) {
		t.Fatalf("(c) wrong appID err=%v", err)
	}

	// (d) non apple-appattest format rejected.
	fmtAuth := appleAuthData(appID)
	fmtNonce := appleNonce(fmtAuth, challenge)
	leafDER, caDER, ca = newAppleAttestLeaf(t, fmtNonce)
	attObj = buildAppleAttObj("fido-u2f", fmtAuth, [][]byte{leafDER, caDER})
	if _, err := VerifyAppleAttestation(attObj, caPool(ca), challenge, appID); !errors.Is(err, ErrAppleAttestInvalid) {
		t.Fatalf("(d) wrong format err=%v", err)
	}

	// (e) indefinite-length CBOR rejected fail-closed.
	if _, err := VerifyAppleAttestation([]byte{0x5f, 0x41, 0x41, 0xff}, caPool(ca), challenge, appID); !errors.Is(err, ErrAppleAttestInvalid) {
		t.Fatalf("(e) indefinite CBOR err=%v", err)
	}

	// (f) chain not rooted in the injected pool rejected.
	rootAuth := appleAuthData(appID)
	sameNonce := appleNonce(rootAuth, challenge)
	leafDER, _, _ = newAppleAttestLeaf(t, sameNonce)
	_, _, otherCA := newAppleAttestLeaf(t, sameNonce)
	attObj = buildAppleAttObj("apple-appattest", rootAuth, [][]byte{leafDER})
	if _, err := VerifyAppleAttestation(attObj, caPool(otherCA), challenge, appID); !errors.Is(err, ErrAppleAttestInvalid) {
		t.Fatalf("(f) unrooted chain err=%v", err)
	}

	// (g) tampered authData breaks the nonce binding even though the chain
	// still verifies.
	originalAuth := appleAuthData(appID)
	tamperedAuth := appleAuthData(appID)
	tamperedAuth[0] ^= 0xff
	tamperNonce := appleNonce(originalAuth, challenge)
	leafDER, caDER, ca = newAppleAttestLeaf(t, tamperNonce)
	attObj = buildAppleAttObj("apple-appattest", tamperedAuth, [][]byte{leafDER, caDER})
	if _, err := VerifyAppleAttestation(attObj, caPool(ca), challenge, appID); !errors.Is(err, ErrAppleAttestChallengeMismatch) {
		t.Fatalf("(g) tampered authData err=%v", err)
	}

	// nil roots pool fails closed before any parsing.
	if _, err := VerifyAppleAttestation(attObj, nil, challenge, appID); !errors.Is(err, ErrAppleAttestRootsUnconfigured) {
		t.Fatalf("nil roots err=%v", err)
	}
}

// appleSecureEnclaveClaim is the claim shape used by the Apple enrollment
// path (secure enclave origin requires blob + OS to pass VerifyClaim).
func appleSecureEnclaveClaim(cbor []byte) AttestationClaim {
	return AttestationClaim{
		KeyOrigin:       KeyOriginSecureEnclave,
		BiometricBound:  true,
		OSVersion:       "iOS 18.0",
		AttestationBlob: "deadbeef",
		AppleAttestCBOR: cbor,
	}
}

// TestEnrollmentVerifiedAppleAttestation drives a full enrollment whose
// SECURE_ENCLAVE claim carries a valid attestation object built against the
// enrollment nonce; the record is stored as AttestationVerified.
func TestEnrollmentVerifiedAppleAttestation(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim: %v", err)
	}
	appID := "TEAM1234.com.integin.fieldapp"

	// Build the attestation object AFTER the challenge so the nonce binds the
	// actual enrollment nonce.
	chal, err := sim.CreateEnrollmentChallenge("ten_test", "insp_test")
	if err != nil {
		t.Fatalf("challenge: %v", err)
	}
	authData := appleAuthData(appID)
	leafDER, caDER, ca := newAppleAttestLeaf(t, appleNonce(authData, []byte(chal.Nonce)))
	attObj := buildAppleAttObj("apple-appattest", authData, [][]byte{leafDER, caDER})
	appleAttestRoots = caPool(ca)
	enrollmentExpectedAppID = appID
	defer func() { appleAttestRoots = nil; enrollmentExpectedAppID = "" }()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	record, err := sim.ProcessDeviceEnrollment(DeviceEnrollmentSubmission{
		ChallengeID:     chal.ChallengeID,
		InspectorID:     "insp_test",
		DevicePublicKey: hex.EncodeToString(pub),
		DeviceModel:     "Test iPhone",
		SignedNonce:     hex.EncodeToString(ed25519.Sign(priv, []byte(chal.Nonce))),
		Attestation:     appleSecureEnclaveClaim(attObj),
	})
	if err != nil {
		t.Fatalf("verified Apple enrollment failed: %v", err)
	}
	if !record.AttestationVerified {
		t.Fatalf("valid Apple attestation did not record AttestationVerified")
	}
	if record.AttestationOrigin != string(KeyOriginSecureEnclave) {
		t.Fatalf("origin not recorded: %q", record.AttestationOrigin)
	}
}

// (h) enrollment with garbage AppleAttestCBOR fails closed and stores nothing.
func TestAppleEnrollmentGarbageCBORFailsClosed(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim: %v", err)
	}
	_, _, ca := newAppleAttestLeaf(t, make([]byte, sha256.Size))
	appleAttestRoots = caPool(ca)
	enrollmentExpectedAppID = "TEAM1234.com.integin.fieldapp"
	defer func() { appleAttestRoots = nil; enrollmentExpectedAppID = "" }()

	_, _, err = enroll(t, sim, appleSecureEnclaveClaim([]byte("not-cbor")))
	if err == nil {
		t.Fatalf("garbage CBOR accepted")
	}
	if !errors.Is(err, ErrAppleAttestInvalid) {
		t.Fatalf("garbage CBOR err=%v", err)
	}
	if len(sim.deviceStore) != 0 {
		t.Fatalf("garbage CBOR stored %d devices", len(sim.deviceStore))
	}
}

// (i) enrollment with AppleAttestCBOR but nil roots returns
// ErrAppleAttestRootsUnconfigured and stores nothing.
func TestAppleEnrollmentNilRootsFailsClosed(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim: %v", err)
	}
	appID := "TEAM1234.com.integin.fieldapp"
	authData := appleAuthData(appID)
	leafDER, caDER, _ := newAppleAttestLeaf(t, appleNonce(authData, []byte("x")))
	attObj := buildAppleAttObj("apple-appattest", authData, [][]byte{leafDER, caDER})
	appleAttestRoots = nil
	enrollmentExpectedAppID = appID
	defer func() { appleAttestRoots = nil; enrollmentExpectedAppID = "" }()

	_, _, err = enroll(t, sim, appleSecureEnclaveClaim(attObj))
	if err == nil {
		t.Fatalf("nil-roots Apple attestation accepted")
	}
	if !errors.Is(err, ErrAppleAttestRootsUnconfigured) {
		t.Fatalf("nil-roots err=%v", err)
	}
	if len(sim.deviceStore) != 0 {
		t.Fatalf("nil-roots stored %d devices", len(sim.deviceStore))
	}
}

// (j) SECURE_ENCLAVE without CBOR enrolls CLAIMED (Verified=false), and so
// does a CBOR-carrying claim while ExpectedAppID is unset (path disabled).
func TestAppleSecureEnclaveWithoutCBOREnrollsClaimed(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim: %v", err)
	}
	enrollmentExpectedAppID = "TEAM1234.com.integin.fieldapp"
	defer func() { enrollmentExpectedAppID = "" }()

	// No CBOR: even with the path armed, there is nothing to verify.
	record, _, err := enroll(t, sim, appleSecureEnclaveClaim(nil))
	if err != nil {
		t.Fatalf("no-CBOR Apple claim rejected: %v", err)
	}
	if record.AttestationVerified {
		t.Fatalf("no-CBOR Apple claim recorded AttestationVerified")
	}
	if record.AttestationOrigin != string(KeyOriginSecureEnclave) {
		t.Fatalf("origin not recorded: %q", record.AttestationOrigin)
	}

	// CBOR present but ExpectedAppID unset: the verification path is
	// disabled, so the claim enrolls CLAIMED rather than failing.
	sim2, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim2: %v", err)
	}
	enrollmentExpectedAppID = ""
	appID := "TEAM1234.com.integin.fieldapp"
	authData := appleAuthData(appID)
	leafDER, caDER, ca := newAppleAttestLeaf(t, appleNonce(authData, []byte("x")))
	attObj := buildAppleAttObj("apple-appattest", authData, [][]byte{leafDER, caDER})
	appleAttestRoots = caPool(ca)
	defer func() { appleAttestRoots = nil }()

	record2, _, err := enroll(t, sim2, appleSecureEnclaveClaim(attObj))
	if err != nil {
		t.Fatalf("unset ExpectedAppID Apple claim rejected: %v", err)
	}
	if record2.AttestationVerified {
		t.Fatalf("disabled Apple path recorded AttestationVerified")
	}
}

// TestAppleOversizedLengthFailsClosed is the orchestrator's regression test:
// a crafted 8-byte definite length (max uint64) must error, never panic the
// slicer via int(n) wrap-around.
func TestAppleOversizedLengthFailsClosed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("oversized CBOR length panicked: %v", r)
		}
	}()
	_, _, ca := newAppleAttestLeaf(t, make([]byte, 32))
	malicious := []byte{0xA1, 0x63, 'f', 'm', 't', 0x5B, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	if _, err := VerifyAppleAttestation(malicious, caPool(ca), []byte("nonce"), "TEAMID.com.example.app"); err == nil {
		t.Fatal("oversized CBOR length accepted")
	}
}
