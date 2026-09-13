// INTEGIN organization-aware authorization tests: MFA enforcement, service-account discrimination, and tenant isolation.
package identity

import (
	"errors"
	"reflect"
	"testing"

	"integin/internal/oidcauth"
)

func TestMFAPolicyEnforceFailClosesOnUnpermittedAuthentication(t *testing.T) {
	policy := MFAPolicy{RequireMFA: true, AllowedAMR: []string{"mfa", "otp", "hwk"}}
	for _, amr := range [][]string{nil, {}, {"pwd"}} {
		if err := policy.Enforce(oidcauth.Principal{Issuer: "issuer", Subject: "subject", AMR: amr}); !errors.Is(err, ErrMFARequired) {
			t.Fatalf("amr=%v expected ErrMFARequired, got %v", amr, err)
		}
	}
	for _, amr := range [][]string{{"otp", "pwd"}, {"otp"}, {"hwk", "pwd"}, {"mfa"}} {
		if err := policy.Enforce(oidcauth.Principal{Issuer: "issuer", Subject: "subject", AMR: amr}); err != nil {
			t.Fatalf("amr=%v expected permitted, got %v", amr, err)
		}
	}
	if err := policy.Enforce(oidcauth.Principal{Issuer: "issuer", Subject: "subject", AMR: []string{"pwd"}}); !errors.Is(err, ErrMFARequired) {
		t.Fatalf("expected MFA required for non-permitted method, got %v", err)
	}
	if err := (MFAPolicy{RequireMFA: false}).Enforce(oidcauth.Principal{Issuer: "issuer", Subject: "subject"}); err != nil {
		t.Fatalf("non-required MFA must pass, got %v", err)
	}
}

func TestMFAPolicyEnforceEmptyAllowedAMRFailsClosed(t *testing.T) {
	if err := (MFAPolicy{RequireMFA: true, AllowedAMR: []string{"mfa"}}).Enforce(oidcauth.Principal{Issuer: "issuer", Subject: "subject", AMR: []string{"mfa"}}); err != nil {
		t.Fatalf("explicit allowed amr must pass, got %v", err)
	}
	if err := (MFAPolicy{RequireMFA: true}).Enforce(oidcauth.Principal{Issuer: "issuer", Subject: "subject", AMR: []string{"mfa"}}); !errors.Is(err, ErrMFARequired) {
		t.Fatalf("empty allowed list under RequireMFA must fail closed, got %v", err)
	}
}

func TestProjectMembershipHumanLoginWithMFAPassAndFail(t *testing.T) {
	membership := Membership{ActorID: "actor-001", TenantID: "tenant-a", OrganizationID: "org-a", WorkOrderRole: "field_engineer", Capabilities: []string{"inspection.write"}}
	principal := oidcauth.Principal{Issuer: "issuer", Subject: "human-001", AMR: []string{"mfa"}}
	policy := MFAPolicy{RequireMFA: true, AllowedAMR: []string{"mfa"}}

	org, err := ProjectMembership(membership, principal, policy)
	if err != nil {
		t.Fatalf("human login with MFA must pass: %v", err)
	}
	if org.PrincipalType != PrincipalTypeHuman {
		t.Fatalf("principal type = %d", org.PrincipalType)
	}
	if org.ActorID != "actor-001" || org.TenantID != "tenant-a" || org.OrganizationID != "org-a" {
		t.Fatalf("context = %+v", org)
	}
	if !reflect.DeepEqual(org.Roles, []string{"field_engineer"}) {
		t.Fatalf("roles = %v", org.Roles)
	}
	if !reflect.DeepEqual(org.Capabilities, []string{"inspection.write"}) {
		t.Fatalf("capabilities = %v", org.Capabilities)
	}

	fail := oidcauth.Principal{Issuer: "issuer", Subject: "human-002", AMR: []string{"pwd"}}
	if _, err := ProjectMembership(membership, fail, policy); !errors.Is(err, ErrMFARequired) {
		t.Fatalf("human login without MFA must fail closed, got %v", err)
	}
}

func TestProjectMembershipServiceAccountRecognitionAndCapabilityScoping(t *testing.T) {
	membership := Membership{ActorID: "sa-pipeline", TenantID: "tenant-a", OrganizationID: "org-a", Capabilities: []string{"data.ingest.write"}}
	policy := MFAPolicy{RequireMFA: false}
	for _, principal := range []oidcauth.Principal{
		{Issuer: "issuer", Subject: "sa:deploy-bot"},
		{Issuer: "issuer", Subject: "service-account:deploy-bot"},
		{Issuer: "issuer", Subject: "deploy-bot", AuthorizedParty: "service-account:keycloak-client"},
	} {
		org, err := ProjectMembership(membership, principal, policy)
		if err != nil {
			t.Fatalf("service account projection %+v: %v", principal, err)
		}
		if org.PrincipalType != PrincipalTypeServiceAccount {
			t.Fatalf("principal %+v classified as %d", principal, org.PrincipalType)
		}
		if !reflect.DeepEqual(org.Capabilities, []string{"data.ingest.write"}) || len(org.Roles) != 0 {
			t.Fatalf("service account scope = %+v", org)
		}
	}
}

