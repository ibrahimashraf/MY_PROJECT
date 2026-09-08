package onboarding

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
)

// Apple App Attest offline verification.
//
// An Apple App Attest attestation object is a CBOR-encoded WebAuthn
// attestation response whose signature is an x509 certificate chain rooted in
// Apple's static App Attest Root CA. Verification is fully offline: parse the
// CBOR, verify the embedded chain against a caller-injected root pool, and
// prove the leaf's nonce binds the attestation to our enrollment challenge.
// Only the device-side generation of attestations needs an Apple Developer
// App ID; verifying them is free and requires no Apple server round trip.

// appAttestNonceOID is the leaf-certificate extension carrying the nonce
// (SHA256(authData || SHA256(clientData))) as defined by Apple App Attest.
// The 8.1 vs 8.2 distinction is moot here: 8.2 alone carries the nonce, so it
// is the only extension consulted. The siblings (8.3 aaguid, 8.4 tang, 8.6
// security posture) are intentionally NOT enforced — they describe credential
// metadata this verifier does not consume, and future Apple fields must not
// break today's verification. The nonce plus chain-to-root already bind
// authentic origin.
var appAttestNonceOID = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 8, 2}

// Sentinel errors for offline Apple App Attest verification.
var (
	ErrAppleAttestInvalid           = errors.New("apple attestation object invalid")
	ErrAppleAttestChallengeMismatch = errors.New("apple attestation challenge mismatch")
	ErrAppleAttestAppIDMismatch     = errors.New("apple attestation appID mismatch")
	ErrAppleAttestRootsUnconfigured = errors.New("apple attestation roots pool not configured")
)

// AppleAttestationInfo is the server-side verdict from VerifyAppleAttestation.
// Every field is true only on full success.
type AppleAttestationInfo struct {
	VerifiedBoot     bool // true on full success (App Attest implies verified boot)
	ChallengeMatched bool // nonce bound the attestation to the enrollment challenge
	AppIDMatched     bool // rpIdHash equaled SHA256(expectedAppID)
}

// appleAttestRoots holds the Apple App Attest Root CA pool used to verify
// SECURE_ENCLAVE claims at enrollment. Roots are ALWAYS injected by the
// operator — nothing is embedded from memory. nil fails closed with
// ErrAppleAttestRootsUnconfigured.
var appleAttestRoots *x509.CertPool

// SetAppleAttestRoots configures the package-wide Apple App Attest roots used
// to verify SECURE_ENCLAVE enrollment claims. Call it once at boot with the
// pool from LoadAppleAttestRoots. Passing nil unconfigures (Apple verification
// then fails closed with ErrAppleAttestRootsUnconfigured).
func SetAppleAttestRoots(pool *x509.CertPool) {
	appleAttestRoots = pool
}

// LoadAppleAttestRoots reads a PEM file of the Apple App Attest Root CA
// certificates and returns them as an x509.CertPool. Shares the Android roots
// loader — the PEM file format is identical.
func LoadAppleAttestRoots(pemPath string) (*x509.CertPool, error) {
	return LoadAttestationRoots(pemPath)
}

// enrollmentExpectedAppID is the orchestrator-configured "<teamID>.<bundleID>"
// that the Apple attestation's rpIdHash is checked against. Defaults to "",
// which disables the Apple verification path entirely: SECURE_ENCLAVE claims
// then enroll as CLAIMED (Verified=false) — an honest no-device-today posture.
var enrollmentExpectedAppID string

// SetEnrollmentExpectedAppID configures the appID expected in Apple App
// Attest attestations at enrollment. Empty (the default) disables the Apple
// verification path.
func SetEnrollmentExpectedAppID(appID string) {
	enrollmentExpectedAppID = appID
}

// cborReader is a minimal definite-length CBOR decoder covering only the
// subset App Attest uses: unsigned ints (0), negative ints (1), byte strings
// (2), text strings (3), arrays (4) and maps (5). Indefinite lengths (31),
// tags (6) and simple/float values (7) are rejected fail-closed.
type cborReader struct {
	data []byte
	off  int
}

// head reads a type/length initial byte and returns the major type and the
// definite-length value. Indefinite lengths and reserved additional info are
// rejected.
func (r *cborReader) head() (major byte, n uint64, err error) {
	if r.off >= len(r.data) {
		return 0, 0, ErrAppleAttestInvalid
	}
	initial := r.data[r.off]
	r.off++
	major = initial >> 5
	ai := initial & 0x1f
	switch {
	case ai < 24:
		n = uint64(ai)
	case ai <= 27:
		width := 1 << (ai - 24)
		if r.off+width > len(r.data) {
			return 0, 0, ErrAppleAttestInvalid
		}
		for i := 0; i < width; i++ {
			n = n<<8 | uint64(r.data[r.off])
			r.off++
		}
	case ai == 31: // indefinite length
		return 0, 0, ErrAppleAttestInvalid
	default: // 28, 29, 30 reserved
		return 0, 0, ErrAppleAttestInvalid
	}
	// No definite length can exceed the unread buffer: cap here so int(n)
	// conversions downstream can never wrap negative and panic the slicer.
	if n > uint64(len(r.data)-r.off) {
		return 0, 0, ErrAppleAttestInvalid
	}
	return major, n, nil
}

