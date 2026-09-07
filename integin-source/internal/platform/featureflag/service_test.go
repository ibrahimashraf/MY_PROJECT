package featureflag

import (
	"testing"
	"time"

	"integin/internal/shared/featureflags"
)

func TestFeatureFlagOverridePrecedenceAndExpiry(t *testing.T) {
	service := New()
	service.SetDefault(featureflags.FlagAIAdvisory, featureflags.Disabled)
	if err := service.SetOverride(Override{Key: featureflags.FlagAIAdvisory, Scope: featureflags.ScopeOrganization, ScopeID: "org-1", State: featureflags.Enabled}); err != nil {
		t.Fatal(err)
	}
	if err := service.SetOverride(Override{Key: featureflags.FlagAIAdvisory, Scope: featureflags.ScopeUser, ScopeID: "user-1", State: featureflags.Disabled}); err != nil {
		t.Fatal(err)
	}
	request := Request{OrganizationID: "org-1", UserID: "user-1"}
	now := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	if got := service.Evaluate(featureflags.FlagAIAdvisory, request, now); got != featureflags.Disabled {
		t.Fatalf("user override should win: %s", got)
	}
	if err := service.SetOverride(Override{Key: featureflags.FlagAIAdvisory, Scope: featureflags.ScopeDevice, ScopeID: "device-1", State: featureflags.Enabled, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	request.DeviceID = "device-1"
	if got := service.Evaluate(featureflags.FlagAIAdvisory, request, now.Add(30*time.Minute)); got != featureflags.Enabled {
		t.Fatalf("device override should win: %s", got)
	}
	if got := service.Evaluate(featureflags.FlagAIAdvisory, request, now.Add(2*time.Hour)); got != featureflags.Disabled {
		t.Fatalf("expired device override should be ignored: %s", got)
	}
}
