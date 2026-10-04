package robotics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"integin/pkg/onboarding"
	"integin/pkg/robotictrust"
)

var (
	ErrNilReceipt           = errors.New("robotics: robotic inspection receipt cannot be nil")
	ErrReceiptTenantMismatch = errors.New("robotics: receipt tenant mismatch with execution context")
)

// IngressService handles the ingestion and adaptation of autonomous robotic
// inspection packages into the core INTEGIN workflow and ledger events.
type IngressService struct {
	// Storage / repo interfaces are injected at server composition time
}

func NewIngressService() *IngressService {
	return &IngressService{}
}

// IngestRoboticInspection validates the robotic hardware attestation and execution
// envelope, returning a normalized core SignedInspectionReceipt.
func (s *IngressService) IngestRoboticInspection(ctx context.Context, tenantID, orgID string, r *robotictrust.RoboticInspectionReceipt) (*onboarding.SignedInspectionReceipt, error) {
	if r == nil {
		return nil, ErrNilReceipt
	}
	if r.TenantID != tenantID || r.OrganizationID != orgID {
		return nil, fmt.Errorf("%w: expected (%s, %s), got (%s, %s)",
			ErrReceiptTenantMismatch, tenantID, orgID, r.TenantID, r.OrganizationID)
	}

	// 1. Enforce fail-closed validation of robot hardware TPM attestation & 4D envelope
	if err := robotictrust.ValidateRoboticSubmission(*r); err != nil {
		return nil, fmt.Errorf("robotic validation failed: %w", err)
	}

	// 2. Synthesize composite payload digest over keyframes and telemetry
	hasher := sha256.New()
	for _, frame := range r.TelemetryFrames {
		hasher.Write([]byte(fmt.Sprintf("%d:%s:%s", frame.SequenceNo, frame.PointcloudSHA256, frame.KeyframeSHA256)))
	}
	compositeTelemetryDigest := hex.EncodeToString(hasher.Sum(nil))

	// 3. Adapt into core SignedInspectionReceipt for the ledger pipeline
	coreReceipt := &onboarding.SignedInspectionReceipt{
		ReceiptID:          r.ReceiptID,
		ManifestID:         "mft-robot-" + r.ReceiptID,
		WorkOrderID:        r.WorkOrderID,
		AssetID:            r.AssetDID,
		CompletedAt:        r.CompletedAt,
		OverallResult:      "PASS_AUTONOMOUS_ATTESTED",
		PayloadDigest:      compositeTelemetryDigest,
		DeviceSignature:    r.RobotSignature,
		ClientRepSignature: "AUTONOMOUS_OPERATOR_GATE_PASSED",
	}

	return coreReceipt, nil
}

// ConvertReceiptJSON is a utility helper for HTTP/CLI ingress decoders.
func ConvertReceiptJSON(raw []byte) (*robotictrust.RoboticInspectionReceipt, error) {
	var receipt robotictrust.RoboticInspectionReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return nil, fmt.Errorf("unmarshal robotic receipt: %w", err)
	}
	if receipt.CompletedAt.IsZero() {
		receipt.CompletedAt = time.Now().UTC()
	}
	return &receipt, nil
}