// skip advances past an entire CBOR item without interpreting its content.
// It is used to ignore fields we do not consume (e.g. the receipt).
func (r *cborReader) skip() error {
	major, n, err := r.head()
	if err != nil {
		return err
	}
	switch major {
	case 0, 1:
		return nil
	case 2, 3:
		if int(n) > len(r.data)-r.off {
			return ErrAppleAttestInvalid
		}
		r.off += int(n)
		return nil
	case 4:
		for i := uint64(0); i < n; i++ {
			if err := r.skip(); err != nil {
				return err
			}
		}
		return nil
	case 5:
		for i := uint64(0); i < n; i++ {
			if err := r.skip(); err != nil { // key
				return err
			}
			if err := r.skip(); err != nil { // value
				return err
			}
		}
		return nil
	default: // tags and simple/float fail closed
		return ErrAppleAttestInvalid
	}
}

func (r *cborReader) byteString() ([]byte, error) {
	major, n, err := r.head()
	if err != nil {
		return nil, err
	}
	if major != 2 {
		return nil, ErrAppleAttestInvalid
	}
	if r.off+int(n) > len(r.data) {
		return nil, ErrAppleAttestInvalid
	}
	out := r.data[r.off : r.off+int(n)]
	r.off += int(n)
	return out, nil
}

func (r *cborReader) textString() (string, error) {
	major, n, err := r.head()
	if err != nil {
		return "", err
	}
	if major != 3 {
		return "", ErrAppleAttestInvalid
	}
	if r.off+int(n) > len(r.data) {
		return "", ErrAppleAttestInvalid
	}
	out := string(r.data[r.off : r.off+int(n)])
	r.off += int(n)
	return out, nil
}

func (r *cborReader) arrayLen() (int, error) {
	major, n, err := r.head()
	if err != nil {
		return 0, err
	}
	if major != 4 {
		return 0, ErrAppleAttestInvalid
	}
	return int(n), nil
}

func (r *cborReader) mapLen() (int, error) {
	major, n, err := r.head()
	if err != nil {
		return 0, err
	}
	if major != 5 {
		return 0, ErrAppleAttestInvalid
	}
	return int(n), nil
}

// appleAttestObject is the parsed subset of the attestation CBOR we consume.
type appleAttestObject struct {
	authData []byte
	x5c      [][]byte
}

// parseAppleAttestObject reads the top-level attestation map. The attStmt
// "receipt" field and any unknown keys are skipped, not rejected, so future
// attestation objects remain verifiable. Trailing bytes after the single
// top-level map are rejected (fail closed on ambiguous input).
func parseAppleAttestObject(attObj []byte) (appleAttestObject, error) {
	var out appleAttestObject
	r := &cborReader{data: attObj}
	fields, err := r.mapLen()
	if err != nil {
		return out, err
	}
	for i := 0; i < fields; i++ {
		key, err := r.textString()
		if err != nil {
			return out, err
		}
		switch key {
		case "fmt":
			format, err := r.textString()
			if err != nil {
				return out, err
			}
			if format != "apple-appattest" {
				return out, fmt.Errorf("%w: attestation format %q is not apple-appattest", ErrAppleAttestInvalid, format)
			}
		case "authData":
			out.authData, err = r.byteString()
			if err != nil {
				return out, err
			}
		case "attStmt":
			stmtFields, err := r.mapLen()
			if err != nil {
				return out, err
			}
			for j := 0; j < stmtFields; j++ {
				stmtKey, err := r.textString()
				if err != nil {
					return out, err
				}
				switch stmtKey {
				case "x5c":
					certs, err := r.arrayLen()
					if err != nil {
						return out, err
					}
					if certs == 0 {
						return out, ErrAppleAttestInvalid
					}
					for k := 0; k < certs; k++ {
						der, err := r.byteString()
						if err != nil {
							return out, err
						}
						out.x5c = append(out.x5c, der)
					}
				default: // "receipt" and future statement fields are ignored.
					if err := r.skip(); err != nil {
						return out, err
					}
				}
			}
		default:
			if err := r.skip(); err != nil {
				return out, err
			}
		}
	}
	if len(out.authData) == 0 || len(out.x5c) == 0 {
		return out, ErrAppleAttestInvalid
	}
	if r.off != len(attObj) {
		return out, ErrAppleAttestInvalid
	}
	return out, nil
}

