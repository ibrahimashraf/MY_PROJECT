package advisoryhttp

import (
	"net/http"
	"strings"
	"time"

	"integin/internal/advisory"
	"integin/internal/shared/httpresponse"
)

// Handler serves the read-only and feedback advisory endpoints under
// /api/v1/advisory. It enforces tenant header isolation, rejects any request
// carrying blocking=true, and refuses AI-gated work for zones outside the
// approved advisory boundary.
type Handler struct {
	registry   *advisory.Registry
	extractor  advisory.DocumentExtractor
	summarizer advisory.TrendSummarizer
}

func NewHandler(registry *advisory.Registry) *Handler {
	return &Handler{
		registry:   registry,
		extractor:  advisory.DocumentExtractor{},
		summarizer: advisory.TrendSummarizer{},
	}
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if h.registry == nil {
		httpresponse.Error(writer, http.StatusServiceUnavailable, "advisory registry not configured")
		return
	}
	tenantID := strings.TrimSpace(request.Header.Get("X-Tenant-ID"))
	orgID := strings.TrimSpace(request.Header.Get("X-Organization-ID"))
	if tenantID == "" || orgID == "" {
		httpresponse.Error(writer, http.StatusBadRequest, "missing required headers: X-Tenant-ID, X-Organization-ID")
		return
	}

	path := strings.Trim(strings.TrimPrefix(request.URL.Path, "/api/v1/advisory"), "/")
	switch {
	case path == "models" && request.Method == http.MethodGet:
		h.handleModels(writer)
	case path == "extract" && request.Method == http.MethodPost:
		h.handleExtract(writer, request, tenantID)
	case path == "trends" && request.Method == http.MethodPost:
		h.handleTrends(writer, request, tenantID)
	case path == "feedback" && request.Method == http.MethodPost:
		h.handleFeedback(writer, request, tenantID)
	case path == "models" || path == "extract" || path == "trends" || path == "feedback":
		httpresponse.Error(writer, http.StatusMethodNotAllowed, "method not allowed")
	default:
		httpresponse.Error(writer, http.StatusNotFound, "endpoint not found")
	}
}

func (h *Handler) handleModels(writer http.ResponseWriter) {
	models := h.registry.ApprovedModels()
	views := make([]modelView, 0, len(models))
	for _, m := range models {
		zones := make([]string, 0, len(m.AllowedZones))
		for _, zone := range m.AllowedZones {
			zones = append(zones, string(zone))
		}
		views = append(views, modelView{
			ID:           m.ID,
			Provider:     m.Provider,
			ModelName:    m.ModelName,
			Version:      m.Version,
			Status:       string(m.Status),
			MaxTokens:    m.MaxTokens,
			AllowedZones: zones,
		})
	}
	httpresponse.JSON(writer, http.StatusOK, map[string]any{"models": views})
}

type extractRequest struct {
	Text     string        `json:"text"`
	Zone     advisory.Zone `json:"zone"`
	Blocking bool          `json:"blocking"`
}

