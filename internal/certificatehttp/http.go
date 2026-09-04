package certificatehttp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"integin/internal/certificatepg"
	"integin/internal/domain/certificateauthority"
	"integin/internal/oidcauth"
	"integin/internal/storage"
)

type TokenValidator interface {
	Validate(context.Context, string) (oidcauth.Principal, error)
}
type ActorResolver interface {
	Resolve(context.Context, oidcauth.Principal) (certificateauthority.ActorContext, error)
}
type Lifecycle interface {
	CreateDraft(context.Context, certificateauthority.ActorContext, certificateauthority.CreateDraftRequest, time.Time) (*certificateauthority.Certificate, error)
	Submit(context.Context, certificateauthority.ActorContext, string, time.Time) error
	Review(context.Context, certificateauthority.ActorContext, string, time.Time) error
	Sign(context.Context, certificateauthority.ActorContext, string, time.Time) error
	Issue(context.Context, certificateauthority.ActorContext, string, time.Time) (certificatepg.IssueResult, error)
	Revoke(context.Context, certificateauthority.ActorContext, string, string, time.Time) error
	Renew(context.Context, certificateauthority.ActorContext, string, string, time.Time) error
	Supersede(context.Context, certificateauthority.ActorContext, string, string, time.Time) error
	Expire(context.Context, certificateauthority.ActorContext, string, time.Time) error
}

type ArtifactRetriever interface {
	GetArtifact(ctx context.Context, actor certificateauthority.ActorContext, certificateID, artifactType string) (certificatepg.ArtifactRecord, error)
	Get(ctx context.Context, key string) (storage.Object, error)
}

