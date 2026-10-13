package queue

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewWebhookDLQNotifierValidation(t *testing.T) {
	if _, err := NewWebhookDLQNotifier("", nil); !errors.Is(err, ErrEmptyEndpoint) {
		t.Fatalf("empty endpoint must fail with ErrEmptyEndpoint, got %v", err)
	}
	if _, err := NewWebhookDLQNotifier("not a url", nil); !errors.Is(err, ErrInvalidEndpoint) {
		t.Fatalf("malformed endpoint must fail with ErrInvalidEndpoint, got %v", err)
	}
	if _, err := NewWebhookDLQNotifier("ftp://hooks.example.com/alert", nil); !errors.Is(err, ErrInvalidEndpoint) {
		t.Fatalf("non-http scheme must fail with ErrInvalidEndpoint, got %v", err)
	}
	if _, err := NewWebhookDLQNotifier("https://hooks.example.com/alert", nil); err != nil {
		t.Fatalf("valid endpoint must construct cleanly, got %v", err)
	}
}

type receivedAlert struct {
	method      string
	contentType string
	body        map[string]interface{}
}

func TestWebhookDLQNotifierPostsQuarantineJSON(t *testing.T) {
	ch := make(chan receivedAlert, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode webhook body: %v", err)
		}
		ch <- receivedAlert{method: r.Method, contentType: r.Header.Get("Content-Type"), body: body}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier, err := NewWebhookDLQNotifier(srv.URL, nil)
	if err != nil {
		t.Fatalf("failed to construct notifier: %v", err)
	}
	record := PoisonRecord{
		JobID:          41,
		Kind:           "certificate_render",
		TenantID:       "t1",
		OrganizationID: "o1",
		Args:           []byte(`{"certificate_id":"c1"}`),
		ErrorText:      "boom",
		Attempt:        2,
		State:          "running",
	}
	if err := notifier.Notify(context.Background(), record); err != nil {
		t.Fatalf("Notify must not fail on 2xx, got %v", err)
	}

	got := <-ch
	if got.method != http.MethodPost {
		t.Fatalf("expected POST delivery, got %s", got.method)
	}
	if !strings.HasPrefix(got.contentType, "application/json") {
		t.Fatalf("expected application/json content type, got %q", got.contentType)
	}
	if got.body["JobID"].(float64) != 41 || got.body["Kind"] != "certificate_render" {
		t.Fatalf("job identity missing from payload: %+v", got.body)
	}
	if got.body["TenantID"] != "t1" || got.body["OrganizationID"] != "o1" {
		t.Fatalf("tenant scope missing from payload: %+v", got.body)
	}
	if got.body["ErrorText"] != "boom" || got.body["Attempt"].(float64) != 2 {
		t.Fatalf("forensic fields missing from payload: %+v", got.body)
	}
	argsJSON, ok := got.body["Args"].(map[string]interface{})
	if !ok || argsJSON["certificate_id"] != "c1" {
		t.Fatalf("encoded args not embedded as JSON: %+v", got.body["Args"])
	}
}

func TestWebhookDLQNotifierFailsOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("sink unavailable"))
	}))
	defer srv.Close()

	notifier, err := NewWebhookDLQNotifier(srv.URL, nil)
	if err != nil {
		t.Fatalf("failed to construct notifier: %v", err)
	}
	err = notifier.Notify(context.Background(), PoisonRecord{JobID: 41})
	if err == nil {
		t.Fatal("non-2xx response must fail Notify")
	}
	if !errors.Is(err, ErrWebhookRejected) {
		t.Fatalf("expected ErrWebhookRejected, got %v", err)
	}
	if !strings.Contains(err.Error(), "sink unavailable") {
		t.Fatalf("rejection body must be surfaced for diagnostics, got %v", err)
	}
}

func TestWebhookDLQNotifierGuardRailChecks(t *testing.T) {
	var notifier *WebhookDLQNotifier
	if err := notifier.Notify(context.Background(), PoisonRecord{JobID: 41}); !errors.Is(err, ErrEmptyEndpoint) {
		t.Fatalf("nil notifier must fail with ErrEmptyEndpoint, got %v", err)
	}
	notifier = &WebhookDLQNotifier{endpoint: "https://hooks.example.com/alert"}
	if err := notifier.Notify(context.Background(), PoisonRecord{JobID: 41}); !errors.Is(err, ErrWebhookHTTPClient) {
		t.Fatalf("nil client must fail with ErrWebhookHTTPClient, got %v", err)
	}
}
