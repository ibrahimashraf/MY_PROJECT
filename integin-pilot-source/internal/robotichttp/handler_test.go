package robotichttp_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"integin/internal/robotichttp"
	"integin/internal/robotics"
	"integin/pkg/onboarding"
	"integin/pkg/robotictrust"
)

func TestRoboticHTTP_ServeHTTP(t *testing.T) {
	service := robotics.NewIngressService()
	handler := robotichttp.NewHandler(service)

	fwHash := sha256.Sum256([]byte("fw-001"))
	pcdHash := sha256.Sum256([]byte("pcd-001"))
	keyframeHash := sha256.Sum256([]byte("frame-001"))

	receipt := robotictrust.RoboticInspectionReceipt{
		ReceiptID:      "rcpt-1",
		WorkOrderID:    "wo-1",
		AssetDID:       "did:integin:asset:1",
		TenantID:       "t-1",
		OrganizationID: "o-1",
		AttestationClaim: robotictrust.RoboticAttestationClaim{
			PlatformID:        "spot-1",
			PlatformType:      robotictrust.PlatformQuadruped,
			Manufacturer:      "Boston Dynamics",
			Model:             "Spot Enterprise",
			SerialNumber:      "SPOT-2026-99",
			FirmwareDigest:    hex.EncodeToString(fwHash[:]),
			TPMQuoteSignature: "SIG_TPM2_QUOTE",
			AttestationTime:   time.Now().UTC(),
			FIPS140_3Level:    "Level 3",
		},
		Envelope: robotictrust.ExecutionEnvelope{
			MinAltitudeMeters: 0,
			MaxAltitudeMeters: 5,
			MaxVelocityMps:    2.0,
			AllowedFirmwares:  []string{hex.EncodeToString(fwHash[:])},
		},
		TelemetryFrames: []robotictrust.SpatialFrameProof{
			{
				SequenceNo:       1,
				Timestamp:        time.Now().UTC(),
				AltitudeMeters:   1.0,
				VelocityMps:      0.5,
				PointcloudSHA256: hex.EncodeToString(pcdHash[:]),
				KeyframeSHA256:   hex.EncodeToString(keyframeHash[:]),
			},
		},
		CompletedAt:    time.Now().UTC(),
		RobotSignature: "ROBOT_SIG",
	}

	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// 1. Success
	req := httptest.NewRequest(http.MethodPost, "/api/v1/robotics/ingress", bytes.NewReader(raw))
	req.Header.Set("X-Tenant-ID", "t-1")
	req.Header.Set("X-Organization-ID", "o-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res onboarding.SignedInspectionReceipt
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.ReceiptID != "rcpt-1" {
		t.Errorf("expected receipt rcpt-1, got %s", res.ReceiptID)
	}

	// 2. Missing headers
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/robotics/ingress", bytes.NewReader(raw))
	recBad := httptest.NewRecorder()
	handler.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", recBad.Code)
	}

	// 3. Method not allowed
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/robotics/ingress", nil)
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", recGet.Code)
	}
}
