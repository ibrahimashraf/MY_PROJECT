// Package timestamp implements the RFC 3161 Time-Stamp Protocol (TSP): the
// TimeStampReq encoder, the TimeStampResp parser and the token verifier used
// to attach independent, non-refutable issuance times to certificates.
//
// Only stdlib encoding/asn1 and crypto are used. Trusted TSA roots are always
// operator-provisioned at boot (see ProvisionFromEnv); nothing is embedded.
// Verification is fail-closed across four independent checks (RFC 3161 §2.4.2
// + RFC 5652 §5.4):
//  1. PKIStatus must be granted;
//  2. the TSTInfo message imprint must match the expected digest exactly;
//  3. the signer's signedAttrs must carry a messageDigest attribute equal to
//     the SHA-256 of the TSTInfo DER;
//  4. the SignerInfo signature over the DER-encoded signedAttrs must verify
//     against the signer certificate's public key (RSA PKCS#1 v1.5 or ECDSA
//     by SignatureAlgorithm OID; any unknown algorithm fails closed), and
//     the signer chain must reach an operator-provisioned root.
package timestamp

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	_ "crypto/sha512"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
	"time"
)

// RFC 3161 / RFC 5652 object identifiers used by the protocol.
var (
	oidSHA1          = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26} // SHA-1 digest (hard-rejected)
	oidSHA256        = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	oidSHA384        = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
	oidSHA512        = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}
	oidSigned        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}        // id-signedData
	oidTSTInfo       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4} // id-ct-TSTInfo
	oidContentType   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	oidMessageDigest = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}

	oidSHA1WithRSA     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 5} // SHA-1 PKCS#1 v1.5 (hard-rejected)
	oidSHA256WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}
	oidSHA384WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 12}
	oidSHA512WithRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 13}
	oidRSASSAPSS       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 10} // id-RSASSA-PSS
	oidEd25519         = asn1.ObjectIdentifier{1, 3, 101, 112}              // id-Ed25519
	oidEcdsaWithSHA256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	oidEcdsaWithSHA384 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 3}
	oidEcdsaWithSHA512 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 4}
)

// PKIStatus values (RFC 3161 §2.4.2).
const (
	pkiStatusGranted  = 0
	pkiStatusRejected = 2
)

// ---- RFC 3161 wire structures (unexported; DER discipline is enforced by
// stdlib encoding/asn1 — it rejects indefinite lengths and trailing bytes). ----

type algorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

type messageImprint struct {
	HashAlgorithm algorithmIdentifier
	HashedMessage []byte
}

type timeStampReq struct {
	Version        int
	MessageImprint messageImprint
	ReqPolicy      asn1.ObjectIdentifier `asn1:"optional"`
	Nonce          *big.Int              `asn1:"optional"`
	CertReq        bool                  `asn1:"optional"`
	Extensions     asn1.RawValue         `asn1:"optional,tag:0"`
}

type pkiStatusInfo struct {
	Status       int
	StatusString asn1.RawValue `asn1:"optional"`
	FailInfo     asn1.RawValue `asn1:"optional"`
}

type timeStampResp struct {
	Status    pkiStatusInfo
	TimeStamp asn1.RawValue `asn1:"optional"`
}

type contentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

