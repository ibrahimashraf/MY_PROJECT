package riskengine

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var (
	ErrFatigueQuarantineTriggered = errors.New("riskengine: asset Palmgren-Miner fatigue index exceeds 1.0 (fail-closed quarantine)")
	ErrUncalibratedToolBreach     = errors.New("riskengine: critical inspection tool calibration expired")
	ErrCriticalDefectsPresent     = errors.New("riskengine: unresolved critical defects block rebate issuance")
	ErrInvalidWeights             = errors.New("riskengine: risk factor weights must sum to 1.0")
	ErrZeroAssetDID               = errors.New("riskengine: asset DID cannot be empty")
	ErrZeroTenantID              = errors.New("riskengine: tenant ID cannot be empty")
)

// Weights configures the actuarial risk calculation formula:
// R = W_Fatigue * D + W_Calibration * C_uncal + W_Defect * D_defect
type Weights struct {
	FatigueWeight     float64 `json:"fatigue_weight"`
	CalibrationWeight float64 `json:"calibration_weight"`
	DefectWeight      float64 `json:"defect_weight"`
}

// DefaultWeights returns calibrated empirical actuarial parameters.
func DefaultWeights() Weights {
	return Weights{
		FatigueWeight:     0.45,
		CalibrationWeight: 0.25,
		DefectWeight:      0.30,
	}
}

// OperationalRiskInputs represents the live physical and operational state of an asset.
type OperationalRiskInputs struct {
	TenantID            string    `json:"tenant_id"`
	OrganizationID      string    `json:"organization_id"`
	AssetDID            string    `json:"asset_did"`
	DamageIndex         float64   `json:"damage_index"`          // Palmgren-Miner cumulative fatigue D in [0.0, 1.0]
	UncalibratedTools   int       `json:"uncalibrated_tools"`   // Count of tools used past calibration expiry
	CriticalDefects     int       `json:"critical_defects"`     // Active unresolved critical defects
	MajorDefects        int       `json:"major_defects"`        // Active unresolved major defects
	ObservationTime     time.Time `json:"observation_time"`
}

// RiskScoreResult encapsulates computed risk metrics and policy qualification.
type RiskScoreResult struct {
	AssetDID            string    `json:"asset_did"`
	CalculatedRiskScore float64   `json:"calculated_risk_score"` // In [0.0, 1.0] where 0.0 is perfect, 1.0 is max risk
	IsQuarantined       bool      `json:"is_quarantined"`
	EligibleForRebate   bool      `json:"eligible_for_rebate"`
	DiscountBasisPoints int       `json:"discount_basis_points"` // e.g. 1500 = 15.00% premium rebate
	EvaluatedAt         time.Time `json:"evaluated_at"`
}

// ComputeOperationalRisk evaluates physical telemetry and defect states into an actuarial score.
func ComputeOperationalRisk(in OperationalRiskInputs, w Weights) (RiskScoreResult, error) {
	if in.AssetDID == "" {
		return RiskScoreResult{}, ErrZeroAssetDID
	}
	if in.TenantID == "" {
		return RiskScoreResult{}, ErrZeroTenantID
	}

	sumW := w.FatigueWeight + w.CalibrationWeight + w.DefectWeight
	if sumW < 0.999 || sumW > 1.001 {
		return RiskScoreResult{}, ErrInvalidWeights
	}

	// 1. Fail-closed safety gates
	if in.DamageIndex >= 1.0 {
		return RiskScoreResult{
			AssetDID:            in.AssetDID,
			CalculatedRiskScore: 1.0,
			IsQuarantined:       true,
			EligibleForRebate:   false,
			DiscountBasisPoints: 0,
			EvaluatedAt:         time.Now().UTC(),
		}, ErrFatigueQuarantineTriggered
	}

	if in.UncalibratedTools > 0 {
		return RiskScoreResult{
			AssetDID:            in.AssetDID,
			CalculatedRiskScore: 1.0,
			IsQuarantined:       false,
			EligibleForRebate:   false,
			DiscountBasisPoints: 0,
			EvaluatedAt:         time.Now().UTC(),
		}, ErrUncalibratedToolBreach
	}

	if in.CriticalDefects > 0 {
		return RiskScoreResult{
			AssetDID:            in.AssetDID,
			CalculatedRiskScore: 1.0,
			IsQuarantined:       false,
			EligibleForRebate:   false,
			DiscountBasisPoints: 0,
			EvaluatedAt:         time.Now().UTC(),
		}, ErrCriticalDefectsPresent
	}

	// 2. Continuous risk score calculation
	// Fatigue component: normalized D in [0.0, 1.0]
	fatigueComp := in.DamageIndex
	if fatigueComp > 1.0 {
		fatigueComp = 1.0
	}

	// Defect component: normalized major defects (capped at 5)
	defectComp := float64(in.MajorDefects) / 5.0
	if defectComp > 1.0 {
		defectComp = 1.0
	}

	// Calibration component: 0 since uncalibrated tools gated above
	calComp := 0.0

	riskScore := (w.FatigueWeight * fatigueComp) + (w.CalibrationWeight * calComp) + (w.DefectWeight * defectComp)
	if riskScore > 1.0 {
		riskScore = 1.0
	}

	// 3. Actuarial rebate qualification
	// Risk score < 0.30 qualifies for dynamic insurance rebate up to 25.00% (2500 bps)
	eligible := riskScore < 0.30
	var bps int
	if eligible {
		// Linear sliding scale: 0.0 risk -> 2500 bps, 0.30 risk -> 500 bps
		discountRatio := (0.30 - riskScore) / 0.30
		bps = 500 + int(discountRatio*2000.0)
	}

	return RiskScoreResult{
		AssetDID:            in.AssetDID,
		CalculatedRiskScore: riskScore,
		IsQuarantined:       false,
		EligibleForRebate:   eligible,
		DiscountBasisPoints: bps,
		EvaluatedAt:         time.Now().UTC(),
	}, nil
}

