package timestamp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"
)

func mustParseSerial(t *testing.T, respDER []byte) *big.Int {
	t.Helper()
	resp, err := ParseResponse(respDER)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != pkiStatusGranted {
		t.Fatalf("response status = %d, want granted", resp.Status)
	}
	info, _, err := parseTSTInfo(resp.Token)
	if err != nil {
		t.Fatal(err)
	}
	return info.SerialNumber
}

func TestBuildResponseHonorsSerial(t *testing.T) {
	cert, key := testTSARoot(t)
	imprint := sha256.Sum256([]byte(testMessage))
	serial := big.NewInt(0).Lsh(big.NewInt(1), 100)
	serial.Add(serial, big.NewInt(7))
	respDER, err := BuildResponse(cert, key, imprint[:], testGenTime, serial)
	if err != nil {
		t.Fatal(err)
	}
	if got := mustParseSerial(t, respDER); got.Cmp(serial) != 0 {
		t.Fatalf("serial = %s, want %s", got, serial)
	}
	pool := poolFor(t, cert)
	if _, err := Verify(pool, respDER, imprint[:], testGenTime.Add(time.Minute)); err != nil {
		t.Fatalf("serial-carrying token rejected: %v", err)
	}
}

func TestBuildResponseRejectsBadSerial(t *testing.T) {
	cert, key := testTSARoot(t)
	imprint := sha256.Sum256([]byte(testMessage))
	for name, serial := range map[string]*big.Int{"nil": nil, "zero": big.NewInt(0), "negative": big.NewInt(-3)} {
		if _, err := BuildResponse(cert, key, imprint[:], testGenTime, serial); err == nil {
			t.Fatalf("%s serial accepted, want error", name)
		}
	}
}

func TestBuildTestResponseKeepsFixedSerial(t *testing.T) {
	cert, key := testTSARoot(t)
	imprint := sha256.Sum256([]byte(testMessage))
	respDER, err := BuildTestResponse(cert, key, imprint[:], testGenTime)
	if err != nil {
		t.Fatal(err)
	}
	if got := mustParseSerial(t, respDER); got.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("legacy serial = %s, want 1", got)
	}
}

func TestRandomSerialPositiveAndUnique(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 4; i++ {
		serial, err := RandomSerial()
		if err != nil {
			t.Fatal(err)
		}
		if serial.Sign() <= 0 || serial.BitLen() > 128 {
			t.Fatalf("serial out of range: %s", serial)
		}
		key := serial.String()
		if _, dup := seen[key]; dup {
			t.Fatalf("duplicate serial: %s", key)
		}
		seen[key] = struct{}{}
	}
}

// TestVerifyAcceptsEKUConstrainedTSACert locks the verifier to the
// timeStamping purpose: a proper RFC 3161 TSA certificate (EKU-constrained,
// like pilot gen-tsa mints) must verify, not just legacy unconstrained roots.
func TestVerifyAcceptsEKUConstrainedTSACert(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := RandomSerial()
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "test TSA EKU"},
		NotBefore:             testGenTime.Add(-time.Hour),
		NotAfter:              testGenTime.Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	imprint := sha256.Sum256([]byte(testMessage))
	tokenSerial, err := RandomSerial()
	if err != nil {
		t.Fatal(err)
	}
	respDER, err := BuildResponse(cert, key, imprint[:], testGenTime, tokenSerial)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(poolFor(t, cert), respDER, imprint[:], testGenTime.Add(time.Minute)); err != nil {
		t.Fatalf("EKU-constrained TSA token rejected: %v", err)
	}
}
