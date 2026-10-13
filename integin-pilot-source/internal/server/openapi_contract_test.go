package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestOpenAPIV1DocumentsEveryLiteralMuxRoute(t *testing.T) {
	source, err := os.ReadFile("http.go")
	if err != nil {
		t.Fatalf("read mux source: %v", err)
	}
	contractPath := filepath.Join("..", "..", "openapi", "integin-v1.json")
	raw, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read OpenAPI v1 contract %s: %v", contractPath, err)
	}

	var document struct {
		Paths map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode OpenAPI v1 contract: %v", err)
	}

	routePattern := regexp.MustCompile(`mux\.Handle(?:Func)?\("([^"]+)"`)
	for _, match := range routePattern.FindAllSubmatch(source, -1) {
		route := string(match[1])
		if _, ok := document.Paths[route]; !ok {
			t.Fatalf("literal mux route %s is not documented in OpenAPI v1", route)
		}
	}
}
