// INTEGIN session auth tests: lifecycle expiry/revocation, clock control, and OrganizationContext binding.
package oidchttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"integin/internal/identity"
	"integin/internal/oidcauth"
)

type fakeClock struct {
	mu      sync.Mutex
	current time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = c.current.Add(d)
}

func newManaged(t *testing.T, maxTTL time.Duration) (*SessionLifecycleManager, *fakeClock) {
	t.Helper()
	manager := NewSessionLifecycleManager(maxTTL)
	clock := &fakeClock{current: time.Now()}
	manager.now = clock.now
	return manager, clock
}

func TestSessionLifecycleOpenActiveAndTTLExpiry(t *testing.T) {
	manager, clock := newManaged(t, time.Hour)
	if err := manager.Open("token-1", "subject-1", time.Now()); err != nil {
		t.Fatalf("open: %v", err)
	}
	if !manager.IsActive("token-1") {
		t.Fatal("fresh session must be active")
	}
	clock.advance(2 * time.Hour)
	if manager.IsActive("token-1") {
		t.Fatal("session must expire after the maximum TTL")
	}
	if !manager.Known("token-1") {
		t.Fatal("expired session must remain known until purge")
	}
	if err := manager.Touch("token-1"); err != ErrSessionExpired {
		t.Fatalf("touching an expired session: got %v", err)
	}
	manager.PurgeExpired()
	if manager.Known("token-1") {
		t.Fatal("purged session must no longer be known")
	}
}

func TestSessionLifecycleInactivityWindowIsSliding(t *testing.T) {
	manager, clock := newManaged(t, time.Hour)
	if err := manager.Open("token-1", "subject-1", clock.now()); err != nil {
		t.Fatalf("open: %v", err)
	}
	clock.advance(30 * time.Minute)
	if err := manager.Touch("token-1"); err != nil {
		t.Fatalf("touch within inactivity window: %v", err)
	}
	clock.advance(45 * time.Minute)
	if manager.IsActive("token-1") {
		t.Fatal("session must expire after 45 minutes of inactivity following a touch at 30 minutes")
	}
}

func TestSessionLifecycleRevocation(t *testing.T) {
	manager, _ := newManaged(t, time.Hour)
	if err := manager.Open("token-1", "subject-1", time.Now()); err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := manager.RevokeSession("token-1"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if !manager.IsRevoked("token-1") {
		t.Fatal("revoked session must report IsRevoked")
	}
	if manager.IsActive("token-1") {
		t.Fatal("revoked session must not be active")
	}
	if err := manager.RevokeSession("token-1"); err != ErrSessionMissing {
		t.Fatalf("revoking an already revoked session: got %v", err)
	}
	if err := manager.Open("token-1", "subject-1", time.Now()); err != ErrSessionRevoked {
		t.Fatalf("reopening a revoked token must fail closed, got %v", err)
	}
	if err := manager.Touch("token-1"); err != ErrSessionRevoked {
		t.Fatalf("touching a revoked token must fail closed, got %v", err)
	}
	if err := manager.RevokeSession(""); err != ErrSessionMissing {
		t.Fatalf("revoking a blank id: got %v", err)
	}
}

func TestSessionLifecycleRevokeSubject(t *testing.T) {
	manager, _ := newManaged(t, time.Hour)
	if err := manager.Open("token-a", "user-1", time.Now()); err != nil {
		t.Fatalf("open a: %v", err)
	}
	if err := manager.Open("token-b", "user-1", time.Now()); err != nil {
		t.Fatalf("open b: %v", err)
	}
	if err := manager.Open("token-c", "user-2", time.Now()); err != nil {
		t.Fatalf("open c: %v", err)
	}

	if err := manager.RevokeSubject("user-1"); err != nil {
		t.Fatalf("revoke subject: %v", err)
	}

	if manager.IsActive("token-a") || manager.IsActive("token-b") {
		t.Fatal("user-1 sessions must both be inactive")
	}
	if !manager.IsRevoked("token-a") || !manager.IsRevoked("token-b") {
		t.Fatal("user-1 tokens must report revoked")
	}
	if !manager.IsActive("token-c") {
		t.Fatal("user-2 session must remain active")
	}
}

func TestSessionLifecycleConcurrentAccess(t *testing.T) {
	manager := NewSessionLifecycleManager(time.Hour)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			token := "token-" + string(rune('a'+n%5))
			_ = manager.Open(token, "subject-"+string(rune('a'+n%5)), time.Now())
			_ = manager.Touch(token)
			_ = manager.IsActive(token)
			_ = manager.Known(token)
			if n%7 == 0 {
				_ = manager.RevokeSession(token)
			}
			_ = manager.IsRevoked(token)
		}(i)
	}
	wg.Wait()
}

