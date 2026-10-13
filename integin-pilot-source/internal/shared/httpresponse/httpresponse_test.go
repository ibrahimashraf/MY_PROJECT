package httpresponse

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These tests pin the exact wire bytes of each error shape. A refactor once
// routed the sync rejection envelope through the bare {"reason":...} helper and
// silently dropped its "outcome" field while the entire suite stayed green.
// Response shapes are a public contract: they need their own tests.
func TestErrorEmitsErrorKey(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, http.StatusBadRequest, "missing tenant")
	if got, want := rec.Body.String(), `{"error":"missing tenant"}`; got != want {
		t.Fatalf("body = %s, want %s", got, want)
	}
}

func TestCodeMessageEmitsCodeAndMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	CodeMessage(rec, http.StatusNotFound, "not_found", "passport not found")
	want := `{"code":"not_found","message":"passport not found"}`
	if got := rec.Body.String(); got != want {
		t.Fatalf("body = %s, want %s", got, want)
	}
}

func TestReasonEmitsReasonKey(t *testing.T) {
	rec := httptest.NewRecorder()
	Reason(rec, http.StatusUnprocessableEntity, "payload rejected")
	if got, want := rec.Body.String(), `{"reason":"payload rejected"}`; got != want {
		t.Fatalf("body = %s, want %s", got, want)
	}
}

func TestSecurityErrorSetsHardeningHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	SecurityError(rec, http.StatusForbidden, "forbidden")
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got, want := rec.Body.String(), `{"error":"forbidden"}`; got != want {
		t.Fatalf("body = %s, want %s", got, want)
	}
}

func TestRawWritesPreEncodedBodyVerbatim(t *testing.T) {
	rec := httptest.NewRecorder()
	body := `{"error":"idempotency_unavailable"}`
	Raw(rec, http.StatusServiceUnavailable, []byte(body))
	if got := rec.Body.String(); got != body {
		t.Fatalf("body = %s, want %s", got, body)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
}

// JSON is the generic writer every typed handler routes through; confirm it
// emits deterministic key order so responses are byte-stable across runs.
func TestJSONIsDeterministic(t *testing.T) {
	first := httptest.NewRecorder()
	second := httptest.NewRecorder()
	JSON(first, http.StatusOK, map[string]string{"b": "2", "a": "1", "c": "3"})
	JSON(second, http.StatusOK, map[string]string{"c": "3", "a": "1", "b": "2"})
	if first.Body.String() != second.Body.String() {
		t.Fatalf("JSON output is not deterministic: %s vs %s", first.Body.String(), second.Body.String())
	}
	if !strings.Contains(first.Body.String(), `"a":"1"`) {
		t.Fatalf("unexpected body: %s", first.Body.String())
	}
}
