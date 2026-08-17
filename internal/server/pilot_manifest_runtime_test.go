package server

import "testing"

func TestPilotManifestHandlerFromEnvironmentIsDisabledByDefault(t *testing.T) {
	for _, key := range []string{
		pilotManifestRetrievalEnvironment,
		pilotRuntimeEnvironment,
		pilotManifestPrivateKeyEnvironment,
		pilotManifestKeyIDEnvironment,
	} {
		t.Setenv(key, "")
	}
	handler, registry, err := PilotManifestHandlerFromEnvironment(nil, nil, nil, IsolatedPilotManifestAddress)
	if err != nil {
		t.Fatalf("disabled pilot manifest retrieval returned error: %v", err)
	}
	if handler != nil || registry != nil {
		t.Fatal("disabled pilot manifest retrieval returned runtime components")
	}
}

func TestPilotManifestHandlerFromEnvironmentRejectsNonPilotEnablement(t *testing.T) {
	t.Setenv(pilotManifestRetrievalEnvironment, "enabled")
	t.Setenv(pilotRuntimeEnvironment, "acceptance")
	_, _, err := PilotManifestHandlerFromEnvironment(nil, nil, nil, "127.0.0.1:8080")
	if err == nil {
		t.Fatal("expected non-pilot manifest retrieval rejection")
	}
}
