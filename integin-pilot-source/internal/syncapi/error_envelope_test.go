package syncapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The sync rejection envelope is a public wire contract: clients branch on
// "outcome", not just "reason". An earlier refactor routed writeError through
// the bare {"reason":...} helper shared with provisioning and silently dropped
// the outcome field. This pins the shape.
func TestWriteErrorKeepsOutcomeEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusConflict, "sequence_conflict")

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("body is not valid JSON: %v (%s)", err, rec.Body.String())
	}
	if decoded["outcome"] == nil {
		t.Fatalf("sync rejection envelope lost its outcome field: %s", rec.Body.String())
	}
	if decoded["reason"] != "sequence_conflict" {
		t.Fatalf("reason = %v, want sequence_conflict", decoded["reason"])
	}
	if !strings.Contains(string(decoded["outcome"].(string)), "") ||
		decoded["outcome"] == "applied" || decoded["outcome"] == "ok" {
		t.Fatalf("outcome = %v, want a rejection outcome", decoded["outcome"])
	}
}
