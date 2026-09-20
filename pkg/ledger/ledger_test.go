package ledger

import (
	"testing"
	"time"

	"integin/internal/domain/auditlog"
)

func shimEntry() Entry {
	return Entry{
		EventType:  "test.event",
		EntityType: EntityInspection,
		EntityID:   "entity-1",
		ActorID:    "actor-1",
		Action:     ActionCreate,
		CreatedAt:  time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
	}
}

func TestShimAppendRecordMatchesAuditlog(t *testing.T) {
	want, err := auditlog.AppendRecord(auditlog.GenesisHash(), auditlog.Entry(shimEntry()))
	if err != nil {
		t.Fatalf("auditlog.AppendRecord: %v", err)
	}
	got, err := AppendRecord(GenesisHash(), shimEntry())
	if err != nil {
		t.Fatalf("ledger.AppendRecord: %v", err)
	}
	if got != Record(want) {
		t.Fatal("ledger.AppendRecord diverges from auditlog.AppendRecord")
	}
}

func TestShimGenesisHashMatches(t *testing.T) {
	if GenesisHash() != auditlog.GenesisHash() {
		t.Fatal("ledger.GenesisHash diverges from auditlog.GenesisHash")
	}
}

func TestShimSentinelsAlias(t *testing.T) {
	if ErrInvalidEntry != auditlog.ErrInvalidEntry {
		t.Fatal("ErrInvalidEntry is not the auditlog sentinel")
	}
	if ErrChainBroken != auditlog.ErrChainBroken {
		t.Fatal("ErrChainBroken is not the auditlog sentinel")
	}
}
