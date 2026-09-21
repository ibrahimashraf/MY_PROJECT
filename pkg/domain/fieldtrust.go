// Package domain carries the field-trust protocol: risk-weighted authority,
// time-horizon enforcement, sub-assembly leasing, execution gating, hardware
// countersigned submissions, and in-chain escrow overrides.
//
// Design authority: phased adversarial review (pitch semantics -> protocol
// design -> enforcement review -> mint-time guards). Every check is
// fail-closed; every sentinel is errors.Is-matchable across the package
// boundary (multi-%w wraps preserve inner sentinels).
package domain

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"
)

// -------------------------------------------------------------------------
// 0. DOMAIN SEPARATION TAGS & SENTINELS
// -------------------------------------------------------------------------

const (
	TagHorizonToken   = "INTEGIN-HORIZON-v1\x00"
	TagSubmissionSeal = "INTEGIN-SEAL-v1\x00"
	TagEscrowOverride = "INTEGIN-OVERRIDE-v1\x00"
)

var (
	ErrHorizonExpired               = errors.New("timeguard: execution exceeds horizon token duration")
	ErrWallClockSkewExceeded        = errors.New("timeguard: wall clock drift exceeds allowable boundary")
	ErrMonotonicBroken              = errors.New("timeguard: monotonic sequence broken without valid reboot state")
	ErrRebootGraceExpired           = errors.New("timeguard: reboot recovery grace window expired; re-anchor required")
	ErrInvalidIssuerSignature       = errors.New("timeguard: token signature invalid against edge authority")
	ErrStaleAuthorityEpoch          = errors.New("timeguard: token authority epoch is below edge high-water epoch")
	ErrHorizonCapExceeded           = errors.New("timeguard: requested horizon exceeds maximum policy duration")
	ErrSkewCapExceeded              = errors.New("timeguard: requested maxSkew exceeds policy allowable skew ceiling")
	ErrBootTamperDetected           = errors.New("timeguard: boot session ID mismatch without valid cold-boot checkpoint")
	ErrCheckpointMissing            = errors.New("timeguard: no checkpoint ledger entry; re-anchor required")
	ErrUnknownCriticality           = errors.New("timeguard: unknown asset criticality class")
	ErrInvalidSeal                  = errors.New("submission: composite cryptographic seal failed verification")
	ErrMalformedP256Sig             = errors.New("submission: malformed P-256 signature (neither DER nor raw IEEE P1363)")
	ErrDomainContextMismatch        = errors.New("submission: context or device DID mismatch against seal payload")
	ErrStatutoryHardwareSealMissing = errors.New("submission: statutory verification requires hardware P-256 seal")
	ErrTombstoneStaleSequence       = errors.New("escrow: presented tombstone sequence does not match edge stored sequence")
	ErrEscrowPermanentFlagRequired  = errors.New("escrow: permanent audit flag must be asserted true")
)

// -------------------------------------------------------------------------
// 1. RISK-WEIGHTED CRITICALITY POLICY & ENUMS
// -------------------------------------------------------------------------

type AssetCriticality string

const (
	CriticalityLifeSafety AssetCriticality = "LIFE_SAFETY" // Hoists, crane hooks, BOP stacks
	CriticalitySecondary  AssetCriticality = "SECONDARY"   // Slings, shackles, pad eyes, baskets
)

type RootCauseCode string

const (
	RootCauseMedicalEvacuation      RootCauseCode = "MEDICAL_EVACUATION"
	RootCauseShiftHandoverAbandoned RootCauseCode = "SHIFT_HANDOVER_ABANDONMENT"
	RootCauseAtmosphericHazard      RootCauseCode = "ATMOSPHERIC_HAZARD_HALT"
	RootCauseHardwareFailureTotal   RootCauseCode = "HARDWARE_FAILURE_TOTAL"
	RootCauseOperationalEmergency   RootCauseCode = "OPERATIONAL_EMERGENCY"
)

