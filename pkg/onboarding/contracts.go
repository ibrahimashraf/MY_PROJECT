package onboarding

import (
	"time"
)

// PlatformGenesisState represents the global root-of-trust state for INTEGIN.
type PlatformGenesisState struct {
	GenesisID            string    `json:"genesis_id"`
	InitializedAt        time.Time `json:"initialized_at"`
	RootCAKeyFingerprint string    `json:"root_ca_key_fingerprint"`
	DatabaseSchemaVer    int       `json:"database_schema_version"`
	RLSPolicyActive       bool      `json:"rls_policy_active"`
	OperatorAdminID      string    `json:"operator_admin_id"`
}

// TenantTier represents the commercial subscription tier of an onboarding organization.
type TenantTier string

const (
	TierStarter    TenantTier = "STARTER"     // SMB lifting shops (up to 5 inspectors)
	TierGrowth     TenantTier = "GROWTH"      // Commercial inspection firms (up to 25 inspectors)
	TierEnterprise TenantTier = "ENTERPRISE"  // Enterprise TIC with private sovereign options
)

// OnboardingStatus represents the discrete lifecycle state machine for tenant onboarding.
type OnboardingStatus string

const (
	StatusDraftSaved       OnboardingStatus = "DRAFT_SAVED"
	StatusPendingReview    OnboardingStatus = "PENDING_QA_REVIEW"
	StatusSandboxActive    OnboardingStatus = "SANDBOX_ACTIVE"
	StatusProductionActive OnboardingStatus = "PRODUCTION_ACTIVE"
	StatusSuspended        OnboardingStatus = "SUSPENDED"
)

// ProvisionTenantRequest is the payload sent to create a new tenant organization.
type ProvisionTenantRequest struct {
	LegalName           string     `json:"legal_name"`
	CommercialRegNo     string     `json:"commercial_registration_no"`
	VATNumber           string     `json:"vat_number"`
	PrimaryContactEmail string     `json:"primary_contact_email"`
	SubdomainSlug       string     `json:"subdomain_slug"` // e.g. "apex" -> apex.integin.com
	PlanTier            TenantTier `json:"plan_tier"`
	PreferredLocale     string     `json:"preferred_locale"` // "ar-SA" | "en-US"
}

// TenantOnboardingDraft represents the persistent, crash-resilient onboarding state machine.
type TenantOnboardingDraft struct {
	TenantID       string           `json:"tenant_id"`
	CurrentStep    int              `json:"current_step"` // 1 to 5
	CompletedSteps []int            `json:"completed_steps"`
	Status         OnboardingStatus `json:"status"`
	LastSavedAt    time.Time        `json:"last_saved_at"`

	CompanyProfile struct {
		LegalName      string `json:"legal_name"`
		CRNumber       string `json:"cr_number"`
		VATNumber      string `json:"vat_number"`
		HeadOfficeAddr string `json:"head_office_address"`
		CountryCode    string `json:"country_code"`
	} `json:"company_profile"`

	Branding struct {
		LogoVaultURL        string   `json:"logo_vault_url"`
		AccreditationBadges []string `json:"accreditation_badges"`
		OfficialStampURL    string   `json:"official_stamp_url"`
		PrimaryBrandColor   string   `json:"primary_brand_color"`
	} `json:"branding"`

	CertificatePolicy struct {
		PrefixPattern      string `json:"prefix_pattern"`
		RequireFourEyesQA  bool   `json:"require_four_eyes_qa"`
		DigitalSealEnabled bool   `json:"digital_seal_enabled"`
		PublicVerifyURL    string `json:"public_verify_url"`
	} `json:"certificate_policy"`

	ActiveDisciplines []string `json:"active_disciplines"`
}

// StagedAssetRecord represents a client asset sitting in the isolation quarantine before live commit.
type StagedAssetRecord struct {
	StagingRowID     string     `json:"staging_row_id"`
	TenantID         string     `json:"tenant_id"`
	BatchID          string     `json:"batch_id"`
	RawSource        string     `json:"raw_source"`
	ClientName       string     `json:"client_name"`
	SiteLocation     string     `json:"site_location"`
	AssetTag         string     `json:"asset_tag"`
	EquipmentType    string     `json:"equipment_type"`
	Manufacturer     string     `json:"manufacturer"`
	Model            string     `json:"model"`
	SerialNumber     string     `json:"serial_number"`
	SafeWorkingLoad  string     `json:"safe_working_load"`
	LastInspectionAt *time.Time `json:"last_inspection_at,omitempty"`
	NextDueAt        *time.Time `json:"next_due_at,omitempty"`
	ValidationStatus string     `json:"validation_status"`
	ValidationErrors []string   `json:"validation_errors,omitempty"`
}

// QualificationVerificationStatus represents QA sign-off on an inspector's credentials.
type QualificationVerificationStatus string

const (
	QualPendingReview QualificationVerificationStatus = "PENDING_QA_REVIEW"
	QualApproved      QualificationVerificationStatus = "APPROVED_BY_TECHNICAL_DIRECTOR"
	QualExpired       QualificationVerificationStatus = "EXPIRED"
	QualRevoked       QualificationVerificationStatus = "REVOKED"
)

