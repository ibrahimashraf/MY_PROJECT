package server

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/idempotency"
	"integin/internal/identity"
	"integin/internal/middleware"
	"integin/internal/oidcauth"
	"integin/internal/packagemanifestapi"
	"integin/internal/storage"
	"integin/internal/syncapi"
	"integin/pkg/httputil"
	"integin/pkg/telemetry"
)

// TokenValidator authenticates OIDC bearer tokens at the HTTP boundary;
// *oidcauth.Validator is the production implementation. It is a minimal
// interface so route gates stay testable without a live issuing authority.
type TokenValidator interface {
	Validate(context.Context, string) (oidcauth.Principal, error)
}

type Dependencies struct {
	DB                             *sql.DB
	SyncProcessor                  *domainsync.Processor
	Devices                        []device_trust.Device
	Authorities                    []device_trust.AuthorityPackage
	EvidenceStore                  storage.Store
	Validator                      TokenValidator
	Resolver                       identity.Resolver
	EvidenceRegistrationHandler    http.Handler
	LocalProvisioning              http.Handler
	OIDCSessionHandler             http.Handler
	SessionRevocationHandler       http.Handler
	SessionRevokeAllHandler        http.Handler
	WorkOrderHandler               http.Handler
	WorkOrderEvidenceHandler       http.Handler
	WorkOrderHandoverHandler       http.Handler
	WorkOrderAssignmentHandler     http.Handler
	WorkOrderReconciliationHandler http.Handler
	CertificateHandler             http.Handler
	CertificatePublicHandler       http.Handler
	PilotManifestHandler           *packagemanifestapi.Handler
	AuthorityRegistry              *syncapi.AuthorityRegistry
	LicenseHandler                 http.Handler
	FlagAdminHandler               http.Handler
	TrainingHandler                http.Handler
	SettingsHandler                http.Handler
	InspectionHandler              http.Handler
	SearchHandler                  http.Handler
	AuditLogHandler                http.Handler
	AnalyticsHandler               http.Handler
	ReportsHandler                 http.Handler
	AdvisoryHandler                http.Handler
	ContextGroundHandler           http.Handler
	ShortLinkHandler               http.Handler
	QRNFCHandler                   http.Handler
	AssuranceHandler               http.Handler
	FormDefinitionHandler          http.Handler
	EvidencePackHandler            http.Handler
	AssetEntitlementHandler        http.Handler
	DPPHandler                     http.Handler
	TUSHandler                     http.Handler
	SchedulingHandler              http.Handler
	WorkbenchHandler               http.Handler
	DeviceEnrollmentHandler        http.Handler
	Readiness                      func(context.Context) error
	ReadinessTimeout               time.Duration
}

// NewMux composes the HTTP boundary without creating global state. Runtime
// startup is responsible for loading the signing secret and registering the
// current authority packages from an authoritative source.
func NewMux(dependencies Dependencies) http.Handler {
	for _, device := range dependencies.Devices {
		if dependencies.SyncProcessor != nil {
			dependencies.SyncProcessor.RegisterDevice(device)
		}
	}
	syncHandler := syncapi.NewHandler(dependencies.SyncProcessor)
	if dependencies.AuthorityRegistry != nil {
		syncHandler.Authorities = dependencies.AuthorityRegistry
	}
	for _, authority := range dependencies.Authorities {
		syncHandler.RegisterAuthority(authority)
	}
	if provisioner, ok := dependencies.LocalProvisioning.(interface {
		SetAuthorityRegistrar(func(device_trust.AuthorityPackage))
	}); ok {
		provisioner.SetAuthorityRegistrar(syncHandler.RegisterAuthority)
	}

	var syncHTTP http.Handler = syncHandler
	var withIdempotency idempotencyWrapper
	if dependencies.DB != nil {
		if store, storeErr := idempotency.NewPostgresStore(dependencies.DB); storeErr == nil {
			withIdempotency = func(h http.Handler) http.Handler {
				return idempotency.NewMiddleware(store, idempotency.EndpointEvidence, idempotency.EvidenceScopeResolver).Wrap(h)
			}
			syncHTTP = idempotency.NewMiddleware(store, idempotency.EndpointSync, idempotency.SyncScopeResolver).Wrap(syncHTTP)
		}
	}

	mux := http.NewServeMux()
	rateLimiter := middleware.DefaultRateLimiter()
	registerCoreRoutes(mux, dependencies, rateLimiter, syncHTTP, withIdempotency)
	registerLicensedAPIRoutes(mux, dependencies)
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writeOperationalJSON(writer, http.StatusOK, `{"status":"ok","service":"integin"}`)
	})
	mux.HandleFunc("/readyz", func(writer http.ResponseWriter, request *http.Request) {
		if dependencies.Readiness != nil {
			context, cancel := context.WithTimeout(request.Context(), readinessTimeout(dependencies.ReadinessTimeout))
			defer cancel()
			if err := dependencies.Readiness(context); err != nil {
				writeOperationalJSON(writer, http.StatusServiceUnavailable, `{"status":"not_ready","service":"integin"}`)
				return
			}
		}
		writeOperationalJSON(writer, http.StatusOK, `{"status":"ready","service":"integin"}`)
	})
	return productionMiddlewareWithLimiter(mux, rateLimiter)
}

func readinessTimeout(value time.Duration) time.Duration {
	if value <= 0 {
		return 3 * time.Second
	}
	return value
}

func writeOperationalJSON(writer http.ResponseWriter, status int, body string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_, _ = writer.Write([]byte(body))
}

func productionMiddleware(next http.Handler) http.Handler {
	rateLimiter := middleware.DefaultRateLimiter()
	return productionMiddlewareWithLimiter(next, rateLimiter)
}