type TTLPolicy struct {
	AuthorityTTL     time.Duration
	MaxHorizon       time.Duration
	MaxAllowableSkew time.Duration // Wall-clock deviation tolerance per criticality class
	MaxRebootGrace   time.Duration // Post-reboot execution allowance before hard re-anchor
	CompetencySlack  time.Duration // Zero-tolerance on LIFE_SAFETY; 24h statutory concession on SECONDARY
}

var CriticalityPolicies = map[AssetCriticality]TTLPolicy{
	CriticalityLifeSafety: {
		AuthorityTTL:     12 * time.Hour,
		MaxHorizon:       8 * time.Hour,
		MaxAllowableSkew: 5 * time.Minute,
		MaxRebootGrace:   15 * time.Minute,
		CompetencySlack:  0,
	},
	CriticalitySecondary: {
		AuthorityTTL:     96 * time.Hour,
		MaxHorizon:       72 * time.Hour,
		MaxAllowableSkew: 30 * time.Minute,
		MaxRebootGrace:   30 * time.Minute,
		CompetencySlack:  24 * time.Hour, // Requires statutory sign-off in deployment jurisdiction
	},
}

// -------------------------------------------------------------------------
// 2. TIME HORIZON TOKEN & REBOOT-RESILIENT TIMEGUARD
// -------------------------------------------------------------------------

type TimeHorizonToken struct {
	TokenID          string           `json:"token_id"`
	WorkPackageID    string           `json:"work_package_id"`
	InspectorID      string           `json:"inspector_id"`
	CriticalityClass AssetCriticality `json:"criticality_class"`
	AnchorWallTime   time.Time        `json:"anchor_wall_time"`
	AnchorMonoNanos  int64            `json:"anchor_mono_nanos"`
	MaxSkewAllowed   time.Duration    `json:"max_skew_allowed"`
	HorizonDuration  time.Duration    `json:"horizon_duration"`
	AuthorityEpoch   uint64           `json:"authority_epoch"`
	IssuerSignature  []byte           `json:"issuer_signature"`
}

type CheckpointLedgerEntry struct {
	WallTimestamp  time.Time `json:"wall_timestamp"`
	MonotonicNanos int64     `json:"monotonic_nanos"`
	BootSessionID  string    `json:"boot_session_id"`
}

type RebootRecoveryState struct {
	ActiveBootSessionID string `json:"active_boot_session_id"`
}

func (t *TimeHorizonToken) Digest() []byte {
	h := sha256.New()
	h.Write([]byte(TagHorizonToken))
	h.Write([]byte(t.TokenID))
	h.Write([]byte(t.WorkPackageID))
	h.Write([]byte(t.InspectorID))
	h.Write([]byte(t.CriticalityClass))
	_ = binary.Write(h, binary.BigEndian, t.AnchorWallTime.UnixNano())
	_ = binary.Write(h, binary.BigEndian, t.AnchorMonoNanos)
	_ = binary.Write(h, binary.BigEndian, t.MaxSkewAllowed.Nanoseconds())
	_ = binary.Write(h, binary.BigEndian, t.HorizonDuration.Nanoseconds())
	_ = binary.Write(h, binary.BigEndian, t.AuthorityEpoch)
	return h.Sum(nil)
}