// InspectorCredential represents a professional qualification (LEEA, ASNT, API).
type InspectorCredential struct {
	CredentialID       string                          `json:"credential_id"`
	InspectorID        string                          `json:"inspector_id"`
	Discipline         string                          `json:"discipline"`
	CertificateNumber  string                          `json:"certificate_number"`
	IssuingBody        string                          `json:"issuing_body"`
	ValidFrom          time.Time                       `json:"valid_from"`
	ExpiresAt          time.Time                       `json:"expires_at"`
	VerificationStatus QualificationVerificationStatus `json:"verification_status"`
	VerifiedByUserID   string                          `json:"verified_by_user_id,omitempty"`
	VerifiedAt         *time.Time                      `json:"verified_at,omitempty"`
	DocumentVaultRef   string                          `json:"document_vault_ref"`
}

// DeviceEnrollmentChallenge is issued by the server to initiate QR scan-to-pair.
type DeviceEnrollmentChallenge struct {
	ChallengeID   string    `json:"challenge_id"`
	TenantID      string    `json:"tenant_id"`
	InspectorID   string    `json:"inspector_id"`
	Nonce         string    `json:"nonce"`
	ExpiresAt     time.Time `json:"expires_at"`
	QRCodePayload string    `json:"qr_code_payload"`
}

// DeviceEnrollmentSubmission is the cryptographic response sent by the field tablet.
type DeviceEnrollmentSubmission struct {
	ChallengeID       string           `json:"challenge_id"`
	InspectorID       string           `json:"inspector_id"`
	DevicePublicKey   string           `json:"device_public_key"`
	DeviceFingerprint string           `json:"device_fingerprint"`
	DeviceModel       string           `json:"device_model"`
	SignedNonce       string           `json:"signed_nonce"`
	Attestation       AttestationClaim `json:"attestation,omitempty"`
}

// KeyOrigin identifies where the device's signing key material is protected.
type KeyOrigin string

const (
	KeyOriginSecureEnclave KeyOrigin = "SECURE_ENCLAVE" // Apple Secure Enclave
	KeyOriginStrongBox     KeyOrigin = "STRONGBOX"      // Android StrongBox / TEE
	KeyOriginSoftware      KeyOrigin = "SOFTWARE"       // portable Ed25519 keypair
	KeyOriginNone          KeyOrigin = "NONE"           // honest "no attestation evidence"
)

// AttestationClaim is hardware-attestation evidence supplied by the field
// tablet during enrollment. The server treats AttestationBlob as opaque and
// never parses platform internals; validation only checks presence, origin,
// and biometric binding. Zero value = unattested legacy device.
type AttestationClaim struct {
	KeyOrigin       KeyOrigin `json:"key_origin"`
	BiometricBound  bool      `json:"biometric_bound"`
	OSVersion       string    `json:"os_version,omitempty"`
	AttestationBlob string    `json:"attestation_blob,omitempty"`
	KeyAlias        string    `json:"key_alias,omitempty"`
}

// DeviceTrustRecord represents an active, authenticated field terminal.
type DeviceTrustRecord struct {
	DeviceID                  string     `json:"device_id"`
	TenantID                  string     `json:"tenant_id"`
	InspectorID               string     `json:"inspector_id"`
	DevicePublicKey           string     `json:"device_public_key"`
	DeviceModel               string     `json:"device_model"`
	IsActive                  bool       `json:"is_active"`
	EnrolledAt                time.Time  `json:"enrolled_at"`
	LastSyncedAt              time.Time  `json:"last_synced_at"`
	RevokedAt                 *time.Time `json:"revoked_at,omitempty"`
	RevocationReason          string     `json:"revocation_reason,omitempty"`
	AttestationOrigin         string     `json:"attestation_origin,omitempty"`
	AttestationBiometricBound bool       `json:"attestation_biometric_bound,omitempty"`
}

// WorkPackageManifest is the server-signed bundle dispatched to the offline tablet.
type WorkPackageManifest struct {
	ManifestID         string    `json:"manifest_id"`
	TenantID           string    `json:"tenant_id"`
	WorkOrderID        string    `json:"work_order_id"`
	AssignedDeviceID   string    `json:"assigned_device_id"`
	InspectorID        string    `json:"inspector_id"`
	IssuedAt           time.Time `json:"issued_at"`
	ValidUntil         time.Time `json:"valid_until"`
	PackageHash        string    `json:"package_hash"`
	AuthoritySignature string    `json:"authority_signature"`
}

// SignedInspectionReceipt is the sealed artifact returned by the offline tablet.
type SignedInspectionReceipt struct {
	ReceiptID          string    `json:"receipt_id"`
	ManifestID         string    `json:"manifest_id"`
	WorkOrderID        string    `json:"work_order_id"`
	AssetID            string    `json:"asset_id"`
	CompletedAt        time.Time `json:"completed_at"`
	OverallResult      string    `json:"overall_result"`
	PayloadDigest      string    `json:"payload_digest"`
	DeviceSignature    string    `json:"device_signature"`
	ClientRepSignature string    `json:"client_rep_signature,omitempty"`
}