type encapsulatedContentInfo struct {
	EContentType asn1.ObjectIdentifier
	EContent     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

type signedData struct {
	Version          int
	DigestAlgorithms []algorithmIdentifier `asn1:"set"`
	EncapContentInfo encapsulatedContentInfo
	Certificates     []asn1.RawValue `asn1:"optional,set,tag:0"`
	CRLs             asn1.RawValue   `asn1:"optional,tag:1"`
	SignerInfos      []signerInfo    `asn1:"set"`
}

type issuerAndSerialNumber struct {
	Issuer       asn1.RawValue // RDNSequence Name
	SerialNumber *big.Int
}

type signerInfo struct {
	Version            int
	SID                asn1.RawValue // [0] IssuerAndSerialNumber or [1] SubjectKeyIdentifier
	DigestAlgorithm    algorithmIdentifier
	SignedAttrs        asn1.RawValue `asn1:"optional,tag:0"`
	SignatureAlgorithm algorithmIdentifier
	Signature          []byte
	UnsignedAttrs      asn1.RawValue `asn1:"optional,tag:1"`
}

// attribute is an ASN.1 Attribute inside the signer's signedAttrs set.
type attribute struct {
	Type   asn1.ObjectIdentifier
	Values []asn1.RawValue `asn1:"set"`
}

type accuracy struct {
	Seconds int `asn1:"optional"`
	Millis  int `asn1:"optional,tag:0"`
	Micros  int `asn1:"optional,tag:1"`
}

type tstInfo struct {
	Version        int
	Policy         asn1.ObjectIdentifier
	MessageImprint messageImprint
	SerialNumber   *big.Int
	GenTime        time.Time     `asn1:"generalized"`
	Accuracy       accuracy      `asn1:"optional"`
	Ordering       bool          `asn1:"optional"`
	Nonce          *big.Int      `asn1:"optional"`
	TSA            asn1.RawValue `asn1:"optional,tag:0"`
	Extensions     asn1.RawValue `asn1:"optional,tag:1"`
}

// Request is a parsed RFC 3161 TimeStampReq.
type Request struct {
	Version        int
	MessageImprint MessageImprint
	CertReq        bool
}

// MessageImprint is the SHA-256 digest algorithm identifier plus digest bytes.
type MessageImprint struct {
	HashAlgorithm asn1.ObjectIdentifier
	HashedMessage []byte
}

// Response is a parsed RFC 3161 TimeStampResp.
type Response struct {
	Status  int
	Token   []byte // whole TimeStampToken (ContentInfo) DER; empty unless granted
	TSTInfo []byte // TSTInfo DER; empty unless granted
	GenTime time.Time
	Imprint []byte // TSTInfo message imprint
}

// BuildRequest encodes a TimeStampReq over message using the SHA-256 imprint
// algorithm and certReq=true (RFC 3161 §2.4.1).
func BuildRequest(message []byte) ([]byte, error) {
	if len(message) == 0 {
		return nil, errors.New("timestamp message is empty")
	}
	sum := sha256.Sum256(message)
	req := timeStampReq{
		Version: 1,
		MessageImprint: messageImprint{
			HashAlgorithm: algorithmIdentifier{Algorithm: oidSHA256},
			HashedMessage: sum[:],
		},
		CertReq: true,
	}
	return asn1.Marshal(req)
}

// ParseRequest decodes a TimeStampReq and returns its structural fields.
func ParseRequest(der []byte) (Request, error) {
	var req timeStampReq
	rest, err := asn1.Unmarshal(der, &req)
	if err != nil {
		return Request{}, fmt.Errorf("timestamp request: %w", err)
	}
	if len(rest) != 0 {
		return Request{}, errors.New("timestamp request has trailing bytes")
	}
	if req.Version != 1 {
		return Request{}, fmt.Errorf("timestamp request unsupported version %d", req.Version)
	}
	if len(req.MessageImprint.HashedMessage) != sha256.Size {
		return Request{}, fmt.Errorf("timestamp request imprint must be %d bytes", sha256.Size)
	}
	return Request{
		Version: req.Version,
		MessageImprint: MessageImprint{
			HashAlgorithm: req.MessageImprint.HashAlgorithm.Algorithm,
			HashedMessage: append([]byte(nil), req.MessageImprint.HashedMessage...),
		},
		CertReq: req.CertReq,
	}, nil
}

// ParseResponse decodes a TimeStampResp. It does not chain- or signature-verify
// the token; use Verify for full acceptance. A non-granted status is returned
// without error so callers can distinguish status conditions from malformed
// input.
func ParseResponse(der []byte) (Response, error) {
	if len(der) == 0 {
		return Response{}, errors.New("empty timestamp response")
	}
	var resp timeStampResp
	rest, err := asn1.Unmarshal(der, &resp)
	if err != nil {
		return Response{}, fmt.Errorf("timestamp response: %w", err)
	}
	if len(rest) != 0 {
		return Response{}, errors.New("timestamp response has trailing bytes")
	}
	out := Response{Status: resp.Status.Status}
	if out.Status != pkiStatusGranted {
		return out, nil
	}
	tokenDER, err := normalizeToken(resp.TimeStamp)
	if err != nil {
		return Response{}, err
	}
	info, infoDER, err := parseTSTInfo(tokenDER)
	if err != nil {
		return Response{}, err
	}
	out.Token = append([]byte(nil), tokenDER...)
	out.TSTInfo = infoDER
	out.GenTime = info.GenTime
	out.Imprint = append([]byte(nil), info.MessageImprint.HashedMessage...)
	return out, nil
}

// normalizeToken extracts the ContentInfo DER of a TimeStampToken, tolerating
// the direct encoding (RFC 3161) and the octet-string wrapper some TSAs use.
func normalizeToken(raw asn1.RawValue) ([]byte, error) {
	candidates := make([][]byte, 0, 2)
	if len(raw.FullBytes) > 0 {
		candidates = append(candidates, raw.FullBytes)
	}
	if len(raw.Bytes) > 0 {
		candidates = append(candidates, raw.Bytes)
	}
	for _, candidate := range candidates {
		var ci contentInfo
		if _, err := asn1.Unmarshal(candidate, &ci); err == nil && ci.ContentType.Equal(oidSigned) {
			return append([]byte(nil), candidate...), nil
		}
	}
	return nil, errors.New("granted timestamp response has an unrecognized timestamp token")
}

func parseTSTInfo(tokenDER []byte) (tstInfo, []byte, error) {
	var ci contentInfo
	rest, err := asn1.Unmarshal(tokenDER, &ci)
	if err != nil {
		return tstInfo{}, nil, fmt.Errorf("timestamp token content info: %w", err)
	}
	if len(rest) != 0 {
		return tstInfo{}, nil, errors.New("timestamp token has trailing bytes")
	}
	if !ci.ContentType.Equal(oidSigned) {
		return tstInfo{}, nil, fmt.Errorf("timestamp token is not signedData (content type %v)", ci.ContentType)
	}
	var sd signedData
	if _, err := asn1.Unmarshal(ci.Content.Bytes, &sd); err != nil {
		return tstInfo{}, nil, fmt.Errorf("timestamp token signed data: %w", err)
	}
	if !sd.EncapContentInfo.EContentType.Equal(oidTSTInfo) {
		return tstInfo{}, nil, errors.New("signed data payload is not TSTInfo")
	}
	payload := sd.EncapContentInfo.EContent.Bytes
	if len(payload) == 0 {
		return tstInfo{}, nil, errors.New("signed data has no TSTInfo payload")
	}
	var oct asn1.RawValue
	if _, err := asn1.Unmarshal(payload, &oct); err != nil {
		return tstInfo{}, nil, fmt.Errorf("timestamp token eContent: %w", err)
	}
	if oct.Tag != asn1.TagOctetString {
		return tstInfo{}, nil, errors.New("timestamp token eContent is not an octet string")
	}
	var info tstInfo
	rest, err = asn1.Unmarshal(oct.Bytes, &info)
	if err != nil {
		return tstInfo{}, nil, fmt.Errorf("tstinfo: %w", err)
	}
	if len(rest) != 0 {
		return tstInfo{}, nil, errors.New("tstinfo has trailing bytes")
	}
	if info.Version != 1 {
		return tstInfo{}, nil, fmt.Errorf("tstinfo unsupported version %d", info.Version)
	}
	if info.GenTime.IsZero() {
		return tstInfo{}, nil, errors.New("tstinfo has no genTime")
	}
	if len(info.MessageImprint.HashedMessage) != sha256.Size {
		return tstInfo{}, nil, fmt.Errorf("tstinfo imprint must be %d bytes", sha256.Size)
	}
	return info, append([]byte(nil), oct.Bytes...), nil
}

// Verify accepts a TimeStampResp only when every fail-closed check holds:
// status granted, token message imprint matches expectedImprint exactly, the
// signer's messageDigest attribute covers the TSTInfo, the SignerInfo
// signature verifies against the signer key, and the signer certificate
// chains to the operator-provisioned roots. now is injected (never read from
// the wall clock) so tests stay deterministic; the caller passes the
// authoritative issuance moment.
func Verify(roots *x509.CertPool, respDER, expectedImprint []byte, now time.Time) (time.Time, error) {
	if roots == nil {
		return time.Time{}, errors.New("tsa roots are not configured")
	}
	if len(expectedImprint) != sha256.Size {
		return time.Time{}, fmt.Errorf("expected imprint must be %d bytes", sha256.Size)
	}
	if now.IsZero() {
		return time.Time{}, errors.New("verification time is required")
	}
	parsed, err := ParseResponse(respDER)
	if err != nil {
		return time.Time{}, err
	}
	if parsed.Status != pkiStatusGranted {
		return time.Time{}, fmt.Errorf("timestamp status %d is not granted", parsed.Status)
	}
	if len(parsed.Token) == 0 {
		return time.Time{}, errors.New("granted timestamp response has no token")
	}
	if !bytes.Equal(parsed.Imprint, expectedImprint) {
		return time.Time{}, errors.New("timestamp message imprint mismatch")
	}
	if err := verifySignerChain(parsed.Token, parsed.TSTInfo, roots, now); err != nil {
		return time.Time{}, err
	}
	return parsed.GenTime, nil
}

func verifySignerChain(tokenDER, tstInfoDER []byte, roots *x509.CertPool, now time.Time) error {
	certs, signer, err := parseSigner(tokenDER)
	if err != nil {
		return err
	}
	leaf, err := selectSignerCertificate(certs, signer)
	if err != nil {
		return err
	}
	if err := verifySignature(signer, leaf, tstInfoDER); err != nil {
		return err
	}
	intermediates := x509.NewCertPool()
	for _, cert := range certs {
		if cert.Equal(leaf) {
			continue
		}
		intermediates.AddCert(cert)
	}
	chains, err := leaf.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   now,
	})
	if err != nil {
		return fmt.Errorf("timestamp signer chain does not verify against provisioned tsa roots: %w", err)
	}
	if len(chains) == 0 {
		return errors.New("timestamp signer certificate chains to no provisioned tsa root")
	}
	return nil
}

