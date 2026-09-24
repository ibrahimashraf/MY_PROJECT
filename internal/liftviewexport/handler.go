package liftviewexport

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	domainrender "integin/internal/domain/certificaterender"
	"integin/pkg/cad/dxf"
	"integin/pkg/cad/engine"
	"integin/pkg/httputil"
	"integin/pkg/rulesengine"
)

// Request carries raw scene state only. Every rating, verdict, and drawing
// is computed server-side by the native engines: cad/engine solves,
// rulesengine gates, cad/dxf draws, certificaterender seals the PDF.
//
//go:embed crane_catalog.json
var embeddedCraneCatalog []byte

type Request struct {
	Crane1      engine.CraneKinematics `json:"crane1"`
	Crane2      engine.CraneKinematics `json:"crane2"`
	Crane1Model string                 `json:"crane1Model"`
	Crane2Model string                 `json:"crane2Model"`
	TotalLoadT  float64                `json:"totalLoadT"`
	CogOffsetM  float64                `json:"cogOffsetM"`
	MatAreaM2   float64                `json:"matAreaM2"`
	AllowableKP float64                `json:"allowableKPa"`
	Crane1CapT  float64                `json:"crane1CapT"`
	Crane2CapT  float64                `json:"crane2CapT"`
	SlingAngleD float64                `json:"slingAngleDeg"`
	SlingMBLT   float64                `json:"slingMblT"`
	RiggingT    float64                `json:"riggingT"`
	Hazards     []HazardControl        `json:"hazards"`
	MethodSteps []MethodStep           `json:"methodSteps"`
	Pages       int                    `json:"pages"`
}

type HazardControl struct {
	ID      string `json:"id"`
	Hazard  string `json:"hazard"`
	Risk    string `json:"risk"`
	Control string `json:"control"`
	Owner   string `json:"owner"`
}

type MethodStep struct {
	Sequence      int    `json:"sequence"`
	Action        string `json:"action"`
	HoldPoint     string `json:"holdPoint"`
	StopCondition string `json:"stopCondition"`
}

type CraneReference struct {
	ID                  string `json:"id"`
	Manufacturer        string `json:"manufacturer"`
	Model               string `json:"model"`
	CraneClass          string `json:"crane_class"`
	VisualizationStatus string `json:"visualization_status"`
	DutyChartStatus     string `json:"duty_chart_status"`
}

func craneCatalog() []CraneReference {
	var refs []CraneReference
	if err := json.Unmarshal(embeddedCraneCatalog, &refs); err != nil {
		return nil
	}
	return refs
}

func findCraneReference(id string) (CraneReference, bool) {
	for _, ref := range craneCatalog() {
		if ref.ID == id {
			return ref, true
		}
	}
	return CraneReference{}, false
}

func validatePlanningSections(hazards []HazardControl, steps []MethodStep) error {
	if len(hazards) == 0 || len(hazards) > 50 {
		return fmt.Errorf("HIRARC requires between 1 and 50 hazards")
	}
	seen := make(map[string]struct{}, len(hazards))
	for _, hazard := range hazards {
		values := []string{hazard.ID, hazard.Hazard, hazard.Risk, hazard.Control, hazard.Owner}
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("HIRARC fields must not be empty")
			}
		}
		if _, exists := seen[hazard.ID]; exists {
			return fmt.Errorf("HIRARC hazard IDs must be unique")
		}
		seen[hazard.ID] = struct{}{}
	}
	if len(steps) == 0 || len(steps) > 100 {
		return fmt.Errorf("sequential method requires between 1 and 100 steps")
	}
	for i, step := range steps {
		if step.Sequence != i+1 {
			return fmt.Errorf("method step sequence must be contiguous from 1")
		}
		if strings.TrimSpace(step.Action) == "" || strings.TrimSpace(step.HoldPoint) == "" || strings.TrimSpace(step.StopCondition) == "" {
			return fmt.Errorf("method step action, hold point and stop condition are required")
		}
	}
	return nil
}

