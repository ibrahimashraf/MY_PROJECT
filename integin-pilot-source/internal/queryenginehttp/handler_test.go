package queryenginehttp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"integin/internal/queryenginehttp"
	"integin/pkg/queryengine"
)

func newTestHandler(t *testing.T) *queryenginehttp.Handler {
	t.Helper()
	reg, err := queryengine.NewSchemaRegistry(queryenginehttp.CanonicalSchema())
	if err != nil {
		t.Fatalf("failed to build schema registry: %v", err)
	}
	return queryenginehttp.NewHandler(reg, nil)
}

func TestHandler_ListTables(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var res struct {
		Tables []string `json:"tables"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(res.Tables) != 4 {
		t.Fatalf("expected 4 tables, got %d: %+v", len(res.Tables), res.Tables)
	}
}

func TestHandler_DryRunQuery(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query/work_order?status=eq.OPEN&limit=10", nil)
	req.Header.Set("X-Tenant-ID", "tenant-100")
	req.Header.Set("X-Organization-ID", "org-200")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Table  string `json:"table"`
		DryRun bool   `json:"dry_run"`
		SQL    string `json:"sql"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !res.DryRun || res.Table != "work_order" {
		t.Fatalf("unexpected response: %+v", res)
	}
	if !strings.Contains(res.SQL, "tenant_id = $1 AND organization_id = $2") {
		t.Fatalf("SQL does not contain tenant guard: %s", res.SQL)
	}
	if !strings.Contains(res.SQL, "status = $3") {
		t.Fatalf("SQL does not contain filter: %s", res.SQL)
	}
}

func TestHandler_MissingTenantHeaders(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query/work_order", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestHandler_UnknownTable(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query/secret_passwords", nil)
	req.Header.Set("X-Tenant-ID", "tenant-100")
	req.Header.Set("X-Organization-ID", "org-200")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_UnauthorizedColumn(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query/work_order?password=eq.hacked", nil)
	req.Header.Set("X-Tenant-ID", "tenant-100")
	req.Header.Set("X-Organization-ID", "org-200")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/query/work_order", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed, got %d", rec.Code)
	}
}

func TestHandler_NilRegistry(t *testing.T) {
	h := queryenginehttp.NewHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/query", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable, got %d", rec.Code)
	}
}
