package reinspectionhttp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"integin/internal/domain/reinspection"
	"integin/internal/shared/httpresponse"
	"integin/internal/shared/types"
)

// Store provides persistence for Corrective Action Requests.
type Store interface {
	Get(id string) (reinspection.CorrectiveActionRequest, bool)
	Save(car reinspection.CorrectiveActionRequest)
}

type MemoryStore struct {
	mu   sync.RWMutex
	cars map[string]reinspection.CorrectiveActionRequest
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{cars: make(map[string]reinspection.CorrectiveActionRequest)}
}

func (s *MemoryStore) Get(id string) (reinspection.CorrectiveActionRequest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	car, ok := s.cars[id]
	return car, ok
}

func (s *MemoryStore) Save(car reinspection.CorrectiveActionRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cars[car.ID] = car
}

// Handler handles CAR lifecycle endpoints under /api/v1/car/.
type Handler struct {
	Store Store
}

func NewHandler(store Store) *Handler {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Handler{Store: store}
}

type issueRequest struct {
	CARID              string         `json:"car_id"`
	AssetID            string         `json:"asset_id"`
	SourceInspectionID string         `json:"source_inspection_id"`
	SourceFindingID    string         `json:"source_finding_id"`
	Severity           types.Severity `json:"severity"`
	Description        string         `json:"description"`
}

type remediateRequest struct {
	CARID        string   `json:"car_id"`
	Notes        string   `json:"notes"`
	EvidenceRefs []string `json:"evidence_refs"`
	RemediatedBy string   `json:"remediated_by"`
}

type verifyRequest struct {
	CARID          string        `json:"car_id"`
	ReinspectionID string        `json:"reinspection_id"`
	Outcome        types.Verdict `json:"outcome"`
	VerifiedBy     string        `json:"verified_by"`
	VerifiedByAlt  string        `json:"verifiedBy,omitempty"`
	TargetItemIDs  []string      `json:"target_item_ids,omitempty"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	orgID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing required headers: X-Tenant-ID, X-Organization-ID")
		return
	}

	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/car"), "/")

	switch {
	case path == "issue" && r.Method == http.MethodPost:
		h.handleIssue(w, r, tenantID, orgID)
	case path == "remediate" && r.Method == http.MethodPost:
		h.handleRemediate(w, r, tenantID, orgID)
	case path == "verify" && r.Method == http.MethodPost:
		h.handleVerify(w, r, tenantID, orgID)
	case strings.HasPrefix(path, "get/") && r.Method == http.MethodGet:
		carID := strings.TrimPrefix(path, "get/")
		car, ok := h.Store.Get(carID)
		if !ok || car.TenantID != tenantID {
			httpresponse.Error(w, http.StatusNotFound, "car not found")
			return
		}
		httpresponse.JSON(w, http.StatusOK, car)
	default:
		httpresponse.Error(w, http.StatusNotFound, "endpoint not found")
	}
}

func (h *Handler) handleIssue(w http.ResponseWriter, r *http.Request, tenantID, orgID string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req issueRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "malformed request JSON")
		return
	}

	car, err := reinspection.IssueCAR(
		req.CARID, tenantID, orgID, req.AssetID,
		req.SourceInspectionID, req.SourceFindingID,
		req.Severity, req.Description, time.Now().UTC(),
	)
	if err != nil {
		httpresponse.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	h.Store.Save(car)
	httpresponse.JSON(w, http.StatusCreated, car)
}

func (h *Handler) handleRemediate(w http.ResponseWriter, r *http.Request, tenantID, orgID string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req remediateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "malformed request JSON")
		return
	}

	car, ok := h.Store.Get(req.CARID)
	if !ok || car.TenantID != tenantID {
		httpresponse.Error(w, http.StatusNotFound, "car not found")
		return
	}

	if err := car.SubmitRemediation(req.Notes, req.EvidenceRefs, req.RemediatedBy, time.Now().UTC()); err != nil {
		httpresponse.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	h.Store.Save(car)
	httpresponse.JSON(w, http.StatusOK, car)
}

func (h *Handler) handleVerify(w http.ResponseWriter, r *http.Request, tenantID, orgID string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "failed to read body")
		return
	}
	var req verifyRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "malformed request JSON")
		return
	}

	car, ok := h.Store.Get(req.CARID)
	if !ok || car.TenantID != tenantID {
		httpresponse.Error(w, http.StatusNotFound, "car not found")
		return
	}

	verifier := req.VerifiedBy
	if verifier == "" {
		verifier = req.VerifiedByAlt
	}
	targetItems := req.TargetItemIDs
	if len(targetItems) == 0 {
		targetItems = []string{car.SourceFindingID}
	}

	if _, err := car.VerifyOutcome(req.ReinspectionID, req.Outcome, verifier, targetItems, time.Now().UTC()); err != nil {
		httpresponse.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	h.Store.Save(car)
	httpresponse.JSON(w, http.StatusOK, car)
}
