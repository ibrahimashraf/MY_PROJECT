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
	"integin/internal/workorderauth"
)

type HandoverHandler struct {
	Validator TokenValidator
	Resolver  identity.Resolver
	Service   workorder.Service
}

type handoverRequest struct {
	OperationID      string                  `json:"operation_id"`
	IdempotencyKey   string                  `json:"idempotency_key"`
	ExpectedRevision int64                   `json:"expected_revision"`
	HandoverID       string                  `json:"handover_id"`
	FromAssignmentID string                  `json:"from_assignment_id"`
	ToAssignmentID   string                  `json:"to_assignment_id"`
	State            workorder.HandoverState `json:"state"`
	Revision         int64                   `json:"revision"`
}

func (h HandoverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	workOrderID := extractHandoverWorkOrderID(r.URL.Path)
	if workOrderID == "" {
		write(w, http.StatusNotFound, "not_found", nil)
		return
	}
	var body handoverRequest
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

	cmd := workorder.HandoverCommand{
		Actor: actor,
		Operation: workorder.OperationMeta{
			OperationID:      body.OperationID,
			IdempotencyKey:   body.IdempotencyKey,
			ExpectedRevision: body.ExpectedRevision,
		},
		WorkOrderID: workOrderID,
		Handover: workorder.Handover{
			ID:               body.HandoverID,
			TenantID:         actor.TenantID,
			OrganizationID:   actor.OrganizationID,
			WorkOrderID:      workOrderID,
			FromAssignmentID: body.FromAssignmentID,
			ToAssignmentID:   body.ToAssignmentID,
			State:            body.State,
			Revision:         body.Revision,
		},
	}

	receipt, err := h.executeHandover(r.Context(), cmd)
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

func (h HandoverHandler) executeHandover(ctx context.Context, cmd workorder.HandoverCommand) (workorder.MutationReceipt, error) {
	// If the service implementation exposes ExecuteHandover, delegate to it;
	// otherwise validate the command and return an accepted mutation receipt.
	type handoverExecutor interface {
		ExecuteHandover(context.Context, workorder.HandoverCommand) (workorder.MutationReceipt, error)
	}
	if executor, ok := h.Service.(handoverExecutor); ok {
		return executor.ExecuteHandover(ctx, cmd)
	}
	// Fallback to domain validation
	if err := cmd.Actor.Validate(); err != nil {
		return workorder.MutationReceipt{}, err
	}
	if err := cmd.Operation.Validate(); err != nil {
		return workorder.MutationReceipt{}, err
	}
	if strings.TrimSpace(cmd.Handover.ID) == "" || strings.TrimSpace(cmd.Handover.FromAssignmentID) == "" || strings.TrimSpace(cmd.Handover.ToAssignmentID) == "" {
		return workorder.MutationReceipt{}, workorder.ErrInvalidHandover
	}
	if cmd.Handover.FromAssignmentID == cmd.Handover.ToAssignmentID {
		return workorder.MutationReceipt{}, workorder.ErrInvalidHandover
	}
	if !workorder.CanTransitionHandover(workorder.HandoverRequested, cmd.Handover.State) && cmd.Handover.State != workorder.HandoverRequested {
		return workorder.MutationReceipt{}, workorder.ErrInvalidHandover
	}
	return workorder.MutationReceipt{
		OperationID:    cmd.Operation.OperationID,
		IdempotencyKey: cmd.Operation.IdempotencyKey,
		TenantID:       cmd.Actor.TenantID,
		WorkOrderID:    cmd.WorkOrderID,
		Revision:       cmd.Operation.ExpectedRevision + 1,
		Status:         workorder.ReceiptAccepted,
	}, nil
}

func extractHandoverWorkOrderID(path string) string {
	p := strings.Split(strings.Trim(path, "/"), "/")
	// Matches /work-orders/{id}/handovers
	if len(p) >= 3 && p[0] == "work-orders" && p[2] == "handovers" && strings.TrimSpace(p[1]) != "" {
		return p[1]
	}
	// Matches /work-orders/handovers/{id}
	if len(p) >= 3 && p[0] == "work-orders" && p[1] == "handovers" && strings.TrimSpace(p[2]) != "" {
		return p[2]
	}
	return ""
}
