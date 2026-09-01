package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/packagemanifestapi"
	"integin/internal/storage"
	"integin/internal/syncapi"
)

type Dependencies struct {
	SyncProcessor               *domainsync.Processor
	Devices                     []device_trust.Device
	Authorities                 []device_trust.AuthorityPackage
	EvidenceStore               storage.Store
	EvidenceRegistrationHandler http.Handler
	LocalProvisioning           http.Handler
	OIDCSessionHandler          http.Handler
	WorkOrderHandler            http.Handler
	WorkOrderEvidenceHandler    http.Handler
	CertificateHandler          http.Handler
	CertificatePublicHandler    http.Handler
	PilotManifestHandler        *packagemanifestapi.Handler
	AuthorityRegistry           *syncapi.AuthorityRegistry
	LicenseHandler              http.Handler
	FlagAdminHandler            http.Handler
	TrainingHandler             http.Handler
	SettingsHandler             http.Handler
	InspectionHandler           http.Handler
	SearchHandler               http.Handler
	AuditLogHandler             http.Handler
	AnalyticsHandler            http.Handler
	ReportsHandler              http.Handler
	Readiness                   func(context.Context) error
	ReadinessTimeout            time.Duration
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

	mux := http.NewServeMux()
	registerCoreRoutes(mux, dependencies, syncHandler)
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
	return productionMiddleware(mux)
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
	return requestLogger(withCorrelationID(withRequestLimit(next, 10<<20)))
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
		wrapped := &statusWriter{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(wrapped, request)
		path := loggedRequestPath(request.URL.EscapedPath())
		slog.Default().Info("http_request", "method", request.Method, "path", path, "status", wrapped.status, "duration_ms", time.Since(started).Milliseconds(), "correlation_id", writer.Header().Get("X-Correlation-ID"))
	})
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
