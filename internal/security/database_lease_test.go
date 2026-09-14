package security

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const dbLeasePath = "kv/production/integin/runtime/postgres"

func seedDatabaseLease(t *testing.T, provider *InMemorySecretProvider, data map[string]string, ttl time.Duration) {
	t.Helper()
	if err := provider.SetSecret(dbLeasePath, data, ttl); err != nil {
		t.Fatalf("failed to set database lease: %v", err)
	}
}

// flakySecretProvider simulates transient upstream (OpenBao) failures.
type flakySecretProvider struct {
	inner SecretProvider
	fail  atomic.Bool
}

func (f *flakySecretProvider) GetSecret(ctx context.Context, path string) (SecretLease, error) {
	if f.fail.Load() {
		return SecretLease{}, errors.New("upstream unavailable")
	}
	return f.inner.GetSecret(ctx, path)
}

func newTestProvider(t *testing.T, provider SecretProvider) (*LeasedDatabaseConfigProvider, *DynamicSecretManager, *InMemorySecretProvider) {
	t.Helper()
	mgr, err := NewDynamicSecretManager(provider)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	dbProvider, err := NewLeasedDatabaseConfigProvider(mgr, dbLeasePath)
	if err != nil {
		t.Fatalf("failed to create leased database provider: %v", err)
	}
	return dbProvider, mgr, provider.(*InMemorySecretProvider)
}

func TestLeasedDatabaseCredentialsResolveAndDSN(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	provider := NewInMemorySecretProvider()
	seedDatabaseLease(t, provider, map[string]string{
		"username": "runtime_user",
		"password": "runtime_password",
		"host":     "db.internal",
		"port":     "5432",
		"dbname":   "integin",
		"sslmode":  "disable",
	}, 60*time.Second)
	mgr, err := NewDynamicSecretManager(provider)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	mgr.now = func() time.Time { return now }
	p, err := NewLeasedDatabaseConfigProvider(mgr, dbLeasePath)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	p.now = func() time.Time { return now }

	creds, err := p.Resolve(ctx)
	if err != nil {
		t.Fatalf("failed to resolve credentials: %v", err)
	}
	if creds.Username != "runtime_user" || creds.Password != "runtime_password" ||
		creds.Host != "db.internal" || creds.Port != 5432 || creds.Database != "integin" ||
		creds.SSLMode != "disable" {
		t.Fatalf("unexpected resolved credentials: username=%q host=%q port=%d db=%q sslmode=%q",
			creds.Username, creds.Host, creds.Port, creds.Database, creds.SSLMode)
	}
	if creds.Version != 1 {
		t.Fatalf("expected version 1, got %d", creds.Version)
	}
	if creds.Expired(now.Add(-time.Second)) || !creds.Expired(now.Add(61*time.Second)) {
		t.Fatal("expiry tracking misreported lease validity")
	}

	dsn, err := p.DSN(ctx)
	if err != nil {
		t.Fatalf("failed to build DSN: %v", err)
	}
	const want = "postgres://runtime_user:runtime_password@db.internal:5432/integin?sslmode=disable"
	if dsn != want {
		t.Fatalf("DSN mismatch (values omitted to avoid leaking credentials): got %d bytes, want %d bytes", len(dsn), len(want))
	}
}