// ParametricRebateVoucher represents an Ed25519-signed actuarial collateral claim.
type ParametricRebateVoucher struct {
	VoucherDID       string    `json:"voucher_did"` // did:integin:rebate:<uuid>
	TenantID         string    `json:"tenant_id"`
	OrganizationID   string    `json:"organization_id"`
	AssetDID         string    `json:"asset_did"`
	CalculatedRisk   float64   `json:"calculated_risk"`
	DiscountBps      int       `json:"discount_bps"`
	IssuedAt         time.Time `json:"issued_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	IssuerDID        string    `json:"issuer_did"`
	Digest           string    `json:"digest"`
	SignatureHex     string    `json:"signature_hex"`
}

// IssueRebateVoucher generates a cryptographically signed voucher for an eligible asset.
func IssueRebateVoucher(
	in OperationalRiskInputs,
	res RiskScoreResult,
	issuerDID string,
	privKey ed25519.PrivateKey,
	validityDuration time.Duration,
) (*ParametricRebateVoucher, error) {
	if !res.EligibleForRebate {
		return nil, errors.New("riskengine: asset is not eligible for parametric rebate voucher")
	}

	var rawUUID [16]byte
	if _, err := rand.Read(rawUUID[:]); err != nil {
		return nil, fmt.Errorf("riskengine: failed to generate voucher UUID: %w", err)
	}
	voucherDID := fmt.Sprintf("did:integin:rebate:%s", hex.EncodeToString(rawUUID[:]))

	now := time.Now().UTC()
	v := &ParametricRebateVoucher{
		VoucherDID:     voucherDID,
		TenantID:       in.TenantID,
		OrganizationID: in.OrganizationID,
		AssetDID:       in.AssetDID,
		CalculatedRisk: res.CalculatedRiskScore,
		DiscountBps:    res.DiscountBasisPoints,
		IssuedAt:       now,
		ExpiresAt:      now.Add(validityDuration),
		IssuerDID:      issuerDID,
	}

	payloadToSign := fmt.Sprintf("%s|%s|%s|%.4f|%d|%d|%d|%s",
		v.VoucherDID, v.TenantID, v.AssetDID, v.CalculatedRisk, v.DiscountBps,
		v.IssuedAt.Unix(), v.ExpiresAt.Unix(), v.IssuerDID,
	)

	digest := sha256.Sum256([]byte(payloadToSign))
	v.Digest = hex.EncodeToString(digest[:])

	sig := ed25519.Sign(privKey, digest[:])
	v.SignatureHex = hex.EncodeToString(sig)

	return v, nil
}

// VerifyRebateVoucher checks cryptographic signature and expiry of a voucher.
func VerifyRebateVoucher(v *ParametricRebateVoucher, pubKey ed25519.PublicKey) (bool, error) {
	if v == nil {
		return false, errors.New("riskengine: voucher is nil")
	}
	if time.Now().UTC().After(v.ExpiresAt) {
		return false, errors.New("riskengine: rebate voucher expired")
	}

	payloadToSign := fmt.Sprintf("%s|%s|%s|%.4f|%d|%d|%d|%s",
		v.VoucherDID, v.TenantID, v.AssetDID, v.CalculatedRisk, v.DiscountBps,
		v.IssuedAt.Unix(), v.ExpiresAt.Unix(), v.IssuerDID,
	)
	expectedDigest := sha256.Sum256([]byte(payloadToSign))
	expectedDigestHex := hex.EncodeToString(expectedDigest[:])
	if expectedDigestHex != v.Digest {
		return false, errors.New("riskengine: voucher digest mismatch")
	}

	sigBytes, err := hex.DecodeString(v.SignatureHex)
	if err != nil {
		return false, fmt.Errorf("riskengine: invalid hex signature: %w", err)
	}

	if !ed25519.Verify(pubKey, expectedDigest[:], sigBytes) {
		return false, errors.New("riskengine: cryptographic signature verification failed")
	}

	return true, nil
}
