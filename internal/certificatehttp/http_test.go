package certificatehttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"integin/internal/certificatepg"
	"integin/internal/domain/certificateauthority"
	"integin/internal/oidcauth"
)

type validatorStub struct{ err error }

func (s validatorStub) Validate(context.Context, string) (oidcauth.Principal, error) {
	return oidcauth.Principal{Issuer: "issuer", Subject: "subject"}, s.err
}

type actorStub struct {
	actor certificateauthority.ActorContext
	err   error
}

func (s actorStub) Resolve(context.Context, oidcauth.Principal) (certificateauthority.ActorContext, error) {
	return s.actor, s.err
}

type lifecycleStub struct {
	actor certificateauthority.ActorContext
}

func (s *lifecycleStub) CreateDraft(_ context.Context, a certificateauthority.ActorContext, r certificateauthority.CreateDraftRequest, _ time.Time) (*certificateauthority.Certificate, error) {
	s.actor = a
	return certificateauthority.NewDraft(a, certificateauthority.CanonicalInspection{ID: r.InspectionID, TenantID: a.TenantID, OrganizationID: a.OrganizationID, AssetID: "asset", InspectorID: a.ActorID, Revision: 1, LifecycleState: "APPROVED", FinalizationState: "FINALIZED"}, certificateauthority.Policy{ID: "policy", TemplateCode: r.TemplateCode, TemplateVersion: r.TemplateVersion, Version: 1, Status: "APPROVED", ValidityDays: 30}, certificateauthority.DraftRequest{ID: r.CertificateID, TemplateCode: r.TemplateCode, TemplateVersion: r.TemplateVersion, Profile: r.Profile, SelfIssueReason: r.SelfIssueReason}, time.Now())
}
func (s *lifecycleStub) Submit(context.Context, certificateauthority.ActorContext, string, time.Time) error {
	return nil
}
func (s *lifecycleStub) Review(context.Context, certificateauthority.ActorContext, string, time.Time) error {
	return nil
}
func (s *lifecycleStub) Sign(context.Context, certificateauthority.ActorContext, string, time.Time) error {
	return nil
}
func (s *lifecycleStub) Issue(context.Context, certificateauthority.ActorContext, string, time.Time) (certificatepg.IssueResult, error) {
	return certificatepg.IssueResult{CertificateNumber: "CERT-00000001", PublicToken: "token-once"}, nil
}
func (s *lifecycleStub) Revoke(context.Context, certificateauthority.ActorContext, string, string, time.Time) error {
	return nil
}
func (s *lifecycleStub) Supersede(context.Context, certificateauthority.ActorContext, string, string, time.Time) error {
	return nil
}
func (s *lifecycleStub) Expire(context.Context, certificateauthority.ActorContext, string, time.Time) error {
	return nil
}

func TestHandlerRejectsMissingAuthenticationNoStore(t *testing.T) {
	h := Handler{Validator: validatorStub{}, Actors: actorStub{}, Lifecycle: &lifecycleStub{}}
	r := httptest.NewRequest(http.MethodPost, "/certificates/drafts", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status=%d cache=%q", w.Code, w.Header().Get("Cache-Control"))
	}
}
func TestHandlerRejectsUnknownDraftField(t *testing.T) {
	h := Handler{Validator: validatorStub{}, Actors: actorStub{actor: certificateauthority.ActorContext{TenantID: "tenant", OrganizationID: "org", ActorID: "actor", Capabilities: map[string]bool{"certificate.prepare": true}}}, Lifecycle: &lifecycleStub{}}
	r := httptest.NewRequest(http.MethodPost, "/certificates/drafts", strings.NewReader(`{"certificate_id":"c","inspection_id":"i","template_code":"t","template_version":1,"profile":"INDEPENDENT_REVIEW","tenant_id":"forged"}`))
	r.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}
func TestHandlerDerivesDraftActor(t *testing.T) {
	life := &lifecycleStub{}
	actor := certificateauthority.ActorContext{TenantID: "tenant", OrganizationID: "org", ActorID: "actor", Capabilities: map[string]bool{"certificate.prepare": true}}
	h := Handler{Validator: validatorStub{}, Actors: actorStub{actor: actor}, Lifecycle: life}
	r := httptest.NewRequest(http.MethodPost, "/certificates/drafts", strings.NewReader(`{"certificate_id":"c","inspection_id":"i","template_code":"t","template_version":1,"profile":"INDEPENDENT_REVIEW"}`))
	r.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated || life.actor.TenantID != "tenant" || life.actor.ActorID != "actor" {
		t.Fatalf("status=%d actor=%#v", w.Code, life.actor)
	}
}

func TestHandlerIssuesTokenOnce(t *testing.T) {
	actor := certificateauthority.ActorContext{TenantID: "tenant", OrganizationID: "org", ActorID: "issuer", Capabilities: map[string]bool{"certificate.issue": true}}
	h := Handler{Validator: validatorStub{}, Actors: actorStub{actor: actor}, Lifecycle: &lifecycleStub{}}
	request := httptest.NewRequest(http.MethodPost, "/certificates/certificate-a/issue", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" || !strings.Contains(response.Body.String(), "token-once") {
		t.Fatalf("status=%d cache=%q body=%s", response.Code, response.Header().Get("Cache-Control"), response.Body.String())
	}
	bad := httptest.NewRequest(http.MethodPost, "/certificates/certificate-a/issue", strings.NewReader(`{}`))
	bad.Header.Set("Authorization", "Bearer token")
	badResponse := httptest.NewRecorder()
	h.ServeHTTP(badResponse, bad)
	if badResponse.Code != http.StatusBadRequest {
		t.Fatalf("non-empty issue body status=%d", badResponse.Code)
	}
}