func TestOrganizationContextContextRoundTrip(t *testing.T) {
	org := identity.OrganizationContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "actor-1", PrincipalType: identity.PrincipalTypeServiceAccount}
	ctx := WithOrganizationContext(context.Background(), org)
	got, ok := OrganizationContextFrom(ctx)
	if !ok || got.TenantID != org.TenantID || got.PrincipalType != org.PrincipalType {
		t.Fatalf("context round-trip failed: %+v ok=%v", got, ok)
	}
	if _, ok := OrganizationContextFrom(context.Background()); ok {
		t.Fatal("absent context must report ok=false")
	}
}

func newAuthenticator(t *testing.T, validator TokenValidator, resolver identity.Resolver, sessions *SessionLifecycleManager, mfa identity.MFAPolicy) *SessionAuthenticator {
	t.Helper()
	auth, err := NewSessionAuthenticator(validator, resolver, sessions, mfa)
	if err != nil {
		t.Fatalf("construct authenticator: %v", err)
	}
	return auth
}

type capturedContext struct {
	Org  identity.OrganizationContext `json:"org"`
	Hits int                          `json:"hits"`
}

func captureBoundOrg(org *capturedContext) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		bound, ok := OrganizationContextFrom(request.Context())
		if !ok {
			writeFailure(writer, http.StatusInternalServerError, "missing_context")
			return
		}
		org.Org = bound
		org.Hits++
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(map[string]string{"tenant": bound.TenantID, "organization": bound.OrganizationID})
	})
}

func TestSessionAuthenticatorBindsOrganizationContextAndEstablishesSession(t *testing.T) {
	sessions, _ := newManaged(t, time.Hour)
	validator := &fakeValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "human-001", AMR: []string{"mfa"}}}
	resolver := &fakeResolver{membership: identity.Membership{ActorID: "actor-001", TenantID: "tenant-a", OrganizationID: "org-a", WorkOrderRole: "field_engineer", Capabilities: []string{"inspection.read"}}}
	auth := newAuthenticator(t, validator, resolver, sessions, identity.MFAPolicy{RequireMFA: true, AllowedAMR: []string{"mfa"}})

	var captured capturedContext
	handler := auth.Middleware(captureBoundOrg(&captured))

	response := serve(handler, http.MethodGet, "Bearer token-1")
	if response.Code != http.StatusOK {
		t.Fatalf("first request status=%d body=%s", response.Code, response.Body.String())
	}
	if !sessions.IsActive("token-1") {
		t.Fatal("first request must establish an active session")
	}
	if captured.Org.TenantID != "tenant-a" || captured.Org.OrganizationID != "org-a" || captured.Org.ActorID != "actor-001" {
		t.Fatalf("bound context = %+v", captured.Org)
	}
	if captured.Org.PrincipalType != identity.PrincipalTypeHuman || len(captured.Org.Roles) != 1 || captured.Org.Roles[0] != "field_engineer" {
		t.Fatalf("bound context = %+v", captured.Org)
	}

	second := serve(handler, http.MethodGet, "Bearer token-1")
	if second.Code != http.StatusOK || captured.Hits != 2 {
		t.Fatalf("second request status=%d hits=%d", second.Code, captured.Hits)
	}
	if !sessions.IsActive("token-1") {
		t.Fatal("second request must refresh the active session")
	}
}

func TestSessionAuthenticatorRejectsMissingBearerToken(t *testing.T) {
	sessions, _ := newManaged(t, time.Hour)
	validator := &fakeValidator{}

	handler := newAuthenticator(t, validator, &fakeResolver{}, sessions, identity.MFAPolicy{}).Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	request := httptest.NewRequest(http.MethodGet, "/secure", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || validator.calls != 0 {
		t.Fatalf("status=%d validator=%d", response.Code, validator.calls)
	}
}

func TestSessionAuthenticatorRejectsRevokedSessionBeforeValidation(t *testing.T) {
	sessions, _ := newManaged(t, time.Hour)
	if err := sessions.Open("token-1", "human-001", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := sessions.RevokeSession("token-1"); err != nil {
		t.Fatal(err)
	}
	validator := &fakeValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "human-001"}}
	auth := newAuthenticator(t, validator, &fakeResolver{}, sessions, identity.MFAPolicy{})
	response := serve(auth.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})), http.MethodGet, "Bearer token-1")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", response.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(response.Body.Bytes(), &body)
	if body["error"] != "session_revoked" || validator.calls != 0 {
		t.Fatalf("body=%v validator=%d", body, validator.calls)
	}
	if err := sessions.Open("token-1", "human-001", time.Now()); err != ErrSessionRevoked {
		t.Fatalf("revoked token must not re-establish, got %v", err)
	}
}

