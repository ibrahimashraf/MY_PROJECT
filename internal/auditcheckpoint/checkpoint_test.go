package auditcheckpoint

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"integin/internal/storage"
)

func TestCheckpointRootIsDeterministicAcrossEntryOrder(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	first := validCheckpoint()
	first.Entries = []Entry{first.Entries[1], first.Entries[0]}
	if err := first.Seal(priv, "key-1"); err != nil {
		t.Fatalf("seal first checkpoint: %v", err)
	}
	second := validCheckpoint()
	if err := second.Seal(priv, "key-1"); err != nil {
		t.Fatalf("seal second checkpoint: %v", err)
	}
	if first.RootSHA256 != second.RootSHA256 {
		t.Fatalf("roots differ by input order: %s != %s", first.RootSHA256, second.RootSHA256)
	}
	if err := first.Verify(pub); err != nil {
		t.Fatalf("verify checkpoint: %v", err)
	}
}

func TestCheckpointRejectsTamperingAndScopeMismatch(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	checkpoint := validCheckpoint()
	if err := checkpoint.Seal(priv, "key-1"); err != nil {
		t.Fatalf("seal checkpoint: %v", err)
	}
	checkpoint.Entries[0].Outcome = "CHANGED"
	if err := checkpoint.Verify(pub); err == nil || !strings.Contains(err.Error(), "root_sha256") {
		t.Fatalf("tamper error = %v", err)
	}
	checkpoint = validCheckpoint()
	checkpoint.Entries[0].TenantID = "tenant-other"
	if _, err := checkpoint.CanonicalPayload(); err == nil || !strings.Contains(err.Error(), "tenant/environment") {
		t.Fatalf("scope mismatch error = %v", err)
	}
}

func TestCheckpointRejectsGapsAndDuplicateSequence(t *testing.T) {
	checkpoint := validCheckpoint()
	checkpoint.Entries[1].Sequence = 3
	if _, err := checkpoint.CanonicalPayload(); err == nil || !strings.Contains(err.Error(), "missing sequence") {
		t.Fatalf("gap error = %v", err)
	}
	checkpoint = validCheckpoint()
	checkpoint.Entries[1].Sequence = checkpoint.Entries[0].Sequence
	if _, err := checkpoint.CanonicalPayload(); err == nil || !strings.Contains(err.Error(), "duplicate sequence") {
		t.Fatalf("duplicate sequence error = %v", err)
	}
}

func TestCheckpointRejectsNonDigestContext(t *testing.T) {
	checkpoint := validCheckpoint()
	checkpoint.Entries[0].ContextSHA256 = "not-a-digest"
	if _, err := checkpoint.CanonicalPayload(); err == nil || !strings.Contains(err.Error(), "context_sha256") {
		t.Fatalf("context digest error = %v", err)
	}
}

func validCheckpoint() Checkpoint {
	return Checkpoint{CheckpointVersion: Version, TenantID: "tenant-a", Environment: "PILOT", SequenceStart: 41, SequenceEnd: 42, PreviousRootSHA256: GenesisRoot, Entries: []Entry{validEntry(41, "record-041", "f"), validEntry(42, "record-042", "e")}}
}

func validEntry(sequence uint64, recordID, digestCharacter string) Entry {
	return Entry{Sequence: sequence, RecordID: recordID, TenantID: "tenant-a", Environment: "PILOT", ActorType: "USER", ActorID: "membership-001", Action: "inspection.complete", ResourceType: "inspection", ResourceID: "inspection-001", Outcome: "ACCEPTED", CorrelationID: "correlation-001", ContextSHA256: strings.Repeat(digestCharacter, 64), CreatedAt: time.Date(2026, 8, 16, 18, int(sequence-40), 0, 0, time.UTC)}
}

func chainEntry(sequence uint64, tenantID, environment, recordID string) Entry {
	entry := validEntry(sequence, recordID, "f")
	entry.TenantID = tenantID
	entry.Environment = environment
	return entry
}

func chainEntries(tenantID, environment string, start, end uint64) []Entry {
	entries := make([]Entry, 0, end-start+1)
	for sequence := start; sequence <= end; sequence++ {
		entries = append(entries, chainEntry(sequence, tenantID, environment, fmt.Sprintf("record-%d", sequence)))
	}
	return entries
}

func checkpointIDFromKey(objectKey string) string {
	parts := strings.Split(objectKey, "/")
	return strings.TrimSuffix(parts[len(parts)-1], ".json")
}

func newTestManager(priv ed25519.PrivateKey) (*Manager, *InMemoryCheckpointRepository, *storage.InMemoryStore) {
	repo := NewInMemoryCheckpointRepository()
	store := storage.NewInMemoryStore()
	manager := NewManager(repo, store, Signer{PrivateKey: priv, KeyID: "key-1"})
	return manager, repo, store
}