// IssueTimeHorizonToken mints a strictly capped, signed token at checkout.
// Signing dispatches on concrete key type: ECDSA rejects Hash(0) opts while
// Ed25519 rejects anything but Hash(0)/SHA-512, so no single crypto.Signer
// opts value serves both — the switch below is load-bearing, not stylistic.
// ECDSA signatures are ASN.1 DER, matching the parseP256Signature arm in
// VerifyToken.
func IssueTimeHorizonToken(
	tokenID string,
	workPackageID string,
	inspectorID string,
	criticality AssetCriticality,
	anchorWall time.Time,
	anchorMono int64,
	maxSkew time.Duration,
	requestedDuration time.Duration,
	currentEpoch uint64,
	issuerPrivKey crypto.Signer,
) (*TimeHorizonToken, error) {
	policy, ok := CriticalityPolicies[criticality]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownCriticality, criticality)
	}

	if requestedDuration > policy.MaxHorizon {
		return nil, fmt.Errorf("%w: requested %s, max %s", ErrHorizonCapExceeded, requestedDuration, policy.MaxHorizon)
	}

	if maxSkew > policy.MaxAllowableSkew {
		return nil, fmt.Errorf("%w: requested %s, max allowable %s", ErrSkewCapExceeded, maxSkew, policy.MaxAllowableSkew)
	}

	tok := &TimeHorizonToken{
		TokenID:          tokenID,
		WorkPackageID:    workPackageID,
		InspectorID:      inspectorID,
		CriticalityClass: criticality,
		AnchorWallTime:   anchorWall,
		AnchorMonoNanos:  anchorMono,
		MaxSkewAllowed:   maxSkew,
		HorizonDuration:  requestedDuration,
		AuthorityEpoch:   currentEpoch,
	}

	var sig []byte
	switch key := issuerPrivKey.(type) {
	case ed25519.PrivateKey:
		sig = ed25519.Sign(key, tok.Digest())
	case *ecdsa.PrivateKey:
		var err error
		sig, err = ecdsa.SignASN1(rand.Reader, key, tok.Digest())
		if err != nil {
			return nil, fmt.Errorf("failed to sign time horizon token: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported issuer private key type %T", issuerPrivKey)
	}
	tok.IssuerSignature = sig

	return tok, nil
}

// VerifyToken validates cryptographic provenance and epoch currency upon
// ingest. Call before trusting any other token field.
func (t *TimeHorizonToken) VerifyToken(issuerPubKey crypto.PublicKey, edgeHighWaterEpoch uint64) error {
	if t.AuthorityEpoch < edgeHighWaterEpoch {
		return fmt.Errorf("%w: token epoch %d < edge high-water %d", ErrStaleAuthorityEpoch, t.AuthorityEpoch, edgeHighWaterEpoch)
	}

	digest := t.Digest()

	switch pub := issuerPubKey.(type) {
	case ed25519.PublicKey:
		if !ed25519.Verify(pub, digest, t.IssuerSignature) {
			return ErrInvalidIssuerSignature
		}
	case *ecdsa.PublicKey:
		r, s, err := parseP256Signature(t.IssuerSignature)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidIssuerSignature, err)
		}
		if !ecdsa.Verify(pub, digest, r, s) {
			return ErrInvalidIssuerSignature
		}
	default:
		return errors.New("unsupported issuer public key type")
	}

	return nil
}

func (t *TimeHorizonToken) ValidateExecutionTime(
	currentWall time.Time,
	currentMonoNanos int64,
	checkpoint CheckpointLedgerEntry,
	rebootState RebootRecoveryState,
) error {
	policy := CriticalityPolicies[t.CriticalityClass]

	// A zero checkpoint carries no continuity basis, and year-1 wall
	// arithmetic overflows int64 durations (wrapping to a negative
	// remaining horizon). Legacy devices without checkpoint support fail
	// closed here, never inside duration math.
	if checkpoint.WallTimestamp.IsZero() {
		return ErrCheckpointMissing
	}

	// 1. Evaluate Wall Clock Skew against Anchor
	expectedElapsedWall := currentWall.Sub(t.AnchorWallTime)
	if expectedElapsedWall < 0 || (t.MaxSkewAllowed > 0 && expectedElapsedWall > t.HorizonDuration+t.MaxSkewAllowed) {
		return ErrWallClockSkewExceeded
	}

	// 2. Monotonic Continuity vs Cold Boot Checkpoint
	rebootDetected := (currentMonoNanos < checkpoint.MonotonicNanos) ||
		(rebootState.ActiveBootSessionID != checkpoint.BootSessionID)

	if rebootDetected {
		if rebootState.ActiveBootSessionID == "" {
			return ErrBootTamperDetected
		}

		remainingHorizon := t.HorizonDuration - checkpoint.WallTimestamp.Sub(t.AnchorWallTime)
		if remainingHorizon < 0 {
			return ErrHorizonExpired
		}

		// Grace window must never exceed policy cap or remaining horizon
		effectiveGrace := policy.MaxRebootGrace
		if remainingHorizon < effectiveGrace {
			effectiveGrace = remainingHorizon
		}

		// Elapsed since reboot is measured directly by the kernel monotonic
		// counter, which resets at boot. Never derive it from checkpoint
		// wall time: checkpoint age would false-lock healthy reboots.
		rebootElapsed := time.Duration(currentMonoNanos)
		if rebootElapsed > effectiveGrace {
			return ErrRebootGraceExpired
		}

		return nil
	}

	// 3. Normal Execution Path: Monotonic Evaluation
	elapsedMono := time.Duration(currentMonoNanos - t.AnchorMonoNanos)
	if elapsedMono < 0 {
		return ErrMonotonicBroken
	}
	if elapsedMono > t.HorizonDuration {
		return ErrHorizonExpired
	}

	return nil
}