func parseSigner(tokenDER []byte) ([]*x509.Certificate, signerInfo, error) {
	var ci contentInfo
	if _, err := asn1.Unmarshal(tokenDER, &ci); err != nil {
		return nil, signerInfo{}, fmt.Errorf("timestamp token: %w", err)
	}
	var sd signedData
	if _, err := asn1.Unmarshal(ci.Content.Bytes, &sd); err != nil {
		return nil, signerInfo{}, fmt.Errorf("timestamp token signed data: %w", err)
	}
	if len(sd.SignerInfos) == 0 {
		return nil, signerInfo{}, errors.New("timestamp token has no signer")
	}
	certs := make([]*x509.Certificate, 0, len(sd.Certificates))
	for _, raw := range sd.Certificates {
		certDER := raw.FullBytes
		if len(certDER) == 0 {
			certDER = raw.Bytes
		}
		cert, err := x509.ParseCertificate(certDER)
		if err != nil {
			return nil, signerInfo{}, fmt.Errorf("timestamp token certificate: %w", err)
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, signerInfo{}, errors.New("timestamp token carries no certificates")
	}
	return certs, sd.SignerInfos[0], nil
}

// selectSignerCertificate locates the signer certificate via the SignerInfo
// identifier (issuer+serial, or subject key id). When the identifier is
// absent and the token carries a single certificate, that certificate is used.
func selectSignerCertificate(certs []*x509.Certificate, signer signerInfo) (*x509.Certificate, error) {
	if signer.SID.Class == asn1.ClassContextSpecific && len(signer.SID.Bytes) > 0 {
		switch signer.SID.Tag {
		case 0:
			var ias issuerAndSerialNumber
			if _, err := asn1.Unmarshal(signer.SID.Bytes, &ias); err != nil {
				if len(certs) == 1 {
					return certs[0], nil
				}
				return nil, fmt.Errorf("signer identifier: %w", err)
			}
			for _, cert := range certs {
				issuerDER := ias.Issuer.FullBytes
				if len(issuerDER) == 0 {
					issuerDER = ias.Issuer.Bytes
				}
				if cert.SerialNumber.Cmp(ias.SerialNumber) == 0 && bytes.Equal(cert.RawIssuer, issuerDER) {
					return cert, nil
				}
			}
			return nil, errors.New("no token certificate matches the signer issuer and serial number")
		case 1:
			sid := signer.SID.Bytes
			for _, cert := range certs {
				if bytes.Equal(cert.SubjectKeyId, sid) {
					return cert, nil
				}
			}
			return nil, errors.New("no token certificate matches the signer subject key identifier")
		}
	}
	if len(certs) == 1 {
		return certs[0], nil
	}
	return nil, errors.New("cannot determine the timestamp signer certificate")
}

type signatureKind int

const (
	sigKindRSA signatureKind = iota
	sigKindECDSA
)

var signatureAlgorithms = map[string]struct {
	hash crypto.Hash
	kind signatureKind
}{
	oidSHA256WithRSA.String():   {hash: crypto.SHA256, kind: sigKindRSA},
	oidSHA384WithRSA.String():   {hash: crypto.SHA384, kind: sigKindRSA},
	oidSHA512WithRSA.String():   {hash: crypto.SHA512, kind: sigKindRSA},
	oidEcdsaWithSHA256.String(): {hash: crypto.SHA256, kind: sigKindECDSA},
	oidEcdsaWithSHA384.String(): {hash: crypto.SHA384, kind: sigKindECDSA},
	oidEcdsaWithSHA512.String(): {hash: crypto.SHA512, kind: sigKindECDSA},
}

var digestAlgorithms = map[string]crypto.Hash{
	oidSHA256.String(): crypto.SHA256,
	oidSHA384.String(): crypto.SHA384,
	oidSHA512.String(): crypto.SHA512,
}

// verifySignature enforces the CMS signing over the signer's signedAttrs
// (RFC 5652 §5.4). Any structural inconsistency or a failing signature check
// fails closed. RSA PKCS#1 v1.5, RSASSA-PSS and ECDSA are keyed on the
// SignatureAlgorithm OID (PSS hash comes from the SignerInfo DigestAlgorithm).
// SHA-1 is rejected in all forms.
func verifySignature(si signerInfo, leaf *x509.Certificate, tstInfoDER []byte) error {
	signedAttrs, err := normalizeSignedAttrs(si.SignedAttrs)
	if err != nil {
		return err
	}
	digest, err := signedMessageDigest(signedAttrs)
	if err != nil {
		return err
	}
	infoDigest := sha256.Sum256(tstInfoDER)
	if !bytes.Equal(digest, infoDigest[:]) {
		return errors.New("signer messageDigest attribute does not match the TSTInfo content")
	}
	if si.DigestAlgorithm.Algorithm.Equal(oidSHA1) || si.SignatureAlgorithm.Algorithm.Equal(oidSHA1WithRSA) {
		return errors.New("SHA-1 digest or signature algorithm is rejected for timestamps (fail closed)")
	}
	return verifySignedAttrsSignature(si, leaf, signedAttrs)
}

// verifySignedAttrsSignature verifies the SignerInfo signature over the
// DER-encoded signedAttrs against the signer certificate key, dispatching on
// the SignatureAlgorithm OID. Unknown OIDs and SHA-1 fail closed.
func verifySignedAttrsSignature(si signerInfo, leaf *x509.Certificate, signedAttrs []byte) error {
	sigAlg := si.SignatureAlgorithm.Algorithm
	digestAlg, ok := digestAlgorithms[si.DigestAlgorithm.Algorithm.String()]
	if !ok {
		return fmt.Errorf("unsupported timestamp digest algorithm %s (fail closed)", si.DigestAlgorithm.Algorithm)
	}
	if !digestAlg.Available() {
		return errors.New("timestamp signing hash is not linked (fail closed)")
	}

	switch {
	case sigAlg.Equal(oidRSASSAPSS):
		pub, ok := leaf.PublicKey.(*rsa.PublicKey)
		if !ok {
			return errors.New("signer certificate key is not rsa for RSASSA-PSS signature")
		}
		h := digestAlg.New()
		h.Write(signedAttrs)
		signedDigest := h.Sum(nil)
		if err := rsa.VerifyPSS(pub, digestAlg, signedDigest, si.Signature, nil); err != nil {
			return fmt.Errorf("timestamp RSASSA-PSS signature verification failed: %w", err)
		}
		return nil

	case sigAlg.Equal(oidEd25519):
		pub, ok := leaf.PublicKey.(ed25519.PublicKey)
		if !ok {
			return errors.New("signer certificate key is not Ed25519 although the signature claims it")
		}
		if !ed25519.Verify(pub, signedAttrs, si.Signature) {
			return errors.New("Ed25519 timestamp signature verification failed")
		}
		return nil
	}

	params, ok := signatureAlgorithms[sigAlg.String()]
	if !ok {
		return fmt.Errorf("unsupported timestamp signature algorithm %s (fail closed)", sigAlg)
	}
	if !params.hash.Available() {
		return errors.New("timestamp signing hash is not linked (fail closed)")
	}
	h := digestAlg.New()
	h.Write(signedAttrs)
	signedDigest := h.Sum(nil)
	switch params.kind {
	case sigKindRSA:
		pub, ok := leaf.PublicKey.(*rsa.PublicKey)
		if !ok {
			return errors.New("signer certificate key is not rsa although the signature claims it")
		}
		if err := rsa.VerifyPKCS1v15(pub, params.hash, signedDigest, si.Signature); err != nil {
			return fmt.Errorf("timestamp rsa signature verification failed: %w", err)
		}
	case sigKindECDSA:
		pub, ok := leaf.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return errors.New("signer certificate key is not ecdsa although the signature claims it")
		}
		if !ecdsa.VerifyASN1(pub, signedDigest, si.Signature) {
			return errors.New("timestamp ecdsa signature verification failed")
		}
	default:
		return errors.New("invalid timestamp signature algorithm kind")
	}
	return nil
}

