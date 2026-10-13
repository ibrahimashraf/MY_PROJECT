package contextground

import (
	"encoding/json"
	"net/http"

	"integin/pkg/httputil"
)

// groundBodyLimit caps the JSON request body (paths, symbols, config) well
// below anything pathological.
const groundBodyLimit = 2 << 20 // 2 MiB

// HTTPHandler returns the handler for POST /v1/context/ground. It decodes a
// GroundingRequest, validates that it names at least one file or symbol, runs
// the grounded extraction, and returns the attested GroundingPayload as JSON.
func HTTPHandler(grounder *ContextGrounder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, groundBodyLimit)
		var req GroundingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteProblem(w, r, http.StatusBadRequest, "Invalid Request", "malformed JSON body")
			return
		}
		if len(req.FilePaths) == 0 && len(req.Symbols) == 0 {
			httputil.WriteProblem(w, r, http.StatusBadRequest, "Invalid Request", "file_paths or symbols must be non-empty")
			return
		}
		payload, err := grounder.Ground(r.Context(), req)
		if err != nil {
			httputil.WriteProblem(w, r, http.StatusBadRequest, "Grounding Failed", err.Error())
			return
		}
		body, err := json.Marshal(payload)
		if err != nil {
			httputil.WriteProblem(w, r, http.StatusInternalServerError, "Internal Server Error", "failed to marshal grounding payload")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(body); err != nil {
			return // client disconnected; response already committed
		}
	}
}
