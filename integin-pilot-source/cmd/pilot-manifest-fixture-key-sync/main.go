// pilot-manifest-fixture-key-sync updates only the public device-key binding in
// an isolated pilot descriptor. It neither prints nor persists the private key.
package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"

	"integin/internal/security"
)

type fixture struct {
	DeviceID                string `json:"device_id"`
	AuthorityID             string `json:"authority_id"`
	TenantID                string `json:"tenant_id"`
	OrganizationID          string `json:"organization_id"`
	InspectionID            string `json:"inspection_id"`
	DeviceKeyID             string `json:"device_key_id"`
	DevicePublicKeyBase64   string `json:"device_public_key_base64url"`
	ManifestKeyID           string `json:"manifest_key_id"`
	ManifestPublicKeyBase64 string `json:"manifest_public_key_base64url"`
	UserID                  string `json:"user_id"`
}

func main() {
	inputPath := flag.String("fixture", "", "approved public fixture descriptor")
	outputPath := flag.String("output", "", "new public fixture descriptor path")
	privateKeyPath := flag.String("private-key-file", "", "approved matrix signing key path")
	flag.Parse()

	if err := run(*inputPath, *outputPath, *privateKeyPath); err != nil {
		os.Exit(1)
	}
}

func run(inputPath, outputPath, privateKeyPath string) error {
	if strings.TrimSpace(inputPath) == "" || strings.TrimSpace(outputPath) == "" || strings.TrimSpace(privateKeyPath) == "" {
		return errors.New("fixture, output, and private key paths are required")
	}
	inputAbsolute, err := filepath.Abs(inputPath)
	if err != nil {
		return err
	}
	outputAbsolute, err := filepath.Abs(outputPath)
	if err != nil {
		return err
	}
	if inputAbsolute == outputAbsolute {
		return errors.New("fixture output path must differ from input path")
	}
	privateKey, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return err
	}
	defer zero(privateKey)

	//nolint:gosec // path is CLI arg/config for test tool
	raw, err := os.ReadFile(inputAbsolute)
	if err != nil {
		return err
	}
	var value fixture
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	if err := value.validate(); err != nil {
		return err
	}

	publicKey := privateKey.Public().(ed25519.PublicKey)
	value.DeviceKeyID = security.DeviceKeyID(publicKey)
	value.DevicePublicKeyBase64 = base64.RawURLEncoding.EncodeToString(publicKey)
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')

	file, err := os.OpenFile(outputAbsolute, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(encoded); err != nil {
		return err
	}
	return file.Sync()
}

func (f fixture) validate() error {
	for name, value := range map[string]string{
		"device_id":                     f.DeviceID,
		"authority_id":                  f.AuthorityID,
		"tenant_id":                     f.TenantID,
		"organization_id":               f.OrganizationID,
		"inspection_id":                 f.InspectionID,
		"device_key_id":                 f.DeviceKeyID,
		"device_public_key_base64url":   f.DevicePublicKeyBase64,
		"manifest_key_id":               f.ManifestKeyID,
		"manifest_public_key_base64url": f.ManifestPublicKeyBase64,
		"user_id":                       f.UserID,
	} {
		if strings.TrimSpace(value) == "" {
			return errors.New(name + " is required")
		}
	}
	return nil
}

//nolint:gosec // path is CLI arg/config for test tool
func loadPrivateKey(path string) (ed25519.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(raw)
	raw = bytes.TrimSpace(raw)
	decoded := make([]byte, base64.RawURLEncoding.DecodedLen(len(raw)))
	n, err := base64.RawURLEncoding.Decode(decoded, raw)
	if err != nil {
		zeroBytes(decoded)
		decoded = make([]byte, base64.StdEncoding.DecodedLen(len(raw)))
		n, err = base64.StdEncoding.Decode(decoded, raw)
	}
	if err != nil || n != ed25519.PrivateKeySize {
		zeroBytes(decoded)
		return nil, errors.New("invalid private key")
	}
	return ed25519.PrivateKey(decoded[:n]), nil
}

func zero(key ed25519.PrivateKey) { zeroBytes(key) }

func zeroBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
