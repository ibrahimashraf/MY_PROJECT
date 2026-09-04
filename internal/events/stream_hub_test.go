package events

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStreamHubBroadcast(t *testing.T) {
	hub := NewStreamHub()
	tenantID := "tenant-alpha"

	ch := hub.Subscribe(tenantID)
	defer hub.Unsubscribe(tenantID, ch)

	event := StationEvent{
		ID:        "evt-001",
		TenantID:  tenantID,
		Type:      "WORKORDER_SUBMITTED",
		Payload:   `{"work_order_id":"wo-999"}`,
		Timestamp: time.Now().UTC(),
	}

	hub.Broadcast(event)

	select {
	case received := <-ch:
		if received.ID != "evt-001" || received.Type != "WORKORDER_SUBMITTED" {
			t.Fatalf("unexpected event: %+v", received)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for broadcast event")
	}
}

func TestStreamHubRejectsMissingTenant(t *testing.T) {
	hub := NewStreamHub()
	handler := hub.SSEHandler()

	req := httptest.NewRequest(http.MethodGet, "/events/stream", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing tenant, got %d", rec.Code)
	}
}
