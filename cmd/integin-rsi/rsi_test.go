package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const sandboxEngine = "package sandbox\n\nfunc Name() string {\n\treturn \"original\"\n}\n"

// mutatePatch rewrites the return value of Name to "mutated".
const mutatePatch = "--- a/engine.go\n+++ b/engine.go\n@@ -4,1 +4,1 @@\n-\treturn \"original\"\n+\treturn \"mutated\"\n"

// breakPatch introduces an uncompilable statement in Name.
const breakPatch = "--- a/engine.go\n+++ b/engine.go\n@@ -4,1 +4,1 @@\n-\treturn \"original\"\n+\treturn broken syntax\n"

const failingTest = "package sandbox\n\nimport \"testing\"\n\nfunc TestBoom(t *testing.T) {\n\tt.Fatal(\"boom\")\n}\n"

// newSandbox creates a throwaway Go module containing engine.go. The go
// directive is mirrored from the repository so the sandbox always compiles
// with the installed toolchain, without any network access.
func newSandbox(t *testing.T) string {
	t.Helper()
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOFLAGS", "-mod=mod")

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(fmt.Sprintf("module integinrsi-sandbox\n\n%s\n", repoGoDirective(t))), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "engine.go"), []byte(sandboxEngine), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func repoGoDirective(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("read repo go.mod: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(line)
		}
	}
	t.Fatal("repo go.mod has no go directive")
	return ""
}

func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func TestApplyAndVerifySuccessRetainsChange(t *testing.T) {
	dir := newSandbox(t)

	res, err := NewRunner(dir).ApplyAndVerify(context.Background(), MutationRequest{
		TargetFiles: []string{"engine.go"},
		Patch:       mutatePatch,
		TestPackage: ".",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected green verification, got failure: %s", res.ErrorReason)
	}
	if res.Attempt != 1 || res.ExecutionDurationMs < 0 {
		t.Fatalf("bad result metadata: %+v", res)
	}
	if !strings.Contains(res.TestOutput, "[no test files]") {
		t.Fatalf("unexpected test output: %q", res.TestOutput)
	}

	content, err := os.ReadFile(filepath.Join(dir, "engine.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `return "mutated"`) {
		t.Fatalf("patched change was not retained:\n%s", content)
	}

	want := []string{"engine.go", "go.mod"}
	if got := dirEntries(t, dir); !equalStrings(got, want) {
		t.Fatalf("unexpected sandbox contents after success: %v", got)
	}
}

func TestRollbackOnCompilationFailure(t *testing.T) {
	dir := newSandbox(t)

	res, err := NewRunner(dir).ApplyAndVerify(context.Background(), MutationRequest{
		TargetFiles: []string{"engine.go"},
		Patch:       breakPatch,
		TestPackage: ".",
	})
	if err != nil {
		t.Fatalf("verification failure is not a hard error, got %v", err)
	}
	if res.Success {
		t.Fatalf("expected failure, got success")
	}
	if !strings.Contains(res.ErrorReason, "vet") {
		t.Fatalf("expected vet failure in reason, got %q", res.ErrorReason)
	}

	content, err := os.ReadFile(filepath.Join(dir, "engine.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != sandboxEngine {
		t.Fatalf("file was not rolled back:\n%s", content)
	}
}

func TestRollbackOnTestFailure(t *testing.T) {
	dir := newSandbox(t)
	if err := os.WriteFile(filepath.Join(dir, "engine_test.go"), []byte(failingTest), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := NewRunner(dir).ApplyAndVerify(context.Background(), MutationRequest{
		TargetFiles: []string{"engine.go"},
		Patch:       mutatePatch,
		TestPackage: ".",
	})
	if err != nil {
		t.Fatalf("verification failure is not a hard error, got %v", err)
	}
	if res.Success {
		t.Fatalf("expected failure, got success")
	}
	if !strings.Contains(res.ErrorReason, "test") {
		t.Fatalf("expected test failure in reason, got %q", res.ErrorReason)
	}

	content, err := os.ReadFile(filepath.Join(dir, "engine.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != sandboxEngine {
		t.Fatalf("file was not rolled back:\n%s", content)
	}
}

func TestBackupIntegrityAndCleanup(t *testing.T) {
	dir := newSandbox(t)
	want := []string{"engine.go", "go.mod"}
	runner := NewRunner(dir)

	res, err := runner.ApplyAndVerify(context.Background(), MutationRequest{
		TargetFiles: []string{"engine.go"},
		Patch:       breakPatch,
		TestPackage: ".",
	})
	if err != nil || res.Success {
		t.Fatalf("expected failed iteration (err=%v, success=%v)", err, res.Success)
	}
	if got := dirEntries(t, dir); !equalStrings(got, want) {
		t.Fatalf("sandbox polluted after rollback: %v", got)
	}

	res, err = runner.ApplyAndVerify(context.Background(), MutationRequest{
		TargetFiles: []string{"engine.go"},
		Patch:       mutatePatch,
		TestPackage: ".",
	})
	if err != nil || !res.Success {
		t.Fatalf("expected green iteration (err=%v, success=%v)", err, res.Success)
	}
	if got := dirEntries(t, dir); !equalStrings(got, want) {
		t.Fatalf("sandbox polluted after success: %v", got)
	}
}

func TestBuildRequestFromStdinJSON(t *testing.T) {
	input := `{"target_files":["pkg/a.go","pkg/b.go"],"patch":"--- a\n+++ b\n@@ -0,0 +1,1 @@\n+x\n","test_package":"./pkg","max_attempts":3}`
	req, err := buildRequest(nil, "", bytes.NewReader([]byte(input)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.TargetFiles) != 2 || req.TargetFiles[0] != "pkg/a.go" || req.TargetFiles[1] != "pkg/b.go" {
		t.Fatalf("target_files not decoded: %v", req.TargetFiles)
	}
	if req.TestPackage != "./pkg" || req.MaxAttempts != 3 {
		t.Fatalf("request fields not decoded: %+v", req)
	}
}

func TestBuildRequestFromFlags(t *testing.T) {
	req, err := buildRequest(targetList{"a.go"}, "./pkg", strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.TargetFiles) != 1 || req.TargetFiles[0] != "a.go" || req.TestPackage != "./pkg" {
		t.Fatalf("flag request wrong: %+v", req)
	}
	if _, err := buildRequest(nil, "./pkg", strings.NewReader("")); err == nil {
		t.Fatal("expected error when --package given without --target")
	}
	if _, err := buildRequest(targetList{"a.go"}, "", strings.NewReader("")); err == nil {
		t.Fatal("expected error when --target given without --package")
	}
	if _, err := buildRequest(nil, "", strings.NewReader("not json")); err == nil {
		t.Fatal("expected JSON decode error")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
