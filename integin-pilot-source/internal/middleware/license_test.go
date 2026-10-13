package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLicenseFromContextMissing(t *testing.T) {
	_, ok := LicenseFromContext(context.Background())
	if ok {
		t.Fatal("expected no license in empty context")
	}
}

func TestLicenseFromContextPresent(t *testing.T) {
	info := LicenseInfo{Valid: true, Tier: "pro"}
	ctx := context.WithValue(context.Background(), licenseKey{}, info)
	result, ok := LicenseFromContext(ctx)
	if !ok {
		t.Fatal("expected license in context")
	}
	if result.Tier != "pro" {
		t.Fatalf("expected tier pro, got %s", result.Tier)
	}
}

func TestLicenseEnforcementMissingHeaders(t *testing.T) {
	middleware := LicenseEnforcement(nil)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 tenant_required, got %d", w.Code)
	}
}
