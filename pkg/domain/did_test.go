package domain

import (
	"testing"
	"time"
)

func TestParseDID(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantType   string
		wantID     string
		wantErr    bool
	}{
		{
			name:     "valid asset did",
			raw:      "did:integin:asset:a8f9c012e45b6789",
			wantType: "asset",
			wantID:   "a8f9c012e45b6789",
			wantErr:  false,
		},
		{
			name:     "valid standard did",
			raw:      "did:integin:standard:ASME_B30_5_2024",
			wantType: "standard",
			wantID:   "ASME_B30_5_2024",
			wantErr:  false,
		},
		{
			name:    "invalid prefix",
			raw:     "did:other:asset:12345",
			wantErr: true,
		},
		{
			name:    "malformed components",
			raw:     "did:integin:asset",
			wantErr: true,
		},
		{
			name:    "empty identifier",
			raw:     "did:integin:asset:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDID(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseDID(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if !tt.wantErr {
				if got.Type != tt.wantType {
					t.Errorf("got Type = %v, want %v", got.Type, tt.wantType)
				}
				if got.Identifier != tt.wantID {
					t.Errorf("got Identifier = %v, want %v", got.Identifier, tt.wantID)
				}
				if got.String() != tt.raw {
					t.Errorf("String() = %v, want %v", got.String(), tt.raw)
				}
			}
		})
	}
}

func TestGenerateAssetDIDDeterminism(t *testing.T) {
	did1 := GenerateAssetDID("Liebherr", "LTM-11200", "SN-998844", 1700000000)
	did2 := GenerateAssetDID("LIEBHERR", "ltm-11200", "sn-998844", 1700000000)
	did3 := GenerateAssetDID("Liebherr", "LTM-11200", "SN-998845", 1700000000)

	if did1.String() != did2.String() {
		t.Errorf("DIDs should match regardless of casing: %s vs %s", did1, did2)
	}

	if did1.String() == did3.String() {
		t.Errorf("Different serials must produce distinct DIDs: %s vs %s", did1, did3)
	}
}

func TestUniversalAssetPassportLifecycle(t *testing.T) {
	did := GenerateAssetDID("Kato", "NK-1000", "RAS-MU-25053043", 1700000000)
	passport := UniversalAssetPassport{
		AssetDID:          did.String(),
		Manufacturer:      "Kato",
		ModelNumber:       "NK-1000",
		ChassisSerial:     "RAS-MU-25053043",
		EquipmentCategory: "MOBILE_CRANE",
		RatedSWL:          "100 Ton",
		CurrentStatus:     AssetStatusOperational,
		JurisdictionCode:  "SA",
		RegisteredAt:      time.Now().UTC(),
	}

	if err := passport.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Test quarantine
	passport.Quarantine("Excessive broken wires on main hoist rope")
	if passport.CurrentStatus != AssetStatusQuarantined {
		t.Errorf("expected status QUARANTINED, got %s", passport.CurrentStatus)
	}
	if passport.CustomAttributes["quarantine_reason"] == "" {
		t.Errorf("expected quarantine reason to be recorded")
	}

	// Test custody transfer
	transfer := CustodyTransferEvent{
		EventID:          "EV-9901",
		PreviousTenantID: "tenant-saudi-contractor",
		NewTenantID:      "tenant-uae-offshore",
		CountryCodeISO2:  "AE",
		TransferDate:     time.Now().UTC(),
		AuthorizedBy:     "QA-Director-Ahmed",
		ReceiptHash:      "hash-123456",
	}

	if err := passport.TransferCustody(transfer); err != nil {
		t.Fatalf("transfer custody failed: %v", err)
	}

	if passport.JurisdictionCode != "AE" {
		t.Errorf("expected jurisdiction code AE, got %s", passport.JurisdictionCode)
	}
	if len(passport.ChainOfCustody) != 1 {
		t.Errorf("expected 1 custody event, got %d", len(passport.ChainOfCustody))
	}
}