func TestProjectMembershipHumanDetectsNoServiceAccountPrefix(t *testing.T) {
	principal := oidcauth.Principal{Issuer: "issuer", Subject: "human-001", AuthorizedParty: "integin-web"}
	org, err := ProjectMembership(Membership{ActorID: "actor-001", TenantID: "tenant-a", OrganizationID: "org-a"}, principal, MFAPolicy{RequireMFA: false})
	if err != nil {
		t.Fatalf("human projection: %v", err)
	}
	if org.PrincipalType != PrincipalTypeHuman {
		t.Fatalf("human classified as %d", org.PrincipalType)
	}
}

func TestProjectMembershipFailsClosedOnMissingBindingAndCrossTenantScope(t *testing.T) {
	principal := oidcauth.Principal{Issuer: "issuer", Subject: "human-001"}
	policy := MFAPolicy{RequireMFA: false}
	for _, membership := range []Membership{
		{ActorID: "actor-001", TenantID: "tenant-a"},
		{ActorID: "", TenantID: "tenant-a", OrganizationID: "org-a"},
		{ActorID: "actor-001", OrganizationID: "org-a"},
	} {
		if _, err := ProjectMembership(membership, principal, policy); err == nil {
			t.Fatalf("expected fail-closed for %+v", membership)
		}
	}
	if _, err := ProjectMembership(Membership{ActorID: "actor-001", OrganizationID: "org-a"}, principal, policy); !errors.Is(err, ErrCrossTenantScope) {
		t.Fatalf("expected cross-tenant scope rejection, got %v", err)
	}
	if _, err := ProjectMembership(Membership{ActorID: "actor-001", TenantID: "tenant-a"}, principal, policy); !errors.Is(err, ErrMissingOrganizationBinding) {
		t.Fatalf("expected missing binding rejection, got %v", err)
	}
	if _, err := ProjectMembership(Membership{}, principal, policy); !errors.Is(err, ErrMissingOrganizationBinding) {
		t.Fatalf("expected missing binding rejection for empty membership, got %v", err)
	}
}

func TestProjectMembershipMultiOrganizationTenantIsolation(t *testing.T) {
	policy := MFAPolicy{RequireMFA: false}
	human := oidcauth.Principal{Issuer: "issuer", Subject: "human-001"}
	tenantA, err := ProjectMembership(Membership{ActorID: "actor-001", TenantID: "tenant-a", OrganizationID: "org-a", WorkOrderRole: "field_engineer", Capabilities: []string{"inspection.write"}}, human, policy)
	if err != nil {
		t.Fatal(err)
	}
	tenantB, err := ProjectMembership(Membership{ActorID: "actor-002", TenantID: "tenant-b", OrganizationID: "org-b", WorkOrderRole: "supervisor", Capabilities: []string{"evidence.approve"}}, human, policy)
	if err != nil {
		t.Fatal(err)
	}
	if tenantA.TenantID != "tenant-a" || tenantB.TenantID != "tenant-b" {
		t.Fatalf("tenant scoping leaked: %+v vs %+v", tenantA, tenantB)
	}
	if tenantA.OrganizationID == tenantB.OrganizationID || tenantA.ActorID == tenantB.ActorID {
		t.Fatalf("organization or actor identity leaked across tenants: %+v vs %+v", tenantA, tenantB)
	}
	if reflect.DeepEqual(tenantA.Roles, tenantB.Roles) || reflect.DeepEqual(tenantA.Capabilities, tenantB.Capabilities) {
		t.Fatalf("authorization leaked across tenants: %+v vs %+v", tenantA, tenantB)
	}
	tenantA.Capabilities = append(tenantA.Capabilities, "escalated.write")
	if len(tenantB.Capabilities) != 1 || tenantB.Capabilities[0] != "evidence.approve" {
		t.Fatalf("mutating one tenant projection affected another: %+v", tenantB)
	}
}

func TestPrincipalTypeOfInspectsSubjectAndAuthorizedParty(t *testing.T) {
	for _, principal := range []oidcauth.Principal{{Subject: "sa:pipeline"}, {Subject: "SERVICE-ACCOUNT:robot"}, {Subject: "plain", AuthorizedParty: "sa:relay"}} {
		if got := PrincipalTypeOf(principal); got != PrincipalTypeServiceAccount {
			t.Fatalf("principal %+v classified as %d", principal, got)
		}
	}
	for _, principal := range []oidcauth.Principal{{Subject: "human-001"}, {Subject: "human-001", AuthorizedParty: "integin-web"}} {
		if got := PrincipalTypeOf(principal); got != PrincipalTypeHuman {
			t.Fatalf("principal %+v classified as %d", principal, got)
		}
	}
}
