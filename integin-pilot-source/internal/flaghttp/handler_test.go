package flaghttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"integin/internal/flagpg"
	"integin/internal/shared/featureflags"
)

type mockFlagRepo struct {
	overrides []flagpg.OverrideRecord
}

func (m *mockFlagRepo) ListOverrides(ctx context.Context, tenantID, orgID string) ([]flagpg.OverrideRecord, error) {
	return m.overrides, nil
}

func (m *mockFlagRepo) GetOverride(ctx context.Context, tenantID, orgID string, key featureflags.Key, scope featureflags.Scope, scopeID string) (flagpg.OverrideRecord, bool, error) {
	return flagpg.OverrideRecord{}, false, nil
}

func (m *mockFlagRepo) UpsertOverride(ctx context.Context, tenantID, orgID string, rec flagpg.OverrideRecord) (flagpg.OverrideRecord, error) {
	return rec, nil
}

func (m *mockFlagRepo) DeleteOverride(ctx context.Context, tenantID, orgID string, key featureflags.Key, scope featureflags.Scope, scopeID string) error {
	return nil
}

func TestHandleList(t *testing.T) {
	repo := &mockFlagRepo{overrides: []flagpg.OverrideRecord{{FlagKey: "test_flag"}}}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feature-flags", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleListMissingHeaders(t *testing.T) {
	repo := &mockFlagRepo{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feature-flags", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleUpsert(t *testing.T) {
	repo := &mockFlagRepo{}
	h := Handler{Repository: repo}

	body := `{"flag_key":"test_flag","scope":"ORGANIZATION","scope_id":"org-1","state":"ENABLED","actor_id":"a1"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/feature-flags", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleDelete(t *testing.T) {
	repo := &mockFlagRepo{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/feature-flags/test_flag/ORGANIZATION/org-1", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
