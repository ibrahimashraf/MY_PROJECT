package onboarding

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"
)

// --- Hermetic DER builders (test-only minimal ASN.1) ---------------------

func derLen(n int) []byte {
	if n < 128 {
		return []byte{byte(n)}
	}
	return []byte{0x81, byte(n)}
}

func derTLV(tag byte, content []byte) []byte {
	out := append([]byte{tag}, derLen(len(content))...)
	return append(out, content...)
}

func derInt(v int) []byte      { return derTLV(0x02, minimalIntBytes(v)) }
func derEnum(v int) []byte     { return derTLV(0x0A, minimalIntBytes(v)) }
func derOctet(b []byte) []byte { return derTLV(0x04, b) }
func derBool(v bool) []byte {
	if v {
		return derTLV(0x01, []byte{0xff})
	}
	return derTLV(0x01, []byte{0x00})
}
func derSeq(items ...[]byte) []byte {
	var content []byte
	for _, it := range items {
		content = append(content, it...)
	}
	return derTLV(0x30, content)
}
func derSet(items ...[]byte) []byte {
	var content []byte
	for _, it := range items {
		content = append(content, it...)
	}
	return derTLV(0x31, content)
}

func minimalIntBytes(v int) []byte {
	if v == 0 {
		return []byte{0x00}
	}
	var b []byte
	for n := v; n != 0; n >>= 8 {
		b = append([]byte{byte(n)}, b...)
	}
	if b[0]&0x80 != 0 {
		b = append([]byte{0x00}, b...)
	}
	return b
}

// derExplicit wraps inner in an EXPLICIT context-specific constructed tag
// (class 2, constructed): [tag] EXPLICIT. Handles high-tag-number form.
func derExplicit(tag int, inner []byte) []byte {
	h := []byte{0xA0 | byte(tag)}
	if tag >= 31 {
		h = []byte{0x9F}
		var t []byte
		for n := tag; n > 0; n /= 128 {
			t = append([]byte{byte(n % 128)}, t...)
		}
		for i := 0; i < len(t)-1; i++ {
			t[i] |= 0x80
		}
		h = append(h, t...)
	}
	return append(append(h, derLen(len(inner))...), inner...)
}

// buildKeyDescription encodes an Android KeyDescription. secLevel is the
// attestationSecurityLevel (0 Software, 1 TEE, 2 StrongBox); bootState is the
// verifiedBootState (0 Verified). teePurposes nil omits the purpose tag;
// bootState < 0 omits the rootOfTrust tag, so callers can build malformed/no-
// enforcement cases that must fail closed.
func buildKeyDescription(secLevel int, challenge []byte, teePurposes []int, bootState int) []byte {
	var tee []byte
	if teePurposes != nil {
		ints := make([][]byte, 0, len(teePurposes))
		for _, p := range teePurposes {
			ints = append(ints, derInt(p))
		}
		tee = derExplicit(kmAuthListPurposeTag, derSet(ints...))
	}
	if bootState >= 0 {
		tee = append(tee, derExplicit(kmTagRootOfTrust, derSeq(
			derOctet([]byte{0xde, 0xad, 0xbe, 0xef}), // verifiedBootKey
			derBool(true),                            // deviceLocked
			derEnum(bootState),                       // verifiedBootState
		))...)
	}
	return derSeq(
		derInt(3),           // attestationVersion
		derEnum(secLevel),   // attestationSecurityLevel
		derInt(100),         // keymasterVersion
		derEnum(secLevel),   // keymasterSecurityLevel
		derOctet(challenge), // attestationChallenge
		derOctet(nil),       // uniqueId
		derSeq(),            // softwareEnforced (empty for hardware keys)
		derSeq(tee),         // teeEnforced
	)
}

// --- Hermetic chain fixtures (own ECDSA CA, injected roots, no network) ---

// newTestAttestationLeaf mints a self-contained CA + leaf signed by it, with
// the given KeyDescription bytes embedded as the leaf's attestation
// extension. Returns the parsed certs and their PEM encodings.
func newTestAttestationLeaf(t *testing.T, extDER []byte) (leaf, ca *x509.Certificate, leafPEM, caPEM []byte) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gen ca key: %v", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Attestation Root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
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
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Test Device"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		ExtraExtensions: []pkix.Extension{
			{Id: attestationExtOID, Critical: false, Value: extDER},
		},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}
	leaf, err = x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}

	leafPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	return leaf, ca, leafPEM, caPEM
}

func caPool(ca *x509.Certificate) *x509.CertPool {
	p := x509.NewCertPool()
	p.AddCert(ca)
	return p
}

