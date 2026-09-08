package onboarding

import (
	"bytes"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// Android key-attestation extension OID (KeyDescription), as published in
// AOSP system/keymaster/include/keymaster/attestation_record.h.
var attestationExtOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 1, 17}

// Keymaster constants used inside the KeyDescription structure.
const (
	// SecurityLevel ::= ENUMERATED { Software, TrustedEnvironment, StrongBox }.
	attestationSecurityLevelTEE       = 1
	attestationSecurityLevelStrongBox = 2
	// VerifiedBootState ::= ENUMERATED { Verified, SelfSigned, Unverified, Failed }.
	verifiedBootStateVerified = 0
	// AuthorizationList field tag for the key purpose set, and KM_PURPOSE_SIGN.
	kmAuthListPurposeTag = 0
	kmPurposeSign        = 2
	kmTagRootOfTrust     = 704 // AuthorizationList field tag 704 (KM_TAG_ROOT_OF_TRUST)
)

// Sentinel errors for offline Android key-attestation verification.
var (
	ErrAttestationChainInvalid      = errors.New("attestation chain invalid")
	ErrAttestationChallengeMismatch = errors.New("attestation challenge mismatch")
	ErrAttestationSecurityLevel     = errors.New("attestation security level is not hardware-backed")
	ErrAttestationBootState         = errors.New("attestation boot state is not verified")
	ErrAttestationPurpose           = errors.New("attestation key purpose does not include SIGN")
	ErrAttestationRootsUnconfigured = errors.New("attestation roots pool not configured")
)

// chainVerifyRoots holds the Google Android Hardware Attestation Root
// certificates used to verify STRONGBOX chain claims at enrollment. It is
// provisioned out-of-band — see LoadAttestationRoots. nil means the operator
// has not configured roots, so chain verification fails closed with
// ErrAttestationRootsUnconfigured rather than trusting an unverifiable claim.
var chainVerifyRoots *x509.CertPool

// SetAttestationRoots configures the package-wide Android attestation roots
// used to verify STRONGBOX chain claims at enrollment. Call it once at boot
// with the pool from LoadAttestationRoots. Passing nil unconfigures (chain
// verification then fails closed with ErrAttestationRootsUnconfigured).
func SetAttestationRoots(pool *x509.CertPool) {
	chainVerifyRoots = pool
}

// AttestationInfo is the server-side verdict from VerifyAttestationChain.
// All fields are true only on full success.
type AttestationInfo struct {
	StrongBox        bool // attested security level was StrongBox (not just TEE)
	VerifiedBoot     bool // verifiedBootState was Verified (GREEN)
	ChallengeMatched bool // attestationChallenge equalled the enrollment nonce
}

// VerifyAttestationChain verifies an Android key-attestation certificate
// chain fully offline using only the standard library. The chain is
// leaf-first; any intermediates come from chain[1:]. roots is the certificate
// pool of Google-published attestation roots (or the operator's injected
// roots in tests/production). expectedChallenge must equal the bytes the
// Keymaster "attestationChallenge" was generated with (the enrollment nonce).
//
// Success requires: chain-of-trust to a configured root, a parseable
// KeyDescription extension, challenge byte-equality, a hardware security
// level (TEE or StrongBox), verifiedBootState == Verified, and SIGN among
// the attested key purposes. Anything else fails closed.
func VerifyAttestationChain(chain []*x509.Certificate, roots *x509.CertPool, expectedChallenge []byte) (AttestationInfo, error) {
	info := AttestationInfo{}
	if len(chain) == 0 {
		return info, ErrAttestationChainInvalid
	}
	if roots == nil {
		return info, ErrAttestationRootsUnconfigured
	}
	intermediates := x509.NewCertPool()
	for _, cert := range chain[1:] {
		intermediates.AddCert(cert)
	}
	if _, err := chain[0].Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return info, fmt.Errorf("%w: %v", ErrAttestationChainInvalid, err)
	}

	kd, err := parseKeyDescription(chain[0])
	if err != nil {
		return info, fmt.Errorf("%w: %v", ErrAttestationChainInvalid, err)
	}
	if !bytes.Equal(kd.challenge, expectedChallenge) {
		return info, ErrAttestationChallengeMismatch
	}
	info.ChallengeMatched = true

	switch kd.securityLevel {
	case attestationSecurityLevelStrongBox:
		info.StrongBox = true
	case attestationSecurityLevelTEE:
		// TEE is accepted hardware attestation; StrongBox stays false.
	default:
		return info, ErrAttestationSecurityLevel
	}

	if kd.bootState != verifiedBootStateVerified {
		return info, ErrAttestationBootState
	}
	info.VerifiedBoot = true

	for _, p := range append(append([]int{}, kd.softwarePurposes...), kd.teePurposes...) {
		if p == kmPurposeSign {
			return info, nil
		}
	}
	return info, ErrAttestationPurpose
}

