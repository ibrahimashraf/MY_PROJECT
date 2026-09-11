package workorderhttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"integin/internal/domain/scheduling"
	"integin/internal/domain/workorder"
	"integin/internal/identity"
	"integin/internal/workorderauth"
	"integin/internal/workorderpg"
	"integin/pkg/onboarding"
)

// CompetencyGate verifies that the technician holds an approved credential
// and current unexpired competencies for all required skills.
type CompetencyGate interface {
	CheckSkills(ctx context.Context, actor workorder.ActorContext, credential onboarding.InspectorCredential, requiredSkills []string, competencies []scheduling.TechnicianCompetency, at time.Time) error
}

type defaultCompetencyGate struct{}

func (defaultCompetencyGate) CheckSkills(_ context.Context, _ workorder.ActorContext, credential onboarding.InspectorCredential, requiredSkills []string, competencies []scheduling.TechnicianCompetency, at time.Time) error {
	return scheduling.CheckAssignmentSkills(at, credential, requiredSkills, competencies)
}

// AssignmentHandler exposes the L6 competency-gated work order assignment endpoint.
type AssignmentHandler struct {
	Validator      TokenValidator
	Resolver       identity.Resolver
	Service        workorder.Service
	CompetencyGate CompetencyGate
	Now            func() time.Time
}

type competencyPayload struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	OrganizationID   string    `json:"organization_id"`
	TechnicianID     string    `json:"technician_id"`
	EquipmentTypeID  string    `json:"equipment_type_id"`
	CertificationRef string    `json:"certification_ref,omitempty"`
	ExpiresAt        time.Time `json:"expires_at"`
	Status           string    `json:"status"`
	VerifiedBy       string    `json:"verified_by,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
}

func (c competencyPayload) toDomain() scheduling.TechnicianCompetency {
	return scheduling.TechnicianCompetency{
		ID:               c.ID,
		TenantID:         c.TenantID,
		OrganizationID:   c.OrganizationID,
		TechnicianID:     c.TechnicianID,
		EquipmentTypeID:  c.EquipmentTypeID,
		CertificationRef: c.CertificationRef,
		ExpiresAt:        c.ExpiresAt,
		Status:           c.Status,
		VerifiedBy:       c.VerifiedBy,
		CreatedAt:        c.CreatedAt,
	}
}

type assignRequest struct {
	OperationID      string                         `json:"operation_id"`
	IdempotencyKey   string                         `json:"idempotency_key"`
	ExpectedRevision int64                          `json:"expected_revision"`
	WorkOrderID      string                         `json:"work_order_id,omitempty"`
	AssignmentID     string                         `json:"assignment_id"`
	InspectorID      string                         `json:"inspector_id"`
	ScopeItemIDs     []string                       `json:"scope_item_ids"`
	RequiredSkills   []string                       `json:"required_skills,omitempty"`
	Credential       onboarding.InspectorCredential `json:"credential"`
	Competencies     []competencyPayload            `json:"competencies"`
}

func (h AssignmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		write(w, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	if h.Validator == nil || h.Resolver == nil || h.Service == nil {
		write(w, http.StatusServiceUnavailable, "service_unavailable", nil)
		return
	}
	raw, ok := bearer(r.Header.Get("Authorization"))
	if !ok {
		write(w, http.StatusUnauthorized, "authentication_failed", nil)
		return
	}
	principal, err := h.Validator.Validate(r.Context(), raw)
	if err != nil {
		write(w, http.StatusUnauthorized, "authentication_failed", nil)
		return
	}
	membership, err := h.Resolver.Resolve(r.Context(), identity.PrincipalKey{Issuer: principal.Issuer, Subject: principal.Subject})
	if err != nil {
		write(w, http.StatusForbidden, "authorization_failed", nil)
		return
	}
	actor, err := workorderauth.ActorFromMembership(membership)
	if err != nil {
		write(w, http.StatusForbidden, "authorization_failed", nil)
		return
	}

	workOrderID := extractAssignWorkOrderID(r.URL.Path)

	var body assignRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		write(w, http.StatusBadRequest, "invalid_request", nil)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		write(w, http.StatusBadRequest, "invalid_request", nil)
		return
	}

	if workOrderID == "" {
		workOrderID = body.WorkOrderID
	}
	if strings.TrimSpace(workOrderID) == "" || strings.TrimSpace(body.AssignmentID) == "" || strings.TrimSpace(body.InspectorID) == "" {
		write(w, http.StatusBadRequest, "invalid_request", nil)
		return
	}

	now := time.Now().UTC()
	if h.Now != nil {
		now = h.Now()
	}

	gate := h.CompetencyGate
	if gate == nil {
		gate = defaultCompetencyGate{}
	}

	comps := make([]scheduling.TechnicianCompetency, len(body.Competencies))
	for i, c := range body.Competencies {
		comps[i] = c.toDomain()
	}

	// L6 dynamic inspector competency verification check
	if err := gate.CheckSkills(r.Context(), actor, body.Credential, body.RequiredSkills, comps, now); err != nil {
		if errors.Is(err, scheduling.ErrAssignmentNotVerified) {
			write(w, http.StatusUnprocessableEntity, "competency_verification_failed", nil)
			return
		}
		write(w, http.StatusBadRequest, "invalid_request", nil)
		return
	}

	assignment := workorder.Assignment{
		ID:             body.AssignmentID,
		TenantID:       actor.TenantID,
		OrganizationID: actor.OrganizationID,
		WorkOrderID:    workOrderID,
		InspectorID:    body.InspectorID,
		ScopeItemIDs:   body.ScopeItemIDs,
		State:          workorder.AssignmentActive,
		Revision:       1,
		EffectiveFrom:  now,
	}

	cmd := workorder.AssignScopeCommand{
		Actor: actor,
		Operation: workorder.OperationMeta{
			OperationID:      body.OperationID,
			IdempotencyKey:   body.IdempotencyKey,
			ExpectedRevision: body.ExpectedRevision,
		},
		WorkOrderID: workOrderID,
		Assignment:  assignment,
	}

	receipt, err := h.Service.AssignScope(r.Context(), cmd)
	if err != nil {
		status := http.StatusBadRequest
		code := "mutation_rejected"
		switch {
		case errors.Is(err, workorderauth.ErrDenied), errors.Is(err, sql.ErrNoRows):
			status = http.StatusForbidden
			code = "authorization_failed"
		case errors.Is(err, workorderpg.ErrStaleRevision):
			status = http.StatusConflict
			code = "stale_revision"
		}
		write(w, status, code, nil)
		return
	}

	write(w, http.StatusOK, "", receipt)
}

func extractAssignWorkOrderID(path string) string {
	p := strings.Split(strings.Trim(path, "/"), "/")
	// Matches /api/v1/work-orders/{id}/assign
	if len(p) >= 5 && p[0] == "api" && p[1] == "v1" && p[2] == "work-orders" && p[4] == "assign" && strings.TrimSpace(p[3]) != "" {
		return p[3]
	}
	// Matches /work-orders/{id}/assign
	if len(p) >= 3 && p[0] == "work-orders" && p[2] == "assign" && strings.TrimSpace(p[1]) != "" {
		return p[1]
	}
	return ""
}