// TestVerifyAttestationChain (a)-(g): hermetic chain verification.
func TestVerifyAttestationChain(t *testing.T) {
	challenge := []byte("test-challenge-123")

	// (a) valid chain + matching challenge + StrongBox passes, StrongBox=true.
	leaf, ca, _, _ := newTestAttestationLeaf(t, buildKeyDescription(2, challenge, []int{2}, 0))
	info, err := VerifyAttestationChain([]*x509.Certificate{leaf, ca}, caPool(ca), challenge)
	if err != nil {
		t.Fatalf("(a) valid StrongBox chain rejected: %v", err)
	}
	if !info.StrongBox || !info.VerifiedBoot || !info.ChallengeMatched {
		t.Fatalf("(a) info not fully true: %+v", info)
	}

	// (b) wrong challenge rejected.
	leaf, ca, _, _ = newTestAttestationLeaf(t, buildKeyDescription(2, challenge, []int{2}, 0))
	if _, err := VerifyAttestationChain([]*x509.Certificate{leaf, ca}, caPool(ca), []byte("wrong")); !errors.Is(err, ErrAttestationChallengeMismatch) {
		t.Fatalf("(b) wrong challenge err=%v", err)
	}

	// (c) TEE level passes with StrongBox=false.
	leaf, ca, _, _ = newTestAttestationLeaf(t, buildKeyDescription(1, challenge, []int{2}, 0))
	info, err = VerifyAttestationChain([]*x509.Certificate{leaf, ca}, caPool(ca), challenge)
	if err != nil {
		t.Fatalf("(c) valid TEE chain rejected: %v", err)
	}
	if info.StrongBox {
		t.Fatalf("(c) TEE mislabeled as StrongBox: %+v", info)
	}

	// (d) software level (0) rejected.
	leaf, ca, _, _ = newTestAttestationLeaf(t, buildKeyDescription(0, challenge, []int{2}, 0))
	if _, err := VerifyAttestationChain([]*x509.Certificate{leaf, ca}, caPool(ca), challenge); !errors.Is(err, ErrAttestationSecurityLevel) {
		t.Fatalf("(d) software level err=%v", err)
	}

	// (e) boot state != Verified rejected.
	leaf, ca, _, _ = newTestAttestationLeaf(t, buildKeyDescription(2, challenge, []int{2}, 1))
	if _, err := VerifyAttestationChain([]*x509.Certificate{leaf, ca}, caPool(ca), challenge); !errors.Is(err, ErrAttestationBootState) {
		t.Fatalf("(e) bad boot state err=%v", err)
	}

	// (f) missing SIGN purpose rejected.
	leaf, ca, _, _ = newTestAttestationLeaf(t, buildKeyDescription(2, challenge, []int{1}, 0))
	if _, err := VerifyAttestationChain([]*x509.Certificate{leaf, ca}, caPool(ca), challenge); !errors.Is(err, ErrAttestationPurpose) {
		t.Fatalf("(f) non-SIGN purpose err=%v", err)
	}

	// (g) root not in pool rejected.
	_, otherCA, _, _ := newTestAttestationLeaf(t, buildKeyDescription(2, challenge, []int{2}, 0))
	foreign, foreignCA, _, _ := newTestAttestationLeaf(t, buildKeyDescription(2, challenge, []int{2}, 0))
	if _, err := VerifyAttestationChain([]*x509.Certificate{foreign, foreignCA}, caPool(otherCA), challenge); !errors.Is(err, ErrAttestationChainInvalid) {
		t.Fatalf("(g) unrooted chain err=%v", err)
	}

	// nil roots pool fails closed with the unconfigured error.
	if _, err := VerifyAttestationChain([]*x509.Certificate{leaf, ca}, nil, challenge); !errors.Is(err, ErrAttestationRootsUnconfigured) {
		t.Fatalf("nil roots err=%v", err)
	}
}

