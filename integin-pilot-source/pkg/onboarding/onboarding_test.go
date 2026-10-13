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

func validCalibratedTool(tenantID, toolID string) CalibratedToolRecord {
	now := time.Now().UTC()
	certHash := sha256.Sum256([]byte("official-calibration-certificate-pdf-bytes"))
	return CalibratedToolRecord{
		ToolID:               toolID,
		TenantID:             tenantID,
		OrganizationID:       "org_apex_01",
		SerialNumber:         "SN-LC-2026-9811",
		ToolType:             "LOAD_CELL",
		Manufacturer:         "Straightpoint Crosby",
		Model:                "Radiolink Plus 50T",
		StandardReference:    "ISO-17020",
		LabCertificateRef:    "CAL-LAB-SAAC-2026-0988",
		LabCertificateDigest: hex.EncodeToString(certHash[:]),
		UncertaintyTolerance: "±0.2% Full Scale (k=2)",
		CalibrationDate:      now.Add(-60 * 24 * time.Hour),
		NextDueDate:          now.Add(305 * 24 * time.Hour),
		IssuingLab:           "Saudi Accreditation Center Lab #104",
		TechnicianID:         "tech_khaled_07",
		Result:               "PASS",
		Status:               ToolCalibrationActive,
		RegisteredAt:         now.Add(-60 * 24 * time.Hour),
	}
}

func TestCalibratedToolRecordValidation(t *testing.T) {
	valid := validCalibratedTool("tenant_1", "tool_lc_01")
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid tool, got: %v", err)
	}

	// Missing tool ID
	invalid := valid
	invalid.ToolID = ""
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error on empty tool ID")
	}

	// Inverted dates
	invalid = valid
	invalid.NextDueDate = invalid.CalibrationDate.Add(-1 * time.Hour)
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error on next due date before calibration date")
	}

	// Invalid certificate digest (not 64 hex characters)
	invalid = valid
	invalid.LabCertificateDigest = "bad-hash"
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error on non-hex SHA256 digest")
	}

	// Invalid status
	invalid = valid
	invalid.Status = "INVALID_STATUS"
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error on unknown status")
	}
}

func TestCalibratedToolCanBeUsedForInspection(t *testing.T) {
	now := time.Now().UTC()
	tool := validCalibratedTool("tenant_1", "tool_lc_01")

	// Active and unexpired
	if err := tool.CanBeUsedForInspection(now); err != nil {
		t.Fatalf("expected tool usable, got: %v", err)
	}

	// Expired
	expiredTool := tool
	expiredTool.NextDueDate = now.Add(-24 * time.Hour)
	if err := expiredTool.CanBeUsedForInspection(now); err == nil {
		t.Fatal("expected expired tool to be rejected")
	}

	// Quarantined
	quarantinedTool := tool
	quarantinedTool.Status = ToolCalibrationQuarantined
	if err := quarantinedTool.CanBeUsedForInspection(now); err == nil {
		t.Fatal("expected quarantined tool to be rejected")
	}

	// Superseded
	supersededTool := tool
	supersededTool.Status = ToolCalibrationSuperseded
	if err := supersededTool.CanBeUsedForInspection(now); err == nil {
		t.Fatal("expected superseded tool to be rejected")
	}
}

