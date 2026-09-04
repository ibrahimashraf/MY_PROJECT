package events

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// StationEvent represents a real-time event dispatched to connected Flutter clients.
type StationEvent struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// StreamHub manages SSE subscriber connections and broadcasts real-time events.
type StreamHub struct {
	mu      sync.RWMutex
	clients map[string]map[chan StationEvent]bool // tenantID -> channels
}

func NewStreamHub() *StreamHub {
	return &StreamHub{
		clients: make(map[string]map[chan StationEvent]bool),
	}
}

// Subscribe registers a channel for a specific tenant.
func (h *StreamHub) Subscribe(tenantID string) chan StationEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan StationEvent, 16)
	if _, ok := h.clients[tenantID]; !ok {
		h.clients[tenantID] = make(map[chan StationEvent]bool)
	}
	h.clients[tenantID][ch] = true
	return ch
}

// Unsubscribe removes a client channel.
func (h *StreamHub) Unsubscribe(tenantID string, ch chan StationEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if tenantClients, ok := h.clients[tenantID]; ok {
		delete(tenantClients, ch)
		close(ch)
		if len(tenantClients) == 0 {
			delete(h.clients, tenantID)
		}
	}
}

// Broadcast sends an event to all subscribers of that tenant via a non-blocking goroutine dispatch.
func (h *StreamHub) Broadcast(event StationEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if tenantClients, ok := h.clients[event.TenantID]; ok {
		for ch := range tenantClients {
			select {
			case ch <- event:
			default:
				// Skip slow clients to prevent blocking the hub
			}
		}
	}
}

// SSEHandler returns an HTTP handler for Flutter clients subscribing to Server-Sent Events.
func (h *StreamHub) SSEHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			tenantID = r.URL.Query().Get("tenant_id")
		}
		if tenantID == "" {
			http.Error(w, "missing tenant_id", http.StatusBadRequest)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ch := h.Subscribe(tenantID)
		defer h.Unsubscribe(tenantID, ch)

		// Initial connection handshake
		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\",\"tenant_id\":%q}\n\n", tenantID)
		flusher.Flush()

		ctx := r.Context()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Fprintf(w, ": heartbeat\n\n")
				flusher.Flush()
			case event, open := <-ch:
				if !open {
					return
				}
				fmt.Fprintf(w, "event: %s\ndata: {\"id\":%q,\"type\":%q,\"payload\":%s,\"timestamp\":%q}\n\n",
					event.Type, event.ID, event.Type, event.Payload, event.Timestamp.Format(time.RFC3339))
				flusher.Flush()
			}
		}
	}
}
