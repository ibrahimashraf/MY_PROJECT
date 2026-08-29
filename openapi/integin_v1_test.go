package openapi

import (
	"encoding/json"
	"os"
	"testing"
)

func TestINTEGINV1ContractHasRequiredClientOperations(t *testing.T) {
	raw, err := os.ReadFile("integin-v1.json")
	if err != nil {
		t.Fatalf("read OpenAPI v1 contract: %v", err)
	}

	var document struct {
		OpenAPI string `json:"openapi"`
		Info    struct {
			Version string `json:"version"`
		} `json:"info"`
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("OpenAPI v1 contract must be valid JSON: %v", err)
	}
	if document.OpenAPI != "3.1.0" {
		t.Fatalf("OpenAPI version = %q, want 3.1.0", document.OpenAPI)
	}
	if document.Info.Version != "1.0.0" {
		t.Fatalf("contract info.version = %q, want 1.0.0", document.Info.Version)
	}

	for path, method := range map[string]string{
		"/healthz":          "get",
		"/readyz":           "get",
		"/sync":             "post",
		"/evidence":         "post",
		"/evidence/metadata-registrations": "post",
		"/local/provision":  "post",
		"/identity/session": "get",
	} {
		operations, ok := document.Paths[path]
		if !ok {
			t.Fatalf("OpenAPI v1 contract omits route %s", path)
		}
		if _, ok := operations[method]; !ok {
			t.Fatalf("OpenAPI v1 contract omits %s %s", method, path)
		}
	}
}
