package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const outputDirectory = "contracts/vectors/v1"

type fixture struct {
	VectorSetVersion             int    `json:"vector_set_version"`
	VectorID                     string `json:"vector_id"`
	Family                       string `json:"family"`
	CanonicalizationVersion      string `json:"canonicalization_version"`
	TestOnlyPrivateSeedBase64URL string `json:"test_only_private_seed_base64url"`
	PublicKeyBase64              string `json:"public_key_base64"`
	KeyIDSHA256                  string `json:"key_id_sha256"`
	CanonicalUTF8                string `json:"canonical_utf8"`
	CanonicalBytesBase64URL      string `json:"canonical_bytes_base64url"`
	SignatureBase64              string `json:"signature_base64"`
	ExpectedAccepted             bool   `json:"expected_accepted"`
	ExpectedReasonCode           string `json:"expected_reason_code"`
}

type bundle struct {
	Valid       fixture `json:"valid"`
	ByteAltered fixture `json:"byte_altered"`
}

func main() {
	seed := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	keyID := sha256.Sum256(publicKey)

	validCanonical := canonicalTransactionV1(
		"txn-vector-001", "tenant-vector-a", "org-vector-a", "TESTING",
		"device-vector-a", "user-vector-a", 7, "inspection.complete", "asset-vector-a",
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"authority-vector-a", 3, "2026-08-15T00:00:00Z", "Ed25519", hex.EncodeToString(keyID[:]),
	)
	validSignature := ed25519.Sign(privateKey, []byte(validCanonical))

	valid := newFixture("signed-transaction-valid-001", seed, publicKey, keyID[:], validCanonical, validSignature, true, "")
	alteredCanonical := strings.Replace(validCanonical, "inspection.complete", "inspection.rejected", 1)
	altered := newFixture("signed-transaction-byte-altered-001", seed, publicKey, keyID[:], alteredCanonical, validSignature, false, "SIGNATURE_INVALID")

	if !ed25519.Verify(publicKey, []byte(valid.CanonicalUTF8), validSignature) {
		fail("valid vector signature did not verify")
	}
	if ed25519.Verify(publicKey, []byte(altered.CanonicalUTF8), validSignature) {
		fail("byte-altered vector unexpectedly verified")
	}

	payload, err := json.MarshalIndent(bundle{Valid: valid, ByteAltered: altered}, "", "  ")
	if err != nil {
		fail(err.Error())
	}
	payload = append(payload, '\n')

	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		fail(err.Error())
	}
	fixturePath := filepath.Join(outputDirectory, "signed_transaction_ed25519_v1.json")
	if err := os.WriteFile(fixturePath, payload, 0o644); err != nil {
		fail(err.Error())
	}
	digest := sha256.Sum256(payload)
	manifest := fmt.Sprintf("%x  signed_transaction_ed25519_v1.json\n", digest)
	if err := os.WriteFile(filepath.Join(outputDirectory, "manifest.sha256"), []byte(manifest), 0o644); err != nil {
		fail(err.Error())
	}
	fmt.Println("PILOT_SIGNED_TRANSACTION_VECTOR_BUNDLE_VERIFIED")
}

func canonicalTransactionV1(transactionID, tenantID, organizationID, environment, deviceID, userID string, sequenceNumber int, operation, entityID, payloadHash, authorityID string, authorityEpoch int, capturedAtUTC, signatureAlgorithm, keyID string) string {
	return fmt.Sprintf("v1|%s|%s|%s|%s|%s|%s|%d|%s|%s|%s|%s|%d|%s|%s|%s",
		transactionID, tenantID, organizationID, environment, deviceID, userID, sequenceNumber,
		operation, entityID, payloadHash, authorityID, authorityEpoch, capturedAtUTC, signatureAlgorithm, keyID,
	)
}

func newFixture(id string, seed []byte, publicKey ed25519.PublicKey, keyID []byte, canonical string, signature []byte, accepted bool, reasonCode string) fixture {
	return fixture{
		VectorSetVersion:             1,
		VectorID:                     id,
		Family:                       "signed_transaction",
		CanonicalizationVersion:      "integin-c14n-1",
		TestOnlyPrivateSeedBase64URL: base64.RawURLEncoding.EncodeToString(seed),
		PublicKeyBase64:              base64.StdEncoding.EncodeToString(publicKey),
		KeyIDSHA256:                  hex.EncodeToString(keyID),
		CanonicalUTF8:                canonical,
		CanonicalBytesBase64URL:      base64.RawURLEncoding.EncodeToString([]byte(canonical)),
		SignatureBase64:              base64.StdEncoding.EncodeToString(signature),
		ExpectedAccepted:             accepted,
		ExpectedReasonCode:           reasonCode,
	}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "contract vector generation failed:", message)
	os.Exit(1)
}