func productionMiddlewareWithLimiter(next http.Handler, rateLimiter *middleware.RateLimiter) http.Handler {
	gate := newConcurrencyGateFromEnv()
	return withSecurityHeaders(requestLogger(withCorrelationID(middleware.EarlyDataMiddleware(withRequestLimit(gate.Middleware(rateLimiter.Middleware(next)), 10<<20)))))
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(writer, request)
	})
}

func withCorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		correlationID := strings.TrimSpace(request.Header.Get("X-Correlation-ID"))
		if !validCorrelationID(correlationID) {
			correlationID = newCorrelationID()
		}
		writer.Header().Set("X-Correlation-ID", correlationID)
		next.ServeHTTP(writer, request)
	})
}

func withRequestLimit(next http.Handler, limit int64) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.ContentLength > limit {
			writeOperationalJSON(writer, http.StatusRequestEntityTooLarge, `{"error":"request_too_large"}`)
			return
		}
		request.Body = http.MaxBytesReader(writer, request.Body, limit)
		next.ServeHTTP(writer, request)
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		correlationID := writer.Header().Get("X-Correlation-ID")
		tenantID := request.Header.Get("X-Tenant-ID")
		orgID := request.Header.Get("X-Organization-ID")

		// W3C trace context: accept a valid inbound traceparent, otherwise
		// generate a fresh one. Propagate downstream via context and echo the
		// header so any client can join the trace.
		traceContext, traceErr := telemetry.ParseTraceParent(request.Header.Get("Traceparent"))
		if traceErr != nil {
			traceContext, traceErr = telemetry.NewTraceContext()
		}
		if traceErr != nil {
			// Cryptographic randomness unavailable; fall back to an unsampled
			// zero-length trace so the pipeline still flows without blocking.
			traceContext = telemetry.TraceContext{Sampled: false}
		}
		writer.Header().Set("Traceparent", traceContext.String())
		traced := request.WithContext(telemetry.WithTraceContext(request.Context(), traceContext))

		// Enrich scoped logger via logger.With() and attach to context
		reqLogger := slog.Default().With(
			"correlation_id", correlationID,
			"trace_id", traceContext.TraceID,
			"method", request.Method,
			"path", loggedRequestPath(request.URL.EscapedPath()),
		)
		if tenantID != "" {
			reqLogger = reqLogger.With("tenant_id", tenantID)
		}
		if orgID != "" {
			reqLogger = reqLogger.With("org_id", orgID)
		}

		ctx := httputil.WithLogger(traced.Context(), reqLogger)
		wrapped := &statusWriter{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(wrapped, request.WithContext(ctx))

		elapsed := time.Since(started)
		reqLogger.Info("http_request",
			"status", wrapped.status,
			"duration_ms", elapsed.Milliseconds(),
		)
		recordHTTPMetrics(request, wrapped.status, elapsed)
	})
}

// recordHTTPMetrics records the request total and duration with contract-safe
// dimensions derived from the normalized route template.
func recordHTTPMetrics(request *http.Request, status int, elapsed time.Duration) {
	statusClass := status / 100
	outcome := "success"
	errorClass := ""
	switch statusClass {
	case 4:
		outcome, errorClass = "client_error", "http_4xx"
	case 5:
		outcome, errorClass = "server_error", "http_5xx"
	}
	labels := map[string]string{
		"service":          metricService(),
		"environment":      metricEnvironment(),
		"route_template":   normalizedRouteTemplate(request.URL.EscapedPath()),
		"method":           request.Method,
		"http_status_code": strconv.Itoa(status),
		"operation":        "http_server",
		"outcome":          outcome,
		"error_class":      errorClass,
	}
	metrics := telemetry.DefaultMetrics()
	metrics.IncHTTPRequest(labels)
	metrics.ObserveHTTPRequestDuration(elapsed.Seconds(), labels)
}

func metricService() string {
	if value := strings.TrimSpace(os.Getenv("INTEGIN_SERVICE_NAME")); value != "" {
		return value
	}
	return "integin"
}

func metricEnvironment() string {
	if value := strings.TrimSpace(os.Getenv("INTEGIN_ENVIRONMENT")); value != "" {
		return value
	}
	return "dev"
}

// normalizedRouteTemplate maps dynamic URL paths onto contract-safe route
// templates so request identifiers never appear as metric dimensions.
func normalizedRouteTemplate(path string) string {
	switch {
	case strings.HasPrefix(path, "/verify/certificates/"):
		return "/verify/certificates/:token"
	case strings.HasPrefix(path, "/certificates/"):
		return "/certificates/:id"
	case strings.HasPrefix(path, "/work-orders/handovers/"):
		return "/work-orders/handovers/:id"
	case strings.HasPrefix(path, "/work-orders/receipts/"):
		return "/work-orders/receipts/:id"
	case strings.HasPrefix(path, "/work-orders/held/"):
		return "/work-orders/held/:id"
	case strings.HasPrefix(path, "/s/"):
		return "/s/:code"
	case path == "":
		return "/"
	default:
		return path
	}
}

func loggedRequestPath(path string) string {
	if strings.HasPrefix(path, "/verify/certificates/") {
		return "/verify/certificates/:token"
	}
	if path == "" {
		return "/"
	}
	return path
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	return w.ResponseWriter.Write(data)
}

// Flush transparently supports chunked HTTP streaming without heap accumulation.
func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func newCorrelationID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "integin-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	}
	return hex.EncodeToString(bytes[:])
}

func validCorrelationID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '-' || character == '_' || character == '.') {
			return false
		}
	}
	return true
}
