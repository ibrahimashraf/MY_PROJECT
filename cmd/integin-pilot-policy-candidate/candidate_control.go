package main

import (
	"context"
	"net/http"
	"time"

	"integin/internal/server"
)

const candidateRollbackPath = "/pilot/candidate/rollback"

// newCandidateRollbackHandler adds a candidate-only, loopback-only operational
// control around the normal handler. It removes the policy before scheduling
// server shutdown, so a later forceful process stop is never the rollback path.
func newCandidateRollbackHandler(next http.Handler, registration *server.PilotPolicyRegistration, shutdown func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", next)
	mux.HandleFunc(candidateRollbackPath, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if registration == nil {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		registration.Disable()
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"status":"rolled_back"}`))

		if shutdown != nil {
			go func() {
				time.Sleep(50 * time.Millisecond)
				_ = shutdown(context.Background())
			}()
		}
	})
	return mux
}
