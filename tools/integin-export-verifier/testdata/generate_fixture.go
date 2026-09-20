// Command generate_fixture produces the golden, Go-signed testdata for the
// Rust export verifier: manifest.json + public_key.hex.
//
// Run from testdata/ inside the integin module:
//
//	go run generate_fixture.go
//
// It is deterministic: a fixed Ed25519 seed and fixed timestamps mean the
// golden files are byte-reproducible. The manifest is intentionally seeded
// with unordered evidence_items and fractional/non-fractional timestamps so
// the Rust verifier must sort and normalize exactly like Go.
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"integin/pkg/evidenceexport"
)

func sha256hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	outDir := dir
	if filepath.Base(dir) != "testdata" {
		outDir = filepath.Join(dir, "tools", "rust-export-verifier", "testdata")
	}

	seed, err := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	if err != nil {
		panic(err)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	items := []evidenceexport.EvidenceItem{
		{ // intentionally placed second to exercise evidence_id sorting
			EvidenceID:          "EV-1002",
			ObjectKey:           "exports/tenant-t1/EV-1002.bin.enc",
			ContentType:         "application/octet-stream",
			CapturedAt:          time.Date(2026, 8, 20, 15, 10, 30, 0, time.UTC),
			PlaintextSHA256:     sha256hex("plaintext-object-2"),
			CiphertextSHA256:    sha256hex("ciphertext-object-2"),
			ByteLength:          8192,
			EncryptionAlgorithm: "AES-256-GCM",
			AuthorityDeviceID:   "device-tablet-7f3a9912",
		},
		{
			EvidenceID:          "EV-1001",
			ObjectKey:           "exports/tenant-t1/EV-1001.bin.enc",
			ContentType:         "application/octet-stream",
			CapturedAt:          time.Date(2026, 8, 20, 14, 5, 0, 250000000, time.UTC), // fractional
			PlaintextSHA256:     sha256hex("plaintext-object-1"),
			CiphertextSHA256:    sha256hex("ciphertext-object-1"),
			ByteLength:          4096,
			EncryptionAlgorithm: "AES-256-GCM",
			AuthorityDeviceID:   "device-tablet-7f3a9912",
		},
		{
			EvidenceID:          "EV-1003",
			ObjectKey:           "exports/tenant-t1/EV-1003.bin.enc",
			ContentType:         "application/octet-stream",
			CapturedAt:          time.Date(2026, 8, 20, 15, 11, 0, 500000000, time.UTC),
			PlaintextSHA256:     sha256hex("plaintext-object-3"),
			CiphertextSHA256:    sha256hex("ciphertext-object-3"),
			ByteLength:          2048,
			EncryptionAlgorithm: "AES-256-GCM",
			AuthorityDeviceID:   "device-tablet-7f3a9912",
		},
	}

	// NewManifest seals with the current wall clock; we then pin a fixed
	// CreatedAt and re-sign so the golden payload is fully reproducible.
	m, err := evidenceexport.NewManifest(
		"EXP-2026-08-21-001",
		"tenant-t1",
		"org-intmec",
		"PROC-EXPORT-V1",
		evidenceexport.ExporterIdentity{
			UserID:        "inspector-ahmed-01",
			Role:          "CERTIFIED_INSPECTOR",
			ClientVersion: "field-app-1.4.0",
		},
		items,
	)
	if err != nil {
		panic(err)
	}
	m.CreatedAt = time.Date(2026, 8, 21, 9, 30, 15, 123456789, time.UTC)

	if err := m.Sign(func(canonical []byte) ([]byte, error) {
		return ed25519.Sign(priv, canonical), nil
	}); err != nil {
		panic(err)
	}

	manifestJSON, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "manifest.json"), manifestJSON, 0o644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "public_key.hex"), []byte(hex.EncodeToString(pub)), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s and %s\n",
		filepath.Join(outDir, "manifest.json"),
		filepath.Join(outDir, "public_key.hex"))
}
