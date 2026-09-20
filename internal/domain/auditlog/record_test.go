package auditlog

import (
	"crypto/sha256"
	"testing"
	"time"
)

func benchmarkEntry() Entry {
	return Entry{
		EventType:  "inspection",
		EntityType: EntityInspection,
		EntityID:   "asset-42",
		ActorID:    "alice",
		Action:     ActionCreate,
		CreatedAt:  time.Unix(0, 1_000_000).UTC(),
	}
}

func TestAppendRecordLayout(t *testing.T) {
	entry := benchmarkEntry()
	rec, err := AppendRecord(genesisHash, entry)
	if err != nil {
		t.Fatalf("AppendRecord: %v", err)
	}

	var zero [32]byte
	if prev := rec.PrevHashPrefix(); prev != zero {
		t.Fatalf("prev hash prefix = %x, want all-zero genesis", prev)
	}

	payload := entry.EventType + string(entry.EntityType) + entry.EntityID + entry.ActorID + string(entry.Action)
	digest := sha256.Sum256([]byte(payload))
	if got := rec.PayloadHash(); got != [16]byte(digest[:16]) {
		t.Fatalf("payload hash = %x, want %x", got, digest[:16])
	}

	if got := rec.CreatedUnixNanos(); got != entry.CreatedAt.UnixNano() {
		t.Fatalf("created nanos = %d, want %d", got, entry.CreatedAt.UnixNano())
	}
	if got := rec.ValidUnixNanos(); got != entry.CreatedAt.UnixNano() {
		t.Fatalf("valid nanos default = %d, want created %d", got, entry.CreatedAt.UnixNano())
	}
}

func TestAppendRecordExplicitValidTime(t *testing.T) {
	entry := benchmarkEntry()
	entry.ValidTime = entry.CreatedAt.Add(-time.Hour)

	rec, err := AppendRecord(genesisHash, entry)
	if err != nil {
		t.Fatalf("AppendRecord: %v", err)
	}
	if got := rec.ValidUnixNanos(); got != entry.ValidTime.UnixNano() {
		t.Fatalf("valid nanos = %d, want %d", got, entry.ValidTime.UnixNano())
	}
	if got := rec.CreatedUnixNanos(); got != entry.CreatedAt.UnixNano() {
		t.Fatalf("created nanos = %d, want %d", got, entry.CreatedAt.UnixNano())
	}
	if got := entry.ValidAt(); !got.Equal(entry.ValidTime) {
		t.Fatalf("ValidAt = %v, want %v", got, entry.ValidTime)
	}
	if got := benchmarkEntry().ValidAt(); !got.Equal(entry.CreatedAt) {
		t.Fatalf("ValidAt default = %v, want CreatedAt %v", got, entry.CreatedAt)
	}
}

func TestAppendRecordDeterministic(t *testing.T) {
	entry := benchmarkEntry()
	prev, err := AppendRecord(genesisHash, entry)
	if err != nil {
		t.Fatalf("AppendRecord: %v", err)
	}
	next, err := AppendRecord(genesisHash, entry)
	if err != nil {
		t.Fatalf("AppendRecord: %v", err)
	}
	if prev != next {
		t.Fatalf("records differ:\n%x\n%x", prev, next)
	}
}

func TestAppendRecordInvalidPrevHash(t *testing.T) {
	entry := benchmarkEntry()
	for _, bad := range []string{"", "abc", "00", "zzzz"} {
		if _, err := AppendRecord(bad, entry); err == nil {
			t.Fatalf("AppendRecord(%q): expected error", bad)
		}
	}
}

func TestAppendRecordPayloadTooLong(t *testing.T) {
	entry := benchmarkEntry()
	entry.ActorID = string(make([]byte, maxPayloadLen+1))
	if _, err := AppendRecord(genesisHash, entry); err == nil {
		t.Fatal("AppendRecord: expected payload-too-long error")
	}
}

func BenchmarkAppendRecord(b *testing.B) {
	entry := benchmarkEntry()
	var rec Record
	var err error
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec, err = AppendRecord(genesisHash, entry)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	_ = rec
}

// TestAppendRecordZeroAlloc asserts the zero-allocation half of the <800ns
// write SLA. Allocation count is deterministic; wall-clock latency is not on
// shared hardware (same code measured 320ns..1890ns/op depending on host
// load), so latency stays a reported benchmark metric
// (BenchmarkAppendRecord ns/op) instead of a pass/fail assertion that would
// be flaky by construction. See BenchmarkAppendRecord.
func TestAppendRecordZeroAlloc(t *testing.T) {
	entry := benchmarkEntry()
	if n := testing.AllocsPerRun(100, func() {
		if _, err := AppendRecord(genesisHash, entry); err != nil {
			t.Fatal(err)
		}
	}); n != 0 {
		t.Fatalf("AppendRecord = %v allocs/run, want 0", n)
	}
}
