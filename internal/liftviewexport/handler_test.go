package liftviewexport

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
		Crane1Model: "LIEBHERR_LTM_1500",
		Crane2Model: "TADANO_ATF_400G",
		TotalLoadT:  45,
		CogOffsetM:  -1,
		MatAreaM2:   3.0,
		AllowableKP: 220,
		Crane1CapT:  40,
		Crane2CapT:  40,
		SlingAngleD: 60,
		SlingMBLT:   30,
		RiggingT:    2,
		Hazards: []HazardControl{{
			ID: "H-01", Hazard: "Suspended load", Risk: "Struck-by", Control: "Exclusion zone and tag lines", Owner: "Appointed Person",
		}},
		MethodSteps: []MethodStep{{
			Sequence: 1, Action: "Inspect slings and connect master link", HoldPoint: "Pre-lift inspection signed", StopCondition: "Any damage or missing evidence",
		}, {
			Sequence: 2, Action: "Trial lift and verify load share", HoldPoint: "Trial lift complete", StopCondition: "Excess tilt or capacity warning",
		}},
		Pages: 8,
	}
}

func planningPayload(t *testing.T) map[string]any {
	t.Helper()
	raw, err := json.Marshal(simRequest())
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	payload["hazards"] = []map[string]any{{
		"id": "H-01", "hazard": "Suspended load", "risk": "Struck-by", "control": "Exclusion zone and tag lines", "owner": "Appointed Person",
	}}
	payload["methodSteps"] = []map[string]any{{
		"sequence": 1, "action": "Inspect slings and connect master link", "holdPoint": "Pre-lift inspection signed", "stopCondition": "Any damage or missing evidence",
	}, {
		"sequence": 2, "action": "Trial lift and verify load share", "holdPoint": "Trial lift complete", "stopCondition": "Excess tilt or capacity warning",
	}}
	return payload
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

func TestCraneCatalogIsReferenceOnly(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/liftviews/cranes", nil)
	rec := httptest.NewRecorder()
	Handler{}.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var refs []struct {
		ID              string `json:"id"`
		DutyChartStatus string `json:"duty_chart_status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refs); err != nil {
		t.Fatal(err)
	}
	if len(refs) != 4 {
		t.Fatalf("catalog entries=%d want=4", len(refs))
	}
	for _, ref := range refs {
		if ref.ID == "" || ref.DutyChartStatus != "NOT_PROVIDED" {
			t.Fatalf("unsafe crane reference: %+v", ref)
		}
	}
}

func TestExportRefusesUnknownCraneModel(t *testing.T) {
	raw, err := json.Marshal(simRequest())
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	payload["crane1Model"] = "UNKNOWN-MODEL"
	rec := post(t, payload)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestExportRefusesMissingHIRARC(t *testing.T) {
	payload := planningPayload(t)
	payload["hazards"] = []map[string]any{}
	rec := post(t, payload)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestExportRefusesNonContiguousMethodSteps(t *testing.T) {
	payload := planningPayload(t)
	payload["methodSteps"] = []map[string]any{{
		"sequence": 2, "action": "Trial lift", "holdPoint": "Hold", "stopCondition": "Stop",
	}}
	rec := post(t, payload)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestExportIncludesHIRARCAndMethodPages(t *testing.T) {
	rec := post(t, planningPayload(t))
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var planPDF, meta []byte
	for _, file := range zr.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		switch file.Name {
		case "liftplan.pdf":
			planPDF = data
		case "export-meta.json":
			meta = data
		}
	}
	if !strings.Contains(string(planPDF), "HIRARC") || !strings.Contains(string(planPDF), "Suspended load") || !strings.Contains(string(planPDF), "SEQUENTIAL METHOD") {
		t.Fatalf("lift plan PDF missing HIRARC or method text: %q", planPDF)
	}
	if !strings.Contains(string(meta), "\"hazards\"") || !strings.Contains(string(meta), "\"method_steps\"") {
		t.Fatalf("export metadata missing planning sections: %s", meta)
	}
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
	for _, k := range []string{"crane1_model", "crane2_model", "hazards", "method_steps", "hook_span_m", "clearance_m", "share1_pct", "fos1", "fos2", "crane_util"} {
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
