package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

var (
	ErrNilPrivateKey = errors.New("private key cannot be nil or empty")
)

// MasterLicenseIssuer is used by vendor operations / CLI to mint verified license tokens.
type MasterLicenseIssuer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// NewMasterLicenseIssuer creates an issuer with an Ed25519 signing key.
func NewMasterLicenseIssuer(priv ed25519.PrivateKey) (*MasterLicenseIssuer, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return nil, ErrNilPrivateKey
	}
	pub := priv.Public().(ed25519.PublicKey)
	return &MasterLicenseIssuer{
		privateKey: priv,
		publicKey:  pub,
	}, nil
}

// GenerateMasterKeyPair creates a fresh asymmetric Ed25519 root authority keypair.
func GenerateMasterKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// IssueLicense signs a license payload and produces a portable SignedLicenseToken.
func (issuer *MasterLicenseIssuer) IssueLicense(payload LicensePayload) (*SignedLicenseToken, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	digest := sha256.Sum256(payloadBytes)
	sig := ed25519.Sign(issuer.privateKey, digest[:])

	return &SignedLicenseToken{
		Payload:   payload,
		Signature: hex.EncodeToString(sig),
		PublicKey: hex.EncodeToString(issuer.publicKey),
	}, nil
}

// AddSignoff appends a second Ed25519 signature over the same payload digest
// from another authorized vendor key, contributing to an m-of-n quorum.
func (issuer *MasterLicenseIssuer) AddSignoff(token *SignedLicenseToken) error {
	if token == nil {
		return errors.New("license token cannot be nil")
	}
	payloadBytes, err := json.Marshal(token.Payload)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(payloadBytes)
	sig := ed25519.Sign(issuer.privateKey, digest[:])
	token.Signoffs = append(token.Signoffs, hex.EncodeToString(sig))
	return nil
}
