package jurisdictionhttp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/jurisdictionhttp"
	"integin/pkg/jurisdictions"
	"integin/pkg/jurisdictions/adapters"
)

func newTestHandler(t *testing.T) *jurisdictionhttp.Handler {
	t.Helper()
	reg, err := jurisdictions.NewRegistry(jurisdictions.TopJurisdictions())
	if err != nil {
		t.Fatalf("failed to build test registry: %v", err)
	}
	return jurisdictionhttp.NewHandler(reg)
}

func TestHandler_ListJurisdictions(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jurisdictions", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var profiles []jurisdictions.CountryProfile
	if err := json.NewDecoder(rec.Body).Decode(&profiles); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(profiles) < 8 {
		t.Fatalf("expected at least 8 seed profiles, got %d", len(profiles))
	}
}

func TestHandler_GetByISO2(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jurisdictions/SA", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var profile jurisdictions.CountryProfile
	if err := json.NewDecoder(rec.Body).Decode(&profile); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if profile.ISO2 != "SA" || profile.ISO3 != "SAU" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestHandler_GetByISO3(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jurisdictions/ARE", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var profile jurisdictions.CountryProfile
	if err := json.NewDecoder(rec.Body).Decode(&profile); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if profile.ISO2 != "AE" || profile.CurrencyCode != "AED" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestHandler_ComplianceChecks(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jurisdictions/SA/compliance", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var results []adapters.Result
	if err := json.NewDecoder(rec.Body).Decode(&results); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected compliance results for SA, got none")
	}
	for _, res := range results {
		if res.HasFailures() {
			t.Errorf("expected passing result for %s, got failures: %+v", res.Pillar, res.Findings)
		}
	}
}

func TestHandler_NotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jurisdictions/ZZ", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", rec.Code)
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jurisdictions", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed, got %d", rec.Code)
	}
}

func TestHandler_NilRegistry(t *testing.T) {
	h := jurisdictionhttp.NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jurisdictions", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable, got %d", rec.Code)
	}
}
