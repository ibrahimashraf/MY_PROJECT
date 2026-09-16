package server

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"integin/internal/evidenceapi"
	"integin/internal/middleware"
	"integin/pkg/telemetry"
)

// idempotencyWrapper wraps an ingress handler with the idempotency middleware.
// It is nil when no database is available, in which case the handler is
// exposed unwrapped (existing behavior).
type idempotencyWrapper func(http.Handler) http.Handler

func wrapOrIdentity(wrapper idempotencyWrapper, handler http.Handler) http.Handler {
	if wrapper == nil {
		return handler
	}
	return wrapper(handler)
}

func registerCoreRoutes(mux *http.ServeMux, d Dependencies, rateLimiter *middleware.RateLimiter, syncHandler http.Handler, withIdempotency idempotencyWrapper) {
	mux.Handle("/metrics", telemetryMetricsHandler(telemetry.DefaultMetrics(), d, rateLimiter))
	mux.Handle("/sync", wrapOrIdentity(withIdempotency, syncHandler))
	if d.PilotManifestHandler != nil {
		mux.Handle("/work-package-manifest", d.PilotManifestHandler)
	}
	if d.OIDCSessionHandler != nil {
		mux.Handle("/identity/session", d.OIDCSessionHandler)
	}
	if d.SessionRevocationHandler != nil {
		mux.Handle("/identity/session/revoke", d.SessionRevocationHandler)
	}
	if d.SessionRevokeAllHandler != nil {
		mux.Handle("/identity/session/revoke-all", d.SessionRevokeAllHandler)
	}
	if d.WorkOrderHandler != nil {
		mux.Handle("/work-orders/partial-submissions", d.WorkOrderHandler)
	}
	if d.WorkOrderHandoverHandler != nil {
		mux.Handle("/work-orders/handovers/", d.WorkOrderHandoverHandler)
	}
	if d.WorkOrderAssignmentHandler != nil {
		mux.Handle("/work-orders/assignments", d.WorkOrderAssignmentHandler)
	}
	if d.WorkOrderEvidenceHandler != nil {
		mux.Handle("/work-orders/", d.WorkOrderEvidenceHandler)
	}
	if d.WorkOrderReconciliationHandler != nil {
		mux.Handle("/work-orders/provisional", d.WorkOrderReconciliationHandler)
		mux.Handle("/work-orders/receipts/", d.WorkOrderReconciliationHandler)
		mux.Handle("/work-orders/held", d.WorkOrderReconciliationHandler)
	}
	if d.CertificateHandler != nil {
		mux.Handle("/certificates/", d.CertificateHandler)
	}
	if d.CertificatePublicHandler != nil {
		mux.Handle("/verify/certificates/", d.CertificatePublicHandler)
	}
	mux.Handle("/verify", publicVerifierHandler())
	mux.Handle("/verify/", publicVerifierHandler())
	if d.EvidenceStore != nil {
		mux.Handle("/evidence", wrapOrIdentity(withIdempotency, newEvidenceHandler(d)))
	}
	if d.EvidenceRegistrationHandler != nil {
		mux.Handle("/evidence/metadata-registrations", d.EvidenceRegistrationHandler)
	}
	if d.LocalProvisioning != nil {
		mux.Handle("/local/provision", d.LocalProvisioning)
	}
	if d.ShortLinkHandler != nil {
		mux.Handle("/s/", d.ShortLinkHandler)
	}
	if d.QRNFCHandler != nil {
		mux.Handle("/qr-nfc/verify", d.QRNFCHandler)
	}
	if d.DeviceEnrollmentHandler != nil {
		mux.Handle("/v1/device-enrollment/", d.DeviceEnrollmentHandler)
		mux.Handle("/v1/devices/", d.DeviceEnrollmentHandler)
	}
}