func hazardPages(hazards []HazardControl) [][]string {
	pages := make([][]string, 0, (len(hazards)+17)/18)
	for start := 0; start < len(hazards); start += 18 {
		end := start + 18
		if end > len(hazards) {
			end = len(hazards)
		}
		lines := []string{"HIRARC", "ID | HAZARD | RISK | CONTROL | OWNER"}
		for _, hazard := range hazards[start:end] {
			lines = append(lines, fmt.Sprintf("%s | %s | %s | %s | %s", hazard.ID, hazard.Hazard, hazard.Risk, hazard.Control, hazard.Owner))
		}
		pages = append(pages, lines)
	}
	return pages
}

func methodPages(steps []MethodStep) [][]string {
	pages := make([][]string, 0, (len(steps)+17)/18)
	for start := 0; start < len(steps); start += 18 {
		end := start + 18
		if end > len(steps) {
			end = len(steps)
		}
		lines := []string{"SEQUENTIAL METHOD", "SEQ | ACTION | HOLD POINT | STOP CONDITION"}
		for _, step := range steps[start:end] {
			lines = append(lines, fmt.Sprintf("%d | %s | %s | %s", step.Sequence, step.Action, step.HoldPoint, step.StopCondition))
		}
		pages = append(pages, lines)
	}
	return pages
}

type Handler struct{}

