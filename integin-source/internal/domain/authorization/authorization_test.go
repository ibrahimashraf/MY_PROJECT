package authorization

import (
	"testing"

	"integin/internal/shared/featureflags"
	"integin/internal/shared/types"
)

func baseRequest() Request {
	return Request{
		Subject:     Subject{UserID: "user-1", TenantID: "tenant-1", OrganizationID: "org-1", Capabilities: map[Capability]bool{CapabilityInspectReview: true}, Scopes: map[string]bool{"site-1": true}},
		Resource:    Resource{ID: "inspection-1", Type: "inspection", TenantID: "tenant-1", OrganizationID: "org-1", ScopeID: "site-1", WorkflowState: "PENDING_REVIEW"},
		Action:      CapabilityInspectReview,
		Environment: types.EnvironmentLive,
	}
}

func TestAuthorizationAllowsValidRequest(t *testing.T) {
	decision := Evaluate(baseRequest())
	if !decision.Allowed || decision.TenantID != "tenant-1" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestAuthorizationDeniesCrossTenantResource(t *testing.T) {
	request := baseRequest()
	request.Resource.TenantID = "tenant-2"
	decision := Evaluate(request)
	if decision.Allowed || decision.Code != types.ErrTenantMismatch {
		t.Fatalf("expected tenant mismatch: %#v", decision)
	}
}

func TestAuthorizationDeniesCapabilityAndScope(t *testing.T) {
	request := baseRequest()
	request.Action = CapabilityInspectApprove
	if decision := Evaluate(request); decision.Allowed || decision.Code != types.ErrUnauthorized {
		t.Fatalf("expected capability denial: %#v", decision)
	}
	request = baseRequest()
	request.Subject.Scopes = map[string]bool{"site-2": true}
	if decision := Evaluate(request); decision.Allowed || decision.Code != types.ErrUnauthorized {
		t.Fatalf("expected scope denial: %#v", decision)
	}
}

func TestAuthorizationEnforcesWorkflowFeatureAndEnvironment(t *testing.T) {
	request := baseRequest()
	request.AllowedWorkflow = map[string]bool{"IN_PROGRESS": true}
	if decision := Evaluate(request); decision.Allowed || decision.Code != types.ErrInvalidTransition {
		t.Fatalf("expected workflow denial: %#v", decision)
	}
	request = baseRequest()
	request.FeatureKey = featureflags.FlagAIAdvisory
	request.FeatureEnabled = false
	if decision := Evaluate(request); decision.Allowed || decision.Code != types.ErrUnauthorized {
		t.Fatalf("expected feature denial: %#v", decision)
	}
	request = baseRequest()
	request.AllowedEnvironments = map[string]bool{types.EnvironmentTesting: true}
	if decision := Evaluate(request); decision.Allowed || decision.Code != types.ErrUnauthorized {
		t.Fatalf("expected environment denial: %#v", decision)
	}
}

func TestAuthorizationEnforcesTrustedOfflineDeviceAndAuthority(t *testing.T) {
	request := baseRequest()
	request.Offline = true
	request.Device = DeviceContext{ID: "device-1", TrustState: types.DevicePending, AuthorityValid: true}
	if decision := Evaluate(request); decision.Allowed || decision.Code != types.ErrUnauthorized {
		t.Fatalf("expected device denial: %#v", decision)
	}
	request.Device.TrustState = types.DeviceTrusted
	request.Device.AuthorityValid = false
	if decision := Evaluate(request); decision.Allowed || decision.Code != types.ErrExpired {
		t.Fatalf("expected authority denial: %#v", decision)
	}
	request.Device.AuthorityValid = true
	if decision := Evaluate(request); !decision.Allowed {
		t.Fatalf("expected trusted offline request to pass: %#v", decision)
	}
}
