package authorization

import (
	"strings"

	"integin/internal/shared/featureflags"
	"integin/internal/shared/types"
)

type Capability string

const (
	CapabilityInspectAssign     Capability = "inspection.assign"
	CapabilityInspectPerform    Capability = "inspection.perform"
	CapabilityInspectReview     Capability = "inspection.review"
	CapabilityInspectApprove    Capability = "inspection.approve"
	CapabilityCertificateIssue  Capability = "certificate.issue"
	CapabilityOrganizationAdmin Capability = "organization.admin"
	CapabilityPlatformAdmin     Capability = "platform.admin"
)

type Subject struct {
	UserID         string
	TenantID       string
	OrganizationID string
	Role           string
	Capabilities   map[Capability]bool
	Scopes         map[string]bool
}

type Resource struct {
	ID             string
	Type           string
	TenantID       string
	OrganizationID string
	ScopeID        string
	WorkflowState  string
}

type DeviceContext struct {
	ID             string
	TrustState     types.DeviceTrustState
	AuthorityValid bool
}

type Request struct {
	Subject             Subject
	Resource            Resource
	Action              Capability
	Environment         string
	Device              DeviceContext
	Offline             bool
	FeatureKey          featureflags.Key
	FeatureEnabled      bool
	AllowedEnvironments map[string]bool
	AllowedWorkflow     map[string]bool
}

type Decision struct {
	Allowed        bool
	Code           types.ErrorCode
	Reason         string
	TenantID       string
	OrganizationID string
}

func Allow(subject Subject) Decision {
	return Decision{Allowed: true, TenantID: subject.TenantID, OrganizationID: subject.OrganizationID}
}
func Deny(request Request, code types.ErrorCode, reason string) Decision {
	return Decision{Allowed: false, Code: code, Reason: reason, TenantID: request.Subject.TenantID, OrganizationID: request.Subject.OrganizationID}
}

// Evaluate is deterministic and side-effect free. It performs every required
// authorization dimension before allowing a request to reach a state-changing
// domain method.
func Evaluate(request Request) Decision {
	if strings.TrimSpace(request.Subject.UserID) == "" {
		return Deny(request, types.ErrUnauthorized, "subject user_id is required")
	}
	if strings.TrimSpace(request.Subject.TenantID) == "" || strings.TrimSpace(request.Subject.OrganizationID) == "" {
		return Deny(request, types.ErrUnauthorized, "subject tenant context is required")
	}
	if strings.TrimSpace(request.Resource.ID) == "" || strings.TrimSpace(request.Resource.Type) == "" {
		return Deny(request, types.ErrValidation, "resource identity is required")
	}
	if request.Resource.TenantID != request.Subject.TenantID {
		return Deny(request, types.ErrTenantMismatch, "resource belongs to another tenant")
	}
	if request.Resource.OrganizationID != request.Subject.OrganizationID {
		return Deny(request, types.ErrUnauthorized, "resource belongs to another organization")
	}
	if request.Action == "" || !request.Subject.Capabilities[request.Action] {
		return Deny(request, types.ErrUnauthorized, "subject lacks required capability")
	}
	if request.Resource.ScopeID != "" && !request.Subject.Scopes[request.Resource.ScopeID] && !request.Subject.Scopes["*"] {
		return Deny(request, types.ErrUnauthorized, "subject lacks resource scope")
	}
	if request.Environment != types.EnvironmentTesting && request.Environment != types.EnvironmentLive {
		return Deny(request, types.ErrValidation, "invalid environment")
	}
	if len(request.AllowedEnvironments) > 0 && !request.AllowedEnvironments[request.Environment] {
		return Deny(request, types.ErrUnauthorized, "environment policy denies action")
	}
	if len(request.AllowedWorkflow) > 0 && !request.AllowedWorkflow[request.Resource.WorkflowState] {
		return Deny(request, types.ErrInvalidTransition, "workflow policy denies action at current state")
	}
	if request.FeatureKey != "" && !request.FeatureEnabled {
		return Deny(request, types.ErrUnauthorized, "feature flag is disabled")
	}
	if request.Offline {
		if request.Device.TrustState != types.DeviceTrusted {
			return Deny(request, types.ErrUnauthorized, "offline action requires trusted device")
		}
		if !request.Device.AuthorityValid {
			return Deny(request, types.ErrExpired, "offline authority is invalid or expired")
		}
	}
	return Allow(request.Subject)
}
