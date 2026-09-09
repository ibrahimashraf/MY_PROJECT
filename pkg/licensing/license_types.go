package licensing

import (
	"time"
)

type LicenseTier string

const (
	TierCommunity  LicenseTier = "COMMUNITY"
	TierEnterprise LicenseTier = "ENTERPRISE"
	TierSovereign  LicenseTier = "SOVEREIGN_AIRGAP"
)

type DeploymentMode string

const (
	ModeCloudSaaS DeploymentMode = "CLOUD_SAAS"
	ModeOnPremise DeploymentMode = "ON_PREMISE"
	ModeAirGapped DeploymentMode = "AIR_GAPPED_EDGE"
)

// PlatformCovenants defines legal and operational operational boundaries granted to an instance.
type PlatformCovenants struct {
	MaxTenants               int      `json:"max_tenants"`
	MaxInspectorsPerTenant   int      `json:"max_inspectors_per_tenant"`
	AllowedJurisdictions     []string `json:"allowed_jurisdictions"`     // e.g. ["SA", "AE", "US", "GLOBAL"]
	EnabledModules           []string `json:"enabled_modules"`           // ["LIFTING", "NDT", "PRESSURE", "OFFSHORE"]
	AirGapOfflineGraceDays   int      `json:"airgap_offline_grace_days"` // Days allowed without license phone-home
	AllowSubcontractorPortal bool     `json:"allow_subcontractor_portal"`
	AllowDynamicRulesAuthor  bool     `json:"allow_dynamic_rules_author"`
}

// LicensePayload represents the unencrypted claims verified by the kernel.
type LicensePayload struct {
	LicenseID      string            `json:"license_id"`    // "LIC-GLOBAL-2026-9901"
	Issuer         string            `json:"issuer"`        // "did:integin:authority:root-pki"
	IssuedToOrg    string            `json:"issued_to_org"` // Customer / Entity Name
	Tier           LicenseTier       `json:"tier"`          // COMMUNITY / ENTERPRISE / SOVEREIGN_AIRGAP
	Mode           DeploymentMode    `json:"mode"`          // CLOUD_SAAS / AIR_GAPPED_EDGE
	IssuedAt       time.Time         `json:"issued_at"`
	NotBefore      time.Time         `json:"not_before"`
	ExpiresAt      time.Time         `json:"expires_at"`
	HardwareLockID string            `json:"hardware_lock_id"` // Optional server hardware hash for air-gapped rigs
	Covenants      PlatformCovenants `json:"covenants"`
}

// SignedLicenseToken is the portable, cryptographically signed token string or file.
type SignedLicenseToken struct {
	Payload   LicensePayload `json:"payload"`
	Signature string         `json:"signature"`  // Hex-encoded Ed25519 signature
	PublicKey string         `json:"public_key"` // Hex-encoded Root Issuer public key
}
