package sync

import (
	"strconv"
	"testing"
)

// TestCurrentSchemaVersionerLineage proves the production registry derives a
// contiguous ladder one step per canonical migration, terminating exactly at
// currentSchemaEpoch.
func TestCurrentSchemaVersionerLineage(t *testing.T) {
	versioner, err := CurrentSchemaVersioner()
	if err != nil {
		t.Fatal(err)
	}
	if versioner == nil {
		t.Fatal("CurrentSchemaVersioner returned nil with " + strconv.FormatUint(uint64(currentSchemaEpoch), 10) + ") migrations, want versioner")
	}
	if got := versioner.CurrentEpoch(); got != currentSchemaEpoch {
		t.Fatalf("CurrentEpoch=%d want %d", got, currentSchemaEpoch)
	}
	for i := SchemaEpoch(0); i < currentSchemaEpoch; i++ {
		if result := versioner.Handshake(i); result.Status != StatusBehindCatchUp {
			t.Fatalf("handshake(epoch=%d)=%s want %s", i, result.Status, StatusBehindCatchUp)
		}
	}
	if result := versioner.Handshake(currentSchemaEpoch); result.Status != StatusSync {
		t.Fatalf("handshake(current)=%s want %s", result.Status, StatusSync)
	}
}

// TestProcessorSchemaHandshakeRegistryEquivalence proves the production-wired
// gate behaves exactly like the current nil-gate fast path for a same-epoch
// tablet (StatusSync), while behind and ahead tablets are now enforced.
func TestProcessorSchemaHandshakeRegistryEquivalence(t *testing.T) {
	processor, err := NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	openGate := processor.SchemaHandshake(currentSchemaEpoch)
	if openGate.Status != StatusSync {
		t.Fatalf("nil gate same-epoch status=%s want %s", openGate.Status, StatusSync)
	}

	versioner, err := CurrentSchemaVersioner()
	if err != nil {
		t.Fatal(err)
	}
	if versioner == nil {
		t.Fatal("CurrentSchemaVersioner returned nil, want versioner")
	}
	processor.SetSchemaVersioner(versioner)

	same := processor.SchemaHandshake(currentSchemaEpoch)
	if same.Status != StatusSync {
		t.Fatalf("wired same-epoch status=%s want %s (fast path must be preserved)", same.Status, StatusSync)
	}
	if same.ServerEpoch != currentSchemaEpoch {
		t.Fatalf("wired ServerEpoch=%d want %d", same.ServerEpoch, currentSchemaEpoch)
	}

	behind := processor.SchemaHandshake(0)
	if behind.Status != StatusBehindCatchUp {
		t.Fatalf("wired behind status=%s want %s", behind.Status, StatusBehindCatchUp)
	}
	if behind.Plan == nil || len(behind.Plan.Migrations) != int(currentSchemaEpoch) {
		t.Fatalf("wired behind plan migrations=%d want %d", len(behind.Plan.Migrations), currentSchemaEpoch)
	}

	ahead := processor.SchemaHandshake(currentSchemaEpoch + 1)
	if ahead.Status != StatusAheadRejected {
		t.Fatalf("wired ahead status=%s want %s", ahead.Status, StatusAheadRejected)
	}
}
