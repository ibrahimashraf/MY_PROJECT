package idempotency

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// Status values recorded in sync_idempotency_cache.status. The set mirrors the
// sync outcome vocabulary plus the transient in-flight marker.
const (
	StatusInProgress      = "IN_PROGRESS"
	StatusApplied         = "APPLIED"
	StatusDuplicate       = "DUPLICATE"
	StatusQueued          = "QUEUED"
	StatusHeld            = "HELD"
	StatusRejected        = "REJECTED"
	StatusConflict        = "CONFLICT"
	StatusSecurityFailure = "SECURITY_FAILURE"
)

// deriveStatus extracts a structured outcome from the recorded response body.
// Ingress handlers that expose an "outcome" field (such as /sync) are cached
// with that structured value; otherwise a conservative HTTP-derived label is
// used so replay preserves the observable contract.
func deriveStatus(body []byte, httpStatus int) string {
	if len(body) > 0 {
		var envelope struct {
			Outcome string `json:"outcome"`
		}
		if err := json.Unmarshal(body, &envelope); err == nil && envelope.Outcome != "" {
			return envelope.Outcome
		}
	}
	switch {
	case httpStatus == http.StatusConflict:
		return StatusConflict
	case httpStatus >= http.StatusBadRequest && httpStatus < http.StatusInternalServerError:
		return StatusRejected
	default:
		return StatusApplied
	}
}

// responseRecorder captures everything a downstream handler writes so the
// outcome can be cached and replayed verbatim for later duplicate requests.
type responseRecorder struct {
	header      http.Header
	status      int
	buffer      bytes.Buffer
	wroteHeader bool
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{header: make(http.Header), status: http.StatusOK}
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.buffer.Write(data)
}

// Flush is a no-op kept for http.Flusher compatibility (e.g. SSE sinks).
func (r *responseRecorder) Flush() {}
