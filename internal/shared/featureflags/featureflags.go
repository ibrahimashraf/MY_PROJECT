package featureflags

type Key string

const (
	FlagOfflineFieldWork Key = "offline_field_work"
	FlagStandardsVault   Key = "standards_vault"
	FlagAIAdvisory       Key = "ai_advisory"
	FlagClientPortal     Key = "client_portal"
	FlagCalibration      Key = "calibration"
	// FlagPilotManifestRetrieval is restricted to the isolated pilot and never implies enforcement.
	FlagPilotManifestRetrieval Key = "pilot_manifest_retrieval"
	FlagPressureTesting        Key = "pressure_testing"
	FlagHullGauging            Key = "hull_gauging"
	FlagFeatureConsole         Key = "feature_console"
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
