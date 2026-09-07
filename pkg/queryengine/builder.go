package queryengine

import (
	"fmt"
	"strings"
)

// BuildSELECT produces a parameterized SELECT statement for a dynamic query.
// It always prepends the mandatory multi-tenant guard and emits only $n
// placeholders for values — never string-concatenating user input.
//
// Placeholder layout:
//
//	$1 => tenant_id
//	$2 => organization_id
//	$3..N => filter values (one per Filter, in order)
//	$N+1 => LIMIT
//	$N+2 => OFFSET
//
// The caller must populate $1/$2 with the authenticated tenant and
// organization IDs before execution.
//
// Parameters:
//   - table:          target table; must match [a-z_]+
//   - allowedFields:  whitelist of permitted field names
//   - q:              the parsed Query
func BuildSELECT(table string, allowedFields []string, q Query) (string, []any, error) {
	if !validIdent.MatchString(table) {
		return "", nil, fmt.Errorf("invalid table name %q", table)
	}

	if err := q.Validate(allowedFields); err != nil {
		return "", nil, err
	}

	var (
		sb   strings.Builder
		args []any
	)

	sb.WriteString("/* queryengine:tenant_guard */ SELECT * FROM ")
	sb.WriteString(table)
	sb.WriteString(" WHERE tenant_id = $1 AND organization_id = $2")
	args = append(args, nil, nil) // callers fill tenant + org placeholders

	idx := 3
	for _, f := range q.Filters {
		sb.WriteString(" AND ")
		sb.WriteString(f.Field)
		switch f.Op {
		case Eq:
			sb.WriteString(fmt.Sprintf(" = $%d", idx))
		case Gte:
			sb.WriteString(fmt.Sprintf(" >= $%d", idx))
		case Lte:
			sb.WriteString(fmt.Sprintf(" <= $%d", idx))
		}
		args = append(args, f.Value)
		idx++
	}

	if len(q.Sorts) > 0 {
		sb.WriteString(" ORDER BY ")
		for i, s := range q.Sorts {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(s.Field)
			if s.Desc {
				sb.WriteString(" DESC")
			} else {
				sb.WriteString(" ASC")
			}
		}
	}

	sb.WriteString(fmt.Sprintf(" LIMIT $%d", idx))
	args = append(args, q.Limit)
	idx++

	sb.WriteString(fmt.Sprintf(" OFFSET $%d", idx))
	args = append(args, q.Offset)

	return sb.String(), args, nil
}