// normalizeSignedAttrs recovers the DER encoding that the signature covers:
// per RFC 5652 §5.4 the [0] IMPLICIT tag that wraps signedAttrs in SignerInfo
// is replaced with the explicit SET OF tag (identical content octets, so only
// the leading tag byte changes). An encoding that is neither [0] IMPLICIT nor
// already a SET fails closed.
func normalizeSignedAttrs(attrs asn1.RawValue) ([]byte, error) {
	der := attrs.FullBytes
	if len(der) == 0 {
		der = attrs.Bytes
	}
	if len(der) == 0 {
		return nil, errors.New("signer has no signed attributes (fail closed)")
	}
	switch der[0] {
	case 0x31:
		return der, nil
	case 0xA0:
		out := make([]byte, len(der))
		out[0] = 0x31
		copy(out[1:], der[1:])
		return out, nil
	default:
		return nil, errors.New("signer signed attributes have an unexpected encoding")
	}
}

// signedMessageDigest extracts the messageDigest attribute value from the DER
// signedAttrs SET OF Attribute, failing closed on any ambiguity (missing or
// duplicate attribute, malformed value).
func signedMessageDigest(setDER []byte) ([]byte, error) {
	attrs, err := parseAttributes(setDER)
	if err != nil {
		return nil, err
	}
	var digest []byte
	for _, attr := range attrs {
		if !attr.Type.Equal(oidMessageDigest) {
			continue
		}
		if digest != nil {
			return nil, errors.New("duplicate messageDigest attribute")
		}
		if len(attr.Values) != 1 {
			return nil, errors.New("messageDigest attribute must have exactly one value")
		}
		value := attr.Values[0]
		der := value.FullBytes
		if len(der) == 0 {
			if len(value.Bytes) == sha256.Size {
				return value.Bytes, nil
			}
			return nil, errors.New("messageDigest attribute value is malformed")
		}
		var oct asn1.RawValue
		if _, err := asn1.Unmarshal(der, &oct); err != nil {
			return nil, fmt.Errorf("messageDigest attribute value: %w", err)
		}
		if oct.Tag != asn1.TagOctetString {
			return nil, errors.New("messageDigest attribute value is not an octet string")
		}
		if len(oct.Bytes) != sha256.Size {
			return nil, fmt.Errorf("messageDigest attribute must be %d bytes", sha256.Size)
		}
		digest = append([]byte(nil), oct.Bytes...)
	}
	if digest == nil {
		return nil, errors.New("signed attributes are missing the messageDigest attribute")
	}
	return digest, nil
}

