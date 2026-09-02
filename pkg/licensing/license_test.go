package licensing

import (
	"testing"
	"time"
)

func TestMasterLicenseIssuanceAndOfflineValidation(t *testing.T) {
	// 1. Generate Root Keypair
	_, priv, err := GenerateMasterKeyPair()
	if err != nil {
		t.Fatalf("GenerateMasterKeyPair error: %v", err)
	}

	issuer, err := NewMasterLicenseIssuer(priv)
	if err != nil {
		t.Fatalf("NewMasterLicenseIssuer error: %v", err)
	}

	validator, err := NewUniversalLicenseValidator(issuer.publicKeyHex())
	if err != nil {
		t.Fatalf("NewUniversalLicenseValidator error: %v", err)
	}

	now := time.Now().UTC()
	payload := LicensePayload{
		LicenseID:   "LIC-INTEGIN-2026-001",
		Issuer:      "did:integin:authority:master-pki",
		IssuedToOrg: "Global Energy & Inspection Ltd",
		Tier:        TierSovereign,
		Mode:        ModeAirGapped,
		IssuedAt:    now,
		NotBefore:   now.Add(-1 * time.Hour),
		ExpiresAt:   now.Add(365 * 24 * time.Hour),
		HardwareLockID: "HW-CHASSIS-SERVER-9912",
		Covenants: PlatformCovenants{
			MaxTenants:             50,
			MaxInspectorsPerTenant: 500,
			AllowedJurisdictions:   []string{"SA", "AE", "US", "DE"},
			EnabledModules:         []string{"LIFTING", "NDT", "PRESSURE", "OFFSHORE"},
			AirGapOfflineGraceDays: 180,
		},
	}

	// 2. Mint Signed License Token
	token, err := issuer.IssueLicense(payload)
	if err != nil {
		t.Fatalf("IssueLicense error: %v", err)
	}

	// 3. Validate with correct Hardware ID
	if err := validator.ValidateToken(token, "HW-CHASSIS-SERVER-9912"); err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}

	// 4. Validate with wrong Hardware ID (Must Fail)
	if err := validator.ValidateToken(token, "HW-WRONG-SERVER-1234"); err != ErrHardwareMismatch {
		t.Errorf("expected ErrHardwareMismatch, got %v", err)
	}

	// 5. Tamper with payload (Must fail signature check)
	tamperedToken := *token
	tamperedToken.Payload.Covenants.MaxInspectorsPerTenant = 999999
	if err := validator.ValidateToken(&tamperedToken, "HW-CHASSIS-SERVER-9912"); err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature for tampered token, got %v", err)
	}

	// 6. Check Jurisdiction Scope
	if err := validator.CheckJurisdictionAllowed(token, "SA"); err != nil {
		t.Errorf("expected SA to be allowed, got error: %v", err)
	}
	if err := validator.CheckJurisdictionAllowed(token, "JP"); err != ErrJurisdictionDenied {
		t.Errorf("expected JP to be denied, got: %v", err)
	}
}

func (issuer *MasterLicenseIssuer) publicKeyHex() string {
	return hexEncode(issuer.publicKey)
}

func hexEncode(b []byte) string {
	return string(hexEncodeBytes(b))
}

func hexEncodeBytes(src []byte) []byte {
	dst := make([]byte, len(src)*2)
	hexDump := "0123456789abcdef"
	for i, v := range src {
		dst[i*2] = hexDump[v>>4]
		dst[i*2+1] = hexDump[v&0x0f]
	}
	return dst
}