// -------------------------------------------------------------------------
// 3. SUB-ASSEMBLY LEASE, TOMBSTONES & ADAPTER PATTERNS
// -------------------------------------------------------------------------

type LeaseState string

const (
	LeaseActive       LeaseState = "ACTIVE"
	LeaseOverrunDraft LeaseState = "OVERRUN_DRAFT"
	LeaseTombstoned   LeaseState = "TOMBSTONED"
)

// NOTE: VersionVector stripped per clean-up decision. Tombstone supremacy is
// strictly epoch-gated (AssertsSupremacy). If air-gapped edge-to-edge gossip
// sync is ever implemented, attach vector clocks here to order concurrent
// tombstones — never to gate mobile client writes.
type SubAssemblyLease struct {
	SubAssemblyID    string           `json:"sub_assembly_id"` // e.g., "CRANE-01:BOOM_LACING"
	ParentAssetDID   string           `json:"parent_asset_did"`
	AssignedDeviceID string           `json:"assigned_device_id"`
	InspectorID      string           `json:"inspector_id"`
	RequiredSkill    string           `json:"required_skill"` // e.g., "NDT-MT-L2", "LEEA-MC"
	CriticalityClass AssetCriticality `json:"criticality_class"`
	LeaseEpoch       uint64           `json:"lease_epoch"`
	LeaseStart       time.Time        `json:"lease_start"`
	ValidUntil       time.Time        `json:"valid_until"`
	State            LeaseState       `json:"state"`
}

type WorkPackageManifest struct {
	ManifestID       string             `json:"manifest_id"`
	TenantID         string             `json:"tenant_id"`
	AssignedDeviceID string             `json:"assigned_device_id"`
	InspectorID      string             `json:"inspector_id"`
	AuthorityEpoch   uint64             `json:"authority_epoch"`
	TimeHorizon      TimeHorizonToken   `json:"time_horizon"`
	LeasedComponents []SubAssemblyLease `json:"leased_components"`
	ValidUntil       time.Time          `json:"valid_until"`
}

// LegacyOnboardingManifest preserves wire-level compatibility with the
// onboarding package's manifest shape (tenant/device/inspector/epoch/validity
// only). Prefer importing the canonical type where the module allows; this
// shadow exists to avoid an import cycle at the domain boundary.
type LegacyOnboardingManifest struct {
	ManifestID       string    `json:"manifest_id"`
	TenantID         string    `json:"tenant_id"`
	AssignedDeviceID string    `json:"assigned_device_id"`
	InspectorID      string    `json:"inspector_id"`
	AuthorityEpoch   uint64    `json:"authority_epoch"`
	ValidUntil       time.Time `json:"valid_until"`
}

func (m *WorkPackageManifest) ToOnboardingManifest() LegacyOnboardingManifest {
	return LegacyOnboardingManifest{
		ManifestID:       m.ManifestID,
		TenantID:         m.TenantID,
		AssignedDeviceID: m.AssignedDeviceID,
		InspectorID:      m.InspectorID,
		AuthorityEpoch:   m.AuthorityEpoch,
		ValidUntil:       m.ValidUntil,
	}
}

