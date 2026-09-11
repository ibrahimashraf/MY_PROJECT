package server

import (
	"net/http"
	"strings"

	"integin/internal/evidenceapi"
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

func registerCoreRoutes(mux *http.ServeMux, d Dependencies, syncHandler http.Handler, withIdempotency idempotencyWrapper) {
	mux.Handle("/sync", wrapOrIdentity(withIdempotency, syncHandler))
	if d.PilotManifestHandler != nil {
		mux.Handle("/work-package-manifest", d.PilotManifestHandler)
	}
	if d.OIDCSessionHandler != nil {
		mux.Handle("/identity/session", d.OIDCSessionHandler)
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
