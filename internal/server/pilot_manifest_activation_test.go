package server

import (
	"testing"

	"integin/internal/packagemanifestapi"
	"integin/internal/shared/featureflags"
)

func validPilotManifestActivationConfig() PilotManifestActivationConfig {
	return PilotManifestActivationConfig{
		RuntimeName:               IsolatedPilotRuntime,
		ListenAddress:             IsolatedPilotManifestAddress,
		ManifestRetrievalState:    featureflags.Enabled,
		PackageEnforcementEnabled: false,
		Handler:                   &packagemanifestapi.Handler{},
	}
}

func TestPilotManifestActivationConfigFailsClosed(t *testing.T) {
	if err := validPilotManifestActivationConfig().Validate(); err != nil {
		t.Fatalf("valid isolated pilot gate rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*PilotManifestActivationConfig)
	}{
		{"acceptance runtime", func(c *PilotManifestActivationConfig) { c.RuntimeName = "acceptance" }},
		{"wrong address", func(c *PilotManifestActivationConfig) { c.ListenAddress = "127.0.0.1:8080" }},
		{"disabled retrieval", func(c *PilotManifestActivationConfig) { c.ManifestRetrievalState = featureflags.Disabled }},
		{"enabled enforcement", func(c *PilotManifestActivationConfig) { c.PackageEnforcementEnabled = true }},
		{"missing handler", func(c *PilotManifestActivationConfig) { c.Handler = nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validPilotManifestActivationConfig()
			test.mutate(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("expected activation gate rejection")
			}
		})
	}
}

func TestPilotManifestObservationValidation(t *testing.T) {
	valid := PilotManifestObservation{
		Outcome:    PilotManifestIssued,
		ReasonCode: "issued",
		ManifestID: "digest-prefix-only",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid observation rejected: %v", err)
	}
	if err := (PilotManifestObservation{Outcome: PilotManifestIssued}).Validate(); err == nil {
		t.Fatal("expected missing reason code rejection")
	}
}
