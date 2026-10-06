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
