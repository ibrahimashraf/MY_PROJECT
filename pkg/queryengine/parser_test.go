package queryengine

import (
	"net/url"
	"testing"
)

func TestParseQueryFilters(t *testing.T) {
	q, err := ParseQuery(url.Values{
		"name":    {"eq.alice"},
		"age":     {"gte.21"},
		"score":   {"lte.99"},
		"ignored": {"x.y.z"},
		"email":   {"hello"}, // no operator → skipped
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Filters) != 3 {
		t.Fatalf("want 3 filters, got %d: %+v", len(q.Filters), q.Filters)
	}
	if len(q.Filters) != 3 {
		t.Fatalf("want 3 filters, got %d: %+v", len(q.Filters), q.Filters)
	}
	checkFilter(t, q.Filters[0], "name", Eq, "alice")
	checkFilter(t, q.Filters[1], "age", Gte, "21")
	checkFilter(t, q.Filters[2], "score", Lte, "99")
}

func TestParseQuerySorts(t *testing.T) {
	q, err := ParseQuery(url.Values{
		"order": {"created_at.desc", "name.asc"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Sorts) != 2 {
		t.Fatalf("want 2 sorts, got %d", len(q.Sorts))
	}
	if q.Sorts[0].Field != "created_at" || !q.Sorts[0].Desc {
		t.Errorf("sort[0] = %+v, want created_at.desc", q.Sorts[0])
	}
	if q.Sorts[1].Field != "name" || q.Sorts[1].Desc {
		t.Errorf("sort[1] = %+v, want name.asc", q.Sorts[1])
	}
}

func TestParseQueryLimitOffset(t *testing.T) {
	q, err := ParseQuery(url.Values{"limit": {"50"}, "offset": {"25"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Limit != 50 || q.Offset != 25 {
		t.Errorf("got limit=%d offset=%d, want 50/25", q.Limit, q.Offset)
	}
}

func TestParseQueryRawValuesBeforeValidate(t *testing.T) {
	q, err := ParseQuery(url.Values{"name": {"eq.bob"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Defaults and clamping are applied only by Validate, not the parser.
	if q.Limit != 0 || q.Offset != 0 {
		t.Errorf("want zero-valued defaults before Validate, got %d/%d", q.Limit, q.Offset)
	}
}

func TestParseQueryErrors(t *testing.T) {
	cases := []struct {
		name   string
		params url.Values
	}{
		{"bad limit", url.Values{"limit": {"abc"}}},
		{"bad offset", url.Values{"offset": {"abc"}}},
		{"bad order", url.Values{"order": {"name"}}},
		{"bad order suffix", url.Values{"order": {"name.median"}}},
		{"duplicate filter", url.Values{"name": {"eq.a", "gte.b"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseQuery(tc.params); err == nil {
				t.Errorf("ParseQuery(%v) want error, got nil", tc.params)
			}
		})
	}
}

func TestParseQueryEmpty(t *testing.T) {
	q, err := ParseQuery(url.Values{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Filters) != 0 || len(q.Sorts) != 0 {
		t.Errorf("want empty query, got %+v", q)
	}
}

func TestParseQueryOrderCSV(t *testing.T) {
	q, err := ParseQuery(url.Values{"order": {"a.desc,b.asc"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Sorts) != 2 {
		t.Fatalf("want 2 sorts from CSV, got %d", len(q.Sorts))
	}
}

func checkFilter(t *testing.T, got Filter, field string, op Op, value string) {
	t.Helper()
	if got.Field != field || got.Op != op || got.Value != value {
		t.Errorf("filter = %+v, want {%s %s %s}", got, field, op, value)
	}
}
