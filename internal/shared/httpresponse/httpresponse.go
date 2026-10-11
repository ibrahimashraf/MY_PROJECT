package httpresponse

import (
	"encoding/json/v2"
	"net/http"
)

// The writers below are the canonical shapes for HTTP error bodies. Handlers
// used to each define their own writeError/writeJSON with divergent
// signatures; the wire formats here are what those handlers already emitted,
// so routing them through this package changes no response bytes.

func JSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.MarshalWrite(w, v, json.Deterministic(true))
}

func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.MarshalWrite(w, map[string]string{"error": message}, json.Deterministic(true))
}

// CodeMessage emits {"code":...,"message":...}. Used where clients branch on a
// stable machine-readable code alongside human-readable detail.
func CodeMessage(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.MarshalWrite(w, map[string]string{"code": code, "message": message}, json.Deterministic(true))
}

// Reason emits {"reason":...}. Used by the sync and provisioning envelopes,
// whose contract names the field "reason" rather than "error".
func Reason(w http.ResponseWriter, status int, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.MarshalWrite(w, map[string]string{"reason": reason}, json.Deterministic(true))
}

// SecurityError emits {"error":...} on a response that additionally pins the
// content type and forbids sniffing, for endpoints whose body is script-adjacent.
func SecurityError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.MarshalWrite(w, map[string]string{"error": code}, json.Deterministic(true))
}

// Raw writes a pre-encoded JSON body verbatim. Used where the middleware builds
// the envelope itself (idempotency) rather than marshalling a struct.
func Raw(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// ReadJSON decodes request JSON directly with strict unknown-field rejection and deterministic validation.
func ReadJSON(r *http.Request, v interface{}) error {
	return json.UnmarshalRead(r.Body, v, json.RejectUnknownMembers(true))
}