// keyDescription is the parsed KeyDescription extension content we care
// about. software/tee enforced purposes are checked for SIGN; the boot state
// always comes from the teeEnforced rootOfTrust.
type keyDescription struct {
	securityLevel    int
	challenge        []byte
	softwarePurposes []int
	teePurposes      []int
	bootState        int
}

func parseKeyDescription(leaf *x509.Certificate) (keyDescription, error) {
	var kd keyDescription
	var extValue []byte
	for _, ext := range leaf.Extensions {
		if ext.Id.Equal(attestationExtOID) {
			extValue = ext.Value
			break
		}
	}
	if extValue == nil {
		return kd, errors.New("attestation extension missing")
	}
	var fields []asn1.RawValue
	rest, err := asn1.Unmarshal(extValue, &fields)
	if err != nil || len(rest) != 0 {
		return kd, fmt.Errorf("attestation extension is not a KeyDescription sequence: %v", err)
	}
	// KeyDescription ::= SEQUENCE {
	//   attestationVersion INTEGER, attestationSecurityLevel SecurityLevel,
	//   keymasterVersion INTEGER, keymasterSecurityLevel SecurityLevel,
	//   attestationChallenge OCTET STRING, uniqueId OCTET STRING,
	//   softwareEnforced AuthorizationList, teeEnforced AuthorizationList }
	if len(fields) < 8 {
		return kd, errors.New("attestation extension truncated")
	}
	kd.securityLevel, err = intField(fields[1])
	if err != nil {
		return kd, fmt.Errorf("attestationSecurityLevel: %w", err)
	}
	if fields[4].Class != asn1.ClassUniversal || fields[4].Tag != asn1.TagOctetString {
		return kd, errors.New("attestationChallenge is not an OCTET STRING")
	}
	kd.challenge = fields[4].Bytes

	software, err := parseAuthorizationList(fields[6].FullBytes)
	if err != nil {
		return kd, err
	}
	tee, err := parseAuthorizationList(fields[7].FullBytes)
	if err != nil {
		return kd, err
	}
	kd.softwarePurposes = software.purposes
	kd.teePurposes = tee.purposes
	kd.bootState = tee.bootState
	return kd, nil
}

// authList is the parsed subset of an authorization list (software- or
// teeEnforced) that this module enforces on.
type authList struct {
	purposes  []int
	bootState int // -1 when no rootOfTrust present
}

// parseAuthorizationList reads a keymaster AuthorizationList: a SEQUENCE of
// EXPLICIT context-specific tags (e.g. [0] EXPLICIT SET OF INTEGER for the
// purpose and [704] EXPLICIT RootOfTrust). listSeq must be the complete
// SEQUENCE value (tag+length+content), i.e. the FullBytes of its element in
// the KeyDescription.
func parseAuthorizationList(listSeq []byte) (authList, error) {
	al := authList{bootState: -1}
	var fields []asn1.RawValue
	rest, err := asn1.Unmarshal(listSeq, &fields)
	if err != nil || len(rest) != 0 {
		return al, fmt.Errorf("malformed authorization list: %v", err)
	}
	for _, f := range fields {
		if f.Class != asn1.ClassContextSpecific {
			return al, errors.New("unexpected tag class in authorization list")
		}
		switch f.Tag {
		case kmAuthListPurposeTag:
			ps, err := intSet(f.Bytes)
			if err != nil {
				return al, err
			}
			al.purposes = append(al.purposes, ps...)
		case kmTagRootOfTrust:
			state, err := rootOfTrustBootState(f.Bytes)
			if err != nil {
				return al, err
			}
			al.bootState = state
		}
	}
	return al, nil
}

// rootOfTrustBootState parses RootOfTrust ::= SEQUENCE { verifiedBootKey
// OCTET STRING, deviceLocked BOOLEAN, verifiedBootState ENUMERATED } and
// returns the verifiedBootState ENUMERATED value.
func rootOfTrustBootState(inner []byte) (int, error) {
	var fields []asn1.RawValue
	rest, err := asn1.Unmarshal(inner, &fields)
	if err != nil || len(rest) != 0 {
		return -1, fmt.Errorf("malformed rootOfTrust: %v", err)
	}
	if len(fields) < 3 {
		return -1, errors.New("rootOfTrust truncated")
	}
	if fields[0].Class != asn1.ClassUniversal || fields[0].Tag != asn1.TagOctetString {
		return -1, errors.New("verifiedBootKey is not an OCTET STRING")
	}
	return intField(fields[2])
}

// intField decodes an ASN.1 INTEGER or ENUMERATED element (an
// AuthorizationList [N] EXPLICIT wrapper, or a KeyDescription level field).
func intField(f asn1.RawValue) (int, error) {
	switch f.Tag {
	case asn1.TagInteger, asn1.TagEnum:
		return decodeIntContent(f.Bytes)
	default:
		return 0, fmt.Errorf("expected INTEGER/ENUMERATED element, got tag %d", f.Tag)
	}
}