type Handler struct {
	Validator TokenValidator
	Actors    ActorResolver
	Lifecycle Lifecycle
	Artifacts ArtifactRetriever
	Now       func() time.Time
}
type draftRequest struct {
	CertificateID   string                       `json:"certificate_id"`
	InspectionID    string                       `json:"inspection_id"`
	TemplateCode    string                       `json:"template_code"`
	TemplateVersion int64                        `json:"template_version"`
	Profile         certificateauthority.Profile `json:"profile"`
	SelfIssueReason string                       `json:"self_issue_reason"`
}
type revocationRequest struct {
	Reason string `json:"reason"`
}
type supersedeRequest struct {
	ReplacementCertificateID string `json:"replacement_certificate_id"`
}
type renewalRequest struct {
	Reason string `json:"reason"`
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.Validator == nil || h.Actors == nil || h.Lifecycle == nil {
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
	actor, err := h.Actors.Resolve(r.Context(), principal)
	if err != nil {
		write(w, http.StatusForbidden, "authorization_failed", nil)
		return
	}
	now := time.Now
	if h.Now != nil {
		now = h.Now
	}

	id, action, valid := route(r.URL.Path)

	if r.Method == http.MethodGet {
		if !valid || (action != "artifact" && action != "pdf") {
			write(w, http.StatusNotFound, "not_found", nil)
			return
		}
		if h.Artifacts == nil {
			write(w, http.StatusServiceUnavailable, "service_unavailable", nil)
			return
		}
		record, err := h.Artifacts.GetArtifact(r.Context(), actor, id, "CERTIFICATE_PDF")
		if err != nil {
			write(w, http.StatusNotFound, "not_found", nil)
			return
		}
		obj, err := h.Artifacts.Get(r.Context(), record.ObjectKey)
		if err != nil {
			write(w, http.StatusNotFound, "not_found", nil)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "inline; filename=\"certificate.pdf\"")
		w.Header().Set("Cache-Control", "private, no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(obj.Data)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		write(w, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	if strings.TrimSuffix(r.URL.Path, "/") == "/certificates/drafts" {
		var body draftRequest
		if !decode(w, r, &body) {
			return
		}
		record, err := h.Lifecycle.CreateDraft(r.Context(), actor, certificateauthority.CreateDraftRequest{CertificateID: body.CertificateID, InspectionID: body.InspectionID, TemplateCode: body.TemplateCode, TemplateVersion: body.TemplateVersion, Profile: body.Profile, SelfIssueReason: body.SelfIssueReason}, now().UTC())
		if err != nil {
			write(w, http.StatusConflict, "certificate_rejected", nil)
			return
		}
		write(w, http.StatusCreated, "", map[string]string{"certificate_id": record.ID(), "status": string(record.Status())})
		return
	}
	if !valid {
		write(w, http.StatusNotFound, "not_found", nil)
		return
	}
	if action == "revoke" {
		var body revocationRequest
		if !decode(w, r, &body) {
			return
		}
		if err := h.Lifecycle.Revoke(r.Context(), actor, id, body.Reason, now().UTC()); err != nil {
			write(w, http.StatusConflict, "certificate_rejected", nil)
			return
		}
		write(w, http.StatusNoContent, "", nil)
		return
	}
	if action == "supersede" {
		var body supersedeRequest
		if !decode(w, r, &body) {
			return
		}
		if err := h.Lifecycle.Supersede(r.Context(), actor, id, body.ReplacementCertificateID, now().UTC()); err != nil {
			write(w, http.StatusConflict, "certificate_rejected", nil)
			return
		}
		write(w, http.StatusNoContent, "", nil)
		return
	}
	if action == "renew" {
		var body renewalRequest
		if !decode(w, r, &body) {
			return
		}
		if err := h.Lifecycle.Renew(r.Context(), actor, id, body.Reason, now().UTC()); err != nil {
			write(w, http.StatusConflict, "certificate_rejected", nil)
			return
		}
		write(w, http.StatusNoContent, "", nil)
		return
	}
	if action == "expire" {
		if !emptyBody(w, r) {
			return
		}
		if err := h.Lifecycle.Expire(r.Context(), actor, id, now().UTC()); err != nil {
			write(w, http.StatusConflict, "certificate_rejected", nil)
			return
		}
		write(w, http.StatusNoContent, "", nil)
		return
	}
	if action == "issue" {
		if !emptyBody(w, r) {
			return
		}
		result, err := h.Lifecycle.Issue(r.Context(), actor, id, now().UTC())
		if err != nil {
			write(w, http.StatusConflict, "certificate_rejected", nil)
			return
		}
		write(w, http.StatusOK, "", map[string]any{"certificate_number": result.CertificateNumber, "public_token": result.PublicToken, "expires_at": result.ExpiresAt})
		return
	}
	if !emptyBody(w, r) {
		return
	}
	var operation error
	switch action {
	case "submit":
		operation = h.Lifecycle.Submit(r.Context(), actor, id, now().UTC())
	case "review":
		operation = h.Lifecycle.Review(r.Context(), actor, id, now().UTC())
	case "sign":
		operation = h.Lifecycle.Sign(r.Context(), actor, id, now().UTC())
	default:
		write(w, http.StatusNotFound, "not_found", nil)
		return
	}
	if operation != nil {
		write(w, http.StatusConflict, "certificate_rejected", nil)
		return
	}
	write(w, http.StatusNoContent, "", nil)
}

func route(path string) (string, string, bool) {
	p := strings.Split(strings.Trim(path, "/"), "/")
	if len(p) != 3 || p[0] != "certificates" || strings.TrimSpace(p[1]) == "" {
		return "", "", false
	}
	return p[1], p[2], true
}
func bearer(value string) (string, bool) {
	p := strings.Fields(value)
	if len(p) != 2 || !strings.EqualFold(p[0], "Bearer") || p[1] == "" {
		return "", false
	}
	return p[1], true
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if d.Decode(target) != nil {
		write(w, http.StatusBadRequest, "invalid_request", nil)
		return false
	}
	if d.Decode(&struct{}{}) == nil {
		write(w, http.StatusBadRequest, "invalid_request", nil)
		return false
	}
	return true
}
func emptyBody(w http.ResponseWriter, r *http.Request) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var value any
	if err := d.Decode(&value); err == io.EOF {
		return true
	}
	write(w, http.StatusBadRequest, "invalid_request", nil)
	return false
}
func write(w http.ResponseWriter, status int, code string, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if code != "" {
		_ = json.NewEncoder(w).Encode(map[string]any{"error": code})
		return
	}
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}
