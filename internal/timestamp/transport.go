package timestamp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func tsaHTTPTimeoutFromEnv() time.Duration {
	if v := strings.TrimSpace(os.Getenv("INTEGIN_TSA_TIMEOUT_SECONDS")); v != "" {
		if s, err := strconv.Atoi(v); err == nil && s > 0 && s <= 120 {
			return time.Duration(s) * time.Second
		}
	}
	return 15 * time.Second
}

var tsaHTTPTimeout = tsaHTTPTimeoutFromEnv()

// TSP wire constants.
const (
	contentTypeQuery = "application/timestamp-query"
	contentTypeReply = "application/timestamp-reply"

	tsaMaxBodyBytes = 1 << 20
)

// Env vars provisioning the timestamp authority at boot (mirrors the
// onboarding roots pattern: the operator supplies the roots file; nothing is
// embedded from memory).
const (
	envTSAURL       = "INTEGIN_TSA_URL"
	envTSARootsFile = "INTEGIN_TSA_ROOTS_FILE"
)

// Transport sends an encoded TimeStampReq and returns the raw TimeStampResp
// bytes. This is the injectable seam: tests fake it, production wiring uses
// HTTPTransport.
type Transport interface {
	RoundTrip(ctx context.Context, request []byte) ([]byte, error)
}

// HTTPTransport is the production RFC 3161 transport (POST
// application/timestamp-query over HTTP).
type HTTPTransport struct {
	URL    string
	Client *http.Client
}

func (t *HTTPTransport) RoundTrip(ctx context.Context, request []byte) ([]byte, error) {
	url := strings.TrimSpace(t.URL)
	if url == "" {
		return nil, errors.New("tsa url is not configured")
	}
	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(request))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentTypeQuery)
	req.Header.Set("Accept", contentTypeReply)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tsa request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{Code: resp.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, tsaMaxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("tsa response read failed: %w", err)
	}
	if len(body) > tsaMaxBodyBytes {
		return nil, errors.New("tsa response exceeds the size limit")
	}
	return body, nil
}

// StatusError reports a non-2xx RFC 3161 HTTP response. It is typed so the
// store-and-forward fallback can classify 5xx responses as transient.
type StatusError struct {
	Code int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("tsa returned http status %d", e.Code)
}

// IsTransient classifies a timestamp round-trip error as retry-safe: transport
// timeouts, other net-level failures, and 5xx responses qualify; 4xx
// rejections and verification failures do not.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr.Code >= 500
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return false
}

// TSA is a provisioned timestamp authority: a signed transport plus the
// trusted roots used to verify tokens at call time.
type TSA struct {
	transport Transport
	roots     *x509.CertPool
}

// New assembles a TSA. A nil transport or roots is a provisioning error
// (fails closed at boot, mirroring the onboarding posture).
func New(transport Transport, roots *x509.CertPool) (*TSA, error) {
	if transport == nil {
		return nil, errors.New("timestamp transport is required")
	}
	if roots == nil {
		return nil, errors.New("timestamp authority roots are required")
	}
	return &TSA{transport: transport, roots: roots}, nil
}

// Token is the verified issuance stamp: the raw TimeStampResp DER plus the
// TSA's genTime.
type Token struct {
	Response []byte
	GenTime  time.Time
}

// Timestamp builds a request over message, round-trips it through the
// transport, and returns the verified token. now (injected, never the wall
// clock) anchors the signer-certificate validity check.
func (t *TSA) Timestamp(ctx context.Context, message []byte, now time.Time) (Token, error) {
	req, err := BuildRequest(message)
	if err != nil {
		return Token{}, err
	}
	resp, err := t.transport.RoundTrip(ctx, req)
	if err != nil {
		return Token{}, err
	}
	sum := sha256.Sum256(message)
	genTime, err := Verify(t.roots, resp, sum[:], now)
	if err != nil {
		return Token{}, err
	}
	return Token{Response: append([]byte(nil), resp...), GenTime: genTime}, nil
}

// LoadRoots reads a PEM file of trusted TSA root certificates.
func LoadRoots(pemPath string) (*x509.CertPool, error) {
	data, err := os.ReadFile(pemPath)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, errors.New("no certificates found in tsa roots file")
	}
	return pool, nil
}

