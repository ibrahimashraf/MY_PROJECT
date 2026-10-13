package queryengine

import (
	"fmt"
	"regexp"
)

// Op represents a comparison operator for filters.
type Op string

const (
	Eq  Op = "eq"
	Gte Op = "gte"
	Lte Op = "lte"
)

// ValidOps lists the operators accepted by the query engine.
var ValidOps = map[Op]bool{
	Eq:  true,
	Gte: true,
	Lte: true,
}

// Filter represents a single field comparison.
type Filter struct {
	Field string
	Op    Op
	Value string
}

// Sort represents a sort clause.
type Sort struct {
	Field string
	Desc  bool
}

// Query holds the full parsed representation of a dynamic query.
type Query struct {
	Filters []Filter
	Sorts   []Sort
	Limit   int
	Offset  int
}

const (
	defaultLimit = 20
	maxLimit     = 100
)

// validIdent matches identifiers permitted in generated SQL. Following the
// PostgREST-inspired constraint, only lowercase letters and underscores are
// accepted; anything else must be rejected outright so no identifier can ever
// smuggle SQL text, even when supplied through a caller-provided allow-list.
var validIdent = regexp.MustCompile(`^[a-z_]+$`)

// Validate checks the Query against the caller-supplied allow-list of fields
// and enforces the limit/offset constraints. Field names must both be present
// in the allow-list and match the [a-z_]+ identifier grammar.
func (q *Query) Validate(allowedFields []string) error {
	allowed := make(map[string]bool, len(allowedFields))
	for _, f := range allowedFields {
		allowed[f] = true
	}

	for _, f := range q.Filters {
		if !allowed[f.Field] {
			return fmt.Errorf("filter field %q not in allowed list", f.Field)
		}
		if !validIdent.MatchString(f.Field) {
			return fmt.Errorf("filter field %q contains disallowed characters", f.Field)
		}
		if !ValidOps[f.Op] {
			return fmt.Errorf("filter operator %q not supported", f.Op)
		}
	}

	for _, s := range q.Sorts {
		if !allowed[s.Field] {
			return fmt.Errorf("sort field %q not in allowed list", s.Field)
		}
		if !validIdent.MatchString(s.Field) {
			return fmt.Errorf("sort field %q contains disallowed characters", s.Field)
		}
	}

	if q.Limit <= 0 {
		q.Limit = defaultLimit
	}
	if q.Limit > maxLimit {
		q.Limit = maxLimit
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	return nil
}
