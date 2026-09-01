package featureflags

type Key string

const (
	FlagOfflineFieldWork        Key = "offline_field_work"
	FlagStandardsVault          Key = "standards_vault"
	FlagAIAdvisory              Key = "ai_advisory"
	FlagClientPortal            Key = "client_portal"
	FlagCalibration             Key = "calibration"
	FlagOpenAPI                 Key = "open_api_readonly"
	FlagProductPassport         Key = "product_passport"
	FlagWordDerived             Key = "word_derived_artifact"
	FlagTimeSheets              Key = "time_sheets"
	FlagCourses                 Key = "courses"
	FlagHierarchicalRegister    Key = "hierarchical_register"
	FlagBulkImportExport        Key = "bulk_import_export"
	FlagSchedulingCalendar      Key = "scheduling_calendar"
	FlagMultiInspectMode        Key = "multi_inspect_mode"
	FlagCommentLibrary          Key = "comment_library"
	FlagEscalationOverdue       Key = "escalation_overdue"
	FlagJobLinkageFailedQueue   Key = "job_linkage_failed_queue"
	FlagCustomDocxTemplates     Key = "custom_docx_templates"
	FlagClientPortalDomainsACLs Key = "client_portal_domains_acls"
	FlagIntegrationsXeroM365    Key = "integrations_xero_m365"
	FlagPartsChargesTimesheet   Key = "parts_charges_timesheet"
	FlagHSNotificationCSVExport Key = "hse_notification_csv_export"
	FlagNFCRFIDQRTagging        Key = "nfc_rfid_qr_tagging"
	FlagFullDPPRegulatory       Key = "full_dpp_regulatory"
	FlagConfigurableSettings    Key = "configurable_settings"
	// FlagPilotManifestRetrieval is restricted to the isolated pilot and never implies enforcement.
	FlagPilotManifestRetrieval Key = "pilot_manifest_retrieval"
	FlagPressureTesting        Key = "pressure_testing"
	FlagHullGauging            Key = "hull_gauging"
	FlagFeatureConsole         Key = "feature_console"
	FlagLicenseEngine          Key = "license_engine"
)

type State string

const (
	Inherited State = "INHERITED"
	Enabled   State = "ENABLED"
	Disabled  State = "DISABLED"
	Expired   State = "EXPIRED"
)

type Scope string

const (
	ScopeOrganization Scope = "ORGANIZATION"
	ScopeUser         Scope = "USER"
	ScopeClient       Scope = "CLIENT"
	ScopeProject      Scope = "PROJECT"
	ScopeDevice       Scope = "DEVICE"
)

type Override struct {
	Key     Key    `json:"key"`
	Scope   Scope  `json:"scope"`
	ScopeID string `json:"scope_id"`
	State   State  `json:"state"`
	Reason  string `json:"reason,omitempty"`
}
