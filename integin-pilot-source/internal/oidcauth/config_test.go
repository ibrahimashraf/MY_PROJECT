// integin OIDC foundation tests: assert opt-in configuration and strict pilot-only HTTP handling.
package oidcauth

import "testing"

func TestLoadConfigDefaultsToDisabled(t *testing.T) {
	config, err := LoadConfig(func(string) string { return "" })
	if err != nil || config.Enabled {
		t.Fatalf("expected disabled default, config=%+v err=%v", config, err)
	}
}

func TestLoadConfigRequiresIssuerAndAudienceWhenEnabled(t *testing.T) {
	_, err := LoadConfig(func(name string) string {
		if name == "INTEGIN_OIDC_ENABLED" {
			return "true"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected enabled configuration to require issuer and audience")
	}
}

func TestLoadConfigAllowsHTTPOnlyForExplicitLoopbackPilot(t *testing.T) {
	values := map[string]string{
		"INTEGIN_OIDC_ENABLED":                 "true",
		"INTEGIN_OIDC_ISSUER":                  "http://127.0.0.1:18180/realms/integin-pilot",
		"INTEGIN_OIDC_AUDIENCE":                "integin-api-pilot",
		"INTEGIN_OIDC_ALLOW_INSECURE_LOOPBACK": "true",
		"INTEGIN_OIDC_REQUIRED_AMR":            "pwd, otp, pwd",
	}
	config, err := LoadConfig(func(name string) string { return values[name] })
	if err != nil {
		t.Fatalf("load loopback configuration: %v", err)
	}
	if !config.Enabled || len(config.RequiredAMR) != 2 || config.RequiredAMR[0] != "pwd" || config.RequiredAMR[1] != "otp" {
		t.Fatalf("unexpected config: %+v", config)
	}
}
