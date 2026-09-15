package contextground

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleSource = `package sample

type MyConfig struct {
	ID    string
	Level int
}

type Runnable interface {
	Run() error
}

func NewConfig(id string) *MyConfig {
	return &MyConfig{ID: id}
}
`

func writeTempFile(t *testing.T, root, name, content string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractTypesAndFunctions(t *testing.T) {
	root := t.TempDir()
	writeTempFile(t, root, "sample.go", sampleSource)

	payload, err := New().Ground(t.Context(), GroundingRequest{
		RepoRoot:  root,
		FilePaths: []string{"sample.go"},
		Symbols:   []string{"MyConfig", "NewConfig", "Runnable", "DoesNotExist"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(payload.Files) != 1 || payload.Files[0].Path != "sample.go" {
		t.Fatalf("unexpected files: %+v", payload.Files)
	}

	got := map[string]SymbolGrounding{}
	for _, s := range payload.Symbols {
		got[s.Name] = s
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 symbols, got %d: %+v", len(got), payload.Symbols)
	}

	cfg, ok := got["MyConfig"]
	if !ok || cfg.Kind != "type" || cfg.File != "sample.go" || cfg.Line != 3 {
		t.Fatalf("MyConfig wrong: %+v", cfg)
	}
	if cfg.Signature != "type MyConfig struct {\n\tID    string\n\tLevel int\n}" {
		t.Fatalf("MyConfig signature: %q", cfg.Signature)
	}

	run, ok := got["Runnable"]
	if !ok || run.Kind != "interface" || run.Line != 8 {
		t.Fatalf("Runnable wrong: %+v", run)
	}
	if run.Signature != "type Runnable interface {\n\tRun() error\n}" {
		t.Fatalf("Runnable signature: %q", run.Signature)
	}

	ncfg, ok := got["NewConfig"]
	if !ok || ncfg.Kind != "func" || ncfg.Line != 12 {
		t.Fatalf("NewConfig wrong: %+v", ncfg)
	}
	if ncfg.Signature != "func NewConfig(id string) *MyConfig" {
		t.Fatalf("NewConfig signature: %q", ncfg.Signature)
	}
}

func TestDigestDeterminism(t *testing.T) {
	const contentA = "package a\n\ntype A struct{ N int }\n"
	const contentB = "package b\n\nfunc B() int { return 2 }\n"
	root := t.TempDir()
	writeTempFile(t, root, "a.go", contentA)
	writeTempFile(t, root, "b.go", contentB)

	req := GroundingRequest{RepoRoot: root, FilePaths: []string{"a.go", "b.go"}}
	first, err := New().Ground(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := New().Ground(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("digest not deterministic: %s vs %s", first.Digest, second.Digest)
	}

	// Full-content attestation: digest must equal SHA-256 over the two
	// files concatenated in sorted path order.
	h := sha256.New()
	h.Write([]byte(contentA))
	h.Write([]byte(contentB))
	if want := hex.EncodeToString(h.Sum(nil)); first.Digest != want {
		t.Fatalf("digest %s != expected %s", first.Digest, want)
	}

	// A content change must change the digest.
	writeTempFile(t, root, "b.go", "package b\n\nfunc B() int { return 3 }\n")
	third, err := New().Ground(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if third.Digest == first.Digest {
		t.Fatal("digest unchanged after content edit")
	}
}

func TestPathTraversalRejection(t *testing.T) {
	root := t.TempDir()
	writeTempFile(t, root, "ok.go", "package ok\n")

	for _, tc := range []struct {
		name string
		path string
	}{
		{"direct parent", "../escape.txt"},
		{"nested traversal", "sub/../../escape.txt"},
		{"absolute", filepath.Join(root, "..", "escape.txt")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New().Ground(t.Context(), GroundingRequest{
				RepoRoot:  root,
				FilePaths: []string{tc.path},
			})
			if err == nil {
				t.Fatalf("expected rejection for %q", tc.path)
			}
			if !strings.Contains(err.Error(), "reject") && !strings.Contains(err.Error(), "escape") &&
				!strings.Contains(err.Error(), "absolute") {
				t.Fatalf("unexpected error text: %v", err)
			}
		})
	}
}

func TestSecretRedaction(t *testing.T) {
	key := "AIza" + strings.Repeat("Ab", 17) + "A" // realistic 39-char Google API key shape
	raw := fmt.Sprintf(`package s

// google placeholder key
const gk = %q

// bearer token in a header helper
const hdr = "Authorization: Bearer aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789abcdef"

const pem = %q
`, key, "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA7d+J\n-----END RSA PRIVATE KEY-----")
	root := t.TempDir()
	writeTempFile(t, root, "secrets.txt", raw)

	payload, err := New().Ground(t.Context(), GroundingRequest{
		RepoRoot:      root,
		FilePaths:     []string{"secrets.txt"},
		RedactSecrets: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	content := payload.Files[0].Content
	if !strings.Contains(content, "[REDACTED]") {
		t.Fatal("expected at least one [REDACTED] marker")
	}
	for _, leak := range []string{"AIza" + strings.Repeat("Ab", 17) + "A", "Bearer aBcDeFgH", "BEGIN RSA PRIVATE KEY"} {
		if strings.Contains(content, leak) {
			t.Fatalf("secret leaked after redaction: %q", leak)
		}
	}

	// Without redaction the raw text must survive intact.
	rawPayload, err := New().Ground(t.Context(), GroundingRequest{
		RepoRoot:      root,
		FilePaths:     []string{"secrets.txt"},
		RedactSecrets: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rawPayload.Files[0].Content != raw {
		t.Fatal("content changed even though redaction was disabled")
	}
	if rawPayload.Digest == payload.Digest {
		t.Fatal("redaction did not affect the digest")
	}
}

func TestHTTPHandlerRoundTrip(t *testing.T) {
	root := t.TempDir()
	writeTempFile(t, root, "sample.go", sampleSource)
	handler := HTTPHandler(New())

	body, err := json.Marshal(GroundingRequest{
		RepoRoot:  root,
		FilePaths: []string{"sample.go"},
		Symbols:   []string{"NewConfig"},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/context/ground", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}
	var payload GroundingPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Files) != 1 || payload.Files[0].Path != "sample.go" {
		t.Fatalf("files: %+v", payload.Files)
	}
	if len(payload.Symbols) != 1 || payload.Symbols[0].Name != "NewConfig" {
		t.Fatalf("symbols: %+v", payload.Symbols)
	}
	if len(payload.Digest) != 64 {
		t.Fatalf("digest not sha256 hex: %q", payload.Digest)
	}
	if payload.TokenEstimate <= 0 {
		t.Fatal("token_estimate not populated")
	}
	if payload.Timestamp.IsZero() {
		t.Fatal("timestamp not populated")
	}

	// Method enforcement.
	req = httptest.NewRequest(http.MethodGet, "/v1/context/ground", nil)
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status %d, want 405", rec.Code)
	}

	// Empty grounding request must be rejected.
	req = httptest.NewRequest(http.MethodPost, "/v1/context/ground", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty request status %d, want 400", rec.Code)
	}

	// Traversal must be rejected through the HTTP path too.
	req = httptest.NewRequest(http.MethodPost, "/v1/context/ground",
		strings.NewReader(`{"repo_root":"`+strings.ReplaceAll(root, `\`, `\\`)+`","file_paths":["../x"]}`))
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("traversal status %d, want 400", rec.Code)
	}
}
