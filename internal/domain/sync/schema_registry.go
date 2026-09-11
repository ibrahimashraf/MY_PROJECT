package sync

import "fmt"

// currentSchemaEpoch is the server's current schema epoch derived from the
// canonical migration ledger in migrations/. There are 75 canonical .sql
// migrations (0001–0073 contiguous, then 0080 and 0081; 0074–0079 are
// unused). Bump this constant when a new canonical migration lands.
const currentSchemaEpoch SchemaEpoch = 75

// currentSchemaMigrations returns one ordered SchemaMigration step per
// canonical migration file. IDs echo the migration ledger numbers so a
// catch-up plan is traceable to the SQL source.
func currentSchemaMigrations() []SchemaMigration {
	migrations := make([]SchemaMigration, 0, currentSchemaEpoch)
	for i := SchemaEpoch(0); i < currentSchemaEpoch; i++ {
		migrations = append(migrations, SchemaMigration{
			ID:        fmt.Sprintf("migration_%04d", i+1),
			FromEpoch: i,
			ToEpoch:   i + 1,
		})
	}
	return migrations
}

// CurrentSchemaVersioner builds the production SchemaVersioner from the
// migration-ledger lineage. It is nil when the schema has no migrations, and
// it fails closed (error) when the ladder is inconsistent, so callers can
// fall back to the original un-wired gate only on a genuinely empty ledger.
func CurrentSchemaVersioner() (*SchemaVersioner, error) {
	if currentSchemaEpoch == 0 {
		return nil, nil
	}
	return NewSchemaVersioner(currentSchemaEpoch, currentSchemaMigrations())
}
