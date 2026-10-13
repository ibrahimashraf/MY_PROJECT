package domain_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	qrcode "github.com/skip2/go-qrcode"
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

func TestAssetPassport_QRScannerOpticalDensity(t *testing.T) {
	passport := domain.UniversalAssetPassport{
		AssetDID:          "did:integin:asset:8f4b1e2c-3a5d-4f6e-9c7b-1a2b3c4d5e6f",
		Manufacturer:      "Tadano Faun GmbH",
		ModelNumber:       "ATF-400G-6",
		ChassisSerial:     "SN-DE-2026-994821",
		EquipmentCategory: "ALL_TERRAIN_CRANE",
		RatedSWL:          "400 Ton",
		CurrentStatus:     domain.AssetStatusOperational,
		JurisdictionCode:  "SA",
		RegisteredAt:      time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC),
		Epoch:             1,
		CustomAttributes: map[string]interface{}{
			"inspection_authority": "SASO",
			"calibration_status":   "VALID",
		},
	}

	// 1. Benchmark CBOR bytes vs raw JSON bytes
	cborBytes, err := passport.ToCBOR()
	if err != nil {
		t.Fatalf("ToCBOR failed: %v", err)
	}

	jsonBytes, err := json.Marshal(passport)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	if len(cborBytes) >= len(jsonBytes) {
		t.Fatalf("CBOR (%d bytes) must be strictly smaller than JSON (%d bytes)", len(cborBytes), len(jsonBytes))
	}

	// 2. Generate QR code with industrial error correction level (Medium)
	qr, err := qrcode.New(string(cborBytes), qrcode.Medium)
	if err != nil {
		t.Fatalf("qrcode.New failed: %v", err)
	}

	// 3. Verify optical matrix dimensions meet handheld industrial scanner criteria (ISO/IEC 18004 <= 90x90)
	bitmap := qr.Bitmap()
	moduleWidth := len(bitmap)
	if moduleWidth > 90 {
		t.Fatalf("QR module matrix size (%dx%d) exceeds optical limit (<=90x90) for rugged field scanning", moduleWidth, moduleWidth)
	}

	// 4. Verify valid PNG rasterization with PNG magic bytes
	pngBytes, err := qr.PNG(256)
	if err != nil {
		t.Fatalf("qr.PNG failed: %v", err)
	}
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if !bytes.HasPrefix(pngBytes, pngMagic) {
		t.Fatal("generated image missing standard PNG magic header")
	}

	// 5. Verify roundtrip decoding from original payload
	roundtripPassport, err := domain.FromCBOR(cborBytes)
	if err != nil {
		t.Fatalf("roundtrip FromCBOR failed: %v", err)
	}
	if roundtripPassport.AssetDID != passport.AssetDID {
		t.Fatalf("DID mismatch: got %s, want %s", roundtripPassport.AssetDID, passport.AssetDID)
	}
}
