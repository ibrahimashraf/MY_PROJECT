package oidchttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"integin/internal/identity"
	"integin/internal/oidcauth"
)

const SessionCapability = "identity.session.read"

type TokenValidator interface {
	Validate(context.Context, string) (oidcauth.Principal, error)
}

type sessionHandler struct {
	validator          TokenValidator
	resolver           identity.Resolver
	requiredCapability string
}

type sessionResponse struct {
	TenantID       string `json:"tenant_id"`
	OrganizationID string `json:"organization_id"`
}

func NewSessionHandler(validator TokenValidator, resolver identity.Resolver, requiredCapability string) (http.Handler, error) {
	if validator == nil {
		return nil, errors.New("OIDC session handler validator is required")
	}
	if resolver == nil {
		return nil, errors.New("OIDC session handler identity resolver is required")
	}
	requiredCapability = strings.TrimSpace(requiredCapability)
	if requiredCapability == "" {
		return nil, errors.New("OIDC session handler local capability is required")
	}
	return sessionHandler{validator: validator, resolver: resolver, requiredCapability: requiredCapability}, nil
}

func (h sessionHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rawToken, ok := bearerToken(request.Header.Get("Authorization"))
	if !ok {
		writeFailure(writer, http.StatusUnauthorized, "authentication_failed")
		return
	}
	principal, err := h.validator.Validate(request.Context(), rawToken)
	if err != nil {
		writeFailure(writer, http.StatusUnauthorized, "authentication_failed")
		return
	}
	membership, err := h.resolver.Resolve(request.Context(), identity.PrincipalKey{Issuer: principal.Issuer, Subject: principal.Subject})
	if err != nil {
		if errors.Is(err, identity.ErrUnknownSubject) || errors.Is(err, identity.ErrAmbiguousMembership) {
			writeFailure(writer, http.StatusForbidden, "authorization_failed")
			return
		}
		writeFailure(writer, http.StatusServiceUnavailable, "authorization_unavailable")
		return
	}
	if strings.TrimSpace(membership.TenantID) == "" || strings.TrimSpace(membership.OrganizationID) == "" || !hasCapability(membership.Capabilities, h.requiredCapability) {
		writeFailure(writer, http.StatusForbidden, "authorization_failed")
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(writer).Encode(sessionResponse{TenantID: membership.TenantID, OrganizationID: membership.OrganizationID})
}

func bearerToken(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func hasCapability(capabilities []string, required string) bool {
	for _, capability := range capabilities {
		if capability == required {
			return true
		}
	}
	return false
}

func writeFailure(writer http.ResponseWriter, status int, reason string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{"error": reason})
}
