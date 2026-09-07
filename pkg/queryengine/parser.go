package queryengine

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseQuery decodes PostgREST-style URL query parameters into a Query.
//
// Recognized parameter shapes:
//
//	field=eq.value    – equality filter
//	field=gte.value   – greater-or-equal filter
//	field=lte.value   – less-or-equal filter
//	order=field.asc   – ascending sort  (repeatable)
//	order=field.desc  – descending sort (repeatable)
//	limit=N
//	offset=N
//
// Unknown parameters are silently ignored so that additional URL params can
// coexist with the query engine.
func ParseQuery(params url.Values) (Query, error) {
	var q Query
	seenOps := make(map[string]bool)

	for key, values := range params {
		for _, raw := range values {
			switch key {
			case "limit":
				n, err := strconv.Atoi(raw)
				if err != nil {
					return Query{}, fmt.Errorf("invalid limit %q: %w", raw, err)
				}
				q.Limit = n

			case "offset":
				n, err := strconv.Atoi(raw)
				if err != nil {
					return Query{}, fmt.Errorf("invalid offset %q: %w", raw, err)
				}
				q.Offset = n

			case "order":
				for _, part := range strings.Split(raw, ",") {
					part = strings.TrimSpace(part)
					if part == "" {
						continue
					}
					lower := strings.ToLower(part)
					if strings.HasSuffix(lower, ".desc") {
						q.Sorts = append(q.Sorts, Sort{
							Field: part[:len(part)-5],
							Desc:  true,
						})
					} else if strings.HasSuffix(lower, ".asc") {
						q.Sorts = append(q.Sorts, Sort{
							Field: part[:len(part)-4],
							Desc:  false,
						})
					} else {
						return Query{}, fmt.Errorf("order value %q must end with .asc or .desc", part)
					}
				}

			default:
				// Try to parse as operator filter: field=op.value
				op, value, ok := parseOpValue(raw)
				if !ok {
					// Not a recognized operator filter — skip.
					continue
				}
				fieldKey := key
				if seenOps[fieldKey] {
					return Query{}, fmt.Errorf("duplicate filter for field %q", fieldKey)
				}
				seenOps[fieldKey] = true
				q.Filters = append(q.Filters, Filter{
					Field: fieldKey,
					Op:    op,
					Value: value,
				})
			}
		}
	}

	return q, nil
}

// parseOpValue attempts to split "op.value" and returns (Op, value, true).
func parseOpValue(raw string) (Op, string, bool) {
	parts := strings.SplitN(raw, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	op := Op(strings.ToLower(parts[0]))
	if !ValidOps[op] {
		return "", "", false
	}
	return op, parts[1], true
}
