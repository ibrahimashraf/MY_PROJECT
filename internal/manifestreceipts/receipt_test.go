package manifestreceipts

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriterEmitsRedactedValidProofReceipt(t *testing.T) {
	directory := t.TempDir()
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	writer, err := NewWriter("0123456789abcdef0123456789abcdef", directory, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	if err := writer.Emit(Observation{Case: CaseValidProof, ObservedOutcome: "proof_valid", HTTPStatus: 200, Before: 7, After: 8, ManifestID: "manifest-internal-id", CorrelationID: "opaque-run-correlation", GeneratedAt: now}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	path := filepath.Join(directory, "receipt-valid_proof-0123456789abcdef0123456789abcdef.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read receipt: %v", err)
	}
	if strings.Contains(string(content), "manifest-internal-id") {
		t.Fatal("receipt leaked raw manifest identifier")
	}
	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	if decoded["status"] != "passed" || decoded["manifest_id_digest"] == "" {
		t.Fatalf("unexpected receipt projection: %#v", decoded)
	}
}

func TestWriterFailsClosedOnDeltaMismatchAndDuplicate(t *testing.T) {
	directory := t.TempDir()
	writer, err := NewWriter("fedcba9876543210fedcba9876543210", directory, time.Now)
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	observation := Observation{Case: CaseReplay, ObservedOutcome: "replay_rejected", HTTPStatus: 409, Before: 9, After: 10}
	if err := writer.Emit(observation); err != nil {
		t.Fatalf("emit failed receipt: %v", err)
	}
	if err := writer.Emit(observation); !errors.Is(err, ErrDuplicateReceipt) {
		t.Fatalf("duplicate error = %v, want %v", err, ErrDuplicateReceipt)
	}
	files, err := os.ReadDir(directory)
	if err != nil || len(files) != 1 {
		t.Fatalf("receipt files = %d, err = %v", len(files), err)
	}
	content, err := os.ReadFile(filepath.Join(directory, files[0].Name()))
	if err != nil {
		t.Fatalf("read failed receipt: %v", err)
	}
	if !strings.Contains(string(content), `"status":"failed"`) {
		t.Fatalf("delta mismatch did not produce failed receipt: %s", content)
	}
}

func TestWriterRejectsUnsafeRunAndDirectory(t *testing.T) {
	if _, err := NewWriter("not-a-run-id", t.TempDir(), time.Now); err == nil {
		t.Fatal("invalid run ID was accepted")
	}
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := NewWriter("0123456789abcdef0123456789abcdef", missing, time.Now); err == nil {
		t.Fatal("missing directory was accepted")
	}
}

func TestWriterRejectsSchemaInvalidOutcomeAndCounts(t *testing.T) {
	writer, err := NewWriter("abcdef0123456789abcdef0123456789", t.TempDir(), time.Now)
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	if err := writer.Emit(Observation{Case: CaseReplay, ObservedOutcome: "proof_invalid", HTTPStatus: 401, Before: 0, After: 0}); err == nil {
		t.Fatal("unknown observed outcome was accepted")
	}
	if err := writer.Emit(Observation{Case: CaseReplay, ObservedOutcome: "replay_rejected", HTTPStatus: 409, Before: -1, After: 0}); err == nil {
		t.Fatal("negative state count was accepted")
	}
}
