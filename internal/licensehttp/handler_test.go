package licensehttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/license"
)

type mockLicenseService struct {
	validateResult license.Validation
	issueResult    license.License
	renewResult    license.License
	revokeErr      error
}

func (m *mockLicenseService) ValidateLicense(ctx context.Context, actor license.ActorContext, at time.Time) (license.Validation, error) {
	return m.validateResult, nil
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

func TestHandleValidate(t *testing.T) {
	svc := &mockLicenseService{
		validateResult: license.Validation{Valid: true, Tier: license.TierPro},
	}
	h := Handler{LicenseService: svc}

	body := `{"tenant_id":"t1","organization_id":"o1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/validate", strings.NewReader(body))
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

func TestHandleIssue(t *testing.T) {
	svc := &mockLicenseService{
		issueResult: license.License{ID: "lic-1", Tier: license.TierStarter},
	}
	h := Handler{LicenseService: svc}

	body := `{"tenant_id":"t1","organization_id":"o1","actor_id":"a1","tier":"starter","max_inspectors":5,"max_inspections_per_month":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestHandleRevoke(t *testing.T) {
	svc := &mockLicenseService{}
	h := Handler{LicenseService: svc}

	body := `{"tenant_id":"t1","organization_id":"o1","actor_id":"a1","license_id":"lic-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/revoke", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleNotFound(t *testing.T) {
	svc := &mockLicenseService{}
	h := Handler{LicenseService: svc}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