// ProvisionFromEnv loads the TSA configuration at boot from:
//
//	INTEGIN_TSA_URL       RFC 3161 endpoint.
//	INTEGIN_TSA_ROOTS_FILE PEM file of trusted TSA roots.
//
// Both must be set together (all-or-nothing, same as the Apple App Attest
// provisioning) or both unset. Unset keeps the package unconfigured posture:
// callers fall back to legacy behavior. A set-but-invalid variable fails boot.
func ProvisionFromEnv() (*TSA, error) {
	url := strings.TrimSpace(os.Getenv(envTSAURL))
	rootsFile := strings.TrimSpace(os.Getenv(envTSARootsFile))
	if url == "" && rootsFile == "" {
		return nil, nil
	}
	if url == "" || rootsFile == "" {
		return nil, fmt.Errorf("%s and %s must be set together (tsa provisioning is all-or-nothing)", envTSAURL, envTSARootsFile)
	}
	roots, err := LoadRoots(rootsFile)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", envTSARootsFile, err)
	}
	transport := &HTTPTransport{URL: url, Client: &http.Client{Timeout: tsaHTTPTimeout}}
	return New(transport, roots)
}

// BuildTestResponse constructs a TimeStampResp for an offline fake TSA: the
// token carries the given imprint and genTime, its signer identity is the
// provided certificate (which must also be provisioned as a root for Verify)
// and the SignerInfo is signed for real with the provided private key (RSA or
// ECDSA). Because Verify now enforces the CMS signature, this helper stays a
// faithful off-line stand-in for a trusted authority — but it is only a test
// and simulation helper, never a substitute for a real TSA operator.
func BuildTestResponse(signer *x509.Certificate, signerKey any, imprint []byte, genTime time.Time) ([]byte, error) {
	return buildResponse(signer, signerKey, imprint, genTime, big.NewInt(1))
}

// BuildResponse assembles a granted TimeStampResp for a real TSA responder:
// like BuildTestResponse but with a caller-supplied serial number, which must
// be positive and unique per token (RFC 3161 section 2.4.2). The signer
// certificate must chain to the roots provisioned alongside the TSA URL for
// Verify to accept the token.
func BuildResponse(signer *x509.Certificate, signerKey any, imprint []byte, genTime time.Time, serial *big.Int) ([]byte, error) {
	if serial == nil || serial.Sign() <= 0 {
		return nil, errors.New("timestamp serial must be a positive number")
	}
	return buildResponse(signer, signerKey, imprint, genTime, serial)
}

// RandomSerial returns a 128-bit positive serial number for BuildResponse.
func RandomSerial() (*big.Int, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("timestamp serial: %w", err)
	}
	return serial.Add(serial, big.NewInt(1)), nil
}

func buildResponse(signer *x509.Certificate, signerKey any, imprint []byte, genTime time.Time, serial *big.Int) ([]byte, error) {
	if signer == nil {
		return nil, errors.New("signer certificate is required")
	}
	if signerKey == nil {
		return nil, errors.New("signer private key is required")
	}
	infoDER, err := buildTSTInfoDERWithSerial(imprint, genTime, serial)
	if err != nil {
		return nil, err
	}
	infoDigest := sha256.Sum256(infoDER)
	attrs, err := buildSignedAttrsDER(infoDigest[:])
	if err != nil {
		return nil, err
	}
	signature, sigAlg, err := signSignedAttrs(signerKey, attrs)
	if err != nil {
		return nil, err
	}
	si, err := newSignerInfo(sigAlg, signature, attrs, signer)
	if err != nil {
		return nil, err
	}
	tokenDER, err := buildTokenDER(infoDER, signer, si)
	if err != nil {
		return nil, err
	}
	resp := timeStampResp{Status: pkiStatusInfo{Status: pkiStatusGranted}, TimeStamp: asn1.RawValue{FullBytes: tokenDER}}
	respDER, err := asn1.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("timestamp response encode: %w", err)
	}
	return respDER, nil
}

// signSignedAttrs produces the SignerInfo signature over the DER-signed
// signedAttrs byte string using the matching algorithm for the key type.
func signSignedAttrs(signerKey any, signedAttrsDER []byte) ([]byte, asn1.ObjectIdentifier, error) {
	digest := sha256.Sum256(signedAttrsDER)
	switch key := signerKey.(type) {
	case *rsa.PrivateKey:
		signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if err != nil {
			return nil, nil, err
		}
		return signature, oidSHA256WithRSA, nil
	case *ecdsa.PrivateKey:
		signature, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
		if err != nil {
			return nil, nil, err
		}
		return signature, oidEcdsaWithSHA256, nil
	default:
		return nil, nil, errors.New("unsupported timestamp signer key (rsa or ecdsa required)")
	}
}

// signSignedAttrsPSS produces an RSASSA-PSS SignerInfo signature over the
// DER-signed signedAttrs using the given hash. Test and simulation use only.
func signSignedAttrsPSS(key *rsa.PrivateKey, hash crypto.Hash, signedAttrsDER []byte) ([]byte, asn1.ObjectIdentifier, error) {
	if !hash.Available() {
		return nil, nil, fmt.Errorf("hash %v is not linked (fail closed)", hash)
	}
	h := hash.New()
	h.Write(signedAttrsDER)
	signature, err := rsa.SignPSS(rand.Reader, key, hash, h.Sum(nil), nil)
	if err != nil {
		return nil, nil, err
	}
	return signature, oidRSASSAPSS, nil
}