func FromOnboardingManifest(
	legacy LegacyOnboardingManifest,
	token TimeHorizonToken,
	leases []SubAssemblyLease,
) WorkPackageManifest {
	return WorkPackageManifest{
		ManifestID:       legacy.ManifestID,
		TenantID:         legacy.TenantID,
		AssignedDeviceID: legacy.AssignedDeviceID,
		InspectorID:      legacy.InspectorID,
		AuthorityEpoch:   legacy.AuthorityEpoch,
		TimeHorizon:      token,
		LeasedComponents: leases,
		ValidUntil:       legacy.ValidUntil,
	}
}

type LeaseTombstoneRecord struct {
	SubAssemblyID       string    `json:"sub_assembly_id"`
	LeaseEpoch          uint64    `json:"lease_epoch"`
	TombstoneSequence   uint64    `json:"tombstone_sequence"`
	RevokedDeviceID     string    `json:"revoked_device_id"`
	RevocationTimestamp time.Time `json:"revocation_timestamp"`
	AuthorizingDID      string    `json:"authorizing_did"`
	Reason              string    `json:"reason"`
}

// AssertsSupremacy strictly suppresses rogue writes from a revoked device.
// Any write from RevokedDeviceID under an epoch <= tombstone.LeaseEpoch is
// suppressed, regardless of the writer's local sequence counter. Writes under
// a newly provisioned epoch pass: re-admission is an explicit re-issue, never
// an accident of counter arithmetic.
func (t *LeaseTombstoneRecord) AssertsSupremacy(incomingDeviceID string, incomingEpoch uint64) bool {
	if t.RevokedDeviceID != incomingDeviceID {
		return false
	}
	return incomingEpoch <= t.LeaseEpoch
}

// -------------------------------------------------------------------------
// 4. GRANULAR COMPETENCY TUPLE (EXECUTION GATE)
// -------------------------------------------------------------------------

type InspectorCompetency struct {
	SkillCode    string    `json:"skill_code"` // e.g., "NDT-UT-L2"
	ValidFrom    time.Time `json:"valid_from"`
	ValidThrough time.Time `json:"valid_through"`
	IsRevoked    bool      `json:"is_revoked"`
}

type ExecutionGateResult struct {
	SubAssemblyID string `json:"sub_assembly_id"`
	SkillCode     string `json:"skill_code"`
	Allowed       bool   `json:"allowed"`
	PartialBlock  bool   `json:"partial_block"`
	FailureReason string `json:"failure_reason,omitempty"`
}

