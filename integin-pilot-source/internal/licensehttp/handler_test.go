package licensehttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/license"
	"integin/internal/identity"
	"integin/internal/oidcauth"
)

type mockLicenseService struct {
	validateResult license.Validation
	issueResult    license.License
	renewResult    license.License
	listResult     []license.License
	revokeErr      error
}

func (m *mockLicenseService) ValidateLicense(ctx context.Context, actor license.ActorContext, at time.Time) (license.Validation, error) {
	return m.validateResult, nil
}

func (m *mockLicenseService) ListLicenses(ctx context.Context, actor license.ActorContext) ([]license.License, error) {
	return m.listResult, nil
}

func (m *mockLicenseService) IssueLicense(ctx context.Context, actor license.ActorContext, lic license.License) (license.License, error) {
	return m.issueResult, nil
}

func (m *mockLicenseService) RenewLicense(ctx context.Context, actor license.ActorContext, licenseID string, newExpiresAt time.Time) (license.License, error) {
	return m.renewResult, nil
}

func (m *mockLicenseService) RevokeLicense(ctx context.Context, actor license.ActorContext, licenseID string) error {
	return m.revokeErr
}

type mockValidator struct {
	principal oidcauth.Principal
	err       error
}

func (m *mockValidator) Validate(ctx context.Context, token string) (oidcauth.Principal, error) {
	return m.principal, m.err
}

type mockResolver struct {
	membership identity.Membership
	err        error
}

func (m *mockResolver) Resolve(ctx context.Context, principal identity.PrincipalKey) (identity.Membership, error) {
	return m.membership, m.err
}

func TestHandleValidate(t *testing.T) {
	svc := &mockLicenseService{
		validateResult: license.Validation{Valid: true, Tier: license.TierPro},
	}
	h := Handler{
		Validator:      &mockValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
		Resolver:       &mockResolver{membership: identity.Membership{TenantID: "t1", OrganizationID: "o1", ActorID: "a1"}},
		LicenseService: svc,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result license.Validation
	json.NewDecoder(w.Body).Decode(&result)
	if !result.Valid {
		t.Fatal("expected valid=true")
	}
}

func TestHandleValidateMissingAuth(t *testing.T) {
	svc := &mockLicenseService{}
	h := Handler{
		Validator:      &mockValidator{},
		Resolver:       &mockResolver{},
		LicenseService: svc,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleIssue(t *testing.T) {
	svc := &mockLicenseService{
		issueResult: license.License{ID: "lic-1", Tier: license.TierStarter},
	}
	h := Handler{
		Validator:      &mockValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
		Resolver:       &mockResolver{membership: identity.Membership{TenantID: "t1", OrganizationID: "o1", ActorID: "a1"}},
		LicenseService: svc,
	}

	body := `{"tier":"starter","max_inspectors":5,"max_inspections_per_month":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestHandleRevoke(t *testing.T) {
	svc := &mockLicenseService{}
	h := Handler{
		Validator:      &mockValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
		Resolver:       &mockResolver{membership: identity.Membership{TenantID: "t1", OrganizationID: "o1", ActorID: "a1"}},
		LicenseService: svc,
	}

	body := `{"license_id":"lic-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/revoke", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleNotFound(t *testing.T) {
	svc := &mockLicenseService{}
	h := Handler{
		Validator:      &mockValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
		Resolver:       &mockResolver{membership: identity.Membership{TenantID: "t1", OrganizationID: "o1", ActorID: "a1"}},
		LicenseService: svc,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleResolverError(t *testing.T) {
	svc := &mockLicenseService{}
	h := Handler{
		Validator:      &mockValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
		Resolver:       &mockResolver{err: errors.New("resolver error")},
		LicenseService: svc,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
