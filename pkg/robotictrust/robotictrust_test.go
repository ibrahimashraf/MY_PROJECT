package robotictrust

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func validTestClaim() RoboticAttestationClaim {
	fwHash := sha256.Sum256([]byte("firmware-v2.4.1-signed"))
	return RoboticAttestationClaim{
		PlatformID:        "spot-rig-alpha-04",
		PlatformType:      PlatformQuadruped,
		Manufacturer:      "Boston Dynamics",
		Model:             "Spot Enterprise C1D1",
		SerialNumber:      "BD-SPOT-2026-991",
		FirmwareDigest:    hex.EncodeToString(fwHash[:]),
		TPMQuoteSignature: "MEYCIQDx8...tpm_pcr_quote_sig...",
		EnclavePublicKey:  "ed25519-public-key-hex",
		AttestationTime:   time.Now().UTC(),
		FIPS140_3Level:    "Level 3",
	}
}

func TestValidateRoboticSubmission_Success(t *testing.T) {
	claim := validTestClaim()
	keyframeHash := sha256.Sum256([]byte("visual-keyframe-001"))
	pcdHash := sha256.Sum256([]byte("pointcloud-lidar-001"))

	receipt := RoboticInspectionReceipt{
		ReceiptID:        "rcpt-robot-001",
		WorkOrderID:      "wo-2026-offshore-77",
		AssetDID:         "did:integin:asset:crane-pedestal-01",
		TenantID:         "tenant-aramco-offshore",
		OrganizationID:   "org-tic-global",
		AttestationClaim: claim,
		Envelope: ExecutionEnvelope{
			MinAltitudeMeters: 0.0,
			MaxAltitudeMeters: 45.0,
			MaxVelocityMps:    1.6,
			AllowedFirmwares:  []string{claim.FirmwareDigest},
		},
		TelemetryFrames: []SpatialFrameProof{
			{
				SequenceNo:       1,
				Timestamp:        time.Now().UTC(),
				AltitudeMeters:   12.4,
				VelocityMps:      0.8,
				PointcloudSHA256: hex.EncodeToString(pcdHash[:]),
				KeyframeSHA256:   hex.EncodeToString(keyframeHash[:]),
			},
		},
		CompletedAt: time.Now().UTC(),
	}

	if err := ValidateRoboticSubmission(receipt); err != nil {
		t.Fatalf("expected valid robotic submission, got: %v", err)
	}
}

func TestValidateRoboticSubmission_FirmwareRejection(t *testing.T) {
	claim := validTestClaim()
	receipt := RoboticInspectionReceipt{
		AttestationClaim: claim,
		Envelope: ExecutionEnvelope{
			MinAltitudeMeters: 0.0,
			MaxAltitudeMeters: 50.0,
			MaxVelocityMps:    2.0,
			AllowedFirmwares:  []string{"0000000000000000000000000000000000000000000000000000000000000000"},
		},
		TelemetryFrames: []SpatialFrameProof{
			{
				SequenceNo:     1,
				AltitudeMeters: 10.0,
				VelocityMps:    1.0,
				KeyframeSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
	}

	err := ValidateRoboticSubmission(receipt)
	if err == nil || !strings.Contains(err.Error(), "firmware") {
		t.Fatalf("expected firmware rejection, got: %v", err)
	}
}

func TestValidateRoboticSubmission_AltitudeVelocityViolation(t *testing.T) {
	claim := validTestClaim()
	receipt := RoboticInspectionReceipt{
		AttestationClaim: claim,
		Envelope: ExecutionEnvelope{
			MinAltitudeMeters: 0.0,
			MaxAltitudeMeters: 20.0,
			MaxVelocityMps:    1.5,
			AllowedFirmwares:  []string{claim.FirmwareDigest},
		},
		TelemetryFrames: []SpatialFrameProof{
			{
				SequenceNo:     1,
				AltitudeMeters: 25.0, // Exceeds 20.0m
				VelocityMps:    1.0,
				KeyframeSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
		},
	}

	if err := ValidateRoboticSubmission(receipt); err == nil || !strings.Contains(err.Error(), "altitude") {
		t.Fatalf("expected altitude breach error, got: %v", err)
	}

	// Test velocity breach
	receipt.TelemetryFrames[0].AltitudeMeters = 15.0
	receipt.TelemetryFrames[0].VelocityMps = 3.2 // Exceeds 1.5 m/s

	if err := ValidateRoboticSubmission(receipt); err == nil || !strings.Contains(err.Error(), "velocity") {
		t.Fatalf("expected velocity breach error, got: %v", err)
	}
}

func BenchmarkValidateRoboticSubmission_HotPath(b *testing.B) {
	claim := validTestClaim()
	receipt := RoboticInspectionReceipt{
		AttestationClaim: claim,
		Envelope: ExecutionEnvelope{
			MinAltitudeMeters: 0.0,
			MaxAltitudeMeters: 100.0,
			MaxVelocityMps:    5.0,
			AllowedFirmwares:  []string{claim.FirmwareDigest},
		},
		TelemetryFrames: []SpatialFrameProof{
			{
				SequenceNo:     1,
				AltitudeMeters: 10.0,
				VelocityMps:    1.0,
				KeyframeSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ValidateRoboticSubmission(receipt); err != nil {
			b.Fatal(err)
		}
	}
}

// Ensure interface compatibility with crypto signature structures
var _ = ed25519.PrivateKeySize