func TestEndToEndCalibratedToolRegistryAndReceiptGating(t *testing.T) {
	sim, err := NewEnrollmentSimulator()
	if err != nil {
		t.Fatalf("Failed to initialize simulator: %v", err)
	}

	tenantID := "ten_apex_01"
	inspectorID := "insp_tariq_01"

	// 1. Register ISO 17020 §6.2 Calibrated Tools in Tenant Store
	tool1 := validCalibratedTool(tenantID, "tool_loadcell_50t")
	if err := sim.RegisterCalibratedTool(tool1); err != nil {
		t.Fatalf("Failed to register tool1: %v", err)
	}

	tool2 := validCalibratedTool(tenantID, "tool_ut_gauge_01")
	tool2.ToolType = "ULTRASONIC_GAUGE"
	tool2.SerialNumber = "SN-UT-2026-441"
	if err := sim.RegisterCalibratedTool(tool2); err != nil {
		t.Fatalf("Failed to register tool2: %v", err)
	}

	// 2. Enroll Field Tablet
	chal, err := sim.CreateEnrollmentChallenge(tenantID, inspectorID)
	if err != nil {
		t.Fatal(err)
	}
	tabletPubKey, tabletPrivKey, _ := ed25519.GenerateKey(rand.Reader)
	signedNonce := ed25519.Sign(tabletPrivKey, []byte(chal.Nonce))
	trustRecord, err := sim.ProcessDeviceEnrollment(DeviceEnrollmentSubmission{
		ChallengeID:       chal.ChallengeID,
		InspectorID:       inspectorID,
		DevicePublicKey:   hex.EncodeToString(tabletPubKey),
		DeviceFingerprint: "hw_tab_active4_sn9982",
		DeviceModel:       "Samsung Tab Active4",
		SignedNonce:       hex.EncodeToString(signedNonce),
	})
	if err != nil {
		t.Fatal(err)
	}

	// 3. Issue Manifest with Required Tools
	manifest, err := sim.IssueWorkPackageManifestWithTools(tenantID, "wo_jubail_001", trustRecord.DeviceID, inspectorID, `{"task":"proof_load"}`, []string{tool1.ToolID, tool2.ToolID})
	if err != nil {
		t.Fatal(err)
	}

	// 4. Tablet completes inspection using both valid calibrated tools
	findingsDigest := sha256.Sum256([]byte(`{"task":"proof_load","load_measured":"48.5T","thickness":"18.2mm"}`))
	payloadDigestHex := hex.EncodeToString(findingsDigest[:])
	receiptID := "rcpt_iso17020_001"
	signPayload := fmt.Sprintf("%s|%s|%s|%s|%s", receiptID, manifest.ManifestID, "crane_01", "PASSED", payloadDigestHex)
	deviceSig := ed25519.Sign(tabletPrivKey, []byte(signPayload))

	receipt := SignedInspectionReceipt{
		ReceiptID:         receiptID,
		ManifestID:        manifest.ManifestID,
		WorkOrderID:       manifest.WorkOrderID,
		AssetID:           "crane_01",
		CompletedAt:       time.Now().UTC(),
		OverallResult:     "PASSED",
		PayloadDigest:     payloadDigestHex,
		DeviceSignature:   hex.EncodeToString(deviceSig),
		CalibratedToolIDs: []string{tool1.ToolID, tool2.ToolID},
	}

	valid, err := sim.VerifyOfflineReceipt(receipt)
	if err != nil || !valid {
		t.Fatalf("expected valid receipt with active calibrated tools, got: %v", err)
	}
	t.Logf("✅ Receipt verified with active ISO 17020 calibrated tools!")
}

func TestOfflineReceiptHardBlockedOnExpiredCalibratedTool(t *testing.T) {
	sim, _ := NewEnrollmentSimulator()
	tenantID := "ten_apex_01"
	inspectorID := "insp_tariq_01"

	// Register an expired tool
	expiredTool := validCalibratedTool(tenantID, "tool_expired_loadcell")
	expiredTool.NextDueDate = time.Now().UTC().Add(-48 * time.Hour) // expired 2 days ago
	expiredTool.Status = ToolCalibrationExpired
	if err := sim.RegisterCalibratedTool(expiredTool); err != nil {
		t.Fatal(err)
	}

	chal, _ := sim.CreateEnrollmentChallenge(tenantID, inspectorID)
	tabletPubKey, tabletPrivKey, _ := ed25519.GenerateKey(rand.Reader)
	signedNonce := ed25519.Sign(tabletPrivKey, []byte(chal.Nonce))
	trustRecord, _ := sim.ProcessDeviceEnrollment(DeviceEnrollmentSubmission{
		ChallengeID:     chal.ChallengeID,
		InspectorID:     inspectorID,
		DevicePublicKey: hex.EncodeToString(tabletPubKey),
		SignedNonce:     hex.EncodeToString(signedNonce),
	})

	manifest, _ := sim.IssueWorkPackageManifest(tenantID, "wo_002", trustRecord.DeviceID, inspectorID, `{}`)

	findingsDigest := sha256.Sum256([]byte(`{"load":"50T"}`))
	payloadDigestHex := hex.EncodeToString(findingsDigest[:])
	receiptID := "rcpt_expired_001"
	signPayload := fmt.Sprintf("%s|%s|%s|%s|%s", receiptID, manifest.ManifestID, "crane_02", "PASSED", payloadDigestHex)
	deviceSig := ed25519.Sign(tabletPrivKey, []byte(signPayload))

	receipt := SignedInspectionReceipt{
		ReceiptID:         receiptID,
		ManifestID:        manifest.ManifestID,
		WorkOrderID:       manifest.WorkOrderID,
		AssetID:           "crane_02",
		CompletedAt:       time.Now().UTC(),
		OverallResult:     "PASSED",
		PayloadDigest:     payloadDigestHex,
		DeviceSignature:   hex.EncodeToString(deviceSig),
		CalibratedToolIDs: []string{expiredTool.ToolID},
	}

	valid, err := sim.VerifyOfflineReceipt(receipt)
	if valid || err == nil {
		t.Fatalf("Hard-block security breach: expired tool calibration was accepted!")
	}
	t.Logf("🛡️ Expired Calibrated Tool Blocked: %v", err)
}

