package traininghttp

import (
	"encoding/json"
	"net/http"
	"strings"

	"integin/internal/domain/training"
	"integin/internal/shared/httpresponse"
)

type Handler struct {
	Repository training.Repository
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tenantID := r.Header.Get("X-Tenant-ID")
	orgID := r.Header.Get("X-Organization-ID")
	if tenantID == "" || orgID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "missing tenant or organization header")
		return
	}
	actor := training.ActorContext{TenantID: tenantID, OrganizationID: orgID, ActorID: r.Header.Get("X-Actor-ID")}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/training/")
	parts := strings.SplitN(path, "/", 2)

	switch {
	case r.Method == http.MethodGet && path == "":
		h.handleListCourses(w, r, actor)
	case r.Method == http.MethodPost && path == "":
		h.handleCreateCourse(w, r, actor)
	case r.Method == http.MethodGet && len(parts) == 2 && parts[1] == "enrollments":
		h.handleListEnrollments(w, r, actor, parts[0])
	case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "enroll":
		h.handleEnroll(w, r, actor, parts[0])
	case r.Method == http.MethodGet && strings.HasPrefix(path, "competencies/"):
		techID := strings.TrimPrefix(path, "competencies/")
		h.handleListCompetencies(w, r, actor, techID)
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/competencies"):
		techID := strings.TrimSuffix(strings.TrimSuffix(path, "/competencies"), "/")
		h.handleUpsertCompetency(w, r, actor, techID)
	default:
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
	}
}

func (h Handler) handleListCourses(w http.ResponseWriter, r *http.Request, actor training.ActorContext) {
	courses, err := h.Repository.ListCourses(r.Context(), actor)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if courses == nil {
		courses = []training.Course{}
	}
	httpresponse.JSON(w, http.StatusOK, courses)
}

func (h Handler) handleCreateCourse(w http.ResponseWriter, r *http.Request, actor training.ActorContext) {
	var course training.Course
	if err := json.NewDecoder(r.Body).Decode(&course); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	course.TenantID = actor.TenantID
	course.OrganizationID = actor.OrganizationID
	result, err := h.Repository.CreateCourse(r.Context(), actor, course)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (h Handler) handleListEnrollments(w http.ResponseWriter, r *http.Request, actor training.ActorContext, courseID string) {
	enrollments, err := h.Repository.ListEnrollments(r.Context(), actor, courseID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if enrollments == nil {
		enrollments = []training.CourseEnrollment{}
	}
	httpresponse.JSON(w, http.StatusOK, enrollments)
}

func (h Handler) handleEnroll(w http.ResponseWriter, r *http.Request, actor training.ActorContext, courseID string) {
	var req struct {
		TechnicianID string `json:"technician_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	enrollment := training.CourseEnrollment{
		TenantID:       actor.TenantID,
		OrganizationID: actor.OrganizationID,
		CourseID:       courseID,
		TechnicianID:   req.TechnicianID,
	}
	result, err := h.Repository.EnrollTechnician(r.Context(), actor, enrollment)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (h Handler) handleListCompetencies(w http.ResponseWriter, r *http.Request, actor training.ActorContext, technicianID string) {
	comps, err := h.Repository.ListCompetencies(r.Context(), actor, technicianID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if comps == nil {
		comps = []training.Competency{}
	}
	httpresponse.JSON(w, http.StatusOK, comps)
}

func (h Handler) handleUpsertCompetency(w http.ResponseWriter, r *http.Request, actor training.ActorContext, technicianID string) {
	var comp training.Competency
	if err := json.NewDecoder(r.Body).Decode(&comp); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid_request")
		return
	}
	comp.TenantID = actor.TenantID
	comp.OrganizationID = actor.OrganizationID
	comp.TechnicianID = technicianID
	result, err := h.Repository.UpsertCompetency(r.Context(), actor, comp)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}
