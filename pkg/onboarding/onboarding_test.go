package onboarding

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func TestEndToEndDeviceOnboardingAndReceiptVerification(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("Failed to initialize simulator: %v", err)
	}

	tenantID := "ten_apex_01"
	inspectorID := "insp_tariq_01"

	// -------------------------------------------------------------
	// 1. Server generates enrollment QR challenge
	// -------------------------------------------------------------
	chal, err := sim.CreateEnrollmentChallenge(tenantID, inspectorID)
	if err != nil {
		t.Fatalf("Failed to create challenge: %v", err)
	}
	t.Logf("✅ Challenge Created: ID=%s Nonce=%s", chal.ChallengeID, chal.Nonce[:12])

	// -------------------------------------------------------------
	// 2. Field Tablet generates Ed25519 keypair and signs challenge nonce
	// -------------------------------------------------------------
	tabletPubKey, tabletPrivKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate tablet key: %v", err)
	}

	signedNonce := ed25519.Sign(tabletPrivKey, []byte(chal.Nonce))
	submission := DeviceEnrollmentSubmission{
		ChallengeID:       chal.ChallengeID,
		InspectorID:       inspectorID,
		DevicePublicKey:   hex.EncodeToString(tabletPubKey),
		DeviceFingerprint: "hw_samsung_tab_active4_sn9982",
		DeviceModel:       "Samsung Galaxy Tab Active4 Pro",
		SignedNonce:       hex.EncodeToString(signedNonce),
	}

	// -------------------------------------------------------------
	// 3. Server processes enrollment and binds device
	// -------------------------------------------------------------
	trustRecord, err := sim.ProcessDeviceEnrollment(submission)
	if err != nil {
		t.Fatalf("Device enrollment failed: %v", err)
	}
	t.Logf("✅ Device Bound: DeviceID=%s InspectorID=%s", trustRecord.DeviceID, trustRecord.InspectorID)

	// -------------------------------------------------------------
	// 4. Server issues a Work Package Manifest
	// -------------------------------------------------------------
	workOrderID := "wo_jubail_crane_001"
	checklistJSON := `{"crane_model":"Kato NK-1000","swl":"100T","items":[{"id":"outriggers","status":"pass"}]}`

	manifest, err := sim.IssueWorkPackageManifest(tenantID, workOrderID, trustRecord.DeviceID, inspectorID, checklistJSON)
	if err != nil {
		t.Fatalf("Failed to issue manifest: %v", err)
	}
	t.Logf("✅ Manifest Issued: ManifestID=%s PackageHash=%s", manifest.ManifestID, manifest.PackageHash[:12])

	// -------------------------------------------------------------
	// 5. Tablet goes OFFLINE, completes inspection, and cryptographically signs receipt
	// -------------------------------------------------------------
	assetID := "asset_kato_nk1000_cr01"
	findingsDigest := sha256.Sum256([]byte(checklistJSON + `{"result":"PASSED","notes":"Wire rope minor wear"}`))
	payloadDigestHex := hex.EncodeToString(findingsDigest[:])
	receiptID := "rcpt_20260902_001"

	receiptSignPayload := fmt.Sprintf("%s|%s|%s|%s|%s", receiptID, manifest.ManifestID, assetID, "PASSED", payloadDigestHex)
	deviceSig := ed25519.Sign(tabletPrivKey, []byte(receiptSignPayload))

	receipt := SignedInspectionReceipt{
		ReceiptID:          receiptID,
		ManifestID:         manifest.ManifestID,
		WorkOrderID:        workOrderID,
		AssetID:            assetID,
		CompletedAt:        time.Now(),
		OverallResult:      "PASSED",
		PayloadDigest:      payloadDigestHex,
		DeviceSignature:    hex.EncodeToString(deviceSig),
		ClientRepSignature: "Fahad Al-Sharq QA",
	}

	// -------------------------------------------------------------
	// 6. Tablet reconnects: Server verifies offline cryptographic receipt
	// -------------------------------------------------------------
	valid, err := sim.VerifyOfflineReceipt(receipt)
	if err != nil || !valid {
		t.Fatalf("Receipt verification failed: %v", err)
	}
	t.Logf("✅ Offline Receipt Cryptographically Verified! Certificate is ready for issuance.")

	// -------------------------------------------------------------
	// 7. Negative Test: Tampered Receipt Detection
	// -------------------------------------------------------------
	tamperedReceipt := receipt
	tamperedReceipt.OverallResult = "FAILED_CRITICAL" // An unauthorized party altered the result

	tamperedValid, err := sim.VerifyOfflineReceipt(tamperedReceipt)
	if tamperedValid || err == nil {
		t.Fatalf("Security failure: Tampered receipt was accepted!")
	}
	t.Logf("🛡️ Tamper Detection Passed: %v", err)
}
