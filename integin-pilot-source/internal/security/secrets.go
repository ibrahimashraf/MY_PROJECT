package security

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// SecretLease models a resolved secret value bound to a path and expiration lease.
type SecretLease struct {
	Path      string
	Version   int
	Data      map[string]string
	LeaseTTL  time.Duration
	ExpiresAt time.Time
}

// SecretProvider abstracts dynamic secret retrieval (OpenBao KV v2, Vault, or isolated memory store).
type SecretProvider interface {
	GetSecret(ctx context.Context, path string) (SecretLease, error)
}

// InMemorySecretProvider provides an isolated thread-safe secret provider compliant with
// the OpenBao KV v2 path naming convention (`kv/production/...`).
type InMemorySecretProvider struct {
	mu      sync.RWMutex
	secrets map[string]SecretLease
	now     func() time.Time
}

// NewInMemorySecretProvider creates a secret provider preloaded with verified secret paths.
func NewInMemorySecretProvider() *InMemorySecretProvider {
	return &InMemorySecretProvider{
		secrets: make(map[string]SecretLease),
		now:     time.Now,
	}
}

// SetSecret sets or updates a secret lease at a specific path.
func (p *InMemorySecretProvider) SetSecret(path string, data map[string]string, ttl time.Duration) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("secret path is required")
	}
	if !strings.HasPrefix(path, "kv/production/") && !strings.HasPrefix(path, "kv/test/") {
		return fmt.Errorf("secret path %q must reside under kv/production/ or kv/test/", path)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := p.now().UTC()
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = now.Add(ttl)
	}

	dataCopy := make(map[string]string, len(data))
	for k, v := range data {
		dataCopy[k] = v
	}

	existing := p.secrets[path]
	version := existing.Version + 1

	p.secrets[path] = SecretLease{
		Path:      path,
		Version:   version,
		Data:      dataCopy,
		LeaseTTL:  ttl,
		ExpiresAt: expiresAt,
	}
	return nil
}

// GetSecret fetches a secret lease by path. Fails closed if the secret is missing or expired.
func (p *InMemorySecretProvider) GetSecret(ctx context.Context, path string) (SecretLease, error) {
	if strings.TrimSpace(path) == "" {
		return SecretLease{}, errors.New("secret path is required")
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	lease, exists := p.secrets[path]
	if !exists {
		return SecretLease{}, fmt.Errorf("secret path not found: %s", path)
	}

	if !lease.ExpiresAt.IsZero() && p.now().UTC().After(lease.ExpiresAt) {
		return SecretLease{}, fmt.Errorf("secret lease expired for path: %s", path)
	}

	// Return defensive copy
	dataCopy := make(map[string]string, len(lease.Data))
	for k, v := range lease.Data {
		dataCopy[k] = v
	}
	lease.Data = dataCopy

	return lease, nil
}

// DynamicSecretManager coordinates background secret refresh, caching, and atomic rotation.
type DynamicSecretManager struct {
	provider SecretProvider
	mu       sync.RWMutex
	cache    map[string]SecretLease
	now      func() time.Time
}

func NewDynamicSecretManager(provider SecretProvider) (*DynamicSecretManager, error) {
	if provider == nil {
		return nil, errors.New("secret provider is required")
	}
	return &DynamicSecretManager{
		provider: provider,
		cache:    make(map[string]SecretLease),
		now:      time.Now,
	}, nil
}

// ResolveSecret gets a cached secret or transparently fetches a new lease from the provider.
func (m *DynamicSecretManager) ResolveSecret(ctx context.Context, path string) (SecretLease, error) {
	m.mu.RLock()
	lease, ok := m.cache[path]
	now := m.now().UTC()
	// Cache hit if lease not expired and has at least 15s safety margin
	if ok && (lease.ExpiresAt.IsZero() || now.Before(lease.ExpiresAt.Add(-15*time.Second))) {
		m.mu.RUnlock()
		return lease, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check after lock
	now = m.now().UTC()
	if lease, ok = m.cache[path]; ok && (lease.ExpiresAt.IsZero() || now.Before(lease.ExpiresAt.Add(-15*time.Second))) {
		return lease, nil
	}

	freshLease, err := m.provider.GetSecret(ctx, path)
	if err != nil {
		// If fetch fails but we have an unexpired cached lease, gracefully use it
		if ok && now.Before(lease.ExpiresAt) {
			return lease, nil
		}
		return SecretLease{}, fmt.Errorf("failed to resolve dynamic secret for %s: %w", path, err)
	}

	m.cache[path] = freshLease
	return freshLease, nil
}
