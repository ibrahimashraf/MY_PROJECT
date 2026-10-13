package sync

import (
	"testing"
)

func ladderSample() []SchemaMigration {
	return []SchemaMigration{
		{ID: "m1", FromEpoch: 0, ToEpoch: 1},
		{ID: "m2", FromEpoch: 1, ToEpoch: 2, Transforms: []DataTransform{{ID: "t2a", Order: 0, Label: "reindex inspection ids"}, {ID: "t2b", Order: 1, Label: "rename status enum"}}},
		{ID: "m3", FromEpoch: 2, ToEpoch: 3, Transforms: []DataTransform{{ID: "t3a", Order: 0, Label: "re-key payload hash"}}},
	}
}

func TestSchemaHandshakeFastPathSameEpoch(t *testing.T) {
	versioner, err := NewSchemaVersioner(3, ladderSample())
	if err != nil {
		t.Fatal(err)
	}
	result := versioner.Handshake(3)
	if result.Status != StatusSync {
		t.Fatalf("status=%s want %s", result.Status, StatusSync)
	}
	if result.Plan != nil {
		t.Fatalf("fast path must not return a plan, got %#v", result.Plan)
	}
	if result.Reason != "" {
		t.Fatalf("fast path must have no reason, got %q", result.Reason)
	}
}

func TestSchemaHandshakeBehindReturnsOrderedCatchUpPlan(t *testing.T) {
	versioner, err := NewSchemaVersioner(3, ladderSample())
	if err != nil {
		t.Fatal(err)
	}
	result := versioner.Handshake(1)
	if result.Status != StatusBehindCatchUp {
		t.Fatalf("status=%s want %s", result.Status, StatusBehindCatchUp)
	}
	plan := result.Plan
	if plan == nil {
		t.Fatal("behind tablet must receive a catch-up plan")
	}
	if plan.MinSchema != 3 {
		t.Fatalf("MinSchema=%d want 3", plan.MinSchema)
	}
	if len(plan.Migrations) != 2 || plan.Migrations[0].ID != "m2" || plan.Migrations[1].ID != "m3" {
		t.Fatalf("ordered migrations=%#v", plan.Migrations)
	}
	if len(plan.Transforms) != 3 {
		t.Fatalf("transforms=%#v want 3 across both steps", plan.Transforms)
	}
	if plan.Transforms[0].ID != "t2a" || plan.Transforms[2].ID != "t3a" {
		t.Fatalf("transform order=%#v", plan.Transforms)
	}
}

func TestSchemaHandshakeBehindAtZeroIncludesEveryMigration(t *testing.T) {
	versioner, err := NewSchemaVersioner(3, ladderSample())
	if err != nil {
		t.Fatal(err)
	}
	result := versioner.Handshake(0)
	if result.Status != StatusBehindCatchUp {
		t.Fatalf("status=%s want %s", result.Status, StatusBehindCatchUp)
	}
	if len(result.Plan.Migrations) != 3 {
		t.Fatalf("epoch-0 tablet should receive all %d migrations, got %d", 3, len(result.Plan.Migrations))
	}
}

func TestSchemaHandshakeAheadRejectedNeverDowngrades(t *testing.T) {
	versioner, err := NewSchemaVersioner(3, ladderSample())
	if err != nil {
		t.Fatal(err)
	}
	for _, ahead := range []SchemaEpoch{4, 5, 1000} {
		result := versioner.Handshake(ahead)
		if result.Status != StatusAheadRejected {
			t.Fatalf("epoch %d status=%s want %s", ahead, result.Status, StatusAheadRejected)
		}
		if result.Plan != nil {
			t.Fatalf("rejected ahead tablet=%d must not receive a plan", ahead)
		}
	}
}