// parseAttributes unmarshals the content of a DER SET OF Attribute, walking
// the elements manually so single-element sets parse deterministically.
func parseAttributes(setDER []byte) ([]attribute, error) {
	_, content, rest, err := splitTLV(setDER)
	if err != nil {
		return nil, fmt.Errorf("signed attributes: %w", err)
	}
	if len(rest) != 0 {
		return nil, errors.New("signed attributes have trailing bytes")
	}
	var attrs []attribute
	for len(content) > 0 {
		var elem []byte
		elem, content, err = nextTLV(content)
		if err != nil {
			return nil, fmt.Errorf("signed attribute: %w", err)
		}
		var attr attribute
		if _, err := asn1.Unmarshal(elem, &attr); err != nil {
			return nil, fmt.Errorf("signed attribute: %w", err)
		}
		attrs = append(attrs, attr)
	}
	return attrs, nil
}

// nextTLV returns the complete DER encoding of the next element (tag, length
// and content) plus the remaining bytes after it.
func nextTLV(b []byte) (tlv, rest []byte, err error) {
	_, content, after, err := splitTLV(b)
	if err != nil {
		return nil, nil, err
	}
	headerLen := len(b) - len(content) - len(after)
	return b[:headerLen+len(content)], after, nil
}

// splitTLV parses one definite-length DER TLV, returning the tag byte, content
// octets and the remaining bytes. Indefinite lengths and over-long lengths are
// rejected so no allocation or scan is ever driven by attacker-controlled
// lengths.
func splitTLV(b []byte) (tag byte, content, rest []byte, err error) {
	if len(b) < 2 {
		return 0, nil, nil, errors.New("tlv is too short")
	}
	tag = b[0]
	idx := 1
	lenByte := int(b[idx])
	idx++
	length := 0
	if lenByte&0x80 != 0 {
		num := lenByte & 0x7f
		if num == 0 {
			return 0, nil, nil, errors.New("indefinite tlv length is not allowed")
		}
		if num > 4 || len(b) < idx+num {
			return 0, nil, nil, errors.New("tlv length is malformed")
		}
		length = int(b[idx])
		for i := 1; i < num; i++ {
			length = length<<8 | int(b[idx+i])
		}
		idx += num
	} else {
		length = lenByte
	}
	if length < 0 || len(b) < idx+length {
		return 0, nil, nil, errors.New("tlv is truncated")
	}
	return tag, b[idx : idx+length], b[idx+length:], nil
}

