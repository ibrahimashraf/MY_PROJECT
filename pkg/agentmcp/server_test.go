package agentmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"integin/pkg/domain"
)

func TestServer_PathSandboxing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mcp-sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	readRoot := filepath.Join(tempDir, "read")
	writeRoot := filepath.Join(tempDir, "write")
	_ = os.MkdirAll(readRoot, 0755)
	_ = os.MkdirAll(writeRoot, 0755)

	dispatcher := domain.NewDispatcher()
	srv, err := NewServer(ServerConfig{
		ReadRoot:  readRoot,
		WriteRoot: writeRoot,
	}, dispatcher)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. Inside write root -> allowed
	validWrite := filepath.Join(writeRoot, "export.json")
	if _, err := srv.ValidatePath(validWrite, true); err != nil {
		t.Fatalf("expected valid write path, got: %v", err)
	}

	// 2. Traversal attempt out of write root -> rejected
	escapeWrite := filepath.Join(writeRoot, "../read/sneaky.txt")
	if _, err := srv.ValidatePath(escapeWrite, true); err == nil {
		t.Fatalf("expected error on path escape, got nil")
	}

	// 3. Absolute path pointing to system root -> rejected
	systemPath := "C:\\Windows\\System32\\calc.exe"
	if _, err := srv.ValidatePath(systemPath, false); err == nil {
		t.Fatalf("expected error on system path escape, got nil")
	}
}

func TestServer_JSONRPC_Execution(t *testing.T) {
	dispatcher := domain.NewDispatcher()
	dispatcher.Register("asset.lock", func(ctx context.Context, cmd domain.Command) (json.RawMessage, error) {
		return json.Marshal(map[string]string{"locked": "true", "tenant": cmd.TenantID})
	})

	srv, err := NewServer(ServerConfig{
		ReadRoot:  os.TempDir(),
		WriteRoot: os.TempDir(),
	}, dispatcher)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()

	// 1. Tool listing request
	listReq := []byte(`{"jsonrpc":"2.0","id":1,"method":"mcp.list_tools"}`)
	res := srv.HandleRequest(ctx, listReq)
	if res.Error != nil {
		t.Fatalf("expected success, got error: %v", res.Error)
	}

	// 2. Domain command execution over JSON-RPC
	cmdReq := []byte(`{
		"jsonrpc": "2.0",
		"id": 2,
		"method": "domain.execute",
		"params": {
			"id": "asset.lock",
			"tenant_id": "tenant-sa-01",
			"organization_id": "aramco-pipe",
			"actor_id": "agent-cursor-1",
			"payload": {}
		}
	}`)
	res = srv.HandleRequest(ctx, cmdReq)
	if res.Error != nil {
		t.Fatalf("unexpected RPC error: %v", res.Error)
	}

	cmdResult, ok := res.Result.(domain.CommandResult)
	if !ok {
		t.Fatalf("expected domain.CommandResult in result, got %T", res.Result)
	}
	if cmdResult.Status != "SUCCESS" {
		t.Fatalf("expected SUCCESS status, got %s", cmdResult.Status)
	}

	// 3. Stream verification over io.Reader / io.Writer pipe
	inBuf := bytes.NewBuffer(listReq)
	inBuf.WriteByte('\n')
	var outBuf bytes.Buffer

	cancelCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	_ = srv.ServeStream(cancelCtx, inBuf, &outBuf)
	if outBuf.Len() == 0 {
		t.Fatalf("expected stream output, got 0 bytes")
	}
}