func TestSchemaHandshakeUnknownEpochRejected(t *testing.T) {
	// Gapped release batching: the server's rungs are 0, 5, 8. Any integer
	// below current that is not a rung is an unknown epoch and fails closed.
	versioner, err := NewSchemaVersioner(8, []SchemaMigration{
		{ID: "batch.a", FromEpoch: 0, ToEpoch: 5},
		{ID: "batch.b", FromEpoch: 5, ToEpoch: 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, unknown := range []SchemaEpoch{1, 3, 6, 7} {
		result := versioner.Handshake(unknown)
		if result.Status != StatusUnknownRejected {
			t.Fatalf("epoch %d status=%s want %s", unknown, result.Status, StatusUnknownRejected)
		}
	}
}

func TestSchemaVersionerRejectsInvalidLadders(t *testing.T) {
	if _, err := NewSchemaVersioner(1, nil); err == nil {
		t.Fatal("non-zero current without migrations must be rejected")
	}
	if _, err := NewSchemaVersioner(3, []SchemaMigration{
		{ID: "a", FromEpoch: 1, ToEpoch: 2},
	}); err == nil {
		t.Fatal("ladder not starting at 0 must be rejected")
	}
	if _, err := NewSchemaVersioner(3, []SchemaMigration{
		{ID: "a", FromEpoch: 0, ToEpoch: 1},
		{ID: "b", FromEpoch: 0, ToEpoch: 2},
	}); err == nil {
		t.Fatal("non-contiguous ladder must be rejected")
	}
	if _, err := NewSchemaVersioner(3, []SchemaMigration{
		{ID: "a", FromEpoch: 0, ToEpoch: 1},
		{ID: "b", FromEpoch: 1, ToEpoch: 2},
	}); err == nil {
		t.Fatal("ladder not reaching current epoch must be rejected")
	}
	if _, err := NewSchemaVersioner(2, []SchemaMigration{
		{ID: "a", FromEpoch: 5, ToEpoch: 2},
	}); err == nil {
		t.Fatal("non-ascending migration must be rejected")
	}
}

// TestSchemaHandshakeThirtyDayOfflineReconcilesWithoutDataLoss simulates a
// tablet that last synced 30+ days ago, during which the server shipped three
// more migration batches. The catch-up plan is derived purely from epoch
// arithmetic: no sleeps, no wall clock.
func TestSchemaHandshakeThirtyDayOfflineReconcilesWithoutDataLoss(t *testing.T) {
	// B1 shipped yesterday, B2 shipped two weeks ago, B3 shipped today.
	migrations := []SchemaMigration{
		{ID: "sprint.b1", FromEpoch: 0, ToEpoch: 1},
		{ID: "sprint.b2-a", FromEpoch: 1, ToEpoch: 2, Transforms: []DataTransform{{ID: "dt-b2-a", Order: 0, Label: "stamp legacy rows"}}},
		{ID: "sprint.b2-b", FromEpoch: 2, ToEpoch: 3},
		{ID: "sprint.b3", FromEpoch: 3, ToEpoch: 4, Transforms: []DataTransform{{ID: "dt-b3", Order: 0, Label: "invert compliance flags"}}},
	}
	versioner, err := NewSchemaVersioner(4, migrations)
	if err != nil {
		t.Fatal(err)
	}
	// The tablet froze after sprint.b1 (epoch 1) and is 30+ days stale.
	result := versioner.Handshake(1)
	if result.Status != StatusBehindCatchUp {
		t.Fatalf("status=%s want %s", result.Status, StatusBehindCatchUp)
	}
	if result.Plan.MinSchema != 4 {
		t.Fatalf("MinSchema=%d want 4", result.Plan.MinSchema)
	}
	want := []string{"sprint.b2-a", "sprint.b2-b", "sprint.b3"}
	if len(result.Plan.Migrations) != len(want) {
		t.Fatalf("migrations=%#v want 3", result.Plan.Migrations)
	}
	for i, expected := range want {
		if result.Plan.Migrations[i].ID != expected {
			t.Fatalf("migration[%d]=%q want %q", i, result.Plan.Migrations[i].ID, expected)
		}
	}
	if len(result.Plan.Transforms) != 2 {
		t.Fatalf("transforms=%#v want dt-b2-a and dt-b3", result.Plan.Transforms)
	}
}

func TestProcessorSchemaHandshakeWiring(t *testing.T) {
	processor, err := NewProcessor(map[string]string{"default": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	unenforced := processor.SchemaHandshake(9)
	if unenforced.Status != StatusSync || unenforced.ServerEpoch != 0 {
		t.Fatalf("nil versioner gate should be open, got %#v", unenforced)
	}
	versioner, err := NewSchemaVersioner(2, []SchemaMigration{
		{ID: "m1", FromEpoch: 0, ToEpoch: 1},
		{ID: "m2", FromEpoch: 1, ToEpoch: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	processor.SetSchemaVersioner(versioner)
	same := processor.SchemaHandshake(2)
	if same.Status != StatusSync {
		t.Fatalf("wired same-epoch status=%s want %s", same.Status, StatusSync)
	}
	behind := processor.SchemaHandshake(0)
	if behind.Status != StatusBehindCatchUp || len(behind.Plan.Migrations) != 2 {
		t.Fatalf("wired behind result=%#v", behind)
	}
}
