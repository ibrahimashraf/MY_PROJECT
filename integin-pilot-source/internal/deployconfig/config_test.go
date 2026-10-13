package deployconfig

import "testing"

func TestLoadDefaultsToDisabledWithoutEndpoint(t *testing.T) {
	config, err := Load(map[string]string{})
	if err != nil || config.IntegrationEnabled || config.Timeout <= 0 {
		t.Fatalf("unexpected disabled config: %#v %v", config, err)
	}
}
func TestLoadEnabledRequiresEndpointAndValidTimeout(t *testing.T) {
	if _, err := Load(map[string]string{"INTEGIN_AI_ENABLED": "true"}); err == nil {
		t.Fatal("enabled integration should require endpoint")
	}
	config, err := Load(map[string]string{"INTEGIN_AI_ENABLED": "true", "INTEGIN_AI_ENDPOINT": "http://ai-service:8000", "INTEGIN_AI_TIMEOUT": "10"})
	if err != nil || !config.IntegrationEnabled || config.Timeout.Seconds() != 10 {
		t.Fatalf("unexpected enabled config: %#v %v", config, err)
	}
	if _, err := Load(map[string]string{"INTEGIN_AI_TIMEOUT": "0"}); err == nil {
		t.Fatal("invalid timeout should fail")
	}
}
