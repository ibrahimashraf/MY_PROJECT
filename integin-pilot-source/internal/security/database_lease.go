package security

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultDatabaseSecretPath is the OpenBao KV v2 production path that holds the
// runtime Postgres database lease (username/password/host/port/dbname/sslmode).
const DefaultDatabaseSecretPath = "kv/production/integin/runtime/postgres"

// ErrDatabaseLeaseExpired signals a resolved database lease whose validity has lapsed.
var ErrDatabaseLeaseExpired = errors.New("database credentials lease expired")

// DatabaseCredentials is the resolved, lease-bound Postgres credential set.
type DatabaseCredentials struct {
	Username  string
	Password  string
	Host      string
	Port      int
	Database  string
	SSLMode   string
	Version   int
	LeaseTTL  time.Duration
	ExpiresAt time.Time
}

// Expired reports whether the credentials have lapsed relative to now.
// A zero ExpiresAt means a non-expiring lease, which never reports as expired.
func (c DatabaseCredentials) Expired(now time.Time) bool {
	return !c.ExpiresAt.IsZero() && now.UTC().After(c.ExpiresAt)
}

// DSN formats the credentials as a postgres:// URL.
func (c DatabaseCredentials) DSN() (string, error) {
	if c.Username == "" || c.Password == "" {
		return "", errors.New("database credentials are incomplete")
	}
	if c.Host == "" || c.Port < 1 || c.Port > 65535 || c.Database == "" {
		return "", errors.New("database credentials are incomplete")
	}
	sslmode := c.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.Username, c.Password),
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   "/" + c.Database,
	}
	q := u.Query()
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// LeasedDatabaseConfigProvider resolves Postgres credentials from a dynamic
// secret lease (OpenBao database engine semantics) with lease expiry tracking.
//
// It fails closed: an expired or unreachable lease yields an error, never a
// connection string built from stale or partial credentials. The last resolved
// unexpired credential set is retained in memory so a temporary refresh failure
// degrades gracefully instead of cutting production at the first blip.
//
// Credential values are never logged or printed.
type LeasedDatabaseConfigProvider struct {
	manager *DynamicSecretManager
	path    string
	now     func() time.Time

	mu        sync.RWMutex
	cached    DatabaseCredentials
	cacheSeen bool
}

// NewLeasedDatabaseConfigProvider binds the provider to a secret manager and an
// OpenBao path that must live under kv/production/ or kv/test/.
func NewLeasedDatabaseConfigProvider(manager *DynamicSecretManager, path string) (*LeasedDatabaseConfigProvider, error) {
	if manager == nil {
		return nil, errors.New("dynamic secret manager is required")
	}
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("secret path is required")
	}
	if !strings.HasPrefix(path, "kv/production/") && !strings.HasPrefix(path, "kv/test/") {
		return nil, fmt.Errorf("secret path %q must reside under kv/production/ or kv/test/", path)
	}
	return &LeasedDatabaseConfigProvider{manager: manager, path: path, now: time.Now}, nil
}

// Resolve returns the current valid database credentials, tracking lease expiry.
// It caches every successful resolution and falls back to unexpired in-memory
// credentials when a refresh temporarily fails. Missing or expired leases fail
// closed.
func (p *LeasedDatabaseConfigProvider) Resolve(ctx context.Context) (DatabaseCredentials, error) {
	lease, err := p.manager.ResolveSecret(ctx, p.path)
	var creds DatabaseCredentials
	if err == nil {
		creds, err = credentialsFromLease(lease, p.now().UTC())
	}
	if err != nil {
		return p.gracefulFallback(err)
	}

	p.mu.Lock()
	p.cached = creds
	p.cacheSeen = true
	p.mu.Unlock()
	return creds, nil
}

// DSN resolves current credentials and formats a postgres:// connection string.
// Each call re-checks lease validity before returning a connection string.
func (p *LeasedDatabaseConfigProvider) DSN(ctx context.Context) (string, error) {
	creds, err := p.Resolve(ctx)
	if err != nil {
		return "", err
	}
	return creds.DSN()
}

// gracefulFallback preserves unexpired, previously resolved credentials when a
// refresh attempt fails. If nothing unexpired is held, the caller's error is
// returned unchanged (fail closed).
func (p *LeasedDatabaseConfigProvider) gracefulFallback(refreshErr error) (DatabaseCredentials, error) {
	now := p.now().UTC()
	p.mu.RLock()
	cached, seen := p.cached, p.cacheSeen
	p.mu.RUnlock()
	if seen && !cached.Expired(now) {
		return cached, nil
	}
	return DatabaseCredentials{}, refreshErr
}

// credentialsFromLease validates a lease payload into typed credentials,
// failing closed on missing or malformed fields.
func credentialsFromLease(lease SecretLease, now time.Time) (DatabaseCredentials, error) {
	if !lease.ExpiresAt.IsZero() && now.After(lease.ExpiresAt) {
		return DatabaseCredentials{}, fmt.Errorf("%w for path %s", ErrDatabaseLeaseExpired, lease.Path)
	}

	username, ok := lease.Data["username"]
	if !ok || strings.TrimSpace(username) == "" {
		return DatabaseCredentials{}, errors.New("database lease missing username")
	}
	password, ok := lease.Data["password"]
	if !ok || password == "" {
		return DatabaseCredentials{}, errors.New("database lease missing password")
	}
	host, ok := lease.Data["host"]
	if !ok || strings.TrimSpace(host) == "" {
		return DatabaseCredentials{}, errors.New("database lease missing host")
	}
	dbname, ok := lease.Data["dbname"]
	if !ok || strings.TrimSpace(dbname) == "" {
		return DatabaseCredentials{}, errors.New("database lease missing dbname")
	}
	port, err := strconv.Atoi(strings.TrimSpace(lease.Data["port"]))
	if err != nil || port < 1 || port > 65535 {
		return DatabaseCredentials{}, errors.New("database lease has no valid port")
	}
	sslmode := strings.TrimSpace(lease.Data["sslmode"])
	if sslmode == "" {
		sslmode = "disable"
	}

	return DatabaseCredentials{
		Username:  username,
		Password:  password,
		Host:      host,
		Port:      port,
		Database:  dbname,
		SSLMode:   sslmode,
		Version:   lease.Version,
		LeaseTTL:  lease.LeaseTTL,
		ExpiresAt: lease.ExpiresAt,
	}, nil
}
