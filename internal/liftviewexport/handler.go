package liftviewexport

import (
	"archive/zip"
	"bytes"
	"encoding/json"
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
type Request struct {
	Crane1      engine.CraneKinematics `json:"crane1"`
	Crane2      engine.CraneKinematics `json:"crane2"`
	TotalLoadT  float64                `json:"totalLoadT"`
	CogOffsetM  float64                `json:"cogOffsetM"`
	MatAreaM2   float64                `json:"matAreaM2"`
	AllowableKP float64                `json:"allowableKPa"`
	Crane1CapT  float64                `json:"crane1CapT"`
	Crane2CapT  float64                `json:"crane2CapT"`
	SlingAngleD float64                `json:"slingAngleDeg"`
	SlingMBLT   float64                `json:"slingMblT"`
	RiggingT    float64                `json:"riggingT"`
	Pages       int                    `json:"pages"`
}

type Handler struct{}

func (Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	}
	if req.Pages == 8 {
		pages = append(pages,
			[]string{"RIGGING", "heavier share " + numeric(maxShare) + "t", "leg tension " + numeric(legT) + "t", "DHL " + numeric(dhl) + "t"},
			[]string{"PLAN VIEW", "see plan.dxf"},
			[]string{"ELEVATION", "see elevation.dxf"},
			[]string{"CHART BINDING", "owner-supplied OEM charts"},
			[]string{"GROUND", "bearing gates passed"},
			[]string{"EXECUTION", "sealed export"},
		)
	}
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
		"hook_span_m":  res.HookSpanMeters,
		"clearance_m":  res.MinBoomClearanceM,
		"share1_pct":   res.LoadShareCrane1,
		"share2_pct":   res.LoadShareCrane2,
		"crane_util":   verdict.CraneUtil,
		"rigging_util": verdict.RiggingUtil,
		"leg_tension":  legT,
		"max_share_t":  maxShare,
		"dhl_t":        dhl,
		"fos1":         fos[0],
		"fos2":         fos[1],
		"renderer":     "integin-engines/v1",
		"total_pages":  req.Pages,
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
