package workorderhttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"integin/internal/domain/workorder"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/workorderauth"
)

type TokenValidator interface {
	Validate(context.Context, string) (oidcauth.Principal, error)
}

type Handler struct {
	Validator TokenValidator
	Resolver  identity.Resolver
	Service   workorder.Service
}

type partialRequest struct {
	OperationID      string   `json:"operation_id"`
	IdempotencyKey   string   `json:"idempotency_key"`
	ExpectedRevision int64    `json:"expected_revision"`
	WorkOrderID      string   `json:"work_order_id"`
	AssignmentID     string   `json:"assignment_id"`
	InspectionIDs    []string `json:"inspection_ids"`
}

type EvidenceHandler struct {
	Validator TokenValidator
	Resolver  identity.Resolver
	Service   workorder.Service
}

type evidenceRequest struct {
	OperationID      string `json:"operation_id"`
	IdempotencyKey   string `json:"idempotency_key"`
	ExpectedRevision int64  `json:"expected_revision"`
	EvidenceID       string `json:"evidence_id"`
	ContentHash      string `json:"content_hash"`
	ReferenceURL     string `json:"reference_url"`
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	var body partialRequest
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
	receipt, err := h.Service.SubmitPartial(r.Context(), workorder.SubmitPartialCommand{Actor: actor, Operation: workorder.OperationMeta{OperationID: body.OperationID, IdempotencyKey: body.IdempotencyKey, ExpectedRevision: body.ExpectedRevision}, WorkOrderID: body.WorkOrderID, AssignmentID: body.AssignmentID, InspectionIDs: body.InspectionIDs})
	if err != nil {
		status := http.StatusBadRequest
		code := "mutation_rejected"
		if errors.Is(err, workorderauth.ErrDenied) || errors.Is(err, sql.ErrNoRows) {
			status = http.StatusForbidden
			code = "authorization_failed"
		}
		write(w, status, code, nil)
		return
	}
	write(w, http.StatusOK, "", receipt)
}

func (h EvidenceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	workOrderID := extractWorkOrderID(r.URL.Path)
	if workOrderID == "" {
		write(w, http.StatusNotFound, "not_found", nil)
		return
	}
	var body evidenceRequest
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
	if body.ReferenceURL != "" && (strings.Contains(body.ReferenceURL, "tenant_id") || strings.Contains(body.ReferenceURL, "organization_id") || strings.Contains(body.ReferenceURL, "actor_id")) {
		write(w, http.StatusBadRequest, "invalid_request", nil)
		return
	}
	evidence := workorder.EvidenceReference{
		ID:             body.EvidenceID,
		TenantID:       actor.TenantID,
		OrganizationID: actor.OrganizationID,
		WorkOrderID:    workOrderID,
		ContentHash:    body.ContentHash,
		ReferenceURL:   body.ReferenceURL,
		CreatedBy:      actor.ActorID,
	}
	receipt, err := h.Service.AddEvidenceReference(r.Context(), workorder.AddEvidenceReferenceCommand{Actor: actor, Operation: workorder.OperationMeta{OperationID: body.OperationID, IdempotencyKey: body.IdempotencyKey, ExpectedRevision: body.ExpectedRevision}, Evidence: evidence})
	if err != nil {
		status := http.StatusBadRequest
		code := "mutation_rejected"
		if errors.Is(err, workorderauth.ErrDenied) || errors.Is(err, sql.ErrNoRows) {
			status = http.StatusForbidden
			code = "authorization_failed"
		}
		write(w, status, code, nil)
		return
	}
	write(w, http.StatusOK, "", receipt)
}

func extractWorkOrderID(path string) string {
	p := strings.Split(strings.Trim(path, "/"), "/")
	if len(p) >= 2 && p[0] == "work-orders" && strings.TrimSpace(p[1]) != "" {
		if len(p) >= 3 && p[2] == "evidence" {
			return p[1]
		}
	}
	return ""
}

func bearer(value string) (string, bool) {
	parts := strings.Fields(value)
	return func() (string, bool) {
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			return "", false
		}
		return parts[1], true
	}()
}
func write(w http.ResponseWriter, status int, code string, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if code != "" {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}
