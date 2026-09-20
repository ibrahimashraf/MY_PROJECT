package httpresponse

import (
	"encoding/json/v2"
	"net/http"
)

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

// ReadJSON decodes request JSON directly with strict unknown-field rejection and deterministic validation.
func ReadJSON(r *http.Request, v interface{}) error {
	return json.UnmarshalRead(r.Body, v, json.RejectUnknownMembers(true))
}