// ---- off-line token construction helpers (shared by BuildTestResponse and
// the forgery/tamper tests so each layer is exercised through the same wire
// builder). ----

func buildTSTInfoDER(imprint []byte, genTime time.Time) ([]byte, error) {
	if len(imprint) != sha256.Size {
		return nil, errors.New("imprint must be 32 bytes")
	}
	if genTime.IsZero() {
		return nil, errors.New("genTime is required")
	}
	info := tstInfo{
		Version: 1,
		Policy:  asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 54321, 1},
		MessageImprint: messageImprint{
			HashAlgorithm: algorithmIdentifier{Algorithm: oidSHA256},
			HashedMessage: append([]byte(nil), imprint...),
		},
		SerialNumber: big.NewInt(1),
		GenTime:      genTime.UTC(),
		Accuracy:     accuracy{Seconds: 1},
	}
	return asn1.Marshal(info)
}

func buildSignedAttrsDER(messageDigest []byte) ([]byte, error) {
	contentType, err := asn1.Marshal(oidTSTInfo)
	if err != nil {
		return nil, err
	}
	attrs := []attribute{
		{Type: oidContentType, Values: []asn1.RawValue{{FullBytes: contentType}}},
		{Type: oidMessageDigest, Values: []asn1.RawValue{{FullBytes: marshalTLV(asn1.TagOctetString, messageDigest)}}},
	}
	der, err := asn1.Marshal(attrs)
	if err != nil {
		return nil, err
	}
	if len(der) == 0 || der[0] != 0x30 {
		return nil, errors.New("signed attributes did not marshal as a sequence")
	}
	// asn1.Marshal of a bare slice emits a SEQUENCE; signedAttrs is a SET OF
	// Attribute, whose content octets are identical, so only the outer tag
	// changes to the universal SET tag.
	der[0] = 0x31
	return der, nil
}

