package reinspectionhttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/domain/reinspection"
	"integin/internal/reinspectionhttp"
	"integin/internal/shared/types"
)

func TestReinspectionHTTP_Lifecycle(t *testing.T) {
	handler := reinspectionhttp.NewHandler(nil)

	// 1. Issue CAR
	issuePayload := map[string]interface{}{
		"car_id":               "car-101",
		"asset_id":             "asset-crane-5",
		"source_inspection_id": "insp-901",
		"source_finding_id":    "find-wire-frayed",
		"severity":             types.SeverityCritical,
		"description":          "Wire rope broken strands exceed discard criteria",
	}
	rawIssue, _ := json.Marshal(issuePayload)

	reqIssue := httptest.NewRequest(http.MethodPost, "/api/v1/car/issue", bytes.NewReader(rawIssue))
	reqIssue.Header.Set("X-Tenant-ID", "tenant-alpha")
	reqIssue.Header.Set("X-Organization-ID", "org-beta")
	recIssue := httptest.NewRecorder()

	handler.ServeHTTP(recIssue, reqIssue)
	if recIssue.Code != http.StatusCreated {
		t.Fatalf("issue expected 201 Created, got %d: %s", recIssue.Code, recIssue.Body.String())
	}

	// 2. Remediate CAR
	remediatePayload := map[string]interface{}{
		"car_id":        "car-101",
		"notes":         "Replaced entire hoist wire rope with new certified spool",
		"evidence_refs": []string{"ev-spool-cert-331"},
		"remediated_by": "tech-lead-07",
	}
	rawRemediate, _ := json.Marshal(remediatePayload)

	reqRemediate := httptest.NewRequest(http.MethodPost, "/api/v1/car/remediate", bytes.NewReader(rawRemediate))
	reqRemediate.Header.Set("X-Tenant-ID", "tenant-alpha")
	reqRemediate.Header.Set("X-Organization-ID", "org-beta")
	recRemediate := httptest.NewRecorder()

	handler.ServeHTTP(recRemediate, reqRemediate)
	if recRemediate.Code != http.StatusOK {
		t.Fatalf("remediate expected 200 OK, got %d: %s", recRemediate.Code, recRemediate.Body.String())
	}

	// 3. Verify Outcome (Pass -> Closed)
	verifyPayload := map[string]interface{}{
		"car_id":          "car-101",
		"reinspection_id": "re-insp-202",
		"outcome":         types.VerdictPass,
		"verifiedBy":      "chief-inspector-01",
	}
	rawVerify, _ := json.Marshal(verifyPayload)

	reqVerify := httptest.NewRequest(http.MethodPost, "/api/v1/car/verify", bytes.NewReader(rawVerify))
	reqVerify.Header.Set("X-Tenant-ID", "tenant-alpha")
	reqVerify.Header.Set("X-Organization-ID", "org-beta")
	recVerify := httptest.NewRecorder()

	handler.ServeHTTP(recVerify, reqVerify)
	if recVerify.Code != http.StatusOK {
		t.Fatalf("verify expected 200 OK, got %d: %s", recVerify.Code, recVerify.Body.String())
	}

	var closedCAR reinspection.CorrectiveActionRequest
	if err := json.NewDecoder(recVerify.Body).Decode(&closedCAR); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if closedCAR.Status != reinspection.CARVerifiedClosed {
		t.Errorf("expected status VERIFIED_CLOSED, got %s", closedCAR.Status)
	}
}

func TestReinspectionHTTP_AutoIssueOnCriticalDefects(t *testing.T) {
	handler := reinspectionhttp.NewHandler(nil)

	autoIssuePayload := map[string]interface{}{
		"source_inspection_id": "insp-shackle-99",
		"findings": []map[string]interface{}{
			{
				"finding_id":  "f-pin-wear",
				"asset_id":    "shackle-alpha",
				"severity":    types.SeverityCritical,
				"description": "Pin diameter reduction exceeds 10% per BS EN 13889",
			},
			{
				"finding_id":  "f-sling-cut",
				"asset_id":    "sling-bravo",
				"severity":    types.SeverityMajor,
				"description": "Core yarn exposed per BS EN 1492-2",
			},
			{
				"finding_id":  "f-cosmetic",
				"asset_id":    "paint-chassis",
				"severity":    types.SeverityMinor,
				"description": "Minor surface paint fading",
			},
		},
	}
	raw, _ := json.Marshal(autoIssuePayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/car/auto-issue", bytes.NewReader(raw))
	req.Header.Set("X-Tenant-ID", "tenant-offshore")
	req.Header.Set("X-Organization-ID", "org-marine")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("auto-issue expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		SourceInspectionID string                                `json:"source_inspection_id"`
		IssuedCount        int                                   `json:"issued_count"`
		CARs               []reinspection.CorrectiveActionRequest `json:"cars"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// Minor defects must NOT trigger CARs, only Critical & Major
	if res.IssuedCount != 2 {
		t.Fatalf("expected 2 CARs issued (Critical and Major), got %d", res.IssuedCount)
	}
	if len(res.CARs) != 2 {
		t.Fatalf("expected 2 cars in payload, got %d", len(res.CARs))
	}
	for _, car := range res.CARs {
		if car.Status != reinspection.CAROpen {
			t.Errorf("expected new CAR to be OPEN, got %s", car.Status)
		}
	}
}

