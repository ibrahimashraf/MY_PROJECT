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
	"integin/internal/evidenceapi"
	"integin/internal/storage"
	"integin/internal/syncapi"
)

type Dependencies struct {
  SyncProcessor      *domainsync.Processor
  Devices            []device_trust.Device
  Authorities        []device_trust.AuthorityPackage
  EvidenceStore      storage.Store
  LocalProvisioning  http.Handler
  Readiness          func(context.Context) error
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
  for _, authority := range dependencies.Authorities {
    syncHandler.RegisterAuthority(authority)
  }
  if provisioner, ok := dependencies.LocalProvisioning.(interface {
    SetAuthorityRegistrar(func(device_trust.AuthorityPackage))
  }); ok {
    provisioner.SetAuthorityRegistrar(syncHandler.RegisterAuthority)
  }

	mux := http.NewServeMux()
	mux.Handle("/sync", syncHandler)
  if dependencies.EvidenceStore != nil {
    mux.Handle("/evidence", evidenceapi.Handler{Store: dependencies.EvidenceStore})
  }
  if dependencies.LocalProvisioning != nil {
    mux.Handle("/local/provision", dependencies.LocalProvisioning)
  }
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{"status":"ok","service":"integin"}`))
		})
	mux.HandleFunc("/readyz", func(writer http.ResponseWriter, request *http.Request) {
		if dependencies.Readiness != nil {
			if err := dependencies.Readiness(request.Context()); err != nil {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusServiceUnavailable)
				_, _ = writer.Write([]byte(`{"status":"not_ready","service":"integin"}`))
				return
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"status":"ready","service":"integin"}`))
	})
	return productionMiddleware(mux)
}

func productionMiddleware(next http.Handler) http.Handler {
	return requestLogger(withCorrelationID(withRequestLimit(next, 10<<20)))
}

func withCorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		correlationID := strings.TrimSpace(request.Header.Get("X-Correlation-ID"))
		if correlationID == "" || len(correlationID) > 128 {
			correlationID = newCorrelationID()
		}
		writer.Header().Set("X-Correlation-ID", correlationID)
		next.ServeHTTP(writer, request)
	})
}

func withRequestLimit(next http.Handler, limit int64) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request.Body = http.MaxBytesReader(writer, request.Body, limit)
		next.ServeHTTP(writer, request)
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		wrapped := &statusWriter{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(wrapped, request)
		slog.Default().Info("http_request", "method", request.Method, "path", request.URL.Path, "status", wrapped.status, "duration_ms", time.Since(started).Milliseconds(), "correlation_id", writer.Header().Get("X-Correlation-ID"))
	})
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
