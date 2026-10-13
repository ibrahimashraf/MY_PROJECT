package licensing

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRevocationCacheRevokedLicenseFailClosed(t *testing.T) {
	now := time.Now().UTC()
	cache := NewRevocationCache()
	err := cache.Load([]RevocationEntry{
		{
			LicenseID: "LIC-REVOKED-001",
			Reason:    "server compromised",
			RevokedAt: now.Add(-2 * time.Hour),
			ExpiresAt: now.Add(2 * time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	err = cache.CheckLicense("LIC-REVOKED-001", now)
	if !errors.Is(err, ErrLicenseRevoked) {
		t.Fatalf("expected ErrLicenseRevoked, got %v", err)
	}
	var revoked *RevokedLicenseError
	if !errors.As(err, &revoked) {
		t.Fatalf("expected RevokedLicenseError, got %T", err)
	}
	if revoked.LicenseID != "LIC-REVOKED-001" || revoked.Reason != "server compromised" {
		t.Errorf("unexpected revocation details: %+v", revoked)
	}

	if err := cache.CheckLicense("LIC-UNKNOWN-999", now); err != nil {
		t.Errorf("unknown license should pass, got %v", err)
	}
}

func TestRevocationCacheExpiredEntryFailClosed(t *testing.T) {
	now := time.Now().UTC()
	cache := NewRevocationCache()
	if err := cache.Load([]RevocationEntry{
		{
			LicenseID: "LIC-STALE-002",
			Reason:    "suspected abuse",
			RevokedAt: now.Add(-48 * time.Hour),
			ExpiresAt: now.Add(-1 * time.Hour),
		},
	}); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if err := cache.CheckLicense("LIC-STALE-002", now); err != ErrRevocationCacheExpired {
		t.Errorf("expected ErrRevocationCacheExpired, got %v", err)
	}
}

func TestRevocationCacheRejectsInvalidAndDuplicateLoad(t *testing.T) {
	now := time.Now().UTC()
	cache := NewRevocationCache()

	if err := cache.Load([]RevocationEntry{
		{LicenseID: "", Reason: "empty id", RevokedAt: now, ExpiresAt: now.Add(time.Hour)},
	}); !errors.Is(err, ErrInvalidRevocationEntry) {
		t.Errorf("empty license_id should be rejected, got %v", err)
	}

	if err := cache.Load([]RevocationEntry{
		{LicenseID: "LIC-DUP", Reason: "a", RevokedAt: now, ExpiresAt: now.Add(time.Hour)},
		{LicenseID: "LIC-DUP", Reason: "b", RevokedAt: now, ExpiresAt: now.Add(time.Hour)},
	}); !errors.Is(err, ErrInvalidRevocationEntry) {
		t.Errorf("duplicate license_id should be rejected, got %v", err)
	}

	if err := cache.Load([]RevocationEntry{
		{LicenseID: "LIC-BAD-WINDOW", Reason: "x", RevokedAt: now.Add(time.Hour), ExpiresAt: now},
	}); err == nil {
		t.Errorf("expires_at before revoked_at should be rejected")
	}
}

func TestValidatorValidateWithRevocation(t *testing.T) {
	_, priv, _ := GenerateMasterKeyPair()
	issuer, err := NewMasterLicenseIssuer(priv)
	if err != nil {
		t.Fatalf("NewMasterLicenseIssuer: %v", err)
	}
	validator, err := NewUniversalLicenseValidator(hexEncode(issuer.publicKey))
	if err != nil {
		t.Fatalf("NewUniversalLicenseValidator: %v", err)
	}

	now := time.Now().UTC()
	token, err := issuer.IssueLicense(LicensePayload{
		LicenseID: "LIC-GATE-003",
		Issuer:    "did:integin:authority:master-pki",
		Tier:      TierEnterprise,
		Mode:      ModeCloudSaaS,
		IssuedAt:  now,
		NotBefore: now.Add(-time.Hour),
		ExpiresAt: now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("IssueLicense: %v", err)
	}

	cache := NewRevocationCache()
	if err := cache.Load([]RevocationEntry{
		{LicenseID: "LIC-GATE-003", Reason: "non-payment", RevokedAt: now, ExpiresAt: now.Add(time.Hour)},
	}); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := validator.ValidateWithRevocation(token, "", cache); !errors.Is(err, ErrLicenseRevoked) {
		t.Errorf("revoked token must be denied, got %v", err)
	}

	cleanCache := NewRevocationCache()
	if err := validator.ValidateWithRevocation(token, "", cleanCache); err != nil {
		t.Errorf("non-revoked token should pass, got %v", err)
	}

	if err := validator.ValidateWithRevocation(token, "", nil); err != ErrRevocationCacheExpired {
		t.Errorf("nil cache must fail closed, got %v", err)
	}
}

func thresholdTokenForTest(t *testing.T, now time.Time, issuers []*MasterLicenseIssuer) *SignedLicenseToken {
	t.Helper()
	token, err := issuers[0].IssueLicense(LicensePayload{
		LicenseID: "LIC-QUORUM-007",
		Issuer:    "did:integin:authority:vendor-ring",
		Tier:      TierEnterprise,
		Mode:      ModeCloudSaaS,
		IssuedAt:  now,
		NotBefore: now.Add(-time.Hour),
		ExpiresAt: now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("IssueLicense: %v", err)
	}
	return token
}

func TestMOfNValidator2Of3Accepts(t *testing.T) {
	now := time.Now().UTC()
	issuers := make([]*MasterLicenseIssuer, 3)
	hexKeys := make([]string, 3)
	for i := range issuers {
		_, priv, err := GenerateMasterKeyPair()
		if err != nil {
			t.Fatalf("GenerateMasterKeyPair: %v", err)
		}
		issuer, err := NewMasterLicenseIssuer(priv)
		if err != nil {
			t.Fatalf("NewMasterLicenseIssuer: %v", err)
		}
		issuers[i] = issuer
		hexKeys[i] = hexEncode(issuer.publicKey)
	}

	validator, err := NewMOfNValidator(hexKeys, 2)
	if err != nil {
		t.Fatalf("NewMOfNValidator: %v", err)
	}
	if validator.Threshold() != 2 || validator.RingSize() != 3 {
		t.Errorf("unexpected config: m=%d n=%d", validator.Threshold(), validator.RingSize())
	}

	token := thresholdTokenForTest(t, now, issuers)
	if err := validator.ValidateToken(token, ""); err != ErrQuorumNotMet {
		t.Errorf("single signature must not meet 2-of-3, got %v", err)
	}

	if err := issuers[1].AddSignoff(token); err != nil {
		t.Fatalf("AddSignoff: %v", err)
	}
	if err := validator.ValidateToken(token, ""); err != nil {
		t.Errorf("2-of-3 quorum should be accepted, got %v", err)
	}
}

func TestMOfNValidator1Of3Fails(t *testing.T) {
	now := time.Now().UTC()
	issuers := make([]*MasterLicenseIssuer, 3)
	hexKeys := make([]string, 3)
	for i := range issuers {
		_, priv, _ := GenerateMasterKeyPair()
		issuer, _ := NewMasterLicenseIssuer(priv)
		issuers[i] = issuer
		hexKeys[i] = hexEncode(issuer.publicKey)
	}

	validator, err := NewMOfNValidator(hexKeys, 2)
	if err != nil {
		t.Fatalf("NewMOfNValidator: %v", err)
	}
	token := thresholdTokenForTest(t, now, issuers)
	if err := validator.ValidateToken(token, ""); err != ErrQuorumNotMet {
		t.Errorf("1-of-3 must fail with ErrQuorumNotMet, got %v", err)
	}
}

func TestMOfNValidatorTamperedSignatureFails(t *testing.T) {
	now := time.Now().UTC()
	issuers := make([]*MasterLicenseIssuer, 3)
	hexKeys := make([]string, 3)
	for i := range issuers {
		_, priv, _ := GenerateMasterKeyPair()
		issuer, _ := NewMasterLicenseIssuer(priv)
		issuers[i] = issuer
		hexKeys[i] = hexEncode(issuer.publicKey)
	}

	validator, err := NewMOfNValidator(hexKeys, 2)
	if err != nil {
		t.Fatalf("NewMOfNValidator: %v", err)
	}

	token := thresholdTokenForTest(t, now, issuers)
	if err := issuers[1].AddSignoff(token); err != nil {
		t.Fatalf("AddSignoff: %v", err)
	}

	tampered := *token
	tampered.Payload.LicenseID = "LIC-EVIL-999"
	if err := validator.ValidateToken(&tampered, ""); err == nil {
		t.Errorf("tampered payload must fail quorum verification")
	}

	tamperedSig := *token
	tamperedSig.Signoffs[0] = strings.Repeat("0", 128)
	if err := validator.ValidateToken(&tamperedSig, ""); err == nil {
		t.Errorf("tampered signoff must fail")
	}
}

func TestMOfNValidatorConfigValidation(t *testing.T) {
	_, priv, _ := GenerateMasterKeyPair()
	issuer, _ := NewMasterLicenseIssuer(priv)
	key := hexEncode(issuer.publicKey)

	if _, err := NewMOfNValidator([]string{key}, 2); err != ErrThresholdExceedsRing {
		t.Errorf("m > n must fail with ErrThresholdExceedsRing, got %v", err)
	}
	if _, err := NewMOfNValidator(nil, 1); !errors.Is(err, ErrInvalidThreshold) {
		t.Errorf("empty ring must fail, got %v", err)
	}
	if _, err := NewMOfNValidator([]string{key}, 0); !errors.Is(err, ErrInvalidThreshold) {
		t.Errorf("m=0 must fail, got %v", err)
	}
	if _, err := NewMOfNValidator([]string{key, key}, 1); err == nil {
		t.Errorf("duplicate ring key must be rejected")
	}
	if _, err := NewMOfNValidator([]string{"zz-not-hex", key}, 1); err == nil {
		t.Errorf("bad hex key must be rejected")
	}
}

func TestMOfNValidatorRevocationIntegration(t *testing.T) {
	now := time.Now().UTC()
	var issuers []*MasterLicenseIssuer
	hexKeys := make([]string, 3)
	for i := range 3 {
		_, priv, _ := GenerateMasterKeyPair()
		issuer, _ := NewMasterLicenseIssuer(priv)
		issuers = append(issuers, issuer)
		hexKeys[i] = hexEncode(issuer.publicKey)
	}
	validator, err := NewMOfNValidator(hexKeys, 2)
	if err != nil {
		t.Fatalf("NewMOfNValidator: %v", err)
	}
	token := thresholdTokenForTest(t, now, issuers)
	if err := issuers[1].AddSignoff(token); err != nil {
		t.Fatalf("AddSignoff: %v", err)
	}

	cache := NewRevocationCache()
	if err := cache.Load([]RevocationEntry{
		{LicenseID: "LIC-QUORUM-007", Reason: "fraud", RevokedAt: now, ExpiresAt: now.Add(time.Hour)},
	}); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := validator.ValidateWithRevocation(token, "", cache); !errors.Is(err, ErrLicenseRevoked) {
		t.Errorf("revoked quorum token must be denied, got %v", err)
	}

	// Snapshot determinism.
	snap := cache.Snapshot()
	if len(snap) != 1 || snap[0].LicenseID != "LIC-QUORUM-007" {
		t.Errorf("unexpected snapshot: %+v", snap)
	}
}
