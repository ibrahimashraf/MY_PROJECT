package evidenceexport

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestExportVerifier_GoldenParityContract guarantees bit-for-bit parity between
// the Go evidenceexport manifest signer and the Rust integin-export-verifier golden testdata.
func TestExportVerifier_GoldenParityContract(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "tools", "integin-export-verifier", "testdata", "manifest.json")
	pubKeyPath := filepath.Join("..", "..", "tools", "integin-export-verifier", "testdata", "public_key.hex")

	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed reading golden manifest: %v", err)
	}

	pubKeyHex, err := os.ReadFile(pubKeyPath)
	if err != nil {
		t.Fatalf("failed reading golden public key: %v", err)
	}

	pubKeyBytes, err := hex.DecodeString(string(pubKeyHex))
	if err != nil {
		t.Fatalf("invalid hex public key: %v", err)
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		t.Fatalf("expected 32-byte Ed25519 key, got %d", len(pubKeyBytes))
	}
	pubKey := ed25519.PublicKey(pubKeyBytes)

	var manifest Manifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatalf("failed unmarshaling golden manifest: %v", err)
	}

	// 1. Rebuild canonical payload byte-for-byte in Go
	canonical, err := manifest.canonicalPayload()
	if err != nil {
		t.Fatalf("failed building canonical payload: %v", err)
	}

	// 2. Assert manifest_sha256 == SHA256(canonicalPayload)
	computedHash := sha256.Sum256(canonical)
	computedHashHex := hex.EncodeToString(computedHash[:])
	if manifest.ManifestSHA256 != computedHashHex {
		t.Fatalf("SHA256 hash mismatch! Manifest declares %s, computed canonical %s", manifest.ManifestSHA256, computedHashHex)
	}

	// 3. Verify Ed25519 signature and full manifest validity over canonical payload
	if err := manifest.VerifyManifest(pubKey); err != nil {
		t.Fatalf("VerifyManifest failed on golden fixture: %v", err)
	}
}
