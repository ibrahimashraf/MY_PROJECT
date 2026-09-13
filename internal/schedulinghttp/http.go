package schedulinghttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"integin/internal/domain/scheduling"
	"integin/internal/identity"
	"integin/internal/oidcauth"
	"integin/pkg/onboarding"
)

type TokenValidator interface {
	Validate(ctx context.Context, token string) (oidcauth.Principal, error)
}

type Handler struct {
	Validator TokenValidator
	Resolver  identity.Resolver
	Now       func() time.Time
}

type checkSkillsRequest struct {
	Credential     onboarding.InspectorCredential    `json:"credential"`
	RequiredSkills []string                          `json:"required_skills"`
	Competencies   []scheduling.TechnicianCompetency `json:"competencies"`
}

type checkSkillsResponse struct {
	Passed bool   `json:"passed"`
	Reason string `json:"reason,omitempty"`
}

func writeJSONError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if h.Validator != nil {
		auth := r.Header.Get("Authorization")
		if len(auth) < 7 || auth[:7] != "Bearer " {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		token := auth[7:]
		principal, err := h.Validator.Validate(r.Context(), token)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if h.Resolver != nil {
			_, err := h.Resolver.Resolve(r.Context(), identity.PrincipalKey{Issuer: principal.Issuer, Subject: principal.Subject})
			if err != nil {
				writeJSONError(w, http.StatusForbidden, "forbidden")
				return
			}
		}
	}

	var req checkSkillsRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request")
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSONError(w, http.StatusBadRequest, "bad_request")
		return
	}

	now := time.Now().UTC()
	if h.Now != nil {
		now = h.Now()
	}

	err := scheduling.CheckAssignmentSkills(now, req.Credential, req.RequiredSkills, req.Competencies)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(checkSkillsResponse{
			Passed: false,
			Reason: err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(checkSkillsResponse{
		Passed: true,
	})
}
