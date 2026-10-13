package exportmanifest

import (
	"strings"
	"testing"
	"time"
)

func TestManifestSealAndVerifyAreDeterministicAcrossRecordOrder(t *testing.T) {
	first := validManifest()
	first.Evidence = []EvidenceRecord{first.Evidence[1], first.Evidence[0]}
	if err := first.Seal(); err != nil {
		t.Fatalf("seal first manifest: %v", err)
	}
	second := validManifest()
	if err := second.Seal(); err != nil {
		t.Fatalf("seal second manifest: %v", err)
	}
	if first.ManifestChecksum != second.ManifestChecksum {
		t.Fatalf("checksums differ by input record order: %s != %s", first.ManifestChecksum, second.ManifestChecksum)
	}
	if err := first.Verify(); err != nil {
		t.Fatalf("verify sealed manifest: %v", err)
	}
}

func TestManifestVerifyRejectsTampering(t *testing.T) {
	manifest := validManifest()
	if err := manifest.Seal(); err != nil {
		t.Fatalf("seal manifest: %v", err)
	}
	manifest.Evidence[0].CiphertextSHA256 = "changed"
	if err := manifest.Verify(); err == nil || !strings.Contains(err.Error(), "manifest_checksum") {
		t.Fatalf("verify tampered manifest error = %v, want checksum mismatch", err)
	}
}

func TestManifestRejectsDuplicateEvidenceAndObjectKeys(t *testing.T) {
	manifest := validManifest()
	manifest.Evidence[1].EvidenceID = manifest.Evidence[0].EvidenceID
	if _, err := manifest.CanonicalPayload(); err == nil || !strings.Contains(err.Error(), "duplicate evidence_id") {
		t.Fatalf("duplicate evidence error = %v", err)
	}

	manifest = validManifest()
	manifest.Evidence[1].ObjectKey = manifest.Evidence[0].ObjectKey
	if _, err := manifest.CanonicalPayload(); err == nil || !strings.Contains(err.Error(), "duplicate object_key") {
		t.Fatalf("duplicate object error = %v", err)
	}
}

func TestManifestRejectsMissingTenantAndIncorrectObjectCount(t *testing.T) {
	manifest := validManifest()
	manifest.TenantID = ""
	if _, err := manifest.CanonicalPayload(); err == nil || !strings.Contains(err.Error(), "tenant_id") {
		t.Fatalf("missing tenant error = %v", err)
	}

	manifest = validManifest()
	manifest.ObjectCount++
	if _, err := manifest.CanonicalPayload(); err == nil || !strings.Contains(err.Error(), "object_count") {
		t.Fatalf("incorrect object count error = %v", err)
	}
}

func validManifest() Manifest {
	capturedAt := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	return Manifest{
		ManifestVersion: Version,
		ExportID:        "export-001",
		TenantID:        "tenant-a",
		OrganizationID:  "org-a",
		CreatedAt:       capturedAt,
		Exporter: ExporterIdentity{
			SubjectReference:    "issuer-subject-hash",
			MembershipReference: "membership-001",
		},
		ProcedureVersion: "procedure-v1",
		Evidence: []EvidenceRecord{
			validEvidence("evidence-002", "tenant-a/evidence-002"),
			validEvidence("evidence-001", "tenant-a/evidence-001"),
		},
		ObjectCount: 2,
		Recovery: RecoveryMetadata{
			MissingObjectIDs:          []string{},
			RestoreVerificationResult: "VERIFIED",
		},
		Privacy: PrivacyMetadata{
			Classification:     "CONFIDENTIAL",
			RetentionReference: "retention-v1",
			HoldState:          "NONE",
			RedactionPolicyRef: "redaction-v1",
		},
		Approval: ApprovalReference{ExportApprovalReference: "approval-001"},
	}
}

func validEvidence(evidenceID, objectKey string) EvidenceRecord {
	return EvidenceRecord{
		EvidenceID:          evidenceID,
		ObjectKey:           objectKey,
		ContentType:         "application/octet-stream",
		CapturedAt:          time.Date(2026, 8, 16, 11, 0, 0, 0, time.UTC),
		InspectionReference: "inspection-001",
		EntityReference:     "asset-001",
		PlaintextSHA256:     "plain-digest",
		CiphertextSHA256:    "cipher-digest-" + evidenceID,
		CiphertextBytes:     42,
		EncryptionAlgorithm: "AES-256-GCM",
		EncryptionKeyRef:    "key-reference-001",
		DeviceID:            "device-001",
		AuthorityID:         "authority-001",
		AuthorityEpoch:      1,
		TransactionID:       "transaction-001",
		ReceiptID:           "receipt-001",
		SignatureAlgorithm:  "Ed25519",
		KeyID:               "key-001",
	}
}
