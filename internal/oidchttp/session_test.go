package oidchttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/identity"
	"integin/internal/oidcauth"
)

type fakeValidator struct {
	principal oidcauth.Principal
	err       error
	calls     int
}

func (f *fakeValidator) Validate(_ context.Context, _ string) (oidcauth.Principal, error) {
	f.calls++
	return f.principal, f.err
}

type fakeResolver struct {
	membership identity.Membership
	err        error
	calls      int
}

func (f *fakeResolver) Resolve(_ context.Context, _ identity.PrincipalKey) (identity.Membership, error) {
	f.calls++
	return f.membership, f.err
}

func TestSessionHandlerRejectsUnauthenticatedInputWithoutResolution(t *testing.T) {
	validator := &fakeValidator{}
	resolver := &fakeResolver{}
	handler, err := NewSessionHandler(validator, resolver, SessionCapability)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/identity/session", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", response.Code)
	}
	if validator.calls != 0 || resolver.calls != 0 {
		t.Fatalf("validator=%d resolver=%d", validator.calls, resolver.calls)
	}
}

func TestSessionHandlerRejectsValidatorFailureWithoutResolution(t *testing.T) {
	validator := &fakeValidator{err: errors.New("bad signature")}
	resolver := &fakeResolver{}
	handler, err := NewSessionHandler(validator, resolver, SessionCapability)
	if err != nil {
		t.Fatal(err)
	}
	response := serve(handler, http.MethodGet, "Bearer token")
	if response.Code != http.StatusUnauthorized || resolver.calls != 0 {
		t.Fatalf("status=%d resolver=%d", response.Code, resolver.calls)
	}
}

func TestSessionHandlerRejectsUnknownMembershipAndMissingCapability(t *testing.T) {
	validator := &fakeValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}}
	unknown := &fakeResolver{err: identity.ErrUnknownSubject}
	handler, err := NewSessionHandler(validator, unknown, SessionCapability)
	if err != nil {
		t.Fatal(err)
	}
	if response := serve(handler, http.MethodGet, "Bearer token"); response.Code != http.StatusForbidden {
		t.Fatalf("unknown status=%d", response.Code)
	}
	missing := &fakeResolver{membership: identity.Membership{TenantID: "tenant-a", OrganizationID: "org-a", Capabilities: []string{"inspection.read"}}}
	handler, err = NewSessionHandler(validator, missing, SessionCapability)
	if err != nil {
		t.Fatal(err)
	}
	if response := serve(handler, http.MethodGet, "Bearer token"); response.Code != http.StatusForbidden {
		t.Fatalf("capability status=%d", response.Code)
	}
}

func TestSessionHandlerReturnsOnlyLocalContext(t *testing.T) {
	validator := &fakeValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject", AMR: []string{"pwd"}}}
	resolver := &fakeResolver{membership: identity.Membership{TenantID: "tenant-a", OrganizationID: "org-a", Capabilities: []string{SessionCapability}}}
	handler, err := NewSessionHandler(validator, resolver, SessionCapability)
	if err != nil {
		t.Fatal(err)
	}
	response := serve(handler, http.MethodGet, "Bearer token")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	body := map[string]string{}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 2 || body["tenant_id"] != "tenant-a" || body["organization_id"] != "org-a" {
		t.Fatalf("body=%#v", body)
	}
}

func TestSessionHandlerReturnsUnavailableForResolverFailureAndRejectsWrongMethod(t *testing.T) {
	validator := &fakeValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}}
	unavailable := &fakeResolver{err: errors.New("database unavailable")}
	handler, err := NewSessionHandler(validator, unavailable, SessionCapability)
	if err != nil {
		t.Fatal(err)
	}
	if response := serve(handler, http.MethodGet, "Bearer token"); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable status=%d", response.Code)
	}
	if response := serve(handler, http.MethodPost, "Bearer token"); response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status=%d", response.Code)
	}
}

func TestSessionHandlerRejectsConflictingTenantScope(t *testing.T) {
	validator := &fakeValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}}
	resolver := &fakeResolver{membership: identity.Membership{TenantID: "tenant-a", OrganizationID: "org-a", Capabilities: []string{SessionCapability}}}
	handler, err := NewSessionHandler(validator, resolver, SessionCapability)
	if err != nil {
		t.Fatal(err)
	}

	for name, spoof := range map[string]func(*http.Request){
		"header": func(r *http.Request) { r.Header.Set("X-Tenant-ID", "tenant-b") },
		"org":    func(r *http.Request) { r.Header.Set("X-Organization-ID", "org-b") },
		"query":  func(r *http.Request) { r.URL.RawQuery = "tenant_id=tenant-b" },
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/identity/session", nil)
			request.Header.Set("Authorization", "Bearer token")
			spoof(request)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("spoofed status=%d body=%s", response.Code, response.Body.String())
			}
			var body map[string]string
			_ = json.Unmarshal(response.Body.Bytes(), &body)
			if body["error"] != "tenant_scope_conflict" {
				t.Fatalf("body=%v", body)
			}
		})
	}

	matching := httptest.NewRequest(http.MethodGet, "/identity/session?tenant_id=tenant-a", nil)
	matching.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, matching)
	if response.Code != http.StatusOK {
		t.Fatalf("matching scoping status=%d body=%s", response.Code, response.Body.String())
	}
}

func serve(handler http.Handler, method, authorization string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/identity/session", nil)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type fakeRevoker struct {
	revokedSessions []string
	revokedSubjects []string
}

func (f *fakeRevoker) RevokeSession(sessionID string) error {
	f.revokedSessions = append(f.revokedSessions, sessionID)
	return nil
}

func (f *fakeRevoker) RevokeSubject(subject string) error {
	f.revokedSubjects = append(f.revokedSubjects, subject)
	return nil
}

func TestSessionRevocationHandlerSingleAndAll(t *testing.T) {
	validator := &fakeValidator{principal: oidcauth.Principal{Issuer: "issuer-1", Subject: "subject-1"}}
	revoker := &fakeRevoker{}

	singleHandler, err := NewSessionRevocationHandler(validator, revoker, false)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/identity/session/revoke", nil)
	req.Header.Set("Authorization", "Bearer tok-123")
	rec := httptest.NewRecorder()
	singleHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("single revoke status = %d", rec.Code)
	}
	if len(revoker.revokedSessions) != 1 || revoker.revokedSessions[0] != "tok-123" {
		t.Fatalf("revoked sessions: %v", revoker.revokedSessions)
	}

	allHandler, err := NewSessionRevocationHandler(validator, revoker, true)
	if err != nil {
		t.Fatal(err)
	}
	reqAll := httptest.NewRequest(http.MethodPost, "/identity/session/revoke-all", nil)
	reqAll.Header.Set("Authorization", "Bearer tok-123")
	recAll := httptest.NewRecorder()
	allHandler.ServeHTTP(recAll, reqAll)
	if recAll.Code != http.StatusOK {
		t.Fatalf("revoke-all status = %d", recAll.Code)
	}
	if len(revoker.revokedSubjects) != 1 || revoker.revokedSubjects[0] != "subject-1" {
		t.Fatalf("revoked subjects: %v", revoker.revokedSubjects)
	}
}
