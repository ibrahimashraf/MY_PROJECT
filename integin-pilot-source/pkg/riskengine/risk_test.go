package riskengine

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestComputeOperationalRisk_GatesAndRebate(t *testing.T) {
	w := DefaultWeights()

	// 1. Palmgren-Miner Fatigue breach (D >= 1.0)
	inFatigue := OperationalRiskInputs{
		TenantID:    "tenant-001",
		AssetDID:    "did:integin:asset:crane-99",
		DamageIndex: 1.05,
	}
	res, err := ComputeOperationalRisk(inFatigue, w)
	if err != ErrFatigueQuarantineTriggered {
		t.Fatalf("expected ErrFatigueQuarantineTriggered, got %v", err)
	}
	if !res.IsQuarantined || res.EligibleForRebate {
		t.Fatalf("expected quarantined and ineligible, got %+v", res)
	}

	// 2. Uncalibrated tools gate
	inTool := OperationalRiskInputs{
		TenantID:          "tenant-001",
		AssetDID:          "did:integin:asset:crane-99",
		DamageIndex:       0.1,
		UncalibratedTools: 1,
	}
	res, err = ComputeOperationalRisk(inTool, w)
	if err != ErrUncalibratedToolBreach {
		t.Fatalf("expected ErrUncalibratedToolBreach, got %v", err)
	}
	if res.EligibleForRebate {
		t.Fatalf("expected ineligible, got %+v", res)
	}

	// 3. Critical defect gate
	inDefect := OperationalRiskInputs{
		TenantID:        "tenant-001",
		AssetDID:        "did:integin:asset:crane-99",
		DamageIndex:     0.1,
		CriticalDefects: 1,
	}
	res, err = ComputeOperationalRisk(inDefect, w)
	if err != ErrCriticalDefectsPresent {
		t.Fatalf("expected ErrCriticalDefectsPresent, got %v", err)
	}
	if res.EligibleForRebate {
		t.Fatalf("expected ineligible, got %+v", res)
	}

	// 4. Low-risk pristine asset eligible for rebate
	inPristine := OperationalRiskInputs{
		TenantID:        "tenant-001",
		OrganizationID:  "org-001",
		AssetDID:        "did:integin:asset:crane-99",
		DamageIndex:     0.05, // 0.05 * 0.45 = 0.0225 risk
		MajorDefects:    0,
		ObservationTime: time.Now().UTC(),
	}
	res, err = ComputeOperationalRisk(inPristine, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.EligibleForRebate {
		t.Fatalf("expected eligible for rebate, got %+v", res)
	}
	if res.DiscountBasisPoints <= 500 {
		t.Fatalf("expected significant discount bps (>500), got %d", res.DiscountBasisPoints)
	}
}

func TestIssueAndVerifyRebateVoucher(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key: %v", err)
	}

	w := DefaultWeights()
	in := OperationalRiskInputs{
		TenantID:       "tenant-001",
		OrganizationID: "org-001",
		AssetDID:       "did:integin:asset:crane-101",
		DamageIndex:    0.10,
	}

	res, err := ComputeOperationalRisk(in, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	voucher, err := IssueRebateVoucher(in, res, "did:integin:issuer:underwriter", priv, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to issue rebate voucher: %v", err)
	}

	ok, err := VerifyRebateVoucher(voucher, pub)
	if err != nil || !ok {
		t.Fatalf("expected voucher verification success, got ok=%v err=%v", ok, err)
	}

	// Verification fails with wrong key
	pubWrong, _, _ := ed25519.GenerateKey(rand.Reader)
	ok, err = VerifyRebateVoucher(voucher, pubWrong)
	if ok || err == nil {
		t.Fatalf("expected verification failure with wrong pubkey")
	}

	// Verification fails on expired voucher
	voucher.ExpiresAt = time.Now().UTC().Add(-1 * time.Minute)
	ok, err = VerifyRebateVoucher(voucher, pub)
	if ok || err == nil {
		t.Fatalf("expected verification failure on expired voucher")
	}
}

func BenchmarkComputeOperationalRisk_HotPath(b *testing.B) {
	w := DefaultWeights()
	in := OperationalRiskInputs{
		TenantID:        "tenant-001",
		OrganizationID:  "org-001",
		AssetDID:        "did:integin:asset:crane-bench",
		DamageIndex:     0.15,
		MajorDefects:    1,
		ObservationTime: time.Now().UTC(),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ComputeOperationalRisk(in, w)
	}
}
