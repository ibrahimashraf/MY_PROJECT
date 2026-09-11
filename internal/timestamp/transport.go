package timestamp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// TSP wire constants.
const (
	contentTypeQuery = "application/timestamp-query"
	contentTypeReply = "application/timestamp-reply"

	tsaHTTPTimeout  = 15 * time.Second
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
		return nil, fmt.Errorf("tsa returned http status %d", resp.StatusCode)
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
