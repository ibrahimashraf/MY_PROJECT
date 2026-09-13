package httputil

import (
	"encoding/json/v2"
	"net/http"
)

// ProblemDetails implements RFC 7807 / RFC 9457 structured error responses.
type ProblemDetails struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitzero"`
	Instance string `json:"instance,omitzero"`
}

// WriteProblem writes a standardized JSON error response conforming to RFC 7807.
func WriteProblem(w http.ResponseWriter, r *http.Request, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	instance := ""
	if r != nil {
		instance = r.URL.Path
	}

	prob := ProblemDetails{
		Type:     "about:blank",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	}

	_ = json.MarshalWrite(w, prob, json.Deterministic(true))
}
