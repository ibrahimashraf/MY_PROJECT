package evidenceexport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"integin/internal/domain/evidence"
	"integin/internal/exportmanifest"
	"integin/internal/storage"
)

func TestExportSealsDeterministicManifestFromReverifiedObject(t *testing.T) {
	actor := evidence.ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "actor-a"}
	data := []byte("ciphertext payload")
	metadata := testMetadata(actor, data)
	store := storage.NewInMemoryStore()
	if err := store.Put(context.Background(), storage.Object{Key: metadata.ObjectKey, ContentType: metadata.ContentType, Data: data}); err != nil {
		t.Fatal(err)
	}
	projection, err := New(fakeRepository{records: []evidence.Metadata{metadata}}, store, func() time.Time { return time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	request := Request{ExportID: "export-a", Exporter: exportmanifest.ExporterIdentity{SubjectReference: "subject-a", MembershipReference: "membership-a"}, ApprovalReference: "approval-a"}
	first, err := projection.Export(context.Background(), actor, request)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	second, err := projection.Export(context.Background(), actor, request)
	if err != nil {
		t.Fatalf("repeat export: %v", err)
	}
	if err := first.Verify(); err != nil {
		t.Fatalf("sealed manifest verification: %v", err)
	}
	if first.ManifestChecksum != second.ManifestChecksum || len(first.Evidence) != 1 || first.Evidence[0].CiphertextSHA256 != metadata.CiphertextSHA256 {
		t.Fatalf("unexpected deterministic projection: %#v", first)
	}
}

func TestExportRejectsMissingObjectAndDigestMismatch(t *testing.T) {
	actor := evidence.ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "actor-a"}
	data := []byte("ciphertext payload")
	metadata := testMetadata(actor, data)
	request := Request{ExportID: "export-a", Exporter: exportmanifest.ExporterIdentity{SubjectReference: "subject-a", MembershipReference: "membership-a"}, ApprovalReference: "approval-a"}
	missing, err := New(fakeRepository{records: []evidence.Metadata{metadata}}, storage.NewInMemoryStore(), fixedClock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := missing.Export(context.Background(), actor, request); !errors.Is(err, ErrMissingObject) {
		t.Fatalf("missing object error = %v", err)
	}
	store := storage.NewInMemoryStore()
	if err := store.Put(context.Background(), storage.Object{Key: metadata.ObjectKey, ContentType: metadata.ContentType, Data: []byte("mutated ciphertext")}); err != nil {
		t.Fatal(err)
	}
	mismatch, err := New(fakeRepository{records: []evidence.Metadata{metadata}}, store, fixedClock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mismatch.Export(context.Background(), actor, request); !errors.Is(err, ErrByteCountMismatch) && !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("digest mismatch error = %v", err)
	}
}

func TestExportRejectsRepositoryScopeLeakAndMixedPrivacy(t *testing.T) {
	actor := evidence.ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "actor-a"}
	data := []byte("ciphertext payload")
	metadata := testMetadata(actor, data)
	request := Request{ExportID: "export-a", Exporter: exportmanifest.ExporterIdentity{SubjectReference: "subject-a", MembershipReference: "membership-a"}, ApprovalReference: "approval-a"}
	projection, err := New(fakeRepository{records: []evidence.Metadata{{TenantID: "tenant-a", OrganizationID: "org-b", ID: "evidence-b"}}}, storage.NewInMemoryStore(), fixedClock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projection.Export(context.Background(), actor, request); !errors.Is(err, ErrScopeMismatch) {
		t.Fatalf("scope leak error = %v", err)
	}
	second := testMetadata(actor, []byte("second ciphertext"))
	second.ID = "evidence-b"
	second.ObjectKey = actor.TenantID + "/" + actor.OrganizationID + "/evidence/" + second.ID
	second.Classification = "RESTRICTED"
	store := storage.NewInMemoryStore()
	for _, record := range []evidence.Metadata{metadata, second} {
		data := []byte("ciphertext payload")
		if record.ID == "evidence-b" {
			data = []byte("second ciphertext")
		}
		if err := store.Put(context.Background(), storage.Object{Key: record.ObjectKey, ContentType: record.ContentType, Data: data}); err != nil {
			t.Fatal(err)
		}
	}
	projection, err = New(fakeRepository{records: []evidence.Metadata{metadata, second}}, store, fixedClock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projection.Export(context.Background(), actor, request); !errors.Is(err, ErrMixedPrivacy) {
		t.Fatalf("mixed privacy error = %v", err)
	}
}

type fakeRepository struct{ records []evidence.Metadata }

func (f fakeRepository) Register(context.Context, evidence.ActorContext, evidence.Metadata) (evidence.Metadata, bool, error) {
	return evidence.Metadata{}, false, errors.New("not implemented")
}
func (f fakeRepository) ListByInspection(context.Context, evidence.ActorContext, string) ([]evidence.Metadata, error) {
	return nil, errors.New("not implemented")
}
func (f fakeRepository) ListByTenantOrganization(context.Context, evidence.ActorContext) ([]evidence.Metadata, error) {
	return f.records, nil
}

func testMetadata(actor evidence.ActorContext, data []byte) evidence.Metadata {
	sum := sha256.Sum256(data)
	return evidence.Metadata{
		ID:                  "evidence-a",
		TenantID:            actor.TenantID,
		OrganizationID:      actor.OrganizationID,
		InspectionID:        "inspection-a",
		ObjectKey:           actor.TenantID + "/" + actor.OrganizationID + "/evidence/evidence-a",
		ContentType:         "application/octet-stream",
		CiphertextBytes:     int64(len(data)),
		PlaintextSHA256:     strings.Repeat("a", 64),
		CiphertextSHA256:    hex.EncodeToString(sum[:]),
		CapturedAt:          fixedClock(),
		DeviceID:            "device-a",
		AuthorityID:         "authority-a",
		AuthorityEpoch:      1,
		TransactionID:       "transaction-a",
		ReceiptID:           "receipt-a",
		SignatureAlgorithm:  "Ed25519",
		KeyID:               "signing-key-a",
		EncryptionAlgorithm: "AES-256-GCM",
		EncryptionKeyRef:    "storage-key-a",
		Classification:      "CONFIDENTIAL",
		RetentionReference:  "retention-v1",
		HoldState:           "NONE",
		RedactionPolicyRef:  "redaction-v1",
		RegisteredBy:        actor.ActorID,
	}
}

func fixedClock() time.Time { return time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC) }