// signSignedAttrsEd25519 produces an Ed25519 SignerInfo signature over the
// DER-signed signedAttrs. Test and simulation use only.
func signSignedAttrsEd25519(key ed25519.PrivateKey, signedAttrsDER []byte) ([]byte, asn1.ObjectIdentifier, error) {
	signature := ed25519.Sign(key, signedAttrsDER)
	return signature, oidEd25519, nil
}

// BuildTestResponsePSS constructs a TimeStampResp whose SignerInfo uses an
// RSASSA-PSS signature over the signed attrs with the given hash. The digest
// field maps hash to its signing OID (SHA-1 rejected).
func BuildTestResponsePSS(signer *x509.Certificate, signerKey *rsa.PrivateKey, hash crypto.Hash, imprint []byte, genTime time.Time) ([]byte, error) {
	if signer == nil {
		return nil, errors.New("signer certificate is required")
	}
	if signerKey == nil {
		return nil, errors.New("signer private key is required")
	}
	infoDER, err := buildTSTInfoDER(imprint, genTime)
	if err != nil {
		return nil, err
	}
	infoDigest := sha256.Sum256(infoDER)
	attrs, err := buildSignedAttrsDER(infoDigest[:])
	if err != nil {
		return nil, err
	}
	digestOID, ok := digestAlgorithmOIDs[hash]
	if !ok {
		return nil, fmt.Errorf("unsupported PSS hash %v (fail closed)", hash)
	}
	signature, sigAlg, err := signSignedAttrsPSS(signerKey, hash, attrs)
	if err != nil {
		return nil, err
	}
	si, err := newSignerInfoWithDigest(sigAlg, signature, attrs, signer, digestOID)
	if err != nil {
		return nil, err
	}
	tokenDER, err := buildTokenDER(infoDER, signer, si)
	if err != nil {
		return nil, err
	}
	resp := timeStampResp{Status: pkiStatusInfo{Status: pkiStatusGranted}, TimeStamp: asn1.RawValue{FullBytes: tokenDER}}
	return asn1.Marshal(resp)
}

// BuildTestResponseEd25519 constructs a TimeStampResp whose SignerInfo uses an
// Ed25519 signature over the signed attrs. Test and simulation use only.
func BuildTestResponseEd25519(signer *x509.Certificate, signerKey ed25519.PrivateKey, imprint []byte, genTime time.Time) ([]byte, error) {
	if signer == nil {
		return nil, errors.New("signer certificate is required")
	}
	if signerKey == nil {
		return nil, errors.New("signer private key is required")
	}
	infoDER, err := buildTSTInfoDER(imprint, genTime)
	if err != nil {
		return nil, err
	}
	infoDigest := sha256.Sum256(infoDER)
	attrs, err := buildSignedAttrsDER(infoDigest[:])
	if err != nil {
		return nil, err
	}
	signature, sigAlg, err := signSignedAttrsEd25519(signerKey, attrs)
	if err != nil {
		return nil, err
	}
	si, err := newSignerInfoWithDigest(sigAlg, signature, attrs, signer, oidSHA512)
	if err != nil {
		return nil, err
	}
	tokenDER, err := buildTokenDER(infoDER, signer, si)
	if err != nil {
		return nil, err
	}
	resp := timeStampResp{Status: pkiStatusInfo{Status: pkiStatusGranted}, TimeStamp: asn1.RawValue{FullBytes: tokenDER}}
	return asn1.Marshal(resp)
}

// digestAlgorithmOIDs maps crypto.Hash to its digest AlgorithmIdentifier OID
// (test fixture helpers only). SHA-1 maps to the rejected OID so tests can
// prove it fails closed.
var digestAlgorithmOIDs = map[crypto.Hash]asn1.ObjectIdentifier{
	crypto.SHA1:   oidSHA1,
	crypto.SHA256: oidSHA256,
	crypto.SHA384: oidSHA384,
	crypto.SHA512: oidSHA512,
}

// marshalTLV builds a single definite-length DER TLV (generator-side helper;
// encoding/asn1 enforces the same definite-length discipline on the read path).
func marshalTLV(tag byte, content []byte) []byte {
	out := []byte{tag}
	switch {
	case len(content) < 128:
		out = append(out, byte(len(content)))
	case len(content) <= 0xFF:
		out = append(out, 0x81, byte(len(content)))
	case len(content) <= 0xFFFF:
		out = append(out, 0x82, byte(len(content)>>8), byte(len(content)))
	default:
		out = append(out, 0x83, byte(len(content)>>16), byte(len(content)>>8), byte(len(content)))
	}
	return append(out, content...)
}
