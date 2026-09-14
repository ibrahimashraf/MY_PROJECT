package queryengine

import (
	"errors"
	"fmt"
	"sort"
)

var (
	// ErrUnknownTable reports a query against a table not registered in the
	// schema registry. Unmapped tables are never allowed to build SQL.
	ErrUnknownTable = errors.New("query table is not registered in the schema guard")
	// ErrInvalidSchema reports a malformed registered schema definition.
	ErrInvalidSchema = errors.New("registered table schema is invalid")
)

// SchemaRegistry is the authoritative column-whitelist guard: it maps
// registered table names to their verified column lists and refuses to build
// any SELECT for an unregistered table or a column outside the whitelist.
// The whitelist is the single source of truth — callers never supply fields
// at build time, so a typo cannot widen a table's exposure.
type SchemaRegistry struct {
	tables map[string][]string // table → sorted allowed columns
}

// NewSchemaRegistry builds a guard from static schema definitions. Each table
// must be a [a-z_]+ identifier with at least one [a-z_]+ column and no
// duplicates.
func NewSchemaRegistry(schemas map[string][]string) (*SchemaRegistry, error) {
	r := &SchemaRegistry{tables: make(map[string][]string, len(schemas))}
	for table, columns := range schemas {
		if err := r.Register(table, columns); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Register adds or replaces the authoritative column whitelist for a table.
func (r *SchemaRegistry) Register(table string, columns []string) error {
	if !validIdent.MatchString(table) {
		return fmt.Errorf("%w: table %q", ErrInvalidSchema, table)
	}
	clean := make([]string, 0, len(columns))
	seen := make(map[string]struct{}, len(columns))
	for _, col := range columns {
		if !validIdent.MatchString(col) {
			return fmt.Errorf("%w: column %q of %q", ErrInvalidSchema, col, table)
		}
		if _, dup := seen[col]; dup {
			return fmt.Errorf("%w: duplicate column %q in %q", ErrInvalidSchema, col, table)
		}
		seen[col] = struct{}{}
		clean = append(clean, col)
	}
	if len(clean) == 0 {
		return fmt.Errorf("%w: table %q must expose at least one column", ErrInvalidSchema, table)
	}
	sort.Strings(clean)
	r.tables[table] = clean
	return nil
}

// Columns returns the registered whitelist for a table.
func (r *SchemaRegistry) Columns(table string) ([]string, bool) {
	cols, ok := r.tables[table]
	return append([]string(nil), cols...), ok
}

// BuildSELECT produces a parameterized SELECT for a registered table. It
// rejects unknown tables and any field outside the registered whitelist
// BEFORE any SQL text is generated.
func (r *SchemaRegistry) BuildSELECT(table string, q Query) (string, []any, error) {
	columns, ok := r.tables[table]
	if !ok {
		return "", nil, fmt.Errorf("%w: %q", ErrUnknownTable, table)
	}
	return buildSELECT(table, columns, q)
}

// Tables returns the sorted list of registered table names.
func (r *SchemaRegistry) Tables() []string {
	out := make([]string, 0, len(r.tables))
	for table := range r.tables {
		out = append(out, table)
	}
	sort.Strings(out)
	return out
}
