package liftviewexport

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/pkg/cad/dxf"
	"integin/pkg/cad/engine"
)

func simRequest() Request {
	return Request{
		Crane1: engine.CraneKinematics{
			BasePosition:       dxf.Point3D{X: -15},
			BoomLengthMeters:   30,
			BoomAngleDeg:       65,
			SlewAngleDeg:       15,
			CounterweightTonne: 20,
			OutriggerSpreadXM:  8.5,
			OutriggerSpreadZM:  8.5,
			ChassisWeightTonne: 60,
		},
		Crane2: engine.CraneKinematics{
			BasePosition:       dxf.Point3D{X: 15},
			BoomLengthMeters:   28,
			BoomAngleDeg:       60,
			SlewAngleDeg:       -20,
			CounterweightTonne: 20,
			OutriggerSpreadXM:  8.5,
			OutriggerSpreadZM:  8.5,
			ChassisWeightTonne: 60,
		},
		TotalLoadT:  45,
		CogOffsetM:  -1,
		MatAreaM2:   3.0,
		AllowableKP: 220,
		Crane1CapT:  40,
		Crane2CapT:  40,
		SlingAngleD: 60,
		SlingMBLT:   30,
		RiggingT:    2,
		Pages:       8,
	}
}

func post(t *testing.T, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/liftviews/export", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	Handler{}.ServeHTTP(rec, req)
	return rec
}

func TestExportZIP(t *testing.T) {
	rec := post(t, simRequest())
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"plan.dxf": false, "elevation.dxf": false, "liftplan.pdf": false, "export-meta.json": false}
	for _, f := range zr.File {
		want[f.Name] = true
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("zip missing %s", name)
		}
	}
}

func TestExportRefusesOverload(t *testing.T) {
	req := simRequest()
	req.TotalLoadT = 400
	rec := post(t, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestVerdictMode(t *testing.T) {
	raw, err := json.Marshal(simRequest())
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/liftviews/export?mode=verdict", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	Handler{}.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var meta map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"hook_span_m", "clearance_m", "share1_pct", "fos1", "fos2", "crane_util"} {
		if _, ok := meta[k]; !ok {
			t.Fatalf("verdict missing %s", k)
		}
	}
	fos1 := meta["fos1"].(float64)
	// Engine FoS ≈ 1.70 on specified 3.0 m² mats: counterweight + slew
	// moments included. Must clear the 1.5 pass threshold with margin.
	if fos1 < 1.5 {
		t.Fatalf("sim scenario FoS must clear 1.5, got %v", fos1)
	}
	if fos1 < 1.6 || fos1 > 1.8 {
		t.Fatalf("sim scenario FoS out of engine band, got %v", fos1)
	}
}

func TestExportMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/liftviews/export", nil)
	rec := httptest.NewRecorder()
	Handler{}.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code=%d", rec.Code)
	}
}