// splitDER walks a DER byte string and returns the encoded sub-elements.
// Unlike asn1.Unmarshal it accepts any outer tag class (including SET, whose
// DER ordering is deliberately not relied upon here).
func splitDER(data []byte) ([]asn1.RawValue, error) {
	var out []asn1.RawValue
	for off := 0; off < len(data); {
		elemStart := off
		b := data[off]
		off++
		class := int(b >> 6)
		compound := b&0x20 != 0
		tag := int(b & 0x1f)
		if tag == 0x1f {
			tag = 0
			done := false
			for !done {
				if off >= len(data) {
					return nil, errors.New("truncated DER tag")
				}
				c := data[off]
				off++
				tag = tag<<7 | int(c&0x7f)
				if c&0x80 == 0 {
					done = true
				}
			}
		}
		if off >= len(data) {
			return nil, errors.New("truncated DER length")
		}
		l := int(data[off])
		off++
		if l&0x80 != 0 {
			n := int(l & 0x7f)
			if n > 2 || off+n > len(data) {
				return nil, errors.New("unsupported DER length")
			}
			l = 0
			for i := 0; i < n; i++ {
				l = l<<8 | int(data[off])
				off++
			}
		}
		if off+l > len(data) {
			return nil, errors.New("truncated DER content")
		}
		contentStart := off
		off += l
		out = append(out, asn1.RawValue{
			Class:      class,
			Tag:        tag,
			IsCompound: compound,
			Bytes:      data[contentStart : contentStart+l],
			FullBytes:  data[elemStart : contentStart+l],
		})
	}
	return out, nil
}

// intSet parses the purpose value: [0] EXPLICIT wrapping a SET OF INTEGER.
// raw is the explicit wrapper's content (the full SET DER); its members must
// all be SIGN-comparable INTEGERs.
func intSet(raw []byte) ([]int, error) {
	outer, err := splitDER(raw)
	if err != nil || len(outer) != 1 {
		return nil, errors.New("purpose tag does not wrap a single SET")
	}
	set := outer[0]
	if set.Class != asn1.ClassUniversal || set.Tag != asn1.TagSet {
		return nil, errors.New("purpose tag does not wrap a SET OF INTEGER")
	}
	elems, err := splitDER(set.Bytes)
	if err != nil {
		return nil, fmt.Errorf("malformed SET OF INTEGER: %v", err)
	}
	out := make([]int, 0, len(elems))
	for _, e := range elems {
		if e.Class != asn1.ClassUniversal || e.Tag != asn1.TagInteger {
			return nil, errors.New("non-integer in purpose set")
		}
		v, err := decodeIntContent(e.Bytes)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// decodeIntContent converts ASN.1 INTEGER/ENUMERATED value bytes
// (big-endian, two's complement, minimal-form checked) to an int.
func decodeIntContent(b []byte) (int, error) {
	if len(b) == 0 || len(b) > 8 {
		return 0, errors.New("invalid integer length")
	}
	if len(b) > 1 {
		if (b[0] == 0 && b[1]&0x80 == 0) || (b[0] == 0xff && b[1]&0x80 != 0) {
			return 0, errors.New("non-minimal integer encoding")
		}
	}
	v := 0
	for _, by := range b {
		v = v<<8 | int(by)
	}
	if len(b) == 8 {
		return 0, errors.New("integer overflows int")
	}
	if b[0]&0x80 != 0 {
		v -= 1 << (8 * len(b))
	}
	return v, nil
}

// verifyPEMChain parses a leaf-first PEM certificate chain and runs
// VerifyAttestationChain against roots. A nil roots pool fails closed before
// any parsing so the error is always ErrAttestationRootsUnconfigured.
func verifyPEMChain(pems []string, roots *x509.CertPool, expectedChallenge []byte) (AttestationInfo, error) {
	if roots == nil {
		return AttestationInfo{}, ErrAttestationRootsUnconfigured
	}
	certs := make([]*x509.Certificate, 0, len(pems))
	for _, p := range pems {
		block, _ := pem.Decode([]byte(p))
		if block == nil || block.Type != "CERTIFICATE" {
			return AttestationInfo{}, ErrAttestationChainInvalid
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return AttestationInfo{}, fmt.Errorf("%w: %v", ErrAttestationChainInvalid, err)
		}
		certs = append(certs, cert)
	}
	return VerifyAttestationChain(certs, roots, expectedChallenge)
}

// LoadAttestationRoots reads a PEM file containing the Google Android
// Hardware Attestation Root certificates and returns them as an x509.CertPool.
//
// The roots file is provisioned out-of-band by the operator. Google publishes
// the roots and the offline-verification pattern in the Android documentation:
// https://developer.android.com/privacy-and-security/keystore/attestation
// The two-root ordering and updates are the operator's responsibility.
func LoadAttestationRoots(pemPath string) (*x509.CertPool, error) {
	data, err := os.ReadFile(pemPath)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, errors.New("no certificates found in attestation roots file")
	}
	return pool, nil
}