func TestDSNRoundTripsThroughURLParser(t *testing.T) {
	creds := DatabaseCredentials{
		Username: "app_runtime",
		Password: "p@ss:w/rd!",
		Host:     "dbserver",
		Port:     5432,
		Database: "integin_pilot",
		SSLMode:  "require",
	}
	dsn, err := creds.DSN()
	if err != nil {
		t.Fatalf("failed to build DSN: %v", err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("DSN is not a parseable URL: %v", err)
	}
	if parsed.Scheme != "postgres" {
		t.Fatalf("unexpected scheme %q", parsed.Scheme)
	}
	if parsed.User.Username() != "app_runtime" {
		t.Fatalf("unexpected username %q", parsed.User.Username())
	}
	pw, ok := parsed.User.Password()
	if !ok || pw != "p@ss:w/rd!" {
		t.Fatalf("password did not survive round trip")
	}
	if parsed.Host != "dbserver:5432" {
		t.Fatalf("unexpected host %q", parsed.Host)
	}
	if parsed.Path != "/integin_pilot" {
		t.Fatalf("unexpected path %q", parsed.Path)
	}
	if parsed.Query().Get("sslmode") != "require" {
		t.Fatalf("unexpected sslmode %q", parsed.Query().Get("sslmode"))
	}
}

func TestLeasedDatabaseRenewalDetection(t *testing.T) {
	ctx := context.Background()
	provider := NewInMemorySecretProvider()
	seedDatabaseLease(t, provider, map[string]string{
		"username": "runtime_user", "password": "v1_pass",
		"host": "db.internal", "port": "5432", "dbname": "integin",
	}, 60*time.Second)

	mgr, err := NewDynamicSecretManager(provider)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	p, err := NewLeasedDatabaseConfigProvider(mgr, dbLeasePath)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	now := time.Now().UTC()
	mgr.now = func() time.Time { return now }
	p.now = func() time.Time { return now }

	creds, err := p.Resolve(ctx)
	if err != nil {
		t.Fatalf("initial resolve failed: %v", err)
	}
	if creds.Username != "runtime_user" {
		t.Fatalf("unexpected initial username %q", creds.Username)
	}

	seedDatabaseLease(t, provider, map[string]string{
		"username": "runtime_user_v2", "password": "v2_pass",
		"host": "db.internal", "port": "5432", "dbname": "integin",
	}, 60*time.Second)

	// Advance both clocks into the 15s refresh margin so the manager must
	// re-resolve and pick up the rotated lease.
	windUp := now.Add(50 * time.Second)
	mgr.now = func() time.Time { return windUp }
	p.now = func() time.Time { return windUp }

	creds, err = p.Resolve(ctx)
	if err != nil {
		t.Fatalf("renewal resolve failed: %v", err)
	}
	if creds.Username != "runtime_user_v2" {
		t.Fatalf("expected rotated username runtime_user_v2, got %q", creds.Username)
	}
	if creds.Version != 2 {
		t.Fatalf("expected version 2 after rotation, got %d", creds.Version)
	}
	if creds.Password != "v2_pass" {
		t.Fatal("expected rotated password")
	}
}

func TestLeasedDatabaseFailsClosedOnMissingLease(t *testing.T) {
	ctx := context.Background()
	provider := NewInMemorySecretProvider()
	mgr, err := NewDynamicSecretManager(provider)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	p, err := NewLeasedDatabaseConfigProvider(mgr, dbLeasePath)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	creds, err := p.Resolve(ctx)
	if err == nil {
		t.Fatal("expected error resolving a missing lease, got nil")
	}
	if creds.Username != "" || creds.Password != "" {
		t.Fatal("fail-closed resolve leaked partial credentials")
	}
	if dsn, err := p.DSN(ctx); err == nil || dsn != "" {
		t.Fatalf("expected DSN to fail closed, got dsn=%q err=%v", dsn, err)
	}
}

func TestLeasedDatabaseFailsClosedOnExpiredLease(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	provider := NewInMemorySecretProvider()
	seedDatabaseLease(t, provider, map[string]string{
		"username": "runtime_user", "password": "ephemeral",
		"host": "db.internal", "port": "5432", "dbname": "integin",
	}, 60*time.Second)

	mgr, err := NewDynamicSecretManager(provider)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	p, err := NewLeasedDatabaseConfigProvider(mgr, dbLeasePath)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	mgr.now = func() time.Time { return now }
	p.now = func() time.Time { return now }

	if _, err := p.Resolve(ctx); err != nil {
		t.Fatalf("initial resolve failed: %v", err)
	}

	// Slip past the lease TTL: manager cache and provider memory both lapse,
	// so resolution must fail closed rather than emit expired credentials.
	lapsed := now.Add(61 * time.Second)
	mgr.now = func() time.Time { return lapsed }
	p.now = func() time.Time { return lapsed }

	creds, err := p.Resolve(ctx)
	if err == nil {
		t.Fatal("expected error after lease expiry, got nil")
	}
	if creds.Username != "" || creds.Password != "" {
		t.Fatal("fail-closed resolve leaked expired credentials")
	}
}

func TestLeasedDatabaseGracefulFallbackPreservesUnexpiredCache(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	provider := NewInMemorySecretProvider()
	seedDatabaseLease(t, provider, map[string]string{
		"username": "runtime_user", "password": "still_valid",
		"host": "db.internal", "port": "5432", "dbname": "integin",
	}, 60*time.Second)

	flaky := &flakySecretProvider{inner: provider}
	mgr, err := NewDynamicSecretManager(flaky)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	p, err := NewLeasedDatabaseConfigProvider(mgr, dbLeasePath)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	mgr.now = func() time.Time { return now }
	p.now = func() time.Time { return now }

	if _, err := p.Resolve(ctx); err != nil {
		t.Fatalf("initial resolve failed: %v", err)
	}

	// Upstream goes dark and the manager cache lapses; only the provider's
	// own still-unexpired in-memory credentials remain. Resolution must
	// fall back gracefully instead of failing.
	flaky.fail.Store(true)
	mgr.now = func() time.Time { return now.Add(61 * time.Second) }

	creds, err := p.Resolve(ctx)
	if err != nil {
		t.Fatalf("graceful fallback failed: %v", err)
	}
	if creds.Username != "runtime_user" || creds.Password != "still_valid" {
		t.Fatalf("unexpected fallback credentials: username=%q", creds.Username)
	}

	// Once the in-memory cache also lapses, fail closed again.
	p.now = func() time.Time { return now.Add(61*time.Second + time.Second) }
	if creds, err := p.Resolve(ctx); err == nil {
		t.Fatal("expected fail-closed after cached credentials lapsed, got nil")
	} else if creds.Username != "" || creds.Password != "" {
		t.Fatal("fail-closed resolve leaked lapsed credentials")
	}
}

func TestLeasedDatabaseFailsClosedOnMalformedLease(t *testing.T) {
	ctx := context.Background()
	provider := NewInMemorySecretProvider()
	seedDatabaseLease(t, provider, map[string]string{
		"username": "runtime_user",
		"host":     "db.internal",
		"port":     "not_a_port",
	}, 60*time.Second)

	mgr, err := NewDynamicSecretManager(provider)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	p, err := NewLeasedDatabaseConfigProvider(mgr, dbLeasePath)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	creds, err := p.Resolve(ctx)
	if err == nil {
		t.Fatal("expected error for malformed lease, got nil")
	}
	if creds.Password != "" {
		t.Fatal("fail-closed resolve leaked partial credentials")
	}
	if _, err := p.DSN(ctx); err == nil {
		t.Fatal("expected DSN to fail closed on malformed lease, got nil")
	}
}

func TestLeasedDatabaseFailsClosedOnMissingSecretField(t *testing.T) {
	lease := SecretLease{
		Path: dbLeasePath,
		Data: map[string]string{
			"username": "runtime_user",
		},
	}
	_, err := credentialsFromLease(lease, time.Now().UTC())
	if err == nil {
		t.Fatal("expected error when password is missing, got nil")
	}
}

func TestCredentialSentinelMarksExpiredDuringParse(t *testing.T) {
	now := time.Now().UTC()
	lease := SecretLease{
		Path:      dbLeasePath,
		ExpiresAt: now.Add(-time.Second),
		Data: map[string]string{
			"username": "runtime_user", "password": "x",
			"host": "db.internal", "port": "5432", "dbname": "integin",
		},
	}
	_, err := credentialsFromLease(lease, now)
	if err == nil {
		t.Fatal("expected expired lease to fail closed")
	}
	if !errors.Is(err, ErrDatabaseLeaseExpired) {
		t.Fatalf("expected ErrDatabaseLeaseExpired, got %v", err)
	}
}

func TestLeasedDatabaseProviderRequiresManagerAndPath(t *testing.T) {
	if _, err := NewLeasedDatabaseConfigProvider(nil, dbLeasePath); err == nil {
		t.Fatal("expected error for nil manager, got nil")
	}
	mgr, err := NewDynamicSecretManager(NewInMemorySecretProvider())
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	if _, err := NewLeasedDatabaseConfigProvider(mgr, ""); err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
	if _, err := NewLeasedDatabaseConfigProvider(mgr, "kv/staging/integin"); err == nil {
		t.Fatal("expected error for path outside kv/production/ or kv/test/, got nil")
	}
}

func TestLeasedDatabaseConcurrentResolution(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	provider := NewInMemorySecretProvider()
	seedDatabaseLease(t, provider, map[string]string{
		"username": "runtime_user", "password": "rotating",
		"host": "db.internal", "port": "5432", "dbname": "integin",
	}, 5*time.Minute)

	flaky := &flakySecretProvider{inner: provider}
	mgr, err := NewDynamicSecretManager(flaky)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	p, err := NewLeasedDatabaseConfigProvider(mgr, dbLeasePath)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	mgr.now = func() time.Time { return now }
	p.now = func() time.Time { return now }

	if _, err := p.Resolve(ctx); err != nil {
		t.Fatalf("warm-up resolve failed: %v", err)
	}

	const workers = 32
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	successes := make(chan string, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Randomly knock upstream dark mid-flight.
			if i%3 == 0 {
				flaky.fail.Store(true)
			}
			creds, err := p.Resolve(ctx)
			if err != nil {
				errs <- err
				return
			}
			successes <- creds.Username
			dsn, err := p.DSN(ctx)
			if err != nil {
				errs <- err
				return
			}
			if strings.TrimSpace(dsn) == "" {
				errs <- errors.New("empty dsn from concurrent resolution")
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	close(successes)
	for err := range errs {
		t.Fatalf("concurrent resolution failed: %v", err)
	}
	for username := range successes {
		if username != "runtime_user" {
			t.Fatalf("unexpected username %q under concurrency", username)
		}
	}
}
