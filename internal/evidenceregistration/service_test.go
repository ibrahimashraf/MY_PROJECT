package evidenceregistration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"integin/internal/domain/evidence"
	"integin/internal/domain/workorder"
	"integin/internal/storage"
	"integin/internal/workorderauth"
)

type repositoryStub struct {
	metadata evidence.Metadata
	inserted bool
	err      error
	called   bool
}

func (r *repositoryStub) RegisterForActiveAssignment(_ context.Context, actor workorder.ActorContext, metadata evidence.Metadata) (evidence.Metadata, bool, error) {
	r.called = true
	if metadata.TenantID != actor.TenantID || metadata.OrganizationID != actor.OrganizationID || metadata.RegisteredBy != actor.ActorID {
		return evidence.Metadata{}, false, errors.New("metadata was not server-derived")
	}
	if r.err != nil {
		return evidence.Metadata{}, false, r.err
	}
	if r.metadata.ID == "" {
		r.metadata = metadata
	}
	return r.metadata, r.inserted, nil
}

func TestRegisterDerivesScopeAndVerifiesObject(t *testing.T) {
	store := storage.NewInMemoryStore()
	data := []byte("ciphertext")
	sum := sha256.Sum256(data)
	actor := workorder.ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "inspector-a", Role: "inspector", Capabilities: []string{workorderauth.CapabilitySubmitPartial}}
	key := objectKey(actor, "evidence-a")
	if err := store.Put(context.Background(), storage.Object{Key: key, ContentType: "application/octet-stream", Data: data}); err != nil {
		t.Fatal(err)
	}
	repo := &repositoryStub{inserted: true}
	service, err := NewService(Dependencies{Repository: repo, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Register(context.Background(), testCommand(actor, hex.EncodeToString(sum[:]), int64(len(data))))
	if err != nil || !result.Inserted || !repo.called || result.Metadata.ObjectKey != key {
		t.Fatalf("result=%#v err=%v called=%v", result, err, repo.called)
	}
}

func TestRegisterRejectsMissingOrMismatchedObjectWithoutRepositoryMutation(t *testing.T) {
	actor := workorder.ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "inspector-a", Role: "inspector", Capabilities: []string{workorderauth.CapabilitySubmitPartial}}
	for _, scenario := range []struct {
		name   string
		store  storage.Store
		digest string
		err    error
	}{
		{name: "missing", store: storage.NewInMemoryStore(), digest: strings64("a"), err: ErrObjectUnavailable},
		{name: "mismatch", store: putObject(t, actor, "evidence-a", []byte("ciphertext")), digest: strings64("a"), err: ErrObjectMismatch},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repo := &repositoryStub{inserted: true}
			service, err := NewService(Dependencies{Repository: repo, Store: scenario.store})
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.Register(context.Background(), testCommand(actor, scenario.digest, int64(len("ciphertext"))))
			if !errors.Is(err, scenario.err) || repo.called {
				t.Fatalf("err=%v called=%v", err, repo.called)
			}
		})
	}
}

func TestRegisterRejectsActorWithoutDerivedCapability(t *testing.T) {
	actor := workorder.ActorContext{TenantID: "tenant-a", OrganizationID: "org-a", ActorID: "inspector-a", Role: "inspector"}
	service, err := NewService(Dependencies{Repository: &repositoryStub{}, Store: storage.NewInMemoryStore()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Register(context.Background(), testCommand(actor, strings64("a"), 1)); !errors.Is(err, ErrDenied) {
		t.Fatalf("error = %v", err)
	}
}

func testCommand(actor workorder.ActorContext, digest string, size int64) RegisterCommand {
	return RegisterCommand{Actor: actor, EvidenceID: "evidence-a", InspectionID: "inspection-a", ContentType: "application/octet-stream", CiphertextBytes: size, PlaintextSHA256: strings64("a"), CiphertextSHA256: digest, CapturedAt: time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC), DeviceID: "device-a", AuthorityID: "authority-a", AuthorityEpoch: 1, TransactionID: "transaction-a", ReceiptID: "receipt-a", SignatureAlgorithm: "Ed25519", KeyID: "key-a", EncryptionAlgorithm: "AES-256-GCM", EncryptionKeyRef: "storage-key-a", Classification: "CONFIDENTIAL", RetentionReference: "retention-v1", HoldState: "NONE", RedactionPolicyRef: "redaction-v1"}
}

func putObject(t *testing.T, actor workorder.ActorContext, id string, data []byte) storage.Store {
	t.Helper()
	store := storage.NewInMemoryStore()
	if err := store.Put(context.Background(), storage.Object{Key: objectKey(actor, id), ContentType: "application/octet-stream", Data: data}); err != nil {
		t.Fatal(err)
	}
	return store
}

func strings64(value string) string {
	result := ""
	for len(result) < 64 {
		result += value
	}
	return result[:64]
}
