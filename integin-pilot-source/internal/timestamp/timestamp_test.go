package timestamp

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	testMessage = "integin-CERT-SNAPSHOT"
	testNow     = "2026-06-01T12:00:00Z"
)

var testGenTime = time.Date(2026, 1, 15, 9, 30, 0, 0, time.UTC)

// goldenMS is the SHA-256 of the fixed test message, as the golden bytes.
var goldenMS = mustHash(testMessage)

func mustHash(msg string) []byte {
	sum := sha256.Sum256([]byte(msg))
	return sum[:]
}

func marshalRejectedResponse() ([]byte, error) {
	rejected := timeStampResp{Status: pkiStatusInfo{Status: pkiStatusRejected}}
	return asn1.Marshal(rejected)
}

func marshalCertPEM(der []byte) ([]byte, error) {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), nil
}

// testTSARoot is a self-signed TSA CA valid across the fixed verification
// time. Serials and keys are fixed so vectors are stable.
func testTSARoot(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "integin Pilot TSA Test Root"},
		NotBefore:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:              time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		SubjectKeyId:          []byte{0x0f, 0x0e, 0x0d, 0x0c, 0x0b},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert, key
}

// testTSARootEd25519 is a self-signed TSA CA with an Ed25519 key, used to
// prove Ed25519 SignerInfo signatures verify.
func testTSARootEd25519(t *testing.T) (*x509.Certificate, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "integin Pilot TSA Test Root Ed25519"},
		NotBefore:             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:              time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		SubjectKeyId:          []byte{0x0f, 0x0e, 0x0d, 0x0c, 0x0b},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, priv)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert, priv
}

func fixedNow(t *testing.T) time.Time {
	t.Helper()
	now, err := time.Parse(time.RFC3339, testNow)
	if err != nil {
		t.Fatal(err)
	}
	return now
}

func poolFor(t *testing.T, certs ...*x509.Certificate) *x509.CertPool {
	t.Helper()
	pool := x509.NewCertPool()
	for _, cert := range certs {
		pool.AddCert(cert)
	}
	return pool
}

func TestBuildRequestParseRequestRoundTrip(t *testing.T) {
	der, err := BuildRequest([]byte(testMessage))
	if err != nil {
		t.Fatal(err)
	}
	req, err := ParseRequest(der)
	if err != nil {
		t.Fatal(err)
	}
	if req.Version != 1 {
		t.Fatalf("version = %d, want 1", req.Version)
	}
	if !req.MessageImprint.HashAlgorithm.Equal(oidSHA256) {
		t.Fatalf("hash algorithm = %v, want sha256", req.MessageImprint.HashAlgorithm)
	}
	if req.MessageImprint.HashedMessage == nil || hex.EncodeToString(req.MessageImprint.HashedMessage) != hex.EncodeToString(goldenMS) {
		t.Fatalf("imprint = %x, want %x", req.MessageImprint.HashedMessage, goldenMS)
	}
	if !req.CertReq {
		t.Fatal("certReq = false, want true")
	}
}

