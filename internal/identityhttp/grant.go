// Package identityhttp exposes the operator identity-grant endpoint: the
// production path for binding an external OIDC issuer-subject to local tenant
// context. Calls are gated on the organization.admin capability inside the
// target tenant and every grant is appended to the tamper-proof audit ledger.
package identityhttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"integin/internal/domain/auditlog"
	"integin/internal/domain/authorization"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/internal/shared/types"
)

// TokenValidator validates OIDC bearer tokens. *oidcauth.Validator satisfies it.
type TokenValidator interface {
	Validate(context.Context, string) (oidcauth.Principal, error)
}

// Auditor appends grant events to the tamper-proof ledger.
// *auditlogpg.Repository satisfies it.
type Auditor interface {
	Append(context.Context, auditlog.Entry) (auditlog.Entry, error)
}

// Granter persists the membership binding. identity.GrantMembership is the
// production implementation; tests substitute fakes.
type Granter func(context.Context, *sql.DB, identity.GrantInput) (int64, error)

type Handler struct {
	DB        *sql.DB
	Validator TokenValidator
	Resolver  identity.Resolver
	Auditor   Auditor
	Grant     Granter
}

type grantRequest struct {
	Issuer         string   `json:"issuer"`
	Subject        string   `json:"subject"`
	TenantID       string   `json:"tenant_id"`
	OrganizationID string   `json:"organization_id"`
	ActorID        string   `json:"actor_id"`
	Role           string   `json:"role"`
	Capabilities   []string `json:"capabilities"`
}

type response struct {
	Outcome      string   `json:"outcome"`
	ActorID      string   `json:"actor_id,omitempty"`
	TenantID     string   `json:"tenant_id,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Reason       string   `json:"reason,omitempty"`
}

func (h Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSON(writer, http.StatusMethodNotAllowed, response{Outcome: "REJECTED", Reason: "POST is required"})
		return
	}
	if h.DB == nil || h.Validator == nil || h.Resolver == nil || h.Auditor == nil {
		writeJSON(writer, http.StatusServiceUnavailable, response{Outcome: "REJECTED", Reason: "identity service is unavailable"})
		return
	}
	grant := h.Grant
	if grant == nil {
		grant = identity.GrantMembership
	}
	raw, ok := bearer(request.Header.Get("Authorization"))
	if !ok {
		writeJSON(writer, http.StatusUnauthorized, response{Outcome: "REJECTED", Reason: "authentication_failed"})
		return
	}
	principal, err := h.Validator.Validate(request.Context(), raw)
	if err != nil {
		writeJSON(writer, http.StatusUnauthorized, response{Outcome: "REJECTED", Reason: "authentication_failed"})
		return
	}
	membership, err := h.Resolver.Resolve(request.Context(), identity.PrincipalKey{Issuer: principal.Issuer, Subject: principal.Subject})
	if err != nil {
		if errors.Is(err, identity.ErrUnknownSubject) {
			writeJSON(writer, http.StatusForbidden, response{Outcome: "REJECTED", Reason: "bootstrap_required"})
			return
		}
		writeJSON(writer, http.StatusForbidden, response{Outcome: "REJECTED", Reason: "authorization_failed"})
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, 1<<20)
	var incoming grantRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&incoming); err != nil {
		writeJSON(writer, http.StatusBadRequest, response{Outcome: "REJECTED", Reason: "invalid JSON request"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(writer, http.StatusBadRequest, response{Outcome: "REJECTED", Reason: "invalid JSON request"})
		return
	}
	// Unspecified scope defaults to the caller's own tenant: grants outside
	// it are rejected by the tenant-mismatch rule below.
	targetTenant := strings.TrimSpace(incoming.TenantID)
	if targetTenant == "" {
		targetTenant = membership.TenantID
	}
	targetOrg := strings.TrimSpace(incoming.OrganizationID)
	if targetOrg == "" {
		targetOrg = membership.OrganizationID
	}
	actorID := strings.TrimSpace(incoming.ActorID)
	if actorID == "" {
		actorID = strings.TrimSpace(incoming.Subject)
	}
	capabilities := make(map[authorization.Capability]bool, len(membership.Capabilities))
	for _, capability := range membership.Capabilities {
		capabilities[authorization.Capability(capability)] = true
	}
	decision := authorization.Evaluate(authorization.Request{
		Subject: authorization.Subject{
			UserID: membership.ActorID, TenantID: membership.TenantID,
			OrganizationID: membership.OrganizationID, Role: membership.WorkOrderRole,
			Capabilities: capabilities,
		},
		Resource:    authorization.Resource{ID: strings.TrimSpace(incoming.Subject), Type: "identity_membership", TenantID: targetTenant, OrganizationID: targetOrg},
		Action:      authorization.CapabilityOrganizationAdmin,
		Environment: types.EnvironmentLive,
	})
	if !decision.Allowed {
		writeJSON(writer, http.StatusForbidden, response{Outcome: "REJECTED", Reason: decision.Reason})
		return
	}
	if _, err := grant(request.Context(), h.DB, identity.GrantInput{
		Issuer: strings.TrimSpace(incoming.Issuer), Subject: strings.TrimSpace(incoming.Subject),
		TenantID: targetTenant, OrganizationID: targetOrg,
		ActorID: actorID, Role: strings.TrimSpace(incoming.Role),
		Capabilities: incoming.Capabilities,
	}); err != nil {
		if errors.Is(err, identity.ErrInvalidGrant) {
			writeJSON(writer, http.StatusBadRequest, response{Outcome: "REJECTED", Reason: err.Error()})
			return
		}
		writeJSON(writer, http.StatusServiceUnavailable, response{Outcome: "REJECTED", Reason: "grant_failed"})
		return
	}
	if _, err := h.Auditor.Append(request.Context(), auditlog.Entry{
		TenantID: targetTenant, OrganizationID: targetOrg,
		EventType: "identity.membership.grant", EntityType: auditlog.EntityIdentity,
		EntityID: strings.TrimSpace(incoming.Subject), ActorID: membership.ActorID,
		Action:   auditlog.ActionCreate,
		NewValue: map[string]interface{}{"role": strings.TrimSpace(incoming.Role), "capabilities": incoming.Capabilities},
		Metadata: map[string]interface{}{"issuer": strings.TrimSpace(incoming.Issuer), "granted_actor": actorID},
	}); err != nil {
		// The grant upserts are idempotent, so a retry after fixing the
		// ledger is safe. Never report success for an unaudited grant.
		writeJSON(writer, http.StatusServiceUnavailable, response{Outcome: "REJECTED", Reason: "grant_applied_audit_failed"})
		return
	}
	writeJSON(writer, http.StatusOK, response{Outcome: "GRANTED", ActorID: actorID, TenantID: targetTenant, Capabilities: incoming.Capabilities})
}

func bearer(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func writeJSON(writer http.ResponseWriter, status int, value response) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
