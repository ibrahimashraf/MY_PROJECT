package advisoryhttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"integin/internal/advisory"
)

const testTenant = "tenant-a"
const testOrg = "org-a"

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	registry := advisory.NewRegistry()
	if err := registry.RegisterModel(advisory.ModelRegistration{
		ID:        "model-lens-1",
		Provider:  "local-mock",
		ModelName: "deterministic-lens",
		Version:   "v1",
		Status:    advisory.ModelApproved,
		MaxTokens: 2048,
		AllowedZones: []advisory.Zone{
			advisory.ZoneMonitoring,
			advisory.ZoneNDTDefect,
		},
	}); err != nil {
		t.Fatalf("register approved model: %v", err)
	}
	if err := registry.RegisterModel(advisory.ModelRegistration{
		ID:           "model-retired",
		Provider:     "local-mock",
		ModelName:    "retired-lens",
		Version:      "v0",
		Status:       advisory.ModelDeprecated,
		MaxTokens:    1024,
		AllowedZones: []advisory.Zone{advisory.ZoneMonitoring},
	}); err != nil {
		t.Fatalf("register deprecated model: %v", err)
	}
	return NewHandler(registry)
}

func perform(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("X-Tenant-ID", testTenant)
	request.Header.Set("X-Organization-ID", testOrg)
	writer := httptest.NewRecorder()
	h.ServeHTTP(writer, request)
	return writer
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(recorder.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, recorder.Body.String())
	}
}

func TestGetModelsReturnsOnlyApproved(t *testing.T) {
	writer := perform(t, newTestHandler(t), http.MethodGet, "/api/v1/advisory/models", "")

	if writer.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", writer.Code, writer.Body.String())
	}
	var payload struct {
		Models []modelView `json:"models"`
	}
	decode(t, writer, &payload)
	if len(payload.Models) != 1 {
		t.Fatalf("expected 1 approved model, got %d", len(payload.Models))
	}
	model := payload.Models[0]
	if model.ID != "model-lens-1" || model.Status != "APPROVED" || model.MaxTokens != 2048 {
		t.Fatalf("unexpected model view: %+v", model)
	}
	if len(model.AllowedZones) != 2 {
		t.Fatalf("expected 2 allowed zones, got %d", len(model.AllowedZones))
	}
}

func TestExtractReturnsDocumentExtract(t *testing.T) {
	body := `{"text":"Title: Lifting Standard 2026\nIssuing authority: ABS\nEffective date: 2026-01-01\nClause 4.2 inspection intervals\nChecklist item 7"}`
	writer := perform(t, newTestHandler(t), http.MethodPost, "/api/v1/advisory/extract", body)

	if writer.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", writer.Code, writer.Body.String())
	}
	var extracted extractView
	decode(t, writer, &extracted)
	if extracted.Title != "Lifting Standard 2026" {
		t.Fatalf("unexpected title: %q", extracted.Title)
	}
	if extracted.Authority != "ABS" {
		t.Fatalf("unexpected authority: %q", extracted.Authority)
	}
	if len(extracted.Clauses) != 1 || extracted.Clauses[0] != "Clause 4.2 inspection intervals" {
		t.Fatalf("unexpected clauses: %+v", extracted.Clauses)
	}
	if extracted.TenantID != testTenant || extracted.Blocking {
		t.Fatalf("unexpected extract view: tenant=%q blocking=%v", extracted.TenantID, extracted.Blocking)
	}
}

func TestExtractAllowsApprovedZone(t *testing.T) {
	writer := perform(t, newTestHandler(t), http.MethodPost, "/api/v1/advisory/extract",
		`{"text":"Title: NDT guidance","zone":"NDT_DEFECT"}`)

	if writer.Code != http.StatusOK {
		t.Fatalf("expected 200 for approved zone, got %d (body=%s)", writer.Code, writer.Body.String())
	}
}

func TestExtractRejectsNonApprovedZone(t *testing.T) {
	for _, zone := range []string{"VERDICT", "CERTIFICATE", "CALIBRATION", "AUTHORIZATION", "SYNC_SECURITY", "PUBLIC_QR", "INSPECTION_APPROVAL"} {
		writer := perform(t, newTestHandler(t), http.MethodPost, "/api/v1/advisory/extract",
			`{"text":"Title: anything","zone":"`+zone+`"}`)

		if writer.Code != http.StatusBadRequest {
			t.Fatalf("zone %s: expected 400, got %d (body=%s)", zone, writer.Code, writer.Body.String())
		}
	}
}