func newSignerInfo(signatureAlgorithm asn1.ObjectIdentifier, signature, signedAttrsDER []byte, cert *x509.Certificate) (signerInfo, error) {
	return newSignerInfoWithDigest(signatureAlgorithm, signature, signedAttrsDER, cert, oidSHA256)
}

func newSignerInfoWithDigest(signatureAlgorithm asn1.ObjectIdentifier, signature, signedAttrsDER []byte, cert *x509.Certificate, digestAlgID asn1.ObjectIdentifier) (signerInfo, error) {
	if len(signedAttrsDER) == 0 {
		return signerInfo{}, errors.New("signed attributes are required")
	}
	setTag, setContent, rest, err := splitTLV(signedAttrsDER)
	if err != nil {
		return signerInfo{}, err
	}
	if setTag != 0x31 || len(rest) != 0 {
		return signerInfo{}, errors.New("signed attributes must be a single DER SET OF")
	}
	ias := issuerAndSerialNumber{Issuer: asn1.RawValue{FullBytes: cert.RawIssuer}, SerialNumber: cert.SerialNumber}
	iasDER, err := asn1.Marshal(ias)
	if err != nil {
		return signerInfo{}, err
	}
	return signerInfo{
		Version:            1,
		SID:                asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: iasDER},
		DigestAlgorithm:    algorithmIdentifier{Algorithm: digestAlgID},
		SignedAttrs:        asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: setContent},
		SignatureAlgorithm: algorithmIdentifier{Algorithm: signatureAlgorithm},
		Signature:          signature,
	}, nil
}

func buildTokenDER(infoDER []byte, cert *x509.Certificate, si signerInfo) ([]byte, error) {
	oct := marshalTLV(asn1.TagOctetString, infoDER)
	eContent := asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, FullBytes: marshalTLV(0xA0, oct)}
	envelope := signedData{
		Version:          1,
		DigestAlgorithms: []algorithmIdentifier{{Algorithm: oidSHA256}},
		EncapContentInfo: encapsulatedContentInfo{EContentType: oidTSTInfo, EContent: eContent},
		Certificates:     []asn1.RawValue{{FullBytes: cert.Raw}},
		SignerInfos:      []signerInfo{si},
	}
	envelopeDER, err := asn1.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("signedData encode: %w", err)
	}
	ci := contentInfo{
		ContentType: oidSigned,
		Content:     asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, FullBytes: marshalTLV(0xA0, envelopeDER)},
	}
	return asn1.Marshal(ci)
}
