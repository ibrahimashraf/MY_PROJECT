package server

import (
	"net/http"
	"os"
	"strings"
)

// corsConfig is the scoped cross-origin policy for browser clients.
// Empty AllowedOrigins disables CORS entirely (fail closed): no
// Access-Control headers are emitted and preflights fall through to the
// mux, which rejects unknown methods per route as before.
type corsConfig struct {
	AllowedOrigins map[string]struct{}
}

func newCORSConfigFromEnv() corsConfig {
	allowed := map[string]struct{}{}
	for _, origin := range strings.Split(os.Getenv("INTEGIN_CORS_ALLOWED_ORIGINS"), ",") {
		if trimmed := strings.TrimSpace(strings.ToLower(origin)); trimmed != "" {
			allowed[trimmed] = struct{}{}
		}
	}
	return corsConfig{AllowedOrigins: allowed}
}

func (c corsConfig) allows(origin string) bool {
	if origin == "" {
		return false
	}
	_, ok := c.AllowedOrigins[strings.ToLower(origin)]
	return ok
}

// Preflight and simple-request headers exposed to browser clients. Bearer
// tokens travel in the Authorization header (no cookies), so no
// Access-Control-Allow-Credentials is emitted and a wildcard origin is
// never used.
const (
	corsAllowMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowHeaders = "Authorization, Content-Type, Idempotency-Key, X-Correlation-ID, X-Tenant-ID, X-Organization-ID, Tus-Resumable, Upload-Length, Upload-Offset, Upload-Metadata, Traceparent"
	corsMaxAge       = "600"
)

// withCORS answers CORS preflights for explicitly allowlisted origins and
// annotates matching simple requests. Non-browser requests (no Origin) and
// non-allowlisted origins pass through untouched.
func withCORS(config corsConfig, next http.Handler) http.Handler {
	if len(config.AllowedOrigins) == 0 {
		return next
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if !config.allows(origin) {
			next.ServeHTTP(writer, request)
			return
		}
		writer.Header().Set("Access-Control-Allow-Origin", origin)
		writer.Header().Set("Vary", "Origin")
		if request.Method == http.MethodOptions {
			writer.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
			writer.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
			writer.Header().Set("Access-Control-Max-Age", corsMaxAge)
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(writer, request)
	})
}
