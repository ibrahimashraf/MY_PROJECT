package evidence

import (
	"strings"
	"testing"
	"time"
)

func TestMetadataRegistrationRequiresServerDerivedScopeAndContainedObjectKey(t *testing.T) {
	actor := ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "actor-a"}
	metadata := validMetadata(actor)
	if err := metadata.ValidateForRegistration(actor); err != nil {
		t.Fatalf("valid metadata rejected: %v", err)
	}
	metadata.OrganizationID = "org-b"
	if err := metadata.ValidateForRegistration(actor); err == nil || !strings.Contains(err.Error(), "server-derived") {
		t.Fatalf("cross-organization metadata error = %v", err)
	}
	metadata = validMetadata(actor)
	metadata.ObjectKey = "tenant-a/org-a/evidence/other"
	if err := metadata.ValidateForRegistration(actor); err == nil || !strings.Contains(err.Error(), "object key") {
		t.Fatalf("uncontained object key error = %v", err)
	}
}

func TestMetadataRegistrationRejectsNonCanonicalDigest(t *testing.T) {
	actor := ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "actor-a"}
	metadata := validMetadata(actor)
	metadata.CiphertextSHA256 = strings.ToUpper(metadata.CiphertextSHA256)
	if err := metadata.ValidateForRegistration(actor); err == nil || !strings.Contains(err.Error(), "digests") {
		t.Fatalf("uppercase digest error = %v", err)
	}
}

func TestMetadataImmutableComparisonExcludesRegistrationTimestamp(t *testing.T) {
	actor := ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "actor-a"}
	first := validMetadata(actor)
	second := first
	second.RegisteredAt = second.RegisteredAt.Add(time.Hour)
	if !first.SameImmutableContent(second) {
		t.Fatal("registration timestamp should not change immutable evidence content")
	}
	second.CiphertextBytes++
	if first.SameImmutableContent(second) {
		t.Fatal("ciphertext byte change must be an immutable conflict")
	}
}

func validMetadata(actor ActorContext) Metadata {
	return Metadata{
		ID:                  "evidence-a",
		TenantID:            actor.TenantID,
		OrganizationID:      actor.OrganizationID,
		InspectionID:        "inspection-a",
		ObjectKey:           actor.TenantID + "/" + actor.OrganizationID + "/evidence/evidence-a",
		ContentType:         "application/octet-stream",
		CiphertextBytes:     42,
		PlaintextSHA256:     strings.Repeat("a", 64),
		CiphertextSHA256:    strings.Repeat("b", 64),
		CapturedAt:          time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
		DeviceID:            "device-a",
		AuthorityID:         "authority-a",
		AuthorityEpoch:      1,
		TransactionID:       "transaction-a",
		ReceiptID:           "receipt-a",
		SignatureAlgorithm:  "Ed25519",
		KeyID:               "key-a",
		EncryptionAlgorithm: "AES-256-GCM",
		EncryptionKeyRef:    "storage-key-a",
		Classification:      "CONFIDENTIAL",
		RetentionReference:  "retention-v1",
		HoldState:           "NONE",
		RedactionPolicyRef:  "redaction-v1",
		RegisteredBy:        actor.ActorID,
	}
}