// EvaluateExecutionGate executes partial-block competency validation.
// currentValidatedTime MUST be sourced from timeguard-validated time, NEVER
// naked wall clock: feeding DateTime.now()/time.Now() reopens the rollback
// vector this gate exists to close.
func EvaluateExecutionGate(
	inspectorID string,
	lease SubAssemblyLease,
	token TimeHorizonToken,
	competencies []InspectorCompetency,
	currentValidatedTime time.Time,
) ExecutionGateResult {
	if lease.InspectorID != inspectorID || token.InspectorID != inspectorID {
		return ExecutionGateResult{
			SubAssemblyID: lease.SubAssemblyID,
			SkillCode:     lease.RequiredSkill,
			Allowed:       false,
			PartialBlock:  true,
			FailureReason: "inspector ID mismatch across lease and authority token",
		}
	}

	// Policy-Shopping Guard: token and lease criticality classes MUST match,
	// or a long-horizon secondary token executes against life-safety hardware.
	if lease.CriticalityClass != token.CriticalityClass {
		return ExecutionGateResult{
			SubAssemblyID: lease.SubAssemblyID,
			SkillCode:     lease.RequiredSkill,
			Allowed:       false,
			PartialBlock:  true,
			FailureReason: fmt.Sprintf("criticality mismatch: lease requires %s, token is %s",
				lease.CriticalityClass, token.CriticalityClass),
		}
	}

	// Replaces unverified map access: fails loudly on unknown or unconfigured class.
	policy, ok := CriticalityPolicies[lease.CriticalityClass]
	if !ok {
		return ExecutionGateResult{
			SubAssemblyID: lease.SubAssemblyID,
			SkillCode:     lease.RequiredSkill,
			Allowed:       false,
			PartialBlock:  true,
			FailureReason: fmt.Sprintf("invalid or unregistered criticality class: %s", lease.CriticalityClass),
		}
	}

	var matched *InspectorCompetency
	for i := range competencies {
		if competencies[i].SkillCode == lease.RequiredSkill {
			matched = &competencies[i]
			break
		}
	}

	if matched == nil {
		return ExecutionGateResult{
			SubAssemblyID: lease.SubAssemblyID,
			SkillCode:     lease.RequiredSkill,
			Allowed:       false,
			PartialBlock:  true,
			FailureReason: fmt.Sprintf("no credential on file for skill %s", lease.RequiredSkill),
		}
	}

	if matched.IsRevoked {
		return ExecutionGateResult{
			SubAssemblyID: lease.SubAssemblyID,
			SkillCode:     lease.RequiredSkill,
			Allowed:       false,
			PartialBlock:  true,
			FailureReason: fmt.Sprintf("credential for %s is explicitly revoked", lease.RequiredSkill),
		}
	}

	effectiveExpiry := matched.ValidThrough.Add(policy.CompetencySlack)

	if currentValidatedTime.Before(matched.ValidFrom) || currentValidatedTime.After(effectiveExpiry) {
		return ExecutionGateResult{
			SubAssemblyID: lease.SubAssemblyID,
			SkillCode:     lease.RequiredSkill,
			Allowed:       false,
			PartialBlock:  true,
			FailureReason: fmt.Sprintf("competency %s expired at %s (slack %s applied: %t)",
				matched.SkillCode, matched.ValidThrough, policy.CompetencySlack, policy.CompetencySlack > 0),
		}
	}

	return ExecutionGateResult{
		SubAssemblyID: lease.SubAssemblyID,
		SkillCode:     lease.RequiredSkill,
		Allowed:       true,
		PartialBlock:  false,
	}
}

// -------------------------------------------------------------------------
// 5. SUBMISSION COUNTERSIGNATURE SEAL (FAIL-CLOSED INVARIANT)
// -------------------------------------------------------------------------

// NonStatutoryRouteAllowlist defines routes explicitly exempted from hardware
// P-256 seals. This is deployment policy, not source truth: prefer loading it
// from configuration. The map must be treated read-only after init.
var NonStatutoryRouteAllowlist = map[string]bool{
	"/api/v1/field/telemetry": true,
	"/api/v1/field/drafts":    true,
	"/api/v1/field/notes":     true,
}

type SubmissionSeal struct {
	DeviceKeyDID  string `json:"device_key_did"`
	TokenID       string `json:"token_id"`
	LeaseEpoch    uint64 `json:"lease_epoch"`
	DataPayload   []byte `json:"data_payload"`
	AppEd25519Sig []byte `json:"app_ed25519_sig"`
	HwP256Sig     []byte `json:"hw_p256_sig"`
}

func (s *SubmissionSeal) DeriveCompositeDigest() []byte {
	h := sha256.New()
	h.Write([]byte(TagSubmissionSeal))
	h.Write([]byte(s.DeviceKeyDID))
	h.Write([]byte(s.TokenID))
	_ = binary.Write(h, binary.BigEndian, s.LeaseEpoch)
	h.Write(s.DataPayload)
	h.Write(s.AppEd25519Sig)
	return h.Sum(nil)
}

