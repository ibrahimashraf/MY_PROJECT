package workorderhttp

import (
	"encoding/json"
	"net/http"
)

// ReconciliationHandler provides stub endpoints for the Server Reconcile Contract (Phase 4 Gate 5).
// It returns 409 AWAITING_RECONCILIATION to signal the client to wait for reconciliation.
type ReconciliationHandler struct{}

func NewReconciliationHandler() http.Handler {
	return ReconciliationHandler{}
}

func (h ReconciliationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "AWAITING_RECONCILIATION",
	})
}
