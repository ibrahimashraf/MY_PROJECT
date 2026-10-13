package shortlinkhttp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/shortlink"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/shortlinkhttp"
	"integin/internal/shortlinksvc"
)

// mockTokenValidator mocks oidcauth.TokenValidator
type mockTokenValidator struct {
	validateFn func(ctx context.Context, rawToken string) (oidcauth.Principal, error)
}

func (m *mockTokenValidator) Validate(ctx context.Context, rawToken string) (oidcauth.Principal, error) {
	if m.validateFn != nil {
		return m.validateFn(ctx, rawToken)
	}
	return oidcauth.Principal{}, errors.New("unexpected call")
}

// mockResolver mocks identity.Resolver
type mockResolver struct {
	resolveFn func(ctx context.Context, principal identity.PrincipalKey) (identity.Membership, error)
}

func (m *mockResolver) Resolve(ctx context.Context, principal identity.PrincipalKey) (identity.Membership, error) {
	if m.resolveFn != nil {
		return m.resolveFn(ctx, principal)
	}
	return identity.Membership{}, errors.New("unexpected call")
}

func TestShortLinkHandler_Security_UnauthenticatedBlocked(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(ctx context.Context, rawToken string) (oidcauth.Principal, error) {
			return oidcauth.Principal{}, errors.New("invalid_token")
		},
	}
	resolver := &mockResolver{}

	svc := shortlinksvc.New(nil, 6, "")
	handler := shortlinkhttp.NewWithAuth(svc, validator, resolver)

	// Attempting to access admin endpoint without Authorization header
	req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/shortlinks/hmac/secrets", nil)
	// Attacker tries header spoofing
	req.Header.Set("X-Tenant-ID", "victim-tenant")
	req.Header.Set("X-Organization-ID", "victim-org")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestShortLinkHandler_Security_CrossTenantTakeoverPrevented(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(ctx context.Context, rawToken string) (oidcauth.Principal, error) {
			if rawToken == "valid-attacker-token" {
				return oidcauth.Principal{
					Issuer:  "https://auth.integin.local",
					Subject: "attacker-user-id",
				}, nil
			}
			return oidcauth.Principal{}, errors.New("invalid_token")
		},
	}
	resolver := &mockResolver{
		resolveFn: func(ctx context.Context, principal identity.PrincipalKey) (identity.Membership, error) {
			// Resolved membership proves attacker belongs to attacker-tenant
			return identity.Membership{
				ActorID:        "actor-attacker",
				TenantID:       "attacker-tenant",
				OrganizationID: "attacker-org",
			}, nil
		},
	}

	svc := shortlinksvc.New(nil, 6, "")
	handler := shortlinkhttp.NewWithAuth(svc, validator, resolver)

	// Attacker authenticates with their own valid token, but attempts to query or create rules
	// in victim-tenant by supplying tenant_id=victim-tenant in the payload / query.
	body := strings.NewReader(`{
		"tenant_id": "victim-tenant",
		"name": "Malicious Rule",
		"type": "scan_burst",
		"config": {"threshold": 10}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/shortlinks/anomaly/rules", body)
	req.Header.Set("Authorization", "Bearer valid-attacker-token")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Must be rejected with 403 Forbidden due to tenant mismatch
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden when requesting cross-tenant access, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "cross-tenant access prohibited") {
		t.Fatalf("expected 'cross-tenant access prohibited' error message, got: %s", rec.Body.String())
	}
}

func TestShortLinkHandler_MountNormalization(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(ctx context.Context, rawToken string) (oidcauth.Principal, error) {
			return oidcauth.Principal{
				Issuer:  "https://auth.integin.local",
				Subject: "user-1",
			}, nil
		},
	}
	resolver := &mockResolver{
		resolveFn: func(ctx context.Context, principal identity.PrincipalKey) (identity.Membership, error) {
			return identity.Membership{
				ActorID:        "actor-1",
				TenantID:       "tenant-1",
				OrganizationID: "org-1",
			}, nil
		},
	}

	svc := shortlinksvc.New(nil, 6, "")
	handler := shortlinkhttp.NewWithAuth(svc, validator, resolver)

	// Mount path in server mux is /api/v1/admin/shortlinks/...
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/shortlinks/anomaly/rules", nil)
	req.Header.Set("Authorization", "Bearer token")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Since DB is nil in svc, it should reach handler and return 500 (db error) or succeed,
	// but NOT 404 (not found).
	if rec.Code == http.StatusNotFound {
		t.Fatalf("expected path normalization to route successfully, got 404: %s", rec.Body.String())
	}
}

func TestShortLinkHandler_ValidationRejection(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(ctx context.Context, rawToken string) (oidcauth.Principal, error) {
			return oidcauth.Principal{Issuer: "https://auth.integin.local", Subject: "user-1"}, nil
		},
	}
	resolver := &mockResolver{
		resolveFn: func(ctx context.Context, principal identity.PrincipalKey) (identity.Membership, error) {
			return identity.Membership{ActorID: "actor-1", TenantID: "tenant-1", OrganizationID: "org-1"}, nil
		},
	}
	svc := shortlinksvc.New(nil, 6, "")
	handler := shortlinkhttp.NewWithAuth(svc, validator, resolver)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/shortlinks", strings.NewReader(`{"target_url":"not-a-valid-url"}`))
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid target_url, got %d: %s", rec.Code, rec.Body.String())
	}

	reqHMAC := httptest.NewRequest(http.MethodPost, "/admin/api/v1/shortlinks/hmac/secrets", strings.NewReader(`{"secret":""}`))
	reqHMAC.Header.Set("Authorization", "Bearer token")
	reqHMAC.Header.Set("Content-Type", "application/json")

	recHMAC := httptest.NewRecorder()
	handler.ServeHTTP(recHMAC, reqHMAC)

	if recHMAC.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for empty secret, got %d: %s", recHMAC.Code, recHMAC.Body.String())
	}
}

func TestShortLinkHandler_OptionalBodyValidation(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(ctx context.Context, rawToken string) (oidcauth.Principal, error) {
			return oidcauth.Principal{Issuer: "https://auth.integin.local", Subject: "user-1"}, nil
		},
	}
	resolver := &mockResolver{
		resolveFn: func(ctx context.Context, principal identity.PrincipalKey) (identity.Membership, error) {
			return identity.Membership{ActorID: "actor-1", TenantID: "tenant-1", OrganizationID: "org-1"}, nil
		},
	}
	svc := shortlinksvc.New(nil, 6, "")
	handler := shortlinkhttp.NewWithAuth(svc, validator, resolver)

	reqBad := httptest.NewRequest(http.MethodPost, "/admin/api/v1/shortlinks/export", strings.NewReader(`{malformed_json`))
	reqBad.Header.Set("Authorization", "Bearer token")
	reqBad.Header.Set("Content-Type", "application/json")

	recBad := httptest.NewRecorder()
	handler.ServeHTTP(recBad, reqBad)

	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for malformed json in optional body, got %d: %s", recBad.Code, recBad.Body.String())
	}

	reqEmpty := httptest.NewRequest(http.MethodPost, "/admin/api/v1/shortlinks/export", nil)
	reqEmpty.Header.Set("Authorization", "Bearer token")

	recEmpty := httptest.NewRecorder()
	handler.ServeHTTP(recEmpty, reqEmpty)

	if recEmpty.Code == http.StatusBadRequest {
		t.Fatalf("expected empty body to be accepted without 400, got: %s", recEmpty.Body.String())
	}
}

// failingRepo embeds the Repository interface so only the methods a test
// exercises need a body; anything else panics and fails loudly.
type failingRepo struct {
	shortlink.Repository
	err error
}

func (f failingRepo) Create(context.Context, string, string, *time.Time, *string, *string, *string, *string, *string) error {
	return f.err
}

// A handler whose repository returns a driver-flavoured error must reach the
// caller as a generic code, never as raw SQL or driver text.
func TestShortLinkHandler_InternalErrorNotLeaked(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(ctx context.Context, rawToken string) (oidcauth.Principal, error) {
			return oidcauth.Principal{Issuer: "https://auth.integin.local", Subject: "user-1"}, nil
		},
	}
	resolver := &mockResolver{
		resolveFn: func(ctx context.Context, principal identity.PrincipalKey) (identity.Membership, error) {
			return identity.Membership{ActorID: "actor-1", TenantID: "tenant-1", OrganizationID: "org-1"}, nil
		},
	}
	driverErr := errors.New(`pq: relation "short_links" does not exist`)
	svc := shortlinksvc.New(failingRepo{err: driverErr}, 6, "")
	handler := shortlinkhttp.NewWithAuth(svc, validator, resolver)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/shortlinks", strings.NewReader(`{"target_url":"https://example.com"}`))
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, leak := range []string{"pq:", "relation", "short_links", "SQL", "driver"} {
		if strings.Contains(body, leak) {
			t.Fatalf("internal error detail %q leaked to caller: %s", leak, body)
		}
	}
	if !strings.Contains(body, "internal_error") {
		t.Fatalf("expected generic internal_error code, got: %s", body)
	}
}
