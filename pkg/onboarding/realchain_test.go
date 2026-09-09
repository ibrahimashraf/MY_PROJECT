package onboarding

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"testing"
)

// Environment-gated device verification. These tests are skipped unless the
// orchestrator supplies a probe captured from a real Android device via adb:
//
//	INTEGIN_REAL_CHAIN_JSON  path to the attest_probe.json pulled from the phone
//	INTEGIN_REAL_ROOTS_PEM   path to the Google attestation roots PEM
//	                        (https://android.googleapis.com/attestation/root)
//	INTEGIN_REAL_NONCE       the exact 64-hex-char nonce the probe was minted with
//
// They exercise the same code path the enrollment engine uses for STRONGBOX
// claims (verifyPEMChain), so a genuine Google-signed chain that passes here
// is exactly the evidence the offline verifier will accept at enrollment.
const (
	testRealChainJSON = "INTEGIN_REAL_CHAIN_JSON"
	testRealRootsPEM  = "INTEGIN_REAL_ROOTS_PEM"
	testRealNonce     = "INTEGIN_REAL_NONCE"
)

func TestRealDeviceChainVerifies(t *testing.T) {
	chainPath := os.Getenv(testRealChainJSON)
	rootsPath := os.Getenv(testRealRootsPEM)
	nonce := os.Getenv(testRealNonce)
	if chainPath == "" || rootsPath == "" || nonce == "" {
		t.Skipf("set %s, %s and %s to verify a real captured device chain",
			testRealChainJSON, testRealRootsPEM, testRealNonce)
	}

	type probe struct {
		NonceHex    string   `json:"nonce_hex"`
		ChainPEM    []string `json:"chain_pem"`
		SecurityLev string   `json:"security_level"`
		DeviceModel string   `json:"device_model"`
		AndroidSDK  int      `json:"android_sdk"`
		Error       string   `json:"error"`
	}
	var p probe
	raw, err := os.ReadFile(chainPath)
	if err != nil {
		t.Fatalf("read probe JSON: %v", err)
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("parse probe JSON: %v", err)
	}
	if p.Error != "" {
		t.Fatalf("probe mint failed: %s", p.Error)
	}
	if p.NonceHex != nonce {
		t.Fatalf("probe nonce %q differs from expected nonce %q", p.NonceHex, nonce)
	}
	if len(p.ChainPEM) == 0 {
		t.Fatalf("probe carried no chain_pem")
	}
	roots, err := LoadAttestationRoots(rootsPath)
	if err != nil {
		t.Fatalf("load Google attestation roots: %v", err)
	}
	if kd, err := parseKeyDescription(chainLeaf(t, p.ChainPEM)); err == nil {
		t.Logf("parsed KeyDescription: securityLevel=%d reportedBootState=%d softwarePurposes=%v teePurposes=%v",
			kd.securityLevel, kd.bootState, kd.softwarePurposes, kd.teePurposes)
	} else {
		t.Logf("parseKeyDescription diagnostic failed: %v", err)
	}

	// The Keymaster attestationChallenge was the UTF-8 bytes of the nonce
	// HEX STRING (not the decoded bytes) — that is what the device bound the
	// challenge to, so it is what expectedChallenge must be.
	info, err := verifyPEMChain(p.ChainPEM, roots, []byte(p.NonceHex))
	if err != nil {
		t.Fatalf("VerifyAttestationChain rejected the genuine device chain: %v", err)
	}
	if !info.VerifiedBoot {
		t.Error("verifiedBootState was not Verified for the genuine chain")
	}
	if !info.ChallengeMatched {
		t.Error("challenge did not match for the genuine chain")
	}
	t.Logf("real device chain verifies offline: StrongBox=%v (claim=%s) ChallengeMatched=%v deviceModel=%s androidSdk=%d",
		info.StrongBox, p.SecurityLev, info.ChallengeMatched, p.DeviceModel, p.AndroidSDK)
}

func chainLeaf(t *testing.T, pems []string) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode([]byte(pems[0]))
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("first chain element is not a PEM certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse leaf certificate: %v", err)
	}
	return cert
}
