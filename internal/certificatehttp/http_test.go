package certificatehttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"integin/internal/certificatepg"
	"integin/internal/domain/certificate"
	"integin/internal/domain/certificateauthority"
	"integin/internal/oidcauth"
	"integin/internal/storage"
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
	event certificate.SignatureEvent
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
func (s *lifecycleStub) Sign(_ context.Context, a certificateauthority.ActorContext, _ string, event certificate.SignatureEvent, _ time.Time) error {
	s.actor = a
	s.event = event
	return event.Validate()
}
func (s *lifecycleStub) Attest(context.Context, certificateauthority.ActorContext, string, string, string, string, string, time.Time) error {
	return nil
}
func (s *lifecycleStub) WaiveSign(context.Context, certificateauthority.ActorContext, string, string, string, string, time.Time) error {
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
func (s *lifecycleStub) Renew(context.Context, certificateauthority.ActorContext, string, string, time.Time) error {
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

func TestHandlerRenewsCertificate(t *testing.T) {
	actor := certificateauthority.ActorContext{TenantID: "tenant", OrganizationID: "org", ActorID: "renewer", Capabilities: map[string]bool{"certificate.renew": true}}
	h := Handler{Validator: validatorStub{}, Actors: actorStub{actor: actor}, Lifecycle: &lifecycleStub{}}
	request := httptest.NewRequest(http.MethodPost, "/certificates/certificate-a/renew", strings.NewReader(`{"reason":"renewed for new period"}`))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	unauthorized := httptest.NewRequest(http.MethodPost, "/certificates/certificate-a/renew", strings.NewReader(`{"reason":"renewed"}`))
	unauthResponse := httptest.NewRecorder()
	h.ServeHTTP(unauthResponse, unauthorized)
	if unauthResponse.Code != http.StatusUnauthorized {
		t.Fatalf("missing auth status=%d", unauthResponse.Code)
	}
}

func TestHandlerSignsWithSignatureEvent(t *testing.T) {
	life := &lifecycleStub{}
	actor := certificateauthority.ActorContext{TenantID: "tenant", OrganizationID: "org", ActorID: "signer", Capabilities: map[string]bool{"certificate.sign": true}}
	h := Handler{Validator: validatorStub{}, Actors: actorStub{actor: actor}, Lifecycle: life}
	body := `{"signer_id":"signer","signer_name":"Sahil Singh","capacity":"client","statement_version":"client_ack_v1","image_sha256_hex":"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08","image_bytes":48210,"image_evidence_id":"evidence-signature-1","snapshot_sha256_hex":"5feceb66ffc86f38d952786c6d696c79c2dbc239dd4e91b46729d73a27fb57e9"}`
	request := httptest.NewRequest(http.MethodPost, "/certificates/certificate-a/sign", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if life.event.SignerName != "Sahil Singh" || life.event.Capacity != "client" {
		t.Fatalf("event not forwarded: %#v", life.event)
	}
	hollow := httptest.NewRequest(http.MethodPost, "/certificates/certificate-a/sign", nil)
	hollow.Header.Set("Authorization", "Bearer token")
	hollowResponse := httptest.NewRecorder()
	h.ServeHTTP(hollowResponse, hollow)
	if hollowResponse.Code != http.StatusBadRequest {
		t.Fatalf("bodyless sign must be rejected, status=%d", hollowResponse.Code)
	}
}

func TestHandlerAttestsFindings(t *testing.T) {
	actor := certificateauthority.ActorContext{TenantID: "tenant", OrganizationID: "org", ActorID: "inspector", Capabilities: map[string]bool{"certificate.attest": true}}
	h := Handler{Validator: validatorStub{}, Actors: actorStub{actor: actor}, Lifecycle: &lifecycleStub{}}
	body := `{"attestor_name":"Inspector I","qualification_basis":"Company Appointed Examiner","statement_version":"inspector_attest_v1","snapshot_sha256_hex":"5feceb66ffc86f38d952786c6d696c79c2dbc239dd4e91b46729d73a27fb57e9"}`
	request := httptest.NewRequest(http.MethodPost, "/certificates/certificate-a/attest", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestHandlerWaivesSignWithReason(t *testing.T) {
	actor := certificateauthority.ActorContext{TenantID: "tenant", OrganizationID: "org", ActorID: "authority", Capabilities: map[string]bool{"certificate.sign": true}}
	h := Handler{Validator: validatorStub{}, Actors: actorStub{actor: actor}, Lifecycle: &lifecycleStub{}}
	body := `{"granted_by":"authority","reason":"client unreachable on site","capacity":"client"}`
	request := httptest.NewRequest(http.MethodPost, "/certificates/certificate-a/sign-waiver", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	hollow := httptest.NewRequest(http.MethodPost, "/certificates/certificate-a/sign-waiver", nil)
	hollow.Header.Set("Authorization", "Bearer token")
	hollowResponse := httptest.NewRecorder()
	h.ServeHTTP(hollowResponse, hollow)
	if hollowResponse.Code != http.StatusBadRequest {
		t.Fatalf("bodyless waiver must be rejected, status=%d", hollowResponse.Code)
	}
}

type artifactRetrieverStub struct {
	record certificatepg.ArtifactRecord
	object storage.Object
	err    error
}

func (s *artifactRetrieverStub) GetArtifact(_ context.Context, _ certificateauthority.ActorContext, _ string, _ string) (certificatepg.ArtifactRecord, error) {
	if s.err != nil {
		return certificatepg.ArtifactRecord{}, s.err
	}
	return s.record, nil
}

func (s *artifactRetrieverStub) Get(_ context.Context, _ string) (storage.Object, error) {
	if s.err != nil {
		return storage.Object{}, s.err
	}
	return s.object, nil
}

func TestHandlerRetrievesArtifact(t *testing.T) {
	actor := certificateauthority.ActorContext{TenantID: "tenant", OrganizationID: "org", ActorID: "viewer", Capabilities: map[string]bool{"certificate.view": true}}
	retriever := &artifactRetrieverStub{
		record: certificatepg.ArtifactRecord{
			ID:            "art-1",
			CertificateID: "cert-123",
			ObjectKey:     "tenants/tenant/certs/cert-123/certificate.pdf",
		},
		object: storage.Object{
			Key:         "tenants/tenant/certs/cert-123/certificate.pdf",
			ContentType: "application/pdf",
			Data:        []byte("%PDF-1.7 mock content"),
		},
	}
	h := Handler{
		Validator: validatorStub{},
		Actors:    actorStub{actor: actor},
		Lifecycle: &lifecycleStub{},
		Artifacts: retriever,
	}

	// 1. Success on GET /certificates/cert-123/artifact
	req := httptest.NewRequest(http.MethodGet, "/certificates/cert-123/artifact", nil)
	req.Header.Set("Authorization", "Bearer token")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if res.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("content-type=%q", res.Header().Get("Content-Type"))
	}
	if res.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("cache-control=%q", res.Header().Get("Cache-Control"))
	}
	if res.Body.String() != "%PDF-1.7 mock content" {
		t.Fatalf("body mismatch: %q", res.Body.String())
	}

	// 2. Success on alias GET /certificates/cert-123/pdf
	req2 := httptest.NewRequest(http.MethodGet, "/certificates/cert-123/pdf", nil)
	req2.Header.Set("Authorization", "Bearer token")
	res2 := httptest.NewRecorder()
	h.ServeHTTP(res2, req2)
	if res2.Code != http.StatusOK {
		t.Fatalf("pdf alias status=%d", res2.Code)
	}

	// 3. Unauthenticated request rejected
	reqUnauth := httptest.NewRequest(http.MethodGet, "/certificates/cert-123/artifact", nil)
	resUnauth := httptest.NewRecorder()
	h.ServeHTTP(resUnauth, reqUnauth)
	if resUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthorized, got %d", resUnauth.Code)
	}
}