// Verify enforces the inverted fail-closed gate: the hardware P-256 seal is
// required on every route except those on the explicit non-statutory
// allowlist. endpointRoute MUST be the matched registered route pattern
// (query strings stripped, path params normalized) — never the raw request
// path — or legitimate allowlisted traffic fails closed on lookup miss.
func (s *SubmissionSeal) Verify(
	endpointRoute string,
	expectedDeviceDID string,
	expectedTokenID string,
	expectedLeaseEpoch uint64,
	appPubKey ed25519.PublicKey,
	hwPubKey *ecdsa.PublicKey,
) error {
	if s.DeviceKeyDID != expectedDeviceDID {
		return fmt.Errorf("%w: device DID mismatch (expected %s, got %s)",
			ErrDomainContextMismatch, expectedDeviceDID, s.DeviceKeyDID)
	}
	if s.TokenID != expectedTokenID || s.LeaseEpoch != expectedLeaseEpoch {
		return fmt.Errorf("%w: token or lease epoch mismatch", ErrDomainContextMismatch)
	}

	// 1. Verify App-level Ed25519 payload signature
	if !ed25519.Verify(appPubKey, s.DataPayload, s.AppEd25519Sig) {
		return fmt.Errorf("%w: invalid app-level Ed25519 signature", ErrInvalidSeal)
	}

	// 2. Inverted Fail-Closed Gate: Require HW P-256 Seal unless on explicit allowlist.
	// Ed25519-alone is NEVER sufficient for regulated payloads.
	if !NonStatutoryRouteAllowlist[endpointRoute] {
		if len(s.HwP256Sig) == 0 || hwPubKey == nil {
			return ErrStatutoryHardwareSealMissing
		}

		compositeDigest := s.DeriveCompositeDigest()
		r, sBig, err := parseP256Signature(s.HwP256Sig)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidSeal, err)
		}

		if !ecdsa.Verify(hwPubKey, compositeDigest, r, sBig) {
			return fmt.Errorf("%w: hardware-rooted P-256 countersignature verification failed", ErrInvalidSeal)
		}
	}

	return nil
}

// parseP256Signature normalizes both platform encodings: 64-byte raw IEEE
// P1363 (R || S) from Android StrongBox / WebCrypto, and ASN.1 DER from Apple
// Secure Enclave / OpenSSL / Go. The multi-%w wrap at call sites preserves
// ErrMalformedP256Sig observability through errors.Is.
func parseP256Signature(sig []byte) (*big.Int, *big.Int, error) {
	if len(sig) == 64 {
		r := new(big.Int).SetBytes(sig[:32])
		s := new(big.Int).SetBytes(sig[32:])
		return r, s, nil
	}

	type dsaSignature struct {
		R, S *big.Int
	}
	var dsaSig dsaSignature
	rest, err := asn1.Unmarshal(sig, &dsaSig)
	if err != nil || len(rest) != 0 || dsaSig.R == nil || dsaSig.S == nil {
		return nil, nil, ErrMalformedP256Sig
	}

	return dsaSig.R, dsaSig.S, nil
}

// -------------------------------------------------------------------------
// 6. IN-CHAIN ESCROW OVERRIDE (BOUND TOMBSTONE SEQUENCE)
// -------------------------------------------------------------------------

type EscrowOverrideRecord struct {
	OverrideID          string        `json:"override_id"`
	SubAssemblyID       string        `json:"sub_assembly_id"`
	LeaseEpoch          uint64        `json:"lease_epoch"`
	TombstoneSequence   uint64        `json:"tombstone_sequence"`
	DraftWorkOrderID    string        `json:"draft_work_order_id"`
	OriginalInspectorID string        `json:"original_inspector_id"`
	TechnicalDirectorID string        `json:"technical_director_id"`
	TdKeyDID            string        `json:"td_key_did"`
	SiteSafetyOfficerID string        `json:"site_safety_officer_id"`
	SsoKeyDID           string        `json:"sso_key_did"`
	RootCause           RootCauseCode `json:"root_cause"`
	TdSignature         []byte        `json:"td_signature"`
	SsoSignature        []byte        `json:"sso_signature"`
	PermanentFlag       bool          `json:"permanent_flag"` // Stamped on final legal cert
}

func (e *EscrowOverrideRecord) CanonicalDigest() []byte {
	h := sha256.New()
	h.Write([]byte(TagEscrowOverride))
	h.Write([]byte(e.OverrideID))
	h.Write([]byte(e.SubAssemblyID))
	_ = binary.Write(h, binary.BigEndian, e.LeaseEpoch)
	_ = binary.Write(h, binary.BigEndian, e.TombstoneSequence)
	h.Write([]byte(e.DraftWorkOrderID))
	h.Write([]byte(e.OriginalInspectorID))
	h.Write([]byte(e.TechnicalDirectorID))
	h.Write([]byte(e.TdKeyDID))
	h.Write([]byte(e.SiteSafetyOfficerID))
	h.Write([]byte(e.SsoKeyDID))
	h.Write([]byte(e.RootCause))
	if e.PermanentFlag {
		h.Write([]byte{1})
	} else {
		h.Write([]byte{0})
	}
	return h.Sum(nil)
}