func TestOfflineReceiptHardBlockedOnQuarantinedCalibratedTool(t *testing.T) {
	sim, _ := NewEnrollmentSimulator()
	tenantID := "ten_apex_01"
	inspectorID := "insp_tariq_01"

	quarantinedTool := validCalibratedTool(tenantID, "tool_quarantined_torque")
	quarantinedTool.Status = ToolCalibrationQuarantined
	_ = sim.RegisterCalibratedTool(quarantinedTool)

	chal, _ := sim.CreateEnrollmentChallenge(tenantID, inspectorID)
	tabletPubKey, tabletPrivKey, _ := ed25519.GenerateKey(rand.Reader)
	signedNonce := ed25519.Sign(tabletPrivKey, []byte(chal.Nonce))
	trustRecord, _ := sim.ProcessDeviceEnrollment(DeviceEnrollmentSubmission{
		ChallengeID:     chal.ChallengeID,
		InspectorID:     inspectorID,
		DevicePublicKey: hex.EncodeToString(tabletPubKey),
		SignedNonce:     hex.EncodeToString(signedNonce),
	})

	manifest, _ := sim.IssueWorkPackageManifest(tenantID, "wo_003", trustRecord.DeviceID, inspectorID, `{}`)
	findingsDigest := sha256.Sum256([]byte(`{"torque":"120Nm"}`))
	payloadDigestHex := hex.EncodeToString(findingsDigest[:])
	receiptID := "rcpt_quarantine_001"
	signPayload := fmt.Sprintf("%s|%s|%s|%s|%s", receiptID, manifest.ManifestID, "crane_03", "PASSED", payloadDigestHex)
	deviceSig := ed25519.Sign(tabletPrivKey, []byte(signPayload))

	receipt := SignedInspectionReceipt{
		ReceiptID:         receiptID,
		ManifestID:        manifest.ManifestID,
		WorkOrderID:       manifest.WorkOrderID,
		AssetID:           "crane_03",
		CompletedAt:       time.Now().UTC(),
		OverallResult:     "PASSED",
		PayloadDigest:     payloadDigestHex,
		DeviceSignature:   hex.EncodeToString(deviceSig),
		CalibratedToolIDs: []string{quarantinedTool.ToolID},
	}

	valid, err := sim.VerifyOfflineReceipt(receipt)
	if valid || err == nil {
		t.Fatalf("Hard-block security breach: quarantined tool was accepted!")
	}
	t.Logf("🛡️ Quarantined Tool Blocked: %v", err)
}

func TestOfflineReceiptHardBlockedOnUnknownCalibratedTool(t *testing.T) {
	sim, _ := NewEnrollmentSimulator()
	tenantID := "ten_apex_01"
	inspectorID := "insp_tariq_01"

	chal, _ := sim.CreateEnrollmentChallenge(tenantID, inspectorID)
	tabletPubKey, tabletPrivKey, _ := ed25519.GenerateKey(rand.Reader)
	signedNonce := ed25519.Sign(tabletPrivKey, []byte(chal.Nonce))
	trustRecord, _ := sim.ProcessDeviceEnrollment(DeviceEnrollmentSubmission{
		ChallengeID:     chal.ChallengeID,
		InspectorID:     inspectorID,
		DevicePublicKey: hex.EncodeToString(tabletPubKey),
		SignedNonce:     hex.EncodeToString(signedNonce),
	})

	manifest, _ := sim.IssueWorkPackageManifest(tenantID, "wo_004", trustRecord.DeviceID, inspectorID, `{}`)
	findingsDigest := sha256.Sum256([]byte(`{"test":"unregistered"}`))
	payloadDigestHex := hex.EncodeToString(findingsDigest[:])
	receiptID := "rcpt_unknown_001"
	signPayload := fmt.Sprintf("%s|%s|%s|%s|%s", receiptID, manifest.ManifestID, "crane_04", "PASSED", payloadDigestHex)
	deviceSig := ed25519.Sign(tabletPrivKey, []byte(signPayload))

	receipt := SignedInspectionReceipt{
		ReceiptID:         receiptID,
		ManifestID:        manifest.ManifestID,
		WorkOrderID:       manifest.WorkOrderID,
		AssetID:           "crane_04",
		CompletedAt:       time.Now().UTC(),
		OverallResult:     "PASSED",
		PayloadDigest:     payloadDigestHex,
		DeviceSignature:   hex.EncodeToString(deviceSig),
		CalibratedToolIDs: []string{"tool_ghost_loadcell"},
	}

	valid, err := sim.VerifyOfflineReceipt(receipt)
	if valid || err == nil {
		t.Fatalf("Hard-block security breach: unverified ghost tool was accepted!")
	}
	t.Logf("🛡️ Unknown Tool Blocked: %v", err)
}
