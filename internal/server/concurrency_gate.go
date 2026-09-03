package server

import (
	"net/http"
	"os"
	"strconv"
)

// concurrencyGate implements an in-flight counting semaphore to protect the Go runtime
// and database connection pool from unbounded concurrency storms under 10,000 clients.
type concurrencyGate struct {
	sem chan struct{}
}

func newConcurrencyGateFromEnv() *concurrencyGate {
	max := 1000
	if raw := os.Getenv("INTEGIN_MAX_CONCURRENT_REQUESTS"); raw != "" {
		if val, err := strconv.Atoi(raw); err == nil && val > 0 {
			max = val
		}
	}
	return &concurrencyGate{
		sem: make(chan struct{}, max),
	}
}

func (cg *concurrencyGate) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health check endpoints are always admitted immediately
		if r.URL.Path == "/healthz" || r.URL.Path == "/healthz/" ||
			r.URL.Path == "/readyz" || r.URL.Path == "/readyz/" {
			next.ServeHTTP(w, r)
			return
		}

		select {
		case cg.sem <- struct{}{}:
			defer func() { <-cg.sem }()
			next.ServeHTTP(w, r)
		default:
			// Fail fast in microseconds: 0 DB queries, 0 stack growth
			w.Header().Set("Retry-After", "2")
			writeOperationalJSON(w, http.StatusServiceUnavailable, `{"error":"server_busy","retry_after_seconds":2}`)
		}
	})
}
