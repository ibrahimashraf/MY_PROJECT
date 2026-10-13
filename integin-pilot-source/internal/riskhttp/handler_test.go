package riskhttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"integin/internal/riskhttp"
	"integin/pkg/riskengine"
)

func TestRiskHTTP_Evaluate(t *testing.T) {
	handler := riskhttp.NewHandler()

	payload := map[string]interface{}{
		"inputs": riskengine.OperationalRiskInputs{
			AssetDID:        "did:integin:asset:crane-44",
			DamageIndex:     0.15,
			CriticalDefects: 0,
			MajorDefects:    1,
			ObservationTime: time.Now().UTC(),
		},
	}
	raw, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/risk/evaluate", bytes.NewReader(raw))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res riskengine.RiskScoreResult
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if res.AssetDID != "did:integin:asset:crane-44" {
		t.Errorf("expected asset did did:integin:asset:crane-44, got %s", res.AssetDID)
	}
}
