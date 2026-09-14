package server

import (
	"net/http"
	"os"
	"strconv"
	"time"
)

// concurrencyGate implements an in-flight counting semaphore to protect the Go runtime
// and database connection pool from unbounded concurrency storms under 10,000 clients.
type concurrencyGate struct {
	sem       chan struct{}
	healthSem chan struct{}
}

func newConcurrencyGateFromEnv() *concurrencyGate {
	max := 1000
	if raw := os.Getenv("INTEGIN_MAX_CONCURRENT_REQUESTS"); raw != "" {
		if val, err := strconv.Atoi(raw); err == nil && val > 0 {
			max = val
		}
	}
	healthMax := 50
	if raw := os.Getenv("INTEGIN_MAX_HEALTH_CONCURRENT_REQUESTS"); raw != "" {
		if val, err := strconv.Atoi(raw); err == nil && val > 0 {
			healthMax = val
		}
	}
	return &concurrencyGate{
		sem:       make(chan struct{}, max),
		healthSem: make(chan struct{}, healthMax),
	}
}

func (cg *concurrencyGate) Middleware(next http.Handler) http.Handler {
	// Enforce an absolute maximum throughput timeout (HARDEN-002)
	timeoutHandler := http.TimeoutHandler(next, 30*time.Second, `{"error":"request_timeout"}`)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health check endpoints are protected by their own dedicated semaphore (max 50)
		if r.URL.Path == "/healthz" || r.URL.Path == "/healthz/" ||
			r.URL.Path == "/readyz" || r.URL.Path == "/readyz/" {
			select {
			case cg.healthSem <- struct{}{}:
				defer func() { <-cg.healthSem }()
				next.ServeHTTP(w, r)
			default:
				w.Header().Set("Retry-After", "1")
				writeOperationalJSON(w, http.StatusServiceUnavailable, `{"error":"health_check_busy","retry_after_seconds":1}`)
			}
			return
		}

		select {
		case cg.sem <- struct{}{}:
			defer func() { <-cg.sem }()
			timeoutHandler.ServeHTTP(w, r)
		default:
			// Fail fast in microseconds: 0 DB queries, 0 stack growth
			w.Header().Set("Retry-After", "2")
			writeOperationalJSON(w, http.StatusServiceUnavailable, `{"error":"server_busy","retry_after_seconds":2}`)
		}
	})
}