func TestSessionAuthenticatorRejectsExpiredEstablishedSession(t *testing.T) {
	sessions, clock := newManaged(t, 30*time.Minute)
	validator := &fakeValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "service-account:pipeline", AMR: []string{"mfa"}}}
	resolver := &fakeResolver{membership: identity.Membership{ActorID: "sa-pipeline", TenantID: "tenant-a", OrganizationID: "org-a", Capabilities: []string{"data.ingest.write"}}}

	client := time.Now()
	if err := sessions.Open("token-1", "sa-pipeline", client); err != nil {
		t.Fatal(err)
	}
	auth := newAuthenticator(t, validator, resolver, sessions, identity.MFAPolicy{RequireMFA: true, AllowedAMR: []string{"mfa"}})
	if !sessions.IsActive("token-1") {
		t.Fatal("session must be active when first opened")
	}
	clock.advance(2 * time.Hour)
	if sessions.Known("token-1") && sessions.IsActive("token-1") {
		t.Fatal("session must be expired after clock advance")
	}
	response := serve(auth.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})), http.MethodGet, "Bearer token-1")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status=%d", response.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(response.Body.Bytes(), &body)
	if body["error"] != "session_expired" {
		t.Fatalf("expired session body=%v", body)
	}
}

func TestSessionAuthenticatorRejectsMFAViolation(t *testing.T) {
	sessions, _ := newManaged(t, time.Hour)
	validator := &fakeValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "human-001", AMR: []string{"pwd"}}}
	resolver := &fakeResolver{membership: identity.Membership{ActorID: "actor-001", TenantID: "tenant-a", OrganizationID: "org-a"}}
	auth := newAuthenticator(t, validator, resolver, sessions, identity.MFAPolicy{RequireMFA: true, AllowedAMR: []string{"mfa"}})
	response := serve(auth.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})), http.MethodGet, "Bearer token-1")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d", response.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(response.Body.Bytes(), &body)
	if body["error"] != "mfa_required" {
		t.Fatalf("mfa body=%v", body)
	}
	if sessions.IsActive("token-1") {
		t.Fatal("failed MFA must not establish a session")
	}
}

func TestSessionAuthenticatorRejectsUnknownMembershipAndResolverFailure(t *testing.T) {
	sessions, _ := newManaged(t, time.Hour)
	validator := &fakeValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "human-001"}}
	unknown := &fakeResolver{err: identity.ErrUnknownSubject}
	auth := newAuthenticator(t, validator, unknown, sessions, identity.MFAPolicy{})
	if response := serve(auth.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})), http.MethodGet, "Bearer token-1"); response.Code != http.StatusForbidden {
		t.Fatalf("unknown status=%d", response.Code)
	}
	unavailable := &fakeResolver{err: context.DeadlineExceeded}
	auth = newAuthenticator(t, validator, unavailable, sessions, identity.MFAPolicy{})
	if response := serve(auth.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})), http.MethodGet, "Bearer token-1"); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable status=%d", response.Code)
	}
	if sessions.Known("token-1") {
		t.Fatal("failed authorization must not establish a session")
	}
}

func TestSessionAuthenticatorRejectsConflictingTenantScope(t *testing.T) {
	sessions, _ := newManaged(t, time.Hour)
	validator := &fakeValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "human-001", AMR: []string{"pwd"}}}
	resolver := &fakeResolver{membership: identity.Membership{ActorID: "actor-001", TenantID: "tenant-a", OrganizationID: "org-a", Capabilities: []string{"inspection.read"}}}
	auth := newAuthenticator(t, validator, resolver, sessions, identity.MFAPolicy{})
	handler := auth.Middleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))

	for name, spoof := range map[string]func(*http.Request){
		"header": func(r *http.Request) { r.Header.Set("X-Tenant-ID", "tenant-b") },
		"org":    func(r *http.Request) { r.Header.Set("X-Organization-ID", "org-attacker") },
		"query":  func(r *http.Request) { r.URL.RawQuery = "tenant_id=tenant-b" },
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/secure", nil)
			request.Header.Set("Authorization", "Bearer token-1")
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
			if sessions.IsActive("token-1") {
				t.Fatal("spoiled scope must not establish a session")
			}
		})
	}

	matching := httptest.NewRequest(http.MethodGet, "/secure", nil)
	matching.Header.Set("Authorization", "Bearer token-1")
	matching.Header.Set("X-Tenant-ID", "tenant-a")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, matching)
	if response.Code != http.StatusOK {
		t.Fatalf("matching header status=%d", response.Code)
	}
}

func TestNewSessionAuthenticatorRejectsNilDependencies(t *testing.T) {
	sessions, _ := newManaged(t, time.Hour)
	if _, err := NewSessionAuthenticator(nil, &fakeResolver{}, sessions, identity.MFAPolicy{}); err == nil {
		t.Fatal("nil validator must be rejected")
	}
	if _, err := NewSessionAuthenticator(&fakeValidator{}, nil, sessions, identity.MFAPolicy{}); err == nil {
		t.Fatal("nil resolver must be rejected")
	}
	if _, err := NewSessionAuthenticator(&fakeValidator{}, &fakeResolver{}, nil, identity.MFAPolicy{}); err == nil {
		t.Fatal("nil lifecycle manager must be rejected")
	}
}