func TestManagerFullChainContinuity(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	manager, _, _ := newTestManager(priv)

	cp1, objectKey1, err := manager.SealAndPersist(context.Background(), "tenant-a", "PILOT", 1, 50, chainEntries("tenant-a", "PILOT", 1, 50))
	if err != nil {
		t.Fatalf("seal checkpoint 1: %v", err)
	}
	if cp1.PreviousRootSHA256 != GenesisRoot {
		t.Fatalf("genesis previous_root = %q, want GENESIS", cp1.PreviousRootSHA256)
	}
	if cp1.SequenceStart != 1 || cp1.SequenceEnd != 50 {
		t.Fatalf("checkpoint 1 bounds = %d..%d", cp1.SequenceStart, cp1.SequenceEnd)
	}
	if !strings.HasPrefix(objectKey1, "checkpoints/tenant-a/cp-") || !strings.HasSuffix(objectKey1, ".json") {
		t.Fatalf("unexpected object key %q", objectKey1)
	}
	if err := manager.VerifyAndAudit(context.Background(), "tenant-a", checkpointIDFromKey(objectKey1), pub); err != nil {
		t.Fatalf("verify checkpoint 1: %v", err)
	}

	cp2, objectKey2, err := manager.SealAndPersist(context.Background(), "tenant-a", "PILOT", 51, 100, chainEntries("tenant-a", "PILOT", 51, 100))
	if err != nil {
		t.Fatalf("seal checkpoint 2: %v", err)
	}
	if cp2.PreviousRootSHA256 != cp1.RootSHA256 {
		t.Fatalf("checkpoint 2 previous_root = %q, want %q", cp2.PreviousRootSHA256, cp1.RootSHA256)
	}
	if cp2.SequenceStart != 51 || cp2.SequenceEnd != 100 {
		t.Fatalf("checkpoint 2 bounds = %d..%d", cp2.SequenceStart, cp2.SequenceEnd)
	}
	if err := manager.VerifyAndAudit(context.Background(), "tenant-a", checkpointIDFromKey(objectKey2), pub); err != nil {
		t.Fatalf("verify checkpoint 2: %v", err)
	}
}

func TestManagerRejectsSequenceGap(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	manager, _, _ := newTestManager(priv)
	if _, _, err := manager.SealAndPersist(context.Background(), "tenant-a", "PILOT", 1, 50, chainEntries("tenant-a", "PILOT", 1, 50)); err != nil {
		t.Fatalf("seal checkpoint 1: %v", err)
	}
	if _, _, err := manager.SealAndPersist(context.Background(), "tenant-a", "PILOT", 55, 100, chainEntries("tenant-a", "PILOT", 55, 100)); err == nil || !strings.Contains(err.Error(), "sequence") {
		t.Fatalf("sequence gap error = %v", err)
	}
}

func TestManagerRejectsChainStartingAfterOne(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	manager, _, _ := newTestManager(priv)
	if _, _, err := manager.SealAndPersist(context.Background(), "tenant-a", "PILOT", 5, 10, chainEntries("tenant-a", "PILOT", 5, 10)); err == nil || !strings.Contains(err.Error(), "sequence 1") {
		t.Fatalf("genesis start error = %v", err)
	}
}

func TestManagerRejectsMismatchedPreviousRoot(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	manager, repo, _ := newTestManager(priv)
	ctx := context.Background()
	cp1, objectKey1, err := manager.SealAndPersist(ctx, "tenant-a", "PILOT", 1, 50, chainEntries("tenant-a", "PILOT", 1, 50))
	if err != nil {
		t.Fatalf("seal checkpoint 1: %v", err)
	}
	tampered := *cp1
	tampered.RootSHA256 = strings.Repeat("a", 64)
	if err := repo.Save(ctx, tampered, objectKey1); err != nil {
		t.Fatalf("corrupt registry head: %v", err)
	}
	if _, _, err := manager.SealAndPersist(ctx, "tenant-a", "PILOT", 51, 100, chainEntries("tenant-a", "PILOT", 51, 100)); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("previous root mismatch error = %v", err)
	}
}

func TestManagerDetectsObjectTampering(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	manager, _, store := newTestManager(priv)
	ctx := context.Background()
	_, objectKey, err := manager.SealAndPersist(ctx, "tenant-a", "PILOT", 1, 50, chainEntries("tenant-a", "PILOT", 1, 50))
	if err != nil {
		t.Fatalf("seal checkpoint: %v", err)
	}
	object, err := store.Get(ctx, objectKey)
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	var tampered Checkpoint
	if err := json.Unmarshal(object.Data, &tampered); err != nil {
		t.Fatalf("deserialize object: %v", err)
	}
	tampered.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	tamperedBytes, err := json.Marshal(tampered)
	if err != nil {
		t.Fatalf("re-marshal tampered object: %v", err)
	}
	if err := store.Put(ctx, storage.Object{Key: objectKey, ContentType: "application/json", Data: tamperedBytes}); err != nil {
		t.Fatalf("put tampered object: %v", err)
	}
	err = manager.VerifyAndAudit(ctx, "tenant-a", checkpointIDFromKey(objectKey), pub)
	if err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("tamper detection error = %v", err)
	}
}

func TestManagerConcurrentSealAndVerifyRaceSafe(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	manager, _, _ := newTestManager(priv)
	const goroutines = 8
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			tenant := fmt.Sprintf("tenant-%d", index)
			ctx := context.Background()
			start := uint64(1)
			end := start + 9
			checkpoint, objectKey, err := manager.SealAndPersist(ctx, tenant, "PILOT", start, end, chainEntries(tenant, "PILOT", start, end))
			if err != nil {
				errs <- err
				return
			}
			if err := manager.VerifyAndAudit(ctx, tenant, checkpointIDFromKey(objectKey), pub); err != nil {
				errs <- err
				return
			}
			if checkpoint.SequenceStart != start || checkpoint.SequenceEnd != end {
				errs <- fmt.Errorf("bounds %d..%d, want %d..%d", checkpoint.SequenceStart, checkpoint.SequenceEnd, start, end)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}
