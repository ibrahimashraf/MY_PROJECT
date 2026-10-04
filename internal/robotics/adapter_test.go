package robotics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"integin/pkg/robotictrust"
)

func TestIngressService_IngestRoboticInspection(t *testing.T) {
	service := NewIngressService()
	fwHash := sha256.Sum256([]byte("firmware-drone-signed-001"))
	pcdHash := sha256.Sum256([]byte("pcd-001"))
	keyframeHash := sha256.Sum256([]byte("frame-001"))

	receipt := &robotictrust.RoboticInspectionReceipt{
		ReceiptID:      "rcpt-drone-99",
		WorkOrderID:    "wo-flare-stack-11",
		AssetDID:       "did:integin:asset:flare-tip-01",
		TenantID:       "tenant-shell-rig",
		OrganizationID: "org-tic-middleeast",
		AttestationClaim: robotictrust.RoboticAttestationClaim{
			PlatformID:        "drone-skydio-x2d",
			PlatformType:      robotictrust.PlatformAerialUAV,
			Manufacturer:      "Skydio",
			Model:             "X2D Autonomous C1D1",
			SerialNumber:      "SK-2026-X2D-044",
			FirmwareDigest:    hex.EncodeToString(fwHash[:]),
			TPMQuoteSignature: "SIG_TPM2_SECURE_ENCLAVE_QUOTE",
			AttestationTime:   time.Now().UTC(),
			FIPS140_3Level:    "Level 3",
		},
		Envelope: robotictrust.ExecutionEnvelope{
			MinAltitudeMeters: 5.0,
			MaxAltitudeMeters: 80.0,
			MaxVelocityMps:    8.0,
			AllowedFirmwares:  []string{hex.EncodeToString(fwHash[:])},
		},
		TelemetryFrames: []robotictrust.SpatialFrameProof{
			{
				SequenceNo:       1,
				Timestamp:        time.Now().UTC(),
				AltitudeMeters:   62.5,
				VelocityMps:      3.4,
				PointcloudSHA256: hex.EncodeToString(pcdHash[:]),
				KeyframeSHA256:   hex.EncodeToString(keyframeHash[:]),
			},
		},
		CompletedAt:    time.Now().UTC(),
		RobotSignature: "ROBOT_SIG_ED25519_ATTESTED",
	}

	// 1. Success case
	coreReceipt, err := service.IngestRoboticInspection(context.Background(), "tenant-shell-rig", "org-tic-middleeast", receipt)
	if err != nil {
		t.Fatalf("unexpected ingress failure: %v", err)
	}
	if coreReceipt.ReceiptID != "rcpt-drone-99" {
		t.Errorf("expected receipt id rcpt-drone-99, got: %s", coreReceipt.ReceiptID)
	}
	if coreReceipt.OverallResult != "PASS_AUTONOMOUS_ATTESTED" {
		t.Errorf("unexpected overall result: %s", coreReceipt.OverallResult)
	}

	// 2. Tenant isolation boundary violation
	_, err = service.IngestRoboticInspection(context.Background(), "attacker-tenant", "org-tic-middleeast", receipt)
	if err == nil {
		t.Fatalf("expected tenant mismatch error, got nil")
	}

	// 3. Nil receipt handling
	_, err = service.IngestRoboticInspection(context.Background(), "tenant-shell-rig", "org-tic-middleeast", nil)
	if err == nil {
		t.Fatalf("expected nil receipt error, got nil")
	}
}
