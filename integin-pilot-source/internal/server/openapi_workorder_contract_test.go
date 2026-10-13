package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAPIWorkOrderPartialSubmissionContractIsComplete(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "openapi", "integin-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	var operation struct {
		OperationID string                     `json:"operationId"`
		RequestBody json.RawMessage            `json:"requestBody"`
		Security    []map[string][]string      `json:"security"`
		Responses   map[string]json.RawMessage `json:"responses"`
	}
	if err := json.Unmarshal(document.Paths["/work-orders/partial-submissions"]["post"], &operation); err != nil {
		t.Fatal(err)
	}
	if operation.OperationID == "" || len(operation.RequestBody) == 0 || string(operation.RequestBody) == "null" {
		t.Fatal("work-order operation lacks a request body contract")
	}
	if len(operation.Security) == 0 {
		t.Fatal("work-order operation lacks bearer security")
	}
	for _, status := range []string{"200", "400", "401", "403", "405", "503"} {
		if _, ok := operation.Responses[status]; !ok {
			t.Fatalf("work-order operation lacks response %s", status)
		}
	}
}