// ValidateQuorum enforces edge-authoritative escrow release. All elapsed-time
// inputs (tombstoneTimestamp, edgeArrivalTimestamp) MUST originate from the
// edge's local clock and stored tombstone record — never deserialized from
// the claimant's submission — or the SLA reduces to a self-asserted claim.
func (e *EscrowOverrideRecord) ValidateQuorum(
	tdPubKey crypto.PublicKey,
	ssoPubKey crypto.PublicKey,
	configuredSLA time.Duration,
	storedTombstoneSequence uint64,
	tombstoneTimestamp time.Time,
	edgeArrivalTimestamp time.Time,
) error {
	if !e.PermanentFlag {
		return ErrEscrowPermanentFlagRequired
	}

	if e.TombstoneSequence != storedTombstoneSequence {
		return fmt.Errorf("%w: record has seq %d, edge stored seq %d",
			ErrTombstoneStaleSequence, e.TombstoneSequence, storedTombstoneSequence)
	}

	authoritativeElapsed := edgeArrivalTimestamp.Sub(tombstoneTimestamp)
	if authoritativeElapsed < configuredSLA {
		return fmt.Errorf("escrow: authoritative SLA threshold (%s) not reached; elapsed was %s",
			configuredSLA, authoritativeElapsed)
	}

	if e.TdKeyDID == "" || len(e.TdSignature) == 0 {
		return errors.New("escrow: technical director signature or DID missing")
	}

	if e.SsoKeyDID == "" || len(e.SsoSignature) == 0 {
		return errors.New("escrow: site safety officer signature or DID missing")
	}

	digest := e.CanonicalDigest()

	if err := verifySignature(tdPubKey, digest, e.TdSignature); err != nil {
		return fmt.Errorf("escrow: invalid TD signature: %w", err)
	}

	if err := verifySignature(ssoPubKey, digest, e.SsoSignature); err != nil {
		return fmt.Errorf("escrow: invalid SSO signature: %w", err)
	}

	return nil
}

func verifySignature(pub crypto.PublicKey, digest, sig []byte) error {
	switch k := pub.(type) {
	case ed25519.PublicKey:
		if !ed25519.Verify(k, digest, sig) {
			return errors.New("ed25519 verification failed")
		}
	case *ecdsa.PublicKey:
		r, s, err := parseP256Signature(sig)
		if err != nil {
			return err
		}
		if !ecdsa.Verify(k, digest, r, s) {
			return errors.New("ecdsa verification failed")
		}
	default:
		return errors.New("unsupported public key type")
	}
	return nil
}

// VerifySignature exports the normalized signature verifier shared across
// packages: Ed25519, raw 64-byte IEEE P1363 P-256, and ASN.1 DER P-256 are
// handled uniformly so no caller forks its own ECDSA branch.
func VerifySignature(pub crypto.PublicKey, digest, sig []byte) error {
	return verifySignature(pub, digest, sig)
}

// MerkleOverrideEvent is the immutable audit record appended per committed
// override. AuthorityEpoch (chain high-water at commit) and LeaseEpoch (the
// tombstone generation the override binds) are tracked independently:
// conflating them mislabels evidence in a legal log.
type MerkleOverrideEvent struct {
	EventID        string               `json:"event_id"`
	Timestamp      time.Time            `json:"timestamp"`
	EventType      string               `json:"event_type"` // "LEASE_OVERRUN_ADMIN_OVERRIDE"
	AuthorityEpoch uint64               `json:"authority_epoch"`
	LeaseEpoch     uint64               `json:"lease_epoch"`
	PayloadHash    [32]byte             `json:"payload_hash"`
	OverrideEscrow EscrowOverrideRecord `json:"override_escrow"`
	PrevRecordHash [32]byte             `json:"prev_record_hash"`
}