// parseAuthData extracts the rpIdHash from the WebAuthn authenticator data:
//
//	rpIdHash[32] || flags[1] || signCount[4] || attestedCredentialData
//
// attestedCredentialData = aaguid[16] || credIdLen[2] || credId ||
// coseCredentialPublicKey. Only rpIdHash is consumed — the remainder is
// covered by the nonce, which hashes the whole authData, so no field-level
// checks are needed here.
func parseAuthData(authData []byte) ([]byte, error) {
	if len(authData) < 32+1+4 {
		return nil, ErrAppleAttestInvalid
	}
	return authData[:32], nil
}

// appleNonce computes the nonce an App Attest certificate must carry:
// SHA256(authData || SHA256(clientData)). clientData is the enrollment
// challenge bytes: in the standard WebAuthn flow clientData is the JSON
// registration payload and its hash is the clientDataHash Apple folds into
// the nonce. We bind the challenge by using the raw enrollment nonce bytes as
// clientData, keeping the computation offline and deterministic — the binding
// is still unforgeable because the leaf must be signed by a chain rooted at
// Apple's CA.
func appleNonce(authData, clientData []byte) []byte {
	clientHash := sha256.Sum256(clientData)
	h := sha256.New()
	h.Write(authData)
	h.Write(clientHash[:])
	return h.Sum(nil)
}

// leafNonce reads the nonce from the leaf's App Attest extension. Go's x509
// parser unwraps the certificate's outer extnValue OCTET STRING, leaving
// either the raw nonce or a DER OCTET STRING wrapping it depending on the
// issuer. Both forms are accepted.
func leafNonce(leaf *x509.Certificate) ([]byte, bool) {
	for _, ext := range leaf.Extensions {
		if !ext.Id.Equal(appAttestNonceOID) {
			continue
		}
		if len(ext.Value) == sha256.Size {
			return ext.Value, true
		}
		var inner asn1.RawValue
		rest, err := asn1.Unmarshal(ext.Value, &inner)
		if err == nil && len(rest) == 0 && inner.Class == asn1.ClassUniversal && inner.Tag == asn1.TagOctetString {
			return inner.Bytes, true
		}
		return nil, false
	}
	return nil, false
}

// VerifyAppleAttestation verifies an Apple App Attest attestation object fully
// offline. expectedChallenge is the enrollment nonce, used as clientData (see
// appleNonce); expectedAppID is the orchestrator-configured "<teamID>.<bundleID>"
// string whose SHA256 must equal the authData rpIdHash — it is never hardcoded.
// roots is the operator-injected App Attest Root CA pool; a nil pool fails
// closed before any parsing returns ErrAppleAttestRootsUnconfigured.
func VerifyAppleAttestation(attObj []byte, roots *x509.CertPool, expectedChallenge []byte, expectedAppID string) (AppleAttestationInfo, error) {
	info := AppleAttestationInfo{}
	if roots == nil {
		return info, ErrAppleAttestRootsUnconfigured
	}
	if expectedAppID == "" {
		return info, ErrAppleAttestInvalid
	}

	obj, err := parseAppleAttestObject(attObj)
	if err != nil {
		return info, err
	}

	rpIDHash, err := parseAuthData(obj.authData)
	if err != nil {
		return info, err
	}

	leaf, err := x509.ParseCertificate(obj.x5c[0])
	if err != nil {
		return info, fmt.Errorf("%w: leaf certificate: %v", ErrAppleAttestInvalid, err)
	}
	intermediates := x509.NewCertPool()
	for _, der := range obj.x5c[1:] {
		c, err := x509.ParseCertificate(der)
		if err != nil {
			return info, fmt.Errorf("%w: intermediate certificate: %v", ErrAppleAttestInvalid, err)
		}
		intermediates.AddCert(c)
	}
	if _, err := leaf.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return info, fmt.Errorf("%w: %v", ErrAppleAttestInvalid, err)
	}

	nonce, ok := leafNonce(leaf)
	if !ok || !bytes.Equal(nonce, appleNonce(obj.authData, expectedChallenge)) {
		return info, ErrAppleAttestChallengeMismatch
	}
	info.ChallengeMatched = true

	appIDHash := sha256.Sum256([]byte(expectedAppID))
	if !bytes.Equal(rpIDHash, appIDHash[:]) {
		return info, ErrAppleAttestAppIDMismatch
	}
	info.AppIDMatched = true
	info.VerifiedBoot = true
	return info, nil
}
