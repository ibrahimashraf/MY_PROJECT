package analyticshttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/domain/analytics"
)

type mockRepository struct {
	dashboard analytics.DashboardResponse
	err       error
}

func (m *mockRepository) GetDashboard(ctx context.Context, req analytics.DashboardRequest) (analytics.DashboardResponse, error) {
	return m.dashboard, m.err
}

func TestHandleDashboardSuccess(t *testing.T) {
	repo := &mockRepository{
		dashboard: analytics.DashboardResponse{
			Summary: []analytics.KPI{
				{Name: "total_work_orders", Value: 10},
			},
		},
	}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/dashboard", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result analytics.DashboardResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Summary) != 1 {
		t.Fatalf("expected 1 summary KPI, got %d", len(result.Summary))
	}
}

func TestHandleDashboardMissingHeaders(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/dashboard", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleDashboardMissingOrgHeader(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/dashboard", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleDashboardWithDateRange(t *testing.T) {
	repo := &mockRepository{
		dashboard: analytics.DashboardResponse{
			Summary: []analytics.KPI{
				{Name: "total_inspections", Value: 25},
			},
		},
	}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/dashboard?from=2024-01-01T00:00:00Z&to=2024-01-31T23:59:59Z", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleDashboardInvalidFromParam(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/dashboard?from=not-a-date", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleDashboardInvalidToParam(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/dashboard?to=bad-date", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleDashboardMethodNotAllowed(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/dashboard", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleDashboardWithFilters(t *testing.T) {
	repo := &mockRepository{
		dashboard: analytics.DashboardResponse{
			InspectorPerformance: []analytics.InspectorPerformance{
				{InspectorID: "i1", Total: 10, PassCount: 8, FailCount: 2, PassRate: 80},
			},
		},
	}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/dashboard?inspector_id=i1&asset_type=pump", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleDashboardEmptyResponse(t *testing.T) {
	repo := &mockRepository{
		dashboard: analytics.DashboardResponse{
			Summary: []analytics.KPI{},
		},
	}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/dashboard", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result analytics.DashboardResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Summary) != 0 {
		t.Fatalf("expected 0 summary KPIs, got %d", len(result.Summary))
	}
}

func TestHandleDashboardRepositoryError(t *testing.T) {
	repo := &mockRepository{
		err: context.DeadlineExceeded,
	}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/dashboard", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