func registerLicensedAPIRoutes(mux *http.ServeMux, d Dependencies) {
	if d.LicenseHandler != nil {
		mux.Handle("/api/v1/licenses/", d.LicenseHandler)
	}
	if d.FlagAdminHandler != nil {
		mux.Handle("/api/v1/admin/feature-flags", d.FlagAdminHandler)
		mux.Handle("/api/v1/admin/feature-flags/", d.FlagAdminHandler)
	}
	if d.TrainingHandler != nil {
		mux.Handle("/api/v1/training/", d.TrainingHandler)
	}
	if d.SettingsHandler != nil {
		mux.Handle("/api/v1/admin/settings", d.SettingsHandler)
		mux.Handle("/api/v1/admin/settings/", d.SettingsHandler)
	}
	if d.InspectionHandler != nil {
		mux.Handle("/api/v1/inspections", d.InspectionHandler)
	}
	if d.SearchHandler != nil {
		mux.Handle("/api/v1/search/", d.SearchHandler)
	}
	if d.AuditLogHandler != nil {
		mux.Handle("/api/v1/audit-log", d.AuditLogHandler)
		mux.Handle("/api/v1/audit-log/", d.AuditLogHandler)
	}
	if d.AnalyticsHandler != nil {
		mux.Handle("/api/v1/analytics/", d.AnalyticsHandler)
	}
	if d.AdvisoryHandler != nil {
		mux.Handle("/api/v1/advisory", d.AdvisoryHandler)
		mux.Handle("/api/v1/advisory/", d.AdvisoryHandler)
	}
	if d.ContextGroundHandler != nil {
		mux.Handle("/api/v1/context/ground", d.ContextGroundHandler)
		mux.Handle("/api/v1/context/", d.ContextGroundHandler)
	}
	if d.ReportsHandler != nil {
		mux.Handle("/api/v1/reports/", d.ReportsHandler)
	}
	if d.ShortLinkHandler != nil {
		mux.Handle("/api/v1/admin/shortlinks", d.ShortLinkHandler)
		mux.Handle("/api/v1/admin/shortlinks/", d.ShortLinkHandler)
	}
	if d.AssuranceHandler != nil {
		mux.Handle("/api/v1/assurance", d.AssuranceHandler)
		mux.Handle("/api/v1/assurance/", d.AssuranceHandler)
	}
	if d.FormDefinitionHandler != nil {
		mux.Handle("/api/v1/form-definitions", d.FormDefinitionHandler)
		mux.Handle("/api/v1/form-definitions/", d.FormDefinitionHandler)
	}
	if d.EvidencePackHandler != nil {
		mux.Handle("/api/v1/evidence-packs", d.EvidencePackHandler)
		mux.Handle("/api/v1/evidence-packs/", d.EvidencePackHandler)
	}
	if d.AssetEntitlementHandler != nil {
		mux.Handle("/api/v1/asset-entitlements", d.AssetEntitlementHandler)
		mux.Handle("/api/v1/asset-entitlements/", d.AssetEntitlementHandler)
	}
	if d.DPPHandler != nil {
		mux.Handle("/dpp/", d.DPPHandler)
		mux.Handle("/api/v1/dpp/", d.DPPHandler)
	}
	if d.TUSHandler != nil {
		gated := requireTUSAuth(d.Validator, d.TUSHandler)
		mux.Handle("/uploads", gated)
		mux.Handle("/uploads/", gated)
	}
	if d.WorkOrderAssignmentHandler != nil {
		mux.Handle("/api/v1/work-orders/", d.WorkOrderAssignmentHandler)
	}
	if d.SchedulingHandler != nil {
		mux.Handle("/api/v1/scheduling/", d.SchedulingHandler)
	}
	if d.WorkbenchHandler != nil {
		mux.Handle("/api/v1/devices", d.WorkbenchHandler)
		mux.Handle("/api/v1/devices/", d.WorkbenchHandler)
		mux.Handle("/api/v1/sync/held", d.WorkbenchHandler)
		mux.Handle("/api/v1/sync/reconcile", d.WorkbenchHandler)
		mux.Handle("/api/v1/legal-holds", d.WorkbenchHandler)
		mux.Handle("/api/v1/legal-holds/", d.WorkbenchHandler)
		mux.Handle("/api/v1/retention-policies", d.WorkbenchHandler)
		mux.Handle("/api/v1/retention-policies/", d.WorkbenchHandler)
		mux.Handle("/api/v1/exports/approvals", d.WorkbenchHandler)
	}
}

