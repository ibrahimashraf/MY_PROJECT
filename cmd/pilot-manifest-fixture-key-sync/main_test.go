package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"integin/internal/security"
)

func TestRunSynchronizesOnlyPublicDeviceKeyBinding(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "fixture.json")
	outputPath := filepath.Join(dir, "fixture-bound.json")
	privateKeyPath := filepath.Join(dir, "private.base64url")
	input := fixture{
		DeviceID:                "device",
		AuthorityID:             "authority",
		TenantID:                "tenant",
		OrganizationID:          "organization",
		InspectionID:            "inspection",
		DeviceKeyID:             "obsolete-key-id",
		DevicePublicKeyBase64:   "obsolete-public-key",
		ManifestKeyID:           "manifest-key",
		ManifestPublicKeyBase64: "manifest-public-key",
		UserID:                  "user",
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(privateKeyPath, []byte(base64.RawURLEncoding.EncodeToString(privateKey)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(inputPath, outputPath, privateKeyPath); err != nil {
		t.Fatal(err)
	}
	outputRaw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	var output fixture
	if err := json.Unmarshal(outputRaw, &output); err != nil {
		t.Fatal(err)
	}
	if output.DeviceKeyID != security.DeviceKeyID(publicKey) {
		t.Fatalf("device key ID was not synchronized")
	}
	if output.DevicePublicKeyBase64 != base64.RawURLEncoding.EncodeToString(publicKey) {
		t.Fatalf("device public key was not synchronized")
	}
	if output.ManifestKeyID != input.ManifestKeyID || output.ManifestPublicKeyBase64 != input.ManifestPublicKeyBase64 || output.InspectionID != input.InspectionID {
		t.Fatal("non-device public fixture fields changed")
	}
}

func TestRunRefusesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.json")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(path, path, filepath.Join(dir, "missing.key")); err == nil {
		t.Fatal("same fixture and output paths were accepted")
	}
}
