package reportshandler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domain "integin/internal/domain/reports"
)

type mockRepository struct {
	configs      []domain.ReportConfig
	config       domain.ReportConfig
	createErr    error
	updateErr    error
	deleteErr    error
	genReport    domain.GeneratedReport
	genErr       error
	genCSVData   func() ([]string, [][]string, error)
	reports      []domain.GeneratedReport
	getReport    domain.GeneratedReport
	getReportErr error
}

func (m *mockRepository) ListConfigs(ctx context.Context, tenantID, orgID string) ([]domain.ReportConfig, error) {
	return m.configs, nil
}

func (m *mockRepository) GetConfig(ctx context.Context, tenantID, orgID, id string) (domain.ReportConfig, error) {
	return m.config, nil
}

func (m *mockRepository) CreateConfig(ctx context.Context, config domain.ReportConfig) (domain.ReportConfig, error) {
	return config, m.createErr
}

func (m *mockRepository) UpdateConfig(ctx context.Context, config domain.ReportConfig) (domain.ReportConfig, error) {
	return config, m.updateErr
}

func (m *mockRepository) DeleteConfig(ctx context.Context, tenantID, orgID, id string) error {
	return m.deleteErr
}

func (m *mockRepository) GenerateReport(ctx context.Context, req domain.GenerateRequest, tenantID, orgID string) (domain.GeneratedReport, error) {
	return m.genReport, m.genErr
}

func (m *mockRepository) GenerateCSVData(ctx context.Context, req domain.GenerateRequest, tenantID, orgID string) ([]string, [][]string, error) {
	if m.genCSVData != nil {
		return m.genCSVData()
	}
	return []string{"col1", "col2"}, [][]string{{"a", "b"}}, nil
}

func (m *mockRepository) ListReports(ctx context.Context, tenantID, orgID string) ([]domain.GeneratedReport, error) {
	return m.reports, nil
}

func (m *mockRepository) GetReport(ctx context.Context, tenantID, orgID, id string) (domain.GeneratedReport, error) {
	return m.getReport, m.getReportErr
}

func TestHandleListConfigs(t *testing.T) {
	repo := &mockRepository{
		configs: []domain.ReportConfig{
			{ID: "c1", Name: "Test Report", Type: domain.ReportInspectionSummary},
		},
	}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/configs", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result []domain.ReportConfig
	json.NewDecoder(w.Body).Decode(&result)
	if len(result) != 1 {
		t.Fatalf("expected 1 config, got %d", len(result))
	}
}

func TestHandleListConfigsMissingHeaders(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/configs", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleCreateConfig(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{Repository: repo}

	body := `{"name":"My Report","type":"inspection_summary","format":"csv","schedule":{"frequency":"daily","time":"08:00","recipients":["admin@test.com"]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/configs", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestHandleUpdateConfig(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{Repository: repo}

	body := `{"name":"Updated Report","type":"work_order_status"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/reports/configs/c1", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleDeleteConfig(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/reports/configs/c1", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleGenerate(t *testing.T) {
	repo := &mockRepository{
		genReport: domain.GeneratedReport{
			ID:     "rpt-1",
			Status: domain.ReportStatusReady,
		},
	}
	h := Handler{Repository: repo}

	body := `{"config_id":"c1","date_from":"2024-01-01","date_to":"2024-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/generate", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestHandleGenerateMissingConfigID(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{Repository: repo}

	body := `{"date_from":"2024-01-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/generate", strings.NewReader(body))
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleListReports(t *testing.T) {
	repo := &mockRepository{
		reports: []domain.GeneratedReport{
			{ID: "r1", Status: domain.ReportStatusReady, RowCount: 10},
		},
	}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleDownload(t *testing.T) {
	repo := &mockRepository{
		getReport: domain.GeneratedReport{
			ID:        "r1",
			ConfigID:  "c1",
			Status:    domain.ReportStatusReady,
			CreatedAt: time.Now(),
		},
		config: domain.ReportConfig{
			ID:   "c1",
			Type: domain.ReportInspectionSummary,
		},
	}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/r1/download", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
		t.Fatalf("expected text/csv, got %s", ct)
	}
}

func TestHandleDownloadNotReady(t *testing.T) {
	repo := &mockRepository{
		getReport: domain.GeneratedReport{
			ID:     "r1",
			Status: domain.ReportStatusPending,
		},
	}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/r1/download", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-Organization-ID", "o1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestHandleNotFound(t *testing.T) {
	repo := &mockRepository{}
	h := Handler{Repository: repo}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
