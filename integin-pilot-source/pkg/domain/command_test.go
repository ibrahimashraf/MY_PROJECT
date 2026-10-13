package domain

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestDispatcher_Execute(t *testing.T) {
	dispatcher := NewDispatcher()

	testHandler := func(ctx context.Context, cmd Command) (json.RawMessage, error) {
		var req struct {
			Echo string `json:"echo"`
		}
		if err := json.Unmarshal(cmd.Payload, &req); err != nil {
			return nil, err
		}
		return json.Marshal(map[string]string{"response": req.Echo})
	}

	dispatcher.Register("test.echo", testHandler)

	ctx := context.Background()

	// 1. Missing tenant ID must fail
	cmdMissingTenant := Command{
		ID:        "test.echo",
		Payload:   json.RawMessage(`{"echo":"hello"}`),
		Timestamp: time.Now(),
	}
	_, err := dispatcher.Execute(ctx, cmdMissingTenant)
	if err == nil || err != ErrTenantRequired {
		t.Fatalf("expected ErrTenantRequired, got %v", err)
	}

	// 2. Unknown command must fail
	cmdUnknown := Command{
		ID:        "test.unknown",
		TenantID:  "tenant-alpha",
		Payload:   json.RawMessage(`{}`),
		Timestamp: time.Now(),
	}
	_, err = dispatcher.Execute(ctx, cmdUnknown)
	if err == nil {
		t.Fatalf("expected error for unknown command, got nil")
	}

	// 3. Valid command execution with revision increment
	cmdValid := Command{
		ID:             "test.echo",
		TenantID:       "tenant-alpha",
		OrganizationID: "org-1",
		ActorID:        "user-42",
		Payload:        json.RawMessage(`{"echo":"integin-monolith"}`),
		Timestamp:      time.Now(),
	}
	res, err := dispatcher.Execute(ctx, cmdValid)
	if err != nil {
		t.Fatalf("unexpected error executing command: %v", err)
	}
	if res.Status != "SUCCESS" {
		t.Fatalf("expected status SUCCESS, got %s", res.Status)
	}
	if res.Revision != 1 {
		t.Fatalf("expected revision 1, got %d", res.Revision)
	}

	var outMap map[string]string
	if err := json.Unmarshal(res.Output, &outMap); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	if outMap["response"] != "integin-monolith" {
		t.Fatalf("expected response 'integin-monolith', got '%s'", outMap["response"])
	}
}
