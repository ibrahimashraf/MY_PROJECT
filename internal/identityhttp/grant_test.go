package identityhttp

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/domain/auditlog"
	"integin/internal/identity"
	"integin/internal/oidcauth"
)

type fakeValidator struct {
	principal oidcauth.Principal
	err       error
}

func (f *fakeValidator) Validate(context.Context, string) (oidcauth.Principal, error) {
	return f.principal, f.err
}

type fakeResolver struct {
	membership identity.Membership
	err        error
}

func (f *fakeResolver) Resolve(context.Context, identity.PrincipalKey) (identity.Membership, error) {
	return f.membership, f.err
}

type fakeAuditor struct {
	appended []auditlog.Entry
	err      error
}

func (f *fakeAuditor) Append(_ context.Context, entry auditlog.Entry) (auditlog.Entry, error) {
	f.appended = append(f.appended, entry)
	return entry, f.err
}

func adminMembership() identity.Membership {
	return identity.Membership{ActorID: "admin-1", TenantID: "tenant-1", OrganizationID: "org-1", WorkOrderRole: "administrator", Capabilities: []string{"organization.admin"}}
}

func callHandler(h *Handler, body string, token string) (int, response) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/identity-grants", bytes.NewBufferString(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	writer := httptest.NewRecorder()
	h.ServeHTTP(writer, request)
	var result response
	_ = json.Unmarshal(writer.Body.Bytes(), &result)
	return writer.Code, result
}

func testHandler() (*Handler, *fakeAuditor, *[]identity.GrantInput) {
	auditor := &fakeAuditor{}
	var grants []identity.GrantInput
	return &Handler{
		DB:        &sql.DB{},
		Validator: &fakeValidator{principal: oidcauth.Principal{Issuer: "https://idp.test", Subject: "admin-sub"}},
		Resolver:  &fakeResolver{membership: adminMembership()},
		Auditor:   auditor,
		Grant: func(_ context.Context, _ *sql.DB, in identity.GrantInput) (int64, error) {
			grants = append(grants, in)
			return 7, nil
		},
	}, auditor, &grants
}

func TestGrantRejectsMissingBearer(t *testing.T) {
	h, _, _ := testHandler()
	code, result := callHandler(h, `{}`, "")
	if code != http.StatusUnauthorized || result.Reason != "authentication_failed" {
		t.Fatalf("code=%d reason=%q", code, result.Reason)
	}
}

func TestGrantRejectsUnknownSubjectAsBootstrap(t *testing.T) {
	h, _, _ := testHandler()
	h.Resolver = &fakeResolver{err: identity.ErrUnknownSubject}
	code, result := callHandler(h, `{"issuer":"x","subject":"y","role":"inspector","capabilities":["workorder.submit_partial"]}`, "tok")
	if code != http.StatusForbidden || result.Reason != "bootstrap_required" {
		t.Fatalf("code=%d reason=%q", code, result.Reason)
	}
}

func TestGrantRejectsNonAdminCaller(t *testing.T) {
	h, auditor, grants := testHandler()
	h.Resolver = &fakeResolver{membership: identity.Membership{ActorID: "insp-1", TenantID: "tenant-1", OrganizationID: "org-1", WorkOrderRole: "inspector", Capabilities: []string{"workorder.submit_partial"}}}
	code, result := callHandler(h, `{"issuer":"https://idp.test","subject":"new-sub","role":"inspector","capabilities":["workorder.submit_partial"]}`, "tok")
	if code != http.StatusForbidden || result.Outcome != "REJECTED" {
		t.Fatalf("code=%d outcome=%q reason=%q", code, result.Outcome, result.Reason)
	}
	if len(*grants) != 0 || len(auditor.appended) != 0 {
		t.Fatal("non-admin call must not grant or audit")
	}
}

func TestGrantRejectsCrossTenantTarget(t *testing.T) {
	h, _, grants := testHandler()
	code, _ := callHandler(h, `{"issuer":"https://idp.test","subject":"new-sub","tenant_id":"tenant-2","organization_id":"org-1","role":"inspector","capabilities":["workorder.submit_partial"]}`, "tok")
	if code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 cross-tenant", code)
	}
	if len(*grants) != 0 {
		t.Fatal("cross-tenant call must not grant")
	}
}

func TestGrantRejectsInvalidInput(t *testing.T) {
	h, auditor, _ := testHandler()
	h.Grant = identity.GrantMembership // real validation rejects before any DB use
	code, _ := callHandler(h, `{"issuer":"https://idp.test","subject":"new-sub","role":"inspector","capabilities":["bogus"]}`, "tok")
	if code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", code)
	}
	if len(auditor.appended) != 0 {
		t.Fatal("invalid grant must not audit")
	}
}

func TestGrantSuccessAuditsLedger(t *testing.T) {
	h, auditor, grants := testHandler()
	code, result := callHandler(h, `{"issuer":"https://idp.test","subject":"new-sub","role":"inspector","capabilities":["workorder.submit_partial"]}`, "tok")
	if code != http.StatusOK || result.Outcome != "GRANTED" {
		t.Fatalf("code=%d outcome=%q reason=%q", code, result.Outcome, result.Reason)
	}
	if len(*grants) != 1 {
		t.Fatalf("grants=%d, want 1", len(*grants))
	}
	got := (*grants)[0]
	if got.TenantID != "tenant-1" || got.OrganizationID != "org-1" || got.ActorID != "new-sub" || got.Role != "inspector" {
		t.Fatalf("grant input = %+v", got)
	}
	if len(auditor.appended) != 1 {
		t.Fatalf("audit appends=%d, want 1", len(auditor.appended))
	}
	entry := auditor.appended[0]
	if entry.EntityType != auditlog.EntityIdentity || entry.EntityID != "new-sub" || entry.ActorID != "admin-1" {
		t.Fatalf("audit entry = %+v", entry)
	}
}

func TestGrantAuditFailureReportsUnsafe(t *testing.T) {
	h, auditor, _ := testHandler()
	auditor.err = errors.New("ledger down")
	code, result := callHandler(h, `{"issuer":"https://idp.test","subject":"new-sub","role":"inspector","capabilities":["workorder.submit_partial"]}`, "tok")
	if code != http.StatusServiceUnavailable || result.Reason != "grant_applied_audit_failed" {
		t.Fatalf("code=%d reason=%q", code, result.Reason)
	}
}
