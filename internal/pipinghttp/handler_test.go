package pipinghttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/pipinghttp"
	"integin/pkg/domain/piping"
)

func TestPipingHTTP_Evaluate(t *testing.T) {
	handler := pipinghttp.NewHandler()

	params := piping.CircuitParameters{
		CircuitDID:           "did:integin:circuit:hydrocarbon-01",
		OuterDiameterMM:      168.3,
		DesignPressureBar:    25.0,
		AllowableStressMPa:   137.9,
		JointEfficiencyE:     1.0,
		CoefficientY:         0.4,
		CorrosionAllowanceMM: 3.0,
		NominalThicknessMM:   7.11,
	}
	point := piping.MeasurementPoint{
		PointID:              "cml-01",
		CurrentThicknessMM:   6.5,
		PreviousThicknessMM:  6.8,
		YearsBetweenReadings: 2.0,
	}

	payload := map[string]interface{}{
		"parameters":        params,
		"measurement_point": point,
	}
	raw, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/piping/evaluate", bytes.NewReader(raw))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res piping.AssessmentResult
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !res.IsSafeToOperate {
		t.Errorf("expected safe to operate, got %+v", res)
	}
}