// requireTUSAuth gates the TUS upload endpoints behind the same OIDC bearer
// authentication the certificate and evidence handlers enforce. Missing
// credentials and invalid tokens return 401; a missing validator fails closed
// with 503 so uploads are never exposed unauthenticated.
func requireTUSAuth(validator TokenValidator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if validator == nil {
			writeOperationalJSON(writer, http.StatusServiceUnavailable, `{"error":"service_unavailable"}`)
			return
		}
		raw, ok := bearerToken(request.Header.Get("Authorization"))
		if !ok {
			writeOperationalJSON(writer, http.StatusUnauthorized, `{"error":"authentication_failed"}`)
			return
		}
		if _, err := validator.Validate(request.Context(), raw); err != nil {
			writeOperationalJSON(writer, http.StatusUnauthorized, `{"error":"authentication_failed"}`)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func bearerToken(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func newEvidenceHandler(d Dependencies) http.Handler {
	return evidenceapi.Handler{Store: d.EvidenceStore, Validator: d.Validator, Resolver: d.Resolver}
}

// telemetryMetricsHandler composes the observability contract exposition with
// the runtime process, rate-limit, and database families.
func telemetryMetricsHandler(metrics *telemetry.Metrics, d Dependencies, rateLimiter *middleware.RateLimiter) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writer.Header().Set("Allow", "GET, HEAD")
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writer.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		if err := metrics.WritePrometheus(writer); err != nil {
			return // client disconnected; nothing further to do
		}
		writeLegacyMetrics(writer, d, rateLimiter)
	})
}

// writeLegacyMetrics emits the process-level families that predate the
// telemetry contract; each value is a concrete observable, never a label.
func writeLegacyMetrics(writer http.ResponseWriter, d Dependencies, rateLimiter *middleware.RateLimiter) {
	allowed, denied := rateLimiter.Stats()
	fmt.Fprintf(writer, "# HELP integin_ratelimit_allowed_total Total requests allowed by rate limiter.\n")
	fmt.Fprintf(writer, "# TYPE integin_ratelimit_allowed_total counter\n")
	fmt.Fprintf(writer, "integin_ratelimit_allowed_total %d\n", allowed)
	fmt.Fprintf(writer, "# HELP integin_ratelimit_denied_total Total requests denied by rate limiter.\n")
	fmt.Fprintf(writer, "# TYPE integin_ratelimit_denied_total counter\n")
	fmt.Fprintf(writer, "integin_ratelimit_denied_total %d\n", denied)
	fmt.Fprintf(writer, "# HELP integin_go_goroutines Current goroutines.\n")
	fmt.Fprintf(writer, "# TYPE integin_go_goroutines gauge\n")
	fmt.Fprintf(writer, "integin_go_goroutines %d\n", runtime.NumGoroutine())
	if d.SyncProcessor != nil {
		fmt.Fprintf(writer, "# HELP integin_sync_offline_hmac_fallback_total Total offline sync transactions accepted via deprecated HMAC fallback.\n")
		fmt.Fprintf(writer, "# TYPE integin_sync_offline_hmac_fallback_total counter\n")
		fmt.Fprintf(writer, "integin_sync_offline_hmac_fallback_total %d\n", d.SyncProcessor.HMACFallbackCount.Load())
	}
	if d.DB != nil {
		stats := d.DB.Stats()
		fmt.Fprintf(writer, "# HELP integin_db_max_open_connections Maximum open database connections.\n")
		fmt.Fprintf(writer, "# TYPE integin_db_max_open_connections gauge\n")
		fmt.Fprintf(writer, "integin_db_max_open_connections %d\n", stats.MaxOpenConnections)
		fmt.Fprintf(writer, "# HELP integin_db_open_connections Current open database connections.\n")
		fmt.Fprintf(writer, "# TYPE integin_db_open_connections gauge\n")
		fmt.Fprintf(writer, "integin_db_open_connections %d\n", stats.OpenConnections)
		fmt.Fprintf(writer, "# HELP integin_db_in_use Current in-use database connections.\n")
		fmt.Fprintf(writer, "# TYPE integin_db_in_use gauge\n")
		fmt.Fprintf(writer, "integin_db_in_use %d\n", stats.InUse)
		fmt.Fprintf(writer, "# HELP integin_db_idle Current idle database connections.\n")
		fmt.Fprintf(writer, "# TYPE integin_db_idle gauge\n")
		fmt.Fprintf(writer, "integin_db_idle %d\n", stats.Idle)
		fmt.Fprintf(writer, "# HELP integin_db_wait_count Total database connections waited for.\n")
		fmt.Fprintf(writer, "# TYPE integin_db_wait_count counter\n")
		fmt.Fprintf(writer, "integin_db_wait_count %d\n", stats.WaitCount)
		fmt.Fprintf(writer, "# HELP integin_db_wait_duration_ms Total database connection wait duration.\n")
		fmt.Fprintf(writer, "# TYPE integin_db_wait_duration_ms counter\n")
		fmt.Fprintf(writer, "integin_db_wait_duration_ms %d\n", stats.WaitDuration.Milliseconds())
	}
}
