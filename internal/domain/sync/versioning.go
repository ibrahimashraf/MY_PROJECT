package sync

import (
	"errors"
	"fmt"
)

// SchemaEpoch is a monotonic, monotonically-increasing integer that identifies
// a server schema generation. Epochs are assigned in release order and never
// reused or decreased, so a tablet that has been offline for 30+ days can be
// located relative to the server purely by integer comparison — no wall-clock
// drift is involved in the handshake.
type SchemaEpoch uint64

// HandshakeStatus is the terminal outcome of a schema epoch handshake.
type HandshakeStatus string

const (
	// StatusSync is the fast path: the tablet's epoch equals the server's.
	// The tablet may sync normally with no schema actions.
	StatusSync HandshakeStatus = "SYNC"
	// StatusBehindCatchUp means the tablet is behind and must fetch the
	// ordered migrations and transforms in the plan before syncing data.
	StatusBehindCatchUp HandshakeStatus = "BEHIND_CATCH_UP"
	// StatusAheadRejected means the tablet reports an epoch above the server's.
	// The server never auto-downgrades: the tablet is rejected and must downgrade.
	StatusAheadRejected HandshakeStatus = "AHEAD_REJECTED"
	// StatusUnknownRejected means the tablet's epoch is not part of the server's
	// known lineage. The server fails closed rather than guessing.
	StatusUnknownRejected HandshakeStatus = "UNKNOWN_REJECTED"
)

// SchemaMigration is one ordered, contiguous schema step in the server's
// known lineage. FromEpoch+1 == ToEpoch for every step in a valid ladder.
type SchemaMigration struct {
	ID        string
	FromEpoch SchemaEpoch
	ToEpoch   SchemaEpoch
	// Transforms are ordered data transforms applied during this step (e.g.
	// re-keying a payload field). They are echoed in the catch-up plan.
	Transforms []DataTransform
}

// DataTransform is one named, ordered data operation the server sends to a
// behind tablet so its local data converges with the new schema without loss.
type DataTransform struct {
	ID    string
	Order int
	Label string
}

// CatchUpPlan is what the server sends to a behind tablet. MinSchema is the
// epoch the tablet must reach; Migrations are the ordered steps in between.
type CatchUpPlan struct {
	MinSchema  SchemaEpoch
	Migrations []SchemaMigration
	Transforms []DataTransform
}

// HandshakeResult is the pure outcome of comparing a tablet epoch against the
// server's known lineage.
type HandshakeResult struct {
	Status      HandshakeStatus
	ServerEpoch SchemaEpoch
	TabletEpoch SchemaEpoch
	Reason      string
	Plan        *CatchUpPlan
}

// SchemaVersioner owns the server's known schema lineage and answers tablet
// handshakes deterministically. It is immutable after construction.
type SchemaVersioner struct {
	current    SchemaEpoch
	migrations []SchemaMigration
	epochs     map[SchemaEpoch]int
}

// NewSchemaVersioner validates the ladder and builds the versioner. The ladder
// must start at epoch 0, advance linearly (each migration's FromEpoch must be
// the previous ToEpoch) and end at current. Epoch jumps between rungs are
// allowed (releases batch many rungs at once); any integer that is not a rung
// is an unknown epoch and fails closed. A server with a single un-migrated
// schema passes current=0 and no migrations.
func NewSchemaVersioner(current SchemaEpoch, migrations []SchemaMigration) (*SchemaVersioner, error) {
	epochs := map[SchemaEpoch]int{0: 0}
	if len(migrations) == 0 {
		if current != 0 {
			return nil, errors.New("schema ladder with no migrations must have current epoch 0")
		}
		return &SchemaVersioner{current: current, epochs: epochs}, nil
	}
	if migrations[0].FromEpoch != 0 {
		return nil, errors.New("schema ladder must start at epoch 0")
	}
	for i, step := range migrations {
		if step.FromEpoch >= step.ToEpoch {
			return nil, fmt.Errorf("schema migration %q is not ascending: %d -> %d", step.ID, step.FromEpoch, step.ToEpoch)
		}
		if i > 0 && step.FromEpoch != migrations[i-1].ToEpoch {
			return nil, fmt.Errorf("schema ladder is not linear: %d -> %d", migrations[i-1].ToEpoch, step.FromEpoch)
		}
		epochs[step.ToEpoch] = i + 1
	}
	if migrations[len(migrations)-1].ToEpoch != current {
		return nil, fmt.Errorf("schema ladder does not reach current epoch %d", current)
	}
	return &SchemaVersioner{current: current, migrations: migrations, epochs: epochs}, nil
}

// CurrentEpoch returns the server's current schema epoch.
func (v *SchemaVersioner) CurrentEpoch() SchemaEpoch {
	return v.current
}

// Handshake compares a tablet's schema epoch against the server's lineage.
// Deterministic: no wall clock or random input participates.
func (v *SchemaVersioner) Handshake(tabletEpoch SchemaEpoch) HandshakeResult {
	base := HandshakeResult{ServerEpoch: v.current, TabletEpoch: tabletEpoch}
	switch {
	case tabletEpoch == v.current:
		base.Status = StatusSync
		return base
	case tabletEpoch > v.current:
		base.Status = StatusAheadRejected
		base.Reason = fmt.Sprintf("tablet schema epoch %d is ahead of server %d; server never auto-downgrades", tabletEpoch, v.current)
		return base
	case !v.known(tabletEpoch):
		base.Status = StatusUnknownRejected
		base.Reason = fmt.Sprintf("tablet schema epoch %d is not part of the server's known lineage", tabletEpoch)
		return base
	}
	base.Status = StatusBehindCatchUp
	v.fillPlan(&base)
	return base
}

func (v *SchemaVersioner) known(epoch SchemaEpoch) bool {
	_, ok := v.epochs[epoch]
	return ok
}

func (v *SchemaVersioner) fillPlan(result *HandshakeResult) {
	plan := &CatchUpPlan{MinSchema: v.current}
	start := v.epochs[result.TabletEpoch]
	plan.Migrations = append([]SchemaMigration(nil), v.migrations[start:]...)
	for _, migration := range plan.Migrations {
		plan.Transforms = append(plan.Transforms, migration.Transforms...)
	}
	result.Plan = plan
}