func TestBuildTestResponseParseAndVerifyRoundTrip(t *testing.T) {
	cert, key := testTSARoot(t)
	resp, err := BuildTestResponse(cert, key, goldenMS, testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Status != pkiStatusGranted {
		t.Fatalf("status = %d, want %d", parsed.Status, pkiStatusGranted)
	}
	if !parsed.GenTime.Equal(testGenTime) {
		t.Fatalf("genTime = %v, want %v", parsed.GenTime, testGenTime)
	}
	if hex.EncodeToString(parsed.Imprint) != hex.EncodeToString(goldenMS) {
		t.Fatalf("imprint = %x, want %x", parsed.Imprint, goldenMS)
	}
	now := fixedNow(t)
	genTime, err := Verify(poolFor(t, cert), resp, goldenMS, now)
	if err != nil {
		t.Fatal(err)
	}
	if !genTime.Equal(testGenTime) {
		t.Fatalf("verified genTime = %v, want %v", genTime, testGenTime)
	}
}

func TestVerifyRootlessRejects(t *testing.T) {
	cert, key := testTSARoot(t)
	resp, err := BuildTestResponse(cert, key, goldenMS, testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(x509.NewCertPool(), resp, goldenMS, fixedNow(t)); err == nil {
		t.Fatal("verify succeeded without provisioned roots, want failure")
	}
}

func TestVerifyDifferentRootRejects(t *testing.T) {
	cert, key := testTSARoot(t)
	other, _ := testTSARoot(t)
	resp, err := BuildTestResponse(cert, key, goldenMS, testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(poolFor(t, other), resp, goldenMS, fixedNow(t)); err == nil {
		t.Fatal("verify succeeded against a different authority's roots, want failure")
	}
}

func TestVerifyImprintTamperRejects(t *testing.T) {
	cert, key := testTSARoot(t)
	tampered := make([]byte, 32)
	copy(tampered, goldenMS)
	tampered[0] ^= 0xff
	resp, err := BuildTestResponse(cert, key, tampered, testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(poolFor(t, cert), resp, goldenMS, fixedNow(t)); err == nil {
		t.Fatal("verify accepted an imprint that does not match the message, want failure")
	}
}

func TestVerifyZeroTimeRejects(t *testing.T) {
	cert, key := testTSARoot(t)
	resp, err := BuildTestResponse(cert, key, goldenMS, testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(poolFor(t, cert), resp, goldenMS, time.Time{}); err == nil {
		t.Fatal("verify accepted a zero verification time, want failure")
	}
}

func TestVerifyRejectedStatus(t *testing.T) {
	rejected, err := marshalRejectedResponse()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseResponse(rejected)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Status != pkiStatusRejected {
		t.Fatalf("status = %d, want %d", parsed.Status, pkiStatusRejected)
	}
	cert, _ := testTSARoot(t)
	if _, err := Verify(poolFor(t, cert), rejected, goldenMS, fixedNow(t)); err == nil {
		t.Fatal("verify accepted a rejected status, want failure")
	}
}

func TestParseResponseEmptyAndGarbage(t *testing.T) {
	if _, err := ParseResponse(nil); err == nil {
		t.Fatal("empty response parsed, want failure")
	}
	if _, err := ParseResponse([]byte{0x30, 0x03, 0x02, 0x01, 0xff}); err == nil {
		t.Fatal("garbage response parsed, want failure")
	}
}

func TestHTTPTransport(t *testing.T) {
	const goodBody = "\x30\x03\x02\x01\x00"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != contentTypeQuery {
			t.Errorf("content-type = %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != contentTypeReply {
			t.Errorf("accept = %q", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(goodBody))
	}))
	defer srv.Close()
	tr := &HTTPTransport{URL: srv.URL, Client: &http.Client{Timeout: time.Second}}
	got, err := tr.RoundTrip(context.Background(), []byte("\x30\x00"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != goodBody {
		t.Fatalf("body = %x, want %x", got, goodBody)
	}
}

func TestHTTPTransportNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	tr := &HTTPTransport{URL: srv.URL, Client: &http.Client{Timeout: time.Second}}
	if _, err := tr.RoundTrip(context.Background(), []byte("\x30\x00")); err == nil {
		t.Fatal("non-200 response accepted, want failure")
	}
}

func TestProvisionFromEnvAllOrNothing(t *testing.T) {
	for _, tc := range []struct {
		url  string
		file string
		want bool // wantErr
	}{
		{url: "", file: "", want: false},
		{url: "https://tsa.example", file: "", want: true},
		{url: "", file: "/tmp/roots.pem", want: true},
		{url: "https://tsa.example", file: filepath.Join(t.TempDir(), "missing.pem"), want: true},
	} {
		t.Setenv(envTSAURL, tc.url)
		t.Setenv(envTSARootsFile, tc.file)
		tsa, err := ProvisionFromEnv()
		if tc.want {
			if err == nil {
				t.Fatalf("ProvisionFromEnv(%q,%q) succeeded, want error", tc.url, tc.file)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ProvisionFromEnv(%q,%q): %v", tc.url, tc.file, err)
		}
		if tsa != nil {
			t.Fatalf("ProvisionFromEnv(%q,%q) returned a TSA, want nil", tc.url, tc.file)
		}
	}
}

func TestProvisionFromEnvWithRootsFile(t *testing.T) {
	cert, _ := testTSARoot(t)
	pemPath := filepath.Join(t.TempDir(), "roots.pem")
	data, err := marshalCertPEM(cert.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pemPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envTSAURL, "https://tsa.example")
	t.Setenv(envTSARootsFile, pemPath)
	tsa, err := ProvisionFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if tsa == nil {
		t.Fatal("ProvisionFromEnv returned nil TSA with roots configured")
	}
	if tsa.roots == nil {
		t.Fatal("provisioned TSA has no roots")
	}
}

func TestTimestampIntegrationAcrossHTTP(t *testing.T) {
	cert, key := testTSARoot(t)
	roots := poolFor(t, cert)
	transport := &echoTransport{cert: cert, key: key, genTime: testGenTime}
	srv := httptest.NewServer(&echoHandler{transport: transport})
	defer srv.Close()
	tsa, err := New(&HTTPTransport{URL: srv.URL, Client: &http.Client{Timeout: time.Second}}, roots)
	if err != nil {
		t.Fatal(err)
	}
	token, err := tsa.Timestamp(context.Background(), []byte(testMessage), fixedNow(t))
	if err != nil {
		t.Fatal(err)
	}
	if !token.GenTime.Equal(testGenTime) {
		t.Fatalf("genTime = %v, want %v", token.GenTime, testGenTime)
	}
	if len(token.Response) == 0 {
		t.Fatal("token response is empty")
	}
}

func TestNewRejectsPartialConfig(t *testing.T) {
	if _, err := New(nil, x509.NewCertPool()); err == nil {
		t.Fatal("New with nil transport succeeded, want failure")
	}
	if _, err := New(&HTTPTransport{URL: "https://x"}, nil); err == nil {
		t.Fatal("New with nil roots succeeded, want failure")
	}
}

type echoHandler struct {
	transport *echoTransport
}

func (h *echoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := h.transport.RoundTrip(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

// echoTransport fakes a TSA: it echoes the requested imprint and returns a
// self-describing, genuinely signed token. Test and simulation use only.
type echoTransport struct {
	cert    *x509.Certificate
	key     *rsa.PrivateKey
	genTime time.Time
}

func (t *echoTransport) RoundTrip(_ context.Context, request []byte) ([]byte, error) {
	req, err := ParseRequest(request)
	if err != nil {
		return nil, err
	}
	return BuildTestResponse(t.cert, t.key, req.MessageImprint.HashedMessage, t.genTime)
}

// TestVerifyForgedGenTimeRejects proves that a token whose TSTInfo genTime was
// changed and whose messageDigest attribute was re-derived to match — exactly
// what an attacker who holds a valid token but no TSA key can do — is still
// rejected because the stale CMS signature no longer covers the altered
// signedAttrs (RFC 5652 §5.4).
func TestVerifyForgedGenTimeRejects(t *testing.T) {
	cert, key := testTSARoot(t)

	originalInfo, err := buildTSTInfoDER(goldenMS, testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	originalDigest := sha256.Sum256(originalInfo)
	originalAttrs, err := buildSignedAttrsDER(originalDigest[:])
	if err != nil {
		t.Fatal(err)
	}
	signature, _, err := signSignedAttrs(key, originalAttrs)
	if err != nil {
		t.Fatal(err)
	}

	forgedInfo, err := buildTSTInfoDER(goldenMS, testGenTime.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	forgedDigest := sha256.Sum256(forgedInfo)
	forgedAttrs, err := buildSignedAttrsDER(forgedDigest[:])
	if err != nil {
		t.Fatal(err)
	}
	forgedSigner, err := newSignerInfo(oidSHA256WithRSA, signature, forgedAttrs, cert)
	if err != nil {
		t.Fatal(err)
	}
	forgedToken, err := buildTokenDER(forgedInfo, cert, forgedSigner)
	if err != nil {
		t.Fatal(err)
	}
	resp := marshalGrantedResponse(t, forgedToken)
	if _, err := Verify(poolFor(t, cert), resp, goldenMS, fixedNow(t)); err == nil {
		t.Fatal("verify accepted a forged token with attacker-chosen genTime, want failure")
	}
}

// TestVerifyTamperedMessageDigestRejects proves the messageDigest attribute is
// coupled to the TSTInfo content: a properly signed token whose signedAttrs
// claim the wrong digest must fail even though the signature itself is valid.
func TestVerifyTamperedMessageDigestRejects(t *testing.T) {
	cert, key := testTSARoot(t)
	info, err := buildTSTInfoDER(goldenMS, testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	wrongDigest := sha256.Sum256([]byte("not the tstinfo"))
	wrongAttrs, err := buildSignedAttrsDER(wrongDigest[:])
	if err != nil {
		t.Fatal(err)
	}
	signature, _, err := signSignedAttrs(key, wrongAttrs)
	if err != nil {
		t.Fatal(err)
	}
	tamperedSigner, err := newSignerInfo(oidSHA256WithRSA, signature, wrongAttrs, cert)
	if err != nil {
		t.Fatal(err)
	}
	tamperedToken, err := buildTokenDER(info, cert, tamperedSigner)
	if err != nil {
		t.Fatal(err)
	}
	resp := marshalGrantedResponse(t, tamperedToken)
	if _, err := Verify(poolFor(t, cert), resp, goldenMS, fixedNow(t)); err == nil {
		t.Fatal("verify accepted a token whose messageDigest attribute does not match its TSTInfo, want failure")
	}
}

// TestVerifyUnknownSignatureAlgorithmRejects proves unknown SignatureAlgorithm
// OIDs fail closed instead of being skipped.
func TestVerifyUnknownSignatureAlgorithmRejects(t *testing.T) {
	cert, key := testTSARoot(t)
	info, err := buildTSTInfoDER(goldenMS, testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	infoDigest := sha256.Sum256(info)
	attrs, err := buildSignedAttrsDER(infoDigest[:])
	if err != nil {
		t.Fatal(err)
	}
	signature, _, err := signSignedAttrs(key, attrs)
	if err != nil {
		t.Fatal(err)
	}
	unknown := asn1.ObjectIdentifier{1, 3, 999, 1, 2, 3}
	opaqueSigner, err := newSignerInfo(unknown, signature, attrs, cert)
	if err != nil {
		t.Fatal(err)
	}
	opaqueToken, err := buildTokenDER(info, cert, opaqueSigner)
	if err != nil {
		t.Fatal(err)
	}
	resp := marshalGrantedResponse(t, opaqueToken)
	if _, err := Verify(poolFor(t, cert), resp, goldenMS, fixedNow(t)); err == nil {
		t.Fatal("verify accepted a token with an unknown signature algorithm, want failure")
	}
}

func marshalGrantedResponse(t *testing.T, tokenDER []byte) []byte {
	t.Helper()
	resp, err := asn1.Marshal(timeStampResp{Status: pkiStatusInfo{Status: pkiStatusGranted}, TimeStamp: asn1.RawValue{FullBytes: tokenDER}})
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// TestVerifyPSSSignatureAccepts proves RSASSA-PSS SignerInfo signatures
// verify across SHA-256/384/512.
func TestVerifyPSSSignatureAccepts(t *testing.T) {
	cert, key := testTSARoot(t)
	for _, hash := range []crypto.Hash{crypto.SHA256, crypto.SHA384, crypto.SHA512} {
		resp, err := BuildTestResponsePSS(cert, key, hash, goldenMS, testGenTime)
		if err != nil {
			t.Fatalf("BuildTestResponsePSS(%v): %v", hash, err)
		}
		if _, err := Verify(poolFor(t, cert), resp, goldenMS, fixedNow(t)); err != nil {
			t.Fatalf("PSS %v verify failed: %v", hash, err)
		}
	}
}

// TestVerifyEd25519SignatureAccepts proves Ed25519 SignerInfo signatures
// verify end to end (certificate chain + CMS signature).
func TestVerifyEd25519SignatureAccepts(t *testing.T) {
	cert, key := testTSARootEd25519(t)
	resp, err := BuildTestResponseEd25519(cert, key, goldenMS, testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(poolFor(t, cert), resp, goldenMS, fixedNow(t)); err != nil {
		t.Fatalf("Ed25519 verify failed: %v", err)
	}
}

// TestVerifySHA1SignatureRejects proves SHA-1 signed tokens are hard-rejected
// even though the messageDigest attribute and certificate chain are valid.
func TestVerifySHA1SignatureRejects(t *testing.T) {
	cert, key := testTSARoot(t)
	info, err := buildTSTInfoDER(goldenMS, testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	infoDigest := sha256.Sum256(info)
	attrs, err := buildSignedAttrsDER(infoDigest[:])
	if err != nil {
		t.Fatal(err)
	}
	sha1Digest := sha1.Sum(attrs)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA1, sha1Digest[:])
	if err != nil {
		t.Fatal(err)
	}
	sha1Signer, err := newSignerInfoWithDigest(oidSHA1WithRSA, signature, attrs, cert, oidSHA1)
	if err != nil {
		t.Fatal(err)
	}
	token, err := buildTokenDER(info, cert, sha1Signer)
	if err != nil {
		t.Fatal(err)
	}
	resp := marshalGrantedResponse(t, token)
	if _, err := Verify(poolFor(t, cert), resp, goldenMS, fixedNow(t)); err == nil {
		t.Fatal("verify accepted a SHA-1 signed token, want failure")
	}
}

// TestVerifyPSSSHA1Rejects proves RSASSA-PSS defaulting to SHA-1 parameters
// fails closed (the SignerInfo digest is SHA-1).
func TestVerifyPSSSHA1Rejects(t *testing.T) {
	cert, key := testTSARoot(t)
	resp, err := BuildTestResponsePSS(cert, key, crypto.SHA1, goldenMS, testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(poolFor(t, cert), resp, goldenMS, fixedNow(t)); err == nil {
		t.Fatal("verify accepted an RSASSA-PSS token with SHA-1 digest, want failure")
	}
}
