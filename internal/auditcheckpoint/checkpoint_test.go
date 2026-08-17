package auditcheckpoint

import (
	"strings"
	"testing"
	"time"
)

func TestCheckpointRootIsDeterministicAcrossEntryOrder(t *testing.T) {
	first := validCheckpoint()
	first.Entries = []Entry{first.Entries[1], first.Entries[0]}
	if err := first.Seal(); err != nil {
		t.Fatalf("seal first checkpoint: %v", err)
	}
	second := validCheckpoint()
	if err := second.Seal(); err != nil {
		t.Fatalf("seal second checkpoint: %v", err)
	}
	if first.RootSHA256 != second.RootSHA256 {
		t.Fatalf("roots differ by input order: %s != %s", first.RootSHA256, second.RootSHA256)
	}
	if err := first.Verify(); err != nil {
		t.Fatalf("verify checkpoint: %v", err)
	}
}

func TestCheckpointRejectsTamperingAndScopeMismatch(t *testing.T) {
	checkpoint := validCheckpoint()
	if err := checkpoint.Seal(); err != nil {
		t.Fatalf("seal checkpoint: %v", err)
	}
	checkpoint.Entries[0].Outcome = "CHANGED"
	if err := checkpoint.Verify(); err == nil || !strings.Contains(err.Error(), "root_sha256") {
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
