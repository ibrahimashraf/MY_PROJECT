package server

import (
	"net/http"

	"integin/internal/evidenceapi"
)

func registerCoreRoutes(mux *http.ServeMux, d Dependencies, syncHandler http.Handler) {
	mux.Handle("/sync", syncHandler)
	if d.PilotManifestHandler != nil {
		mux.Handle("/work-package-manifest", d.PilotManifestHandler)
	}
	if d.OIDCSessionHandler != nil {
		mux.Handle("/identity/session", d.OIDCSessionHandler)
	}
	if d.WorkOrderHandler != nil {
		mux.Handle("/work-orders/partial-submissions", d.WorkOrderHandler)
	}
	if d.CertificateHandler != nil {
		mux.Handle("/certificates/", d.CertificateHandler)
	}
	if d.CertificatePublicHandler != nil {
		mux.Handle("/verify/certificates/", d.CertificatePublicHandler)
	}
	if d.EvidenceStore != nil {
		mux.Handle("/evidence", newEvidenceHandler(d))
	}
	if d.EvidenceRegistrationHandler != nil {
		mux.Handle("/evidence/metadata-registrations", d.EvidenceRegistrationHandler)
	}
	if d.LocalProvisioning != nil {
		mux.Handle("/local/provision", d.LocalProvisioning)
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
}

func newEvidenceHandler(d Dependencies) http.Handler {
	return evidenceapi.Handler{Store: d.EvidenceStore}
}
