package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DLQAlertNotifier is notified whenever a job is quarantined to the poison
// differential processing lane (DLQ).
type DLQAlertNotifier interface {
	Notify(ctx context.Context, record PoisonRecord) error
}

// PoisonRecord is the quarantine snapshot delivered to alert sinks. It aliases
// PoisonQuarantineEntry so a single forensic record serves both the database
// column and the outbound webhook payload.
type PoisonRecord = PoisonQuarantineEntry

var (
	ErrEmptyEndpoint     = errors.New("queue: webhook endpoint is required")
	ErrInvalidEndpoint   = errors.New("queue: webhook endpoint is not a valid http(s) URL")
	ErrWebhookRejected   = errors.New("queue: webhook endpoint rejected the alert")
	ErrWebhookHTTPClient = errors.New("queue: http client is required")
)

// WebhookDLQNotifier posts quarantine alerts to a configured HTTP endpoint as
// a JSON document.
type WebhookDLQNotifier struct {
	endpoint string
	client   *http.Client
}

// NewWebhookDLQNotifier validates the endpoint and returns a notifier that
// POSTs JSON alerts. A nil client falls back to one with a 5s timeout so a
// stalled sink cannot hang the quarantine path.
func NewWebhookDLQNotifier(endpoint string, client *http.Client) (*WebhookDLQNotifier, error) {
	if endpoint == "" {
		return nil, ErrEmptyEndpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, ErrInvalidEndpoint
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &WebhookDLQNotifier{endpoint: endpoint, client: client}, nil
}

// Notify marshals the poison record and POSTs it to the configured endpoint.
// A non-2xx response is treated as delivery failure.
func (n *WebhookDLQNotifier) Notify(ctx context.Context, record PoisonRecord) error {
	if n == nil || n.endpoint == "" {
		return ErrEmptyEndpoint
	}
	if n.client == nil {
		return ErrWebhookHTTPClient
	}
	body, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("queue: marshal poison record: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("queue: build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("queue: webhook delivery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		reply, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("%w: status %d: %s", ErrWebhookRejected, resp.StatusCode, string(reply))
	}
	return nil
}
