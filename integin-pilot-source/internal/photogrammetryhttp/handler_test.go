package photogrammetryhttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/photogrammetryhttp"
	"integin/pkg/photogrammetry"
)

func TestPhotogrammetryHTTP_Survey(t *testing.T) {
	handler := photogrammetryhttp.NewHandler()

	points := []photogrammetry.Point3D{
		{X: 0, Y: 0, Z: 5},
		{X: 10, Y: 0, Z: 5.2},
		{X: 10, Y: 10, Z: 4.8},
		{X: 0, Y: 10, Z: 5.1},
	}

	payload := map[string]interface{}{
		"asset_did":                  "did:integin:asset:stockpile-12",
		"points":                     points,
		"baseline_plane_z":           5.0,
		"max_permissible_deflection": 1.0,
	}
	raw, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/photogrammetry/survey", bytes.NewReader(raw))
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Organization-ID", "org-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res photogrammetry.VolumetricAnalysisResult
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if res.AssetDID != "did:integin:asset:stockpile-12" {
		t.Errorf("expected asset did did:integin:asset:stockpile-12, got %s", res.AssetDID)
	}
}
