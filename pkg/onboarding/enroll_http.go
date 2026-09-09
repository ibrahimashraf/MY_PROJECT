package onboarding

import (
	"encoding/json"
	"net/http"
	"sync"
)

// EnrollServer is the concrete HTTP glue for the pilot enrollment path: it
// exposes the in-memory EnrollmentSimulator over HTTP so a field_app
// simulator submission flows through the same ProcessDeviceEnrollment gate
// the Go tests exercise. It is a dev harness only — loopback-bound pilot use
// (INTEGIN_PILOT_ENROLL_ENABLED in cmd/integin-server), no database, no tenant
// scoping, no interface.
//
// The simulator's challenge/device maps are not safe for concurrent access,
// so every simulator call runs under a single mutex.
type EnrollServer struct {
	sim *EnrollmentSimulator
	mu  sync.Mutex
}

// NewEnrollServer wraps an EnrollmentSimulator for HTTP use.
func NewEnrollServer(sim *EnrollmentSimulator) *EnrollServer {
	return &EnrollServer{sim: sim}
}

// Handler mounts the enrollment routes. Mount under "/enroll/" in the server
// router: POST /enroll/challenge and POST /enroll/submit.
func (s *EnrollServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/challenge", s.createChallenge)
	mux.HandleFunc("/submit", s.submit)
	return mux
}

func (s *EnrollServer) createChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		TenantID    string `json:"tenant_id"`
		InspectorID string `json:"inspector_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeEnrollJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid challenge request body"})
		return
	}
	s.mu.Lock()
	challenge, err := s.sim.CreateEnrollmentChallenge(req.TenantID, req.InspectorID)
	s.mu.Unlock()
	if err != nil {
		writeEnrollJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeEnrollJSON(w, http.StatusCreated, challenge)
}

func (s *EnrollServer) submit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var sub DeviceEnrollmentSubmission
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		writeEnrollJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid enrollment submission body"})
		return
	}
	s.mu.Lock()
	record, err := s.sim.ProcessDeviceEnrollment(sub)
	s.mu.Unlock()
	if err != nil {
		writeEnrollJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeEnrollJSON(w, http.StatusOK, record)
}

// writeEnrollJSON marshals payload and writes it with the given status.
// Marshal errors never reach the client as partial JSON.
func writeEnrollJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		return // client disconnected; the response is already committed
	}
}