func (h *Handler) handleExtract(writer http.ResponseWriter, request *http.Request, tenantID string) {
	var req extractRequest
	if err := httpresponse.ReadJSON(request, &req); err != nil {
		httpresponse.Error(writer, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Blocking {
		httpresponse.Error(writer, http.StatusBadRequest, "blocking must be false")
		return
	}
	if !zoneAllowed(req.Zone) {
		httpresponse.Error(writer, http.StatusBadRequest, "AI invocation is forbidden in this zone")
		return
	}
	extracted, err := h.extractor.Extract(tenantID, req.Text, time.Now())
	if err != nil {
		httpresponse.Error(writer, http.StatusBadRequest, err.Error())
		return
	}
	httpresponse.JSON(writer, http.StatusOK, extractView{
		TenantID:      extracted.TenantID,
		Title:         extracted.Title,
		Authority:     extracted.Authority,
		EffectiveDate: extracted.EffectiveDate,
		Clauses:       extracted.Clauses,
		ChecklistRefs: extracted.ChecklistRefs,
		Blocking:      extracted.Blocking,
		ExtractedAt:   extracted.ExtractedAt,
	})
}

type observationInput struct {
	ID         string        `json:"id"`
	Zone       advisory.Zone `json:"zone"`
	Discipline string        `json:"discipline"`
	Location   string        `json:"location"`
	Severity   string        `json:"severity"`
	ReportedAt time.Time     `json:"reported_at"`
	Findings   []string      `json:"findings"`
}

func (o observationInput) toDomain() advisory.DefectObservation {
	return advisory.DefectObservation{
		ID:         o.ID,
		Zone:       o.Zone,
		Discipline: o.Discipline,
		Location:   o.Location,
		Severity:   o.Severity,
		ReportedAt: o.ReportedAt,
		Findings:   append([]string(nil), o.Findings...),
	}
}

type trendsRequest struct {
	Lens         string             `json:"lens"`
	Zone         advisory.Zone      `json:"zone"`
	Observations []observationInput `json:"observations"`
	EvidenceRefs []string           `json:"evidence_refs"`
	Blocking     bool               `json:"blocking"`
}

func (h *Handler) handleTrends(writer http.ResponseWriter, request *http.Request, tenantID string) {
	var req trendsRequest
	if err := httpresponse.ReadJSON(request, &req); err != nil {
		httpresponse.Error(writer, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Blocking {
		httpresponse.Error(writer, http.StatusBadRequest, "blocking must be false")
		return
	}
	if req.Zone == "" {
		httpresponse.Error(writer, http.StatusBadRequest, "zone is required")
		return
	}
	if !zoneAllowed(req.Zone) {
		httpresponse.Error(writer, http.StatusBadRequest, "AI invocation is forbidden in this zone")
		return
	}
	observations := make([]advisory.DefectObservation, 0, len(req.Observations))
	for _, observation := range req.Observations {
		if !zoneAllowed(observation.Zone) {
			httpresponse.Error(writer, http.StatusBadRequest, "AI invocation is forbidden in this zone")
			return
		}
		observations = append(observations, observation.toDomain())
	}
	insight, err := h.summarizer.Summarize(tenantID, req.Lens, req.Zone, observations, req.EvidenceRefs, time.Now())
	if err != nil {
		httpresponse.Error(writer, http.StatusBadRequest, err.Error())
		return
	}
	if err := advisory.EnsureAdvisoryStrict(req.Zone, insight); err != nil {
		httpresponse.Error(writer, http.StatusBadRequest, err.Error())
		return
	}
	httpresponse.JSON(writer, http.StatusOK, insightView{
		ID:            insight.ID,
		TenantID:      insight.TenantID,
		Lens:          insight.Lens,
		Title:         insight.Title,
		Summary:       insight.Summary,
		Severity:      insight.Severity,
		Confidence:    insight.Confidence,
		Rationale:     insight.Rationale,
		EvidenceRefs:  insight.EvidenceRefs,
		Provider:      insight.Provider,
		Model:         insight.Model,
		PromptVersion: insight.PromptVersion,
		Blocking:      insight.Blocking,
		CreatedAt:     insight.CreatedAt,
		Limitations:   insight.Limitations,
	})
}

type feedbackRequest struct {
	ID          string `json:"id"`
	AuditID     string `json:"audit_id"`
	InspectorID string `json:"inspector_id"`
	Disposition string `json:"disposition"`
	Notes       string `json:"notes"`
	Blocking    bool   `json:"blocking"`
}

func (h *Handler) handleFeedback(writer http.ResponseWriter, request *http.Request, tenantID string) {
	var req feedbackRequest
	if err := httpresponse.ReadJSON(request, &req); err != nil {
		httpresponse.Error(writer, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Blocking {
		httpresponse.Error(writer, http.StatusBadRequest, "blocking must be false")
		return
	}
	feedback := advisory.InspectorFeedback{
		ID:          req.ID,
		TenantID:    tenantID,
		AuditID:     req.AuditID,
		InspectorID: req.InspectorID,
		Disposition: req.Disposition,
		Notes:       req.Notes,
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.registry.SubmitFeedback(feedback); err != nil {
		httpresponse.Error(writer, http.StatusBadRequest, err.Error())
		return
	}
	httpresponse.JSON(writer, http.StatusCreated, feedbackView{
		ID:          feedback.ID,
		TenantID:    feedback.TenantID,
		AuditID:     feedback.AuditID,
		InspectorID: feedback.InspectorID,
		Disposition: feedback.Disposition,
		Notes:       feedback.Notes,
		CreatedAt:   feedback.CreatedAt,
	})
}

func zoneAllowed(zone advisory.Zone) bool {
	if zone == "" {
		return true
	}
	return advisory.AIAllowed(zone)
}

type modelView struct {
	ID           string   `json:"id"`
	Provider     string   `json:"provider"`
	ModelName    string   `json:"model_name"`
	Version      string   `json:"version"`
	Status       string   `json:"status"`
	MaxTokens    int      `json:"max_tokens"`
	AllowedZones []string `json:"allowed_zones"`
}

type extractView struct {
	TenantID      string    `json:"tenant_id"`
	Title         string    `json:"title"`
	Authority     string    `json:"authority"`
	EffectiveDate string    `json:"effective_date"`
	Clauses       []string  `json:"clauses"`
	ChecklistRefs []string  `json:"checklist_refs"`
	Blocking      bool      `json:"blocking"`
	ExtractedAt   time.Time `json:"extracted_at"`
}

type insightView struct {
	ID            string    `json:"id,omitempty"`
	TenantID      string    `json:"tenant_id"`
	Lens          string    `json:"lens"`
	Title         string    `json:"title"`
	Summary       string    `json:"summary"`
	Severity      string    `json:"severity"`
	Confidence    float64   `json:"confidence"`
	Rationale     string    `json:"rationale"`
	EvidenceRefs  []string  `json:"evidence_refs"`
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	PromptVersion string    `json:"prompt_version"`
	Blocking      bool      `json:"blocking"`
	CreatedAt     time.Time `json:"created_at"`
	Limitations   []string  `json:"limitations"`
}

type feedbackView struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	AuditID     string    `json:"audit_id"`
	InspectorID string    `json:"inspector_id"`
	Disposition string    `json:"disposition"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
}
