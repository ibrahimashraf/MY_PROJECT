package httputil

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProblemDetails_Serialization(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)

	WriteProblem(w, r, http.StatusNotFound, "Not Found", "Item does not exist")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %s, want application/problem+json", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"title":"Not Found"`) || !strings.Contains(body, `"instance":"/api/v1/test"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestProblemDetails_OmitZero(t *testing.T) {
	w := httptest.NewRecorder()
	WriteProblem(w, nil, http.StatusBadRequest, "Bad Request", "")

	body := w.Body.String()
	// omitzero must omit empty detail and empty instance
	if strings.Contains(body, `"detail"`) || strings.Contains(body, `"instance"`) {
		t.Fatalf("expected detail and instance to be omitted, got: %s", body)
	}
}

func TestLoggerContext(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	baseLogger := slog.New(handler)

	ctx := WithLogger(context.Background(), baseLogger)
	retrieved := LoggerFromContext(ctx)
	if retrieved != baseLogger {
		t.Fatal("LoggerFromContext failed to retrieve stored logger")
	}

	scoped := ScopedLogger(ctx, "t-123", "org-456", "billing")
	scoped.Info("event occurred", "count", 1)

	logged := buf.String()
	if !strings.Contains(logged, `"tenant_id":"t-123"`) ||
		!strings.Contains(logged, `"org_id":"org-456"`) ||
		!strings.Contains(logged, `"subsystem":"billing"`) {
		t.Fatalf("scoped logger missing tags: %s", logged)
	}
}
