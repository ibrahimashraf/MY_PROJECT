package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/hkdf"
)

var (
	ErrEmptyTenantID     = errors.New("tenant_id is required for key isolation")
	ErrSecretNotFound    = errors.New("secret not found for tenant")
	ErrDecryptionFailed  = errors.New("decryption failed: ciphertext corrupted or wrong tenant key")
	ErrMasterKeyTooShort = errors.New("master key must be at least 32 bytes")
)

// StoredSecret represents the ciphertext payload stored in PostgreSQL.
type StoredSecret struct {
	TenantID       string    `json:"tenant_id"`
	OrganizationID string    `json:"organization_id"`
	Provider       string    `json:"provider"` // e.g. "openai", "anthropic", "gemini"
	KeyFingerprint string    `json:"key_fingerprint"`
	CiphertextB64  string    `json:"ciphertext_b64"`
	NonceB64       string    `json:"nonce_b64"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Vault manages tenant-isolated BYOK encryption using AES-256-GCM and HKDF-derived keys.
type Vault struct {
	masterKey []byte
	mu        sync.RWMutex
	store     map[string]StoredSecret // In-memory/test storage, mirrors DB table
}

// NewVault initializes the vault with a cryptographically secure root master key.
func NewVault(masterKey []byte) (*Vault, error) {
	if len(masterKey) < 32 {
		return nil, ErrMasterKeyTooShort
	}
	return &Vault{
		masterKey: masterKey,
		store:     make(map[string]StoredSecret),
	}, nil
}

// deriveTenantKey generates a deterministic, isolated 256-bit AES key per tenant via HKDF-SHA256.
func (v *Vault) deriveTenantKey(tenantID, orgID string) ([]byte, error) {
	if tenantID == "" {
		return nil, ErrEmptyTenantID
	}
	info := fmt.Sprintf("integin-byok-v1:%s:%s", tenantID, orgID)
	hkdfReader := hkdf.New(sha256.New, v.masterKey, nil, []byte(info))
	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("failed deriving tenant key: %w", err)
	}
	return key, nil
}

// StoreSecret encrypts an external provider API key with the tenant's isolated AEAD key.
func (v *Vault) StoreSecret(ctx context.Context, tenantID, orgID, provider, rawApiKey string) (*StoredSecret, error) {
	tenantKey, err := v.deriveTenantKey(tenantID, orgID)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(tenantKey)
	if err != nil {
		return nil, fmt.Errorf("failed creating cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed creating GCM AEAD: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed generating nonce: %w", err)
	}

	// Additional Authenticated Data (AAD) binds the ciphertext to the tenant and provider
	aad := []byte(fmt.Sprintf("%s:%s:%s", tenantID, orgID, provider))
	ciphertext := gcm.Seal(nil, nonce, []byte(rawApiKey), aad)

	// Fingerprint for safe audit logging (first 4 and last 4 chars)
	fingerprint := "key-****"
	if len(rawApiKey) > 8 {
		fingerprint = fmt.Sprintf("%s...%s", rawApiKey[:4], rawApiKey[len(rawApiKey)-4:])
	}

	record := StoredSecret{
		TenantID:       tenantID,
		OrganizationID: orgID,
		Provider:       provider,
		KeyFingerprint: fingerprint,
		CiphertextB64:  base64.StdEncoding.EncodeToString(ciphertext),
		NonceB64:       base64.StdEncoding.EncodeToString(nonce),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	v.mu.Lock()
	v.store[fmt.Sprintf("%s:%s", tenantID, provider)] = record
	v.mu.Unlock()

	return &record, nil
}

// ResolveSecret decrypts the external provider key strictly asserting tenant ownership.
func (v *Vault) ResolveSecret(ctx context.Context, tenantID, orgID, provider string) (string, error) {
	v.mu.RLock()
	record, exists := v.store[fmt.Sprintf("%s:%s", tenantID, provider)]
	v.mu.RUnlock()

	if !exists {
		return "", ErrSecretNotFound
	}

	tenantKey, err := v.deriveTenantKey(tenantID, orgID)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(tenantKey)
	if err != nil {
		return "", fmt.Errorf("failed creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed creating GCM: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(record.CiphertextB64)
	if err != nil {
		return "", fmt.Errorf("invalid ciphertext base64: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(record.NonceB64)
	if err != nil {
		return "", fmt.Errorf("invalid nonce base64: %w", err)
	}

	aad := []byte(fmt.Sprintf("%s:%s:%s", tenantID, orgID, provider))
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}
