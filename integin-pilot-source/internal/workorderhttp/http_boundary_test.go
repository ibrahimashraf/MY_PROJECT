package workorderhttp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"integin/internal/identity"
	"integin/internal/oidcauth"
)

func TestHandlerRejectsUnsupportedMethodBeforeDependencies(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/work-orders/partial-submissions", nil)
	Handler{}.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if response.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("Allow=%q, want %q", response.Header().Get("Allow"), http.MethodPost)
	}
}

func TestHandlerMapsAuthenticationAndResolverFailures(t *testing.T) {
	tests := []struct {
		name          string
		validator     TokenValidator
		resolver      identity.Resolver
		authorization string
		wantStatus    int
	}{
		{
			name:          "malformed bearer",
			validator:     testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
			resolver:      testResolver{membership: validHTTPMembership()},
			authorization: "Basic token",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "token validation",
			validator:     testValidator{err: errors.New("invalid token")},
			resolver:      testResolver{membership: validHTTPMembership()},
			authorization: "Bearer invalid",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "membership resolution",
			validator:     testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
			resolver:      testResolver{err: errors.New("membership unavailable")},
			authorization: "Bearer valid",
			wantStatus:    http.StatusForbidden,
		},
		{
			name:          "incomplete derived actor",
			validator:     testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
			resolver:      testResolver{membership: identity.Membership{ActorID: "actor", TenantID: "tenant", OrganizationID: "org"}},
			authorization: "Bearer valid",
			wantStatus:    http.StatusForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := Handler{Validator: test.validator, Resolver: test.resolver, Service: &testService{}}
			request := httptest.NewRequest(http.MethodPost, "/work-orders/partial-submissions", nil)
			request.Header.Set("Authorization", test.authorization)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func validHTTPMembership() identity.Membership {
	return identity.Membership{
		ActorID:        "actor",
		TenantID:       "tenant",
		OrganizationID: "org",
		WorkOrderRole:  "inspector",
		Capabilities:   []string{"workorder.submit_partial"},
	}
}

func validEvidenceMembership() identity.Membership {
	return identity.Membership{
		ActorID:        "actor",
		TenantID:       "tenant",
		OrganizationID: "org",
		WorkOrderRole:  "inspector",
		Capabilities:   []string{"workorder.add_evidence_reference"},
	}
}

func TestEvidenceHandlerRejectsUnsupportedMethodBeforeDependencies(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/work-orders/wo/evidence", nil)
	EvidenceHandler{}.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if response.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("Allow=%q, want %q", response.Header().Get("Allow"), http.MethodPost)
	}
}

func TestEvidenceHandlerMapsAuthenticationAndResolverFailures(t *testing.T) {
	tests := []struct {
		name          string
		validator     TokenValidator
		resolver      identity.Resolver
		authorization string
		wantStatus    int
	}{
		{
			name:          "malformed bearer",
			validator:     testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
			resolver:      testResolver{membership: validEvidenceMembership()},
			authorization: "Basic token",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "token validation",
			validator:     testValidator{err: errors.New("invalid token")},
			resolver:      testResolver{membership: validEvidenceMembership()},
			authorization: "Bearer invalid",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "membership resolution",
			validator:     testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
			resolver:      testResolver{err: errors.New("membership unavailable")},
			authorization: "Bearer valid",
			wantStatus:    http.StatusForbidden,
		},
		{
			name:          "incomplete derived actor",
			validator:     testValidator{principal: oidcauth.Principal{Issuer: "issuer", Subject: "subject"}},
			resolver:      testResolver{membership: identity.Membership{ActorID: "actor", TenantID: "tenant", OrganizationID: "org"}},
			authorization: "Bearer valid",
			wantStatus:    http.StatusForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := EvidenceHandler{Validator: test.validator, Resolver: test.resolver, Service: &testService{}}
			request := httptest.NewRequest(http.MethodPost, "/work-orders/wo/evidence", nil)
			request.Header.Set("Authorization", test.authorization)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}
