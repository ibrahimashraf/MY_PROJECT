package security

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

func GenerateDeviceKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}
	return publicKey, privateKey, nil
}

func SignDeviceMutation(privateKey ed25519.PrivateKey, message []byte) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key")
	}
	return ed25519.Sign(privateKey, message), nil
}

func VerifyDeviceMutation(publicKey ed25519.PublicKey, message, signature []byte) bool {
	return len(publicKey) == ed25519.PublicKeySize && len(signature) == ed25519.SignatureSize && ed25519.Verify(publicKey, message, signature)
}

func DeviceKeyID(publicKey ed25519.PublicKey) string {
	hash := sha256.Sum256(publicKey)
	return hex.EncodeToString(hash[:])
}
