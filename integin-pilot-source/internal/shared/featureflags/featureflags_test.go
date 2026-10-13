package featureflags

import "testing"

func TestFeatureFlagStatesAndScopes(t *testing.T) {
	if Enabled == Disabled || Inherited == Expired {
		t.Fatal("feature flag states must be distinct")
	}
	if ScopeOrganization == ScopeDevice || ScopeUser == ScopeProject {
		t.Fatal("feature flag scopes must be distinct")
	}
	override := Override{Key: FlagAIAdvisory, Scope: ScopeOrganization, ScopeID: "org-1", State: Disabled, Reason: "tenant policy"}
	if override.Key != FlagAIAdvisory || override.ScopeID != "org-1" {
		t.Fatalf("unexpected override: %#v", override)
	}
}