func TestExtractRejectsBlocking(t *testing.T) {
	writer := perform(t, newTestHandler(t), http.MethodPost, "/api/v1/advisory/extract",
		`{"text":"Title: anything","blocking":true}`)

	if writer.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", writer.Code, writer.Body.String())
	}
}

func TestTrendsReturnsInsight(t *testing.T) {
	body := `{
		"zone":"NDT_DEFECT",
		"lens":"ndt-recurrence",
		"evidence_refs":["ev-1"],
		"observations":[
			{"id":"obs-1","zone":"NDT_DEFECT","discipline":"MPI","severity":"ADVISORY"},
			{"id":"obs-2","zone":"NDT_DEFECT","discipline":"MPI","severity":"ADVISORY"}
		]
	}`
	writer := perform(t, newTestHandler(t), http.MethodPost, "/api/v1/advisory/trends", body)

	if writer.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", writer.Code, writer.Body.String())
	}
	var insight insightView
	decode(t, writer, &insight)
	if insight.Title == "" || insight.Summary == "" || insight.Severity != "ADVISORY" {
		t.Fatalf("unexpected insight: %+v", insight)
	}
	if insight.Blocking {
		t.Fatal("insight must never be blocking")
	}
	if insight.TenantID != testTenant {
		t.Fatalf("unexpected tenant: %q", insight.TenantID)
	}
}

func TestTrendsRejectsMissingZone(t *testing.T) {
	writer := perform(t, newTestHandler(t), http.MethodPost, "/api/v1/advisory/trends",
		`{"observations":[]}`)

	if writer.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", writer.Code, writer.Body.String())
	}
}

func TestTrendsRejectsNonApprovedZone(t *testing.T) {
	writer := perform(t, newTestHandler(t), http.MethodPost, "/api/v1/advisory/trends",
		`{"zone":"CERTIFICATE","observations":[]}`)

	if writer.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", writer.Code, writer.Body.String())
	}
}

func TestTrendsRejectsBlocking(t *testing.T) {
	writer := perform(t, newTestHandler(t), http.MethodPost, "/api/v1/advisory/trends",
		`{"zone":"NDT_DEFECT","blocking":true,"observations":[]}`)

	if writer.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", writer.Code, writer.Body.String())
	}
}

func TestFeedbackSubmits(t *testing.T) {
	body := `{"id":"fb-1","audit_id":"audit-1","inspector_id":"insp-7","disposition":"ACCEPTED","notes":"confirmed"}`
	writer := perform(t, newTestHandler(t), http.MethodPost, "/api/v1/advisory/feedback", body)

	if writer.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", writer.Code, writer.Body.String())
	}
	var feedback feedbackView
	decode(t, writer, &feedback)
	if feedback.ID != "fb-1" || feedback.TenantID != testTenant || feedback.Disposition != "ACCEPTED" {
		t.Fatalf("unexpected feedback view: %+v", feedback)
	}
}

func TestFeedbackRejectsInvalidDisposition(t *testing.T) {
	writer := perform(t, newTestHandler(t), http.MethodPost, "/api/v1/advisory/feedback",
		`{"id":"fb-2","audit_id":"audit-1","inspector_id":"insp-7","disposition":"MAYBE"}`)

	if writer.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", writer.Code, writer.Body.String())
	}
}

func TestFeedbackRejectsBlocking(t *testing.T) {
	writer := perform(t, newTestHandler(t), http.MethodPost, "/api/v1/advisory/feedback",
		`{"id":"fb-3","audit_id":"audit-1","inspector_id":"insp-7","disposition":"ACCEPTED","blocking":true}`)

	if writer.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", writer.Code, writer.Body.String())
	}
}

func TestRejectsMissingTenantHeaders(t *testing.T) {
	for _, path := range []string{
		"/api/v1/advisory/models",
		"/api/v1/advisory/extract",
		"/api/v1/advisory/trends",
		"/api/v1/advisory/feedback",
	} {
		handler := newTestHandler(t)
		request := httptest.NewRequest(http.MethodGet, path, nil)
		if path != "/api/v1/advisory/models" {
			request = httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
		}
		writer := httptest.NewRecorder()
		handler.ServeHTTP(writer, request)

		if writer.Code != http.StatusBadRequest {
			t.Fatalf("path %s without tenant headers: expected 400, got %d", path, writer.Code)
		}
	}
}
