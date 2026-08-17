package main

import "testing"

func TestLocalProvisioningFlagDefaultsDisabledAndRejectsInvalidValue(t *testing.T) {
	t.Setenv("INTEGIN_LOCAL_PROVISIONING_ENABLED", "")
	if envBool("INTEGIN_LOCAL_PROVISIONING_ENABLED") {
		t.Fatal("absent local provisioning flag must default disabled")
	}
	t.Setenv("INTEGIN_LOCAL_PROVISIONING_ENABLED", "not-a-boolean")
	if envBool("INTEGIN_LOCAL_PROVISIONING_ENABLED") {
		t.Fatal("invalid local provisioning flag must default disabled")
	}
	t.Setenv("INTEGIN_LOCAL_PROVISIONING_ENABLED", "true")
	if !envBool("INTEGIN_LOCAL_PROVISIONING_ENABLED") {
		t.Fatal("explicit true local provisioning flag was not recognized")
	}
}

func TestLocalProvisioningRequiresLoopbackBinding(t *testing.T) {
	for _, address := range []string{"127.0.0.1:8080", "localhost:8080", "[::1]:8080"} {
		if !isLoopbackAddress(address) {
			t.Fatalf("expected loopback address %q to be allowed", address)
		}
	}
	for _, address := range []string{":8080", "0.0.0.0:8080", "192.0.2.10:8080", "example.test:8080"} {
		if isLoopbackAddress(address) {
			t.Fatalf("expected non-loopback address %q to be rejected", address)
		}
	}
}