// chainEnrollment drives a full enrollment whose STRONGBOX claim carries a
// PEM chain built against the given nonce.
func chainEnrollment(t *testing.T, sim *EnrollmentSimulator, chainPEM []string) (*DeviceTrustRecord, error) {
	t.Helper()
	chal, err := sim.CreateEnrollmentChallenge("ten_test", "insp_test")
	if err != nil {
		t.Fatalf("challenge: %v", err)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	return sim.ProcessDeviceEnrollment(DeviceEnrollmentSubmission{
		ChallengeID:     chal.ChallengeID,
		InspectorID:     "insp_test",
		DevicePublicKey: hex.EncodeToString(pub),
		DeviceModel:     "Test Tablet",
		SignedNonce:     hex.EncodeToString(ed25519.Sign(priv, []byte(chal.Nonce))),
		Attestation: AttestationClaim{
			KeyOrigin:       KeyOriginStrongBox,
			OSVersion:       "Android 14",
			AttestationBlob: "deadbeef",
			ChainPEM:        chainPEM,
		},
	})
}

// TestEnrollmentVerifiedStrongBoxChain records AttestationVerified for a
// valid chain whose challenge matches the enrollment nonce.
func TestEnrollmentVerifiedStrongBoxChain(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim: %v", err)
	}

	// Prevent the enrollment nonce from escaping our control: build the leaf
	// AFTER the challenge is created so the nonce is known.
	chal, err := sim.CreateEnrollmentChallenge("ten_test", "insp_test")
	if err != nil {
		t.Fatalf("challenge: %v", err)
	}
	_, ca, leafPEM, caPEM := newTestAttestationLeaf(t, buildKeyDescription(2, []byte(chal.Nonce), []int{2}, 0))
	pool := caPool(ca)
	chainVerifyRoots = pool
	defer func() { chainVerifyRoots = nil }()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	record, err := sim.ProcessDeviceEnrollment(DeviceEnrollmentSubmission{
		ChallengeID:     chal.ChallengeID,
		InspectorID:     "insp_test",
		DevicePublicKey: hex.EncodeToString(pub),
		DeviceModel:     "Test Tablet",
		SignedNonce:     hex.EncodeToString(ed25519.Sign(priv, []byte(chal.Nonce))),
		Attestation: AttestationClaim{
			KeyOrigin:       KeyOriginStrongBox,
			OSVersion:       "Android 14",
			AttestationBlob: "deadbeef",
			ChainPEM:        []string{string(leafPEM), string(caPEM)},
		},
	})
	if err != nil {
		t.Fatalf("verified StrongBox enrollment failed: %v", err)
	}
	if !record.AttestationVerified {
		t.Fatalf("valid chain did not record AttestationVerified")
	}
	if record.AttestationOrigin != string(KeyOriginStrongBox) {
		t.Fatalf("origin not recorded: %q", record.AttestationOrigin)
	}
}

// (h) enrollment with garbage ChainPEM fails closed and stores nothing.
func TestEnrollmentGarbageChainFailsClosed(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim: %v", err)
	}
	_, ca, _, _ := newTestAttestationLeaf(t, buildKeyDescription(2, []byte("x"), []int{2}, 0))
	chainVerifyRoots = caPool(ca)
	defer func() { chainVerifyRoots = nil }()

	_, err = chainEnrollment(t, sim, []string{"not a certificate"})
	if err == nil {
		t.Fatalf("garbage chain accepted")
	}
	if !errors.Is(err, ErrAttestationChainInvalid) {
		t.Fatalf("garbage chain err=%v", err)
	}
	if len(sim.deviceStore) != 0 {
		t.Fatalf("garbage chain stored %d devices", len(sim.deviceStore))
	}
}

// (i) enrollment with ChainPEM but nil roots pool returns
// ErrAttestationRootsUnconfigured and stores nothing.
func TestEnrollmentNilRootsFailsClosed(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim: %v", err)
	}
	_, _, leafPEM, caPEM := newTestAttestationLeaf(t, buildKeyDescription(2, []byte("x"), []int{2}, 0))
	chainVerifyRoots = nil
	defer func() { chainVerifyRoots = nil }()

	_, err = chainEnrollment(t, sim, []string{string(leafPEM), string(caPEM)})
	if err == nil {
		t.Fatalf("nil-roots chain accepted")
	}
	if !errors.Is(err, ErrAttestationRootsUnconfigured) {
		t.Fatalf("nil-roots err=%v", err)
	}
	if len(sim.deviceStore) != 0 {
		t.Fatalf("nil-roots stored %d devices", len(sim.deviceStore))
	}
}

// (h-cont) non-STRONGBOX origins ignore ChainPEM entirely, even garbage.
func TestNonAndroidOriginIgnoresChain(t *testing.T) {
	chainVerifyRoots = nil
	defer func() { chainVerifyRoots = nil }()

	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("sim: %v", err)
	}
	chal, err := sim.CreateEnrollmentChallenge("ten_test", "insp_test")
	if err != nil {
		t.Fatalf("challenge: %v", err)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	record, err := sim.ProcessDeviceEnrollment(DeviceEnrollmentSubmission{
		ChallengeID:     chal.ChallengeID,
		InspectorID:     "insp_test",
		DevicePublicKey: hex.EncodeToString(pub),
		DeviceModel:     "Test Tablet",
		SignedNonce:     hex.EncodeToString(ed25519.Sign(priv, []byte(chal.Nonce))),
		Attestation: AttestationClaim{
			KeyOrigin:       KeyOriginSecureEnclave,
			BiometricBound:  true,
			OSVersion:       "iOS 17.2",
			AttestationBlob: "abcd",
			ChainPEM:        []string{"garbage"},
		},
	})
	if err != nil {
		t.Fatalf("Secure Enclave chain ignored, but enrollment failed: %v", err)
	}
	if record.AttestationVerified {
		t.Fatalf("non-Android origin recorded AttestationVerified")
	}
}