func (Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/liftviews/cranes" {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			httputil.WriteProblem(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Allowed methods: GET")
			return
		}
		refs := craneCatalog()
		if len(refs) == 0 {
			httputil.WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "crane catalog unavailable")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(refs); err != nil {
			httputil.WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "crane catalog unavailable")
		}
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		httputil.WriteProblem(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Allowed methods: POST")
		return
	}
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		httputil.WriteProblem(w, r, http.StatusUnsupportedMediaType, "Unsupported Media Type", "Content-Type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid scene JSON")
		return
	}
	if req.Pages != 4 {
		req.Pages = 8
	}
	fail := func(msg string) {
		httputil.WriteProblem(w, r, http.StatusUnprocessableEntity, "Unprocessable Entity", msg)
	}
	crane1Ref, crane1OK := findCraneReference(req.Crane1Model)
	crane2Ref, crane2OK := findCraneReference(req.Crane2Model)
	if !crane1OK || !crane2OK {
		fail("crane model reference is missing or unknown")
		return
	}
	if err := validatePlanningSections(req.Hazards, req.MethodSteps); err != nil {
		fail(err.Error())
		return
	}
	// 1. Solve.
	res, err := engine.SolveTandemLift(req.Crane1, req.Crane2, req.TotalLoadT, req.CogOffsetM)
	if err != nil {
		fail("solve refused: " + err.Error())
		return
	}
	// 2. Rigging tension on the heavier share.
	maxShare := res.Crane1LoadTonnes
	if res.Crane2LoadTonnes > maxShare {
		maxShare = res.Crane2LoadTonnes
	}
	legT, ok, lockout, err := rulesengine.SlingTension2Leg(maxShare, req.SlingAngleD)
	if err != nil || !ok || lockout {
		fail("sling gate refused")
		return
	}
	// 3. DHL ledger + four-gate capacity.
	dhl, err := rulesengine.DynamicHookLoad(req.TotalLoadT, req.RiggingT)
	if err != nil {
		fail("DHL ledger refused: " + err.Error())
		return
	}
	_ = dhl
	verdict, err := rulesengine.EvaluateCapacityGates(rulesengine.CapacityGates{
		CraneLoadT: res.Crane1LoadTonnes, CraneCapT: req.Crane1CapT,
		RiggingLoadT: legT, RiggingCapT: req.SlingMBLT,
		SteelLoadT: maxShare, SteelCapT: req.Crane1CapT,
		ObjectLoadT: maxShare, ObjectCapT: req.Crane2CapT,
	})
	if err != nil {
		fail("capacity gates refused: " + err.Error())
		return
	}
	if !verdict.Passed {
		fail("capacity gates fail-closed")
		return
	}
	// 4. Bearing on the worst pad of each crane.
	fos := make([]float64, 0, 2)
	for _, maxPad := range []float64{res.Crane1Outriggers.MaxLoad, res.Crane2Outriggers.MaxLoad} {
		gate, err := rulesengine.CheckBearingPressure(maxPad, req.MatAreaM2, req.AllowableKP)
		if err != nil || !gate.Passed {
			fail("bearing gate fail-closed")
			return
		}
		fos = append(fos, gate.FactorOfSafety)
	}
	// 5. Draw from the solved state.
	plan, err := engine.PlanView(req.Crane1, req.Crane2, res)
	if err != nil {
		fail("plan view refused: " + err.Error())
		return
	}
	elev, err := engine.ElevationView(req.Crane1, req.Crane2, res)
	if err != nil {
		fail("elevation refused: " + err.Error())
		return
	}
	var planBuf, elevBuf bytes.Buffer
	if err := dxf.NewWriter(&planBuf).Write(plan); err != nil {
		fail("dxf render failed")
		return
	}
	if err := dxf.NewWriter(&elevBuf).Write(elev); err != nil {
		fail("dxf render failed")
		return
	}
	// 6. Seal the pack PDF through the one shared engine.
	pages := [][]string{
		{"TANDEM LIFT PLAN — solved, not drawn"},
		{"span " + numeric(res.HookSpanMeters) + "m clear " +
			numeric(res.MinBoomClearanceM) + "m share " +
			numeric(res.LoadShareCrane1) + "/" + numeric(res.LoadShareCrane2) +
			" crane-util " + numeric(verdict.CraneUtil)},
		{"RIGGING", "heavier share " + numeric(maxShare) + "t", "leg tension " + numeric(legT) + "t", "DHL " + numeric(dhl) + "t"},
		{"PLAN VIEW", "see plan.dxf"},
		{"ELEVATION", "see elevation.dxf"},
		{"CHART BINDING", crane1Ref.Manufacturer + " " + crane1Ref.Model + " duty chart NOT_PROVIDED", crane2Ref.Manufacturer + " " + crane2Ref.Model + " duty chart NOT_PROVIDED"},
		{"GROUND", "bearing gates passed"},
		{"EXECUTION", "sealed export"},
	}
	pages = append(pages, hazardPages(req.Hazards)...)
	pages = append(pages, methodPages(req.MethodSteps)...)
	req.Pages = len(pages)
	raw, err := domainrender.BuildDeterministicPDF("TANDEM LIFT PLAN", pages)
	if err != nil {
		fail("pdf engine refused: " + err.Error())
		return
	}
	if err := domainrender.ValidatePDFSecurity(raw); err != nil {
		fail("pdf security refused")
		return
	}
	meta, err := json.Marshal(map[string]any{
		"crane1_model":      crane1Ref.ID,
		"crane2_model":      crane2Ref.ID,
		"duty_chart_status": crane1Ref.DutyChartStatus,
		"hazards":           req.Hazards,
		"method_steps":      req.MethodSteps,
		"hook_span_m":       res.HookSpanMeters,
		"clearance_m":       res.MinBoomClearanceM,
		"share1_pct":        res.LoadShareCrane1,
		"share2_pct":        res.LoadShareCrane2,
		"crane_util":        verdict.CraneUtil,
		"rigging_util":      verdict.RiggingUtil,
		"leg_tension":       legT,
		"max_share_t":       maxShare,
		"dhl_t":             dhl,
		"fos1":              fos[0],
		"fos2":              fos[1],
		"renderer":          "integin-engines/v1",
		"total_pages":       req.Pages,
	})
	if err != nil {
		fail("meta render failed")
		return
	}
	if r.URL.Query().Get("mode") == "verdict" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(meta)
		return
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range []struct {
		name string
		data []byte
	}{
		{"plan.dxf", planBuf.Bytes()},
		{"elevation.dxf", elevBuf.Bytes()},
		{"liftplan.pdf", raw},
		{"export-meta.json", meta},
	} {
		fw, err := zw.Create(f.name)
		if err != nil {
			fail("zip failed")
			return
		}
		if _, err := fw.Write(f.data); err != nil {
			fail("zip failed")
			return
		}
	}
	if err := zw.Close(); err != nil {
		fail("zip failed")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="liftview-export.zip"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func numeric(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func planSummary(res *engine.TandemLiftResult, craneUtil float64) string {
	return "span " + numeric(res.HookSpanMeters) + "m clear " +
		numeric(res.MinBoomClearanceM) + "m share " +
		numeric(res.LoadShareCrane1) + "/" + numeric(res.LoadShareCrane2) +
		" crane-util " + numeric(craneUtil)
}
