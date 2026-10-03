package domain_test

import (
	"testing"
	"time"

	"integin/pkg/domain"
)

func TestAssetPassport_CBOR(t *testing.T) {
	passport := domain.UniversalAssetPassport{
		AssetDID:          "did:integin:asset:test-1234",
		Manufacturer:      "Liebherr",
		ModelNumber:       "LTM-11200",
		ChassisSerial:     "SN-998877",
		EquipmentCategory: "MOBILE_CRANE",
		RatedSWL:          "1200 Ton",
		CurrentStatus:     domain.AssetStatusOperational,
		JurisdictionCode:  "SA",
		RegisteredAt:      time.Now().UTC().Truncate(time.Second),
	}

	data, err := passport.ToCBOR()
	if err != nil {
		t.Fatalf("ToCBOR failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty CBOR bytes")
	}

	decoded, err := domain.FromCBOR(data)
	if err != nil {
		t.Fatalf("FromCBOR failed: %v", err)
	}

	if decoded.AssetDID != passport.AssetDID {
		t.Errorf("expected DID %s, got %s", passport.AssetDID, decoded.AssetDID)
	}
	if decoded.ChassisSerial != passport.ChassisSerial {
		t.Errorf("expected serial %s, got %s", passport.ChassisSerial, decoded.ChassisSerial)
	}
}
