package queryengine

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildSELECTTenantGuardAlwaysPresent(t *testing.T) {
	sql, args, err := BuildSELECT("licenses", []string{"name"}, Query{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "WHERE tenant_id = $1 AND organization_id = $2") {
		t.Errorf("tenant guard missing from SQL: %s", sql)
	}
	if !strings.Contains(sql, "/* queryengine:tenant_guard */") {
		t.Errorf("tenant guard comment missing: %s", sql)
	}
	if len(args) != 4 {
		t.Fatalf("want 4 args (tenant, org, limit, offset), got %d: %v", len(args), args)
	}
}

func TestBuildSELECTPlaceholderNumbering(t *testing.T) {
	sql, args, err := BuildSELECT("licenses", []string{"name", "age"}, Query{
		Filters: []Filter{
			{Field: "name", Op: Eq, Value: "alice"},
			{Field: "age", Op: Gte, Value: "21"},
		},
		Limit:  10,
		Offset: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "/* queryengine:tenant_guard */ SELECT * FROM licenses WHERE tenant_id = $1 AND organization_id = $2" +
		" AND name = $3 AND age >= $4" +
		" LIMIT $5 OFFSET $6"
	if sql != want {
		t.Fatalf("sql mismatch:\n got: %s\nwant: %s", sql, want)
	}

	if len(args) != 6 {
		t.Fatalf("want 6 args, got %d: %v", len(args), args)
	}
	if args[2] != "alice" || args[3] != "21" || args[4] != 10 || args[5] != 5 {
		t.Errorf("arg mismatch: %v", args)
	}
}

func TestBuildSELECTAllOps(t *testing.T) {
	sql, _, err := BuildSELECT("tickets", []string{"a", "b", "c"}, Query{
		Filters: []Filter{
			{Field: "a", Op: Eq, Value: "1"},
			{Field: "b", Op: Gte, Value: "2"},
			{Field: "c", Op: Lte, Value: "3"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Exact operators and sequential placeholder numbering.
	if !strings.Contains(sql, "AND a = $3 AND b >= $4 AND c <= $5") {
		t.Errorf("operators/placeholders wrong: %s", sql)
	}
}

func TestBuildSELECTSorts(t *testing.T) {
	sql, _, err := BuildSELECT("licenses", []string{"name", "created_at"}, Query{
		Sorts: []Sort{
			{Field: "created_at", Desc: true},
			{Field: "name", Desc: false},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "ORDER BY created_at DESC, name ASC") {
		t.Errorf("order by clause wrong: %s", sql)
	}
}

func TestBuildSELECTInjectionRejected(t *testing.T) {
	cases := []struct {
		name  string
		q     Query
		table string
	}{
		{
			name:  "sql in filter field",
			table: "licenses",
			q:     Query{Filters: []Filter{{Field: "1=1;DROP", Op: Eq, Value: "x"}}},
		},
		{
			name:  "semicolon in filter field",
			table: "licenses",
			q:     Query{Filters: []Filter{{Field: "name; DROP TABLE x", Op: Eq, Value: "y"}}},
		},
		{
			name:  "uppercase filter field",
			table: "licenses",
			q:     Query{Filters: []Filter{{Field: "NAME", Op: Eq, Value: "y"}}},
		},
		{
			name:  "space in sort field",
			table: "licenses",
			q:     Query{Sorts: []Sort{{Field: "name DESC --", Desc: true}}},
		},
		{
			name:  "sql in table name",
			table: "licenses; DROP TABLE users",
			q:     Query{},
		},
		{
			name:  "quoted table name",
			table: `"licenses"`,
			q:     Query{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := BuildSELECT(tc.table, nil, tc.q); err == nil {
				t.Fatalf("want error for %q, got nil", tc.name)
			}
		})
	}
}

func TestBuildSELECTUnknownFieldRejected(t *testing.T) {
	_, _, err := BuildSELECT("licenses", []string{"name"}, Query{
		Filters: []Filter{{Field: "email", Op: Eq, Value: "a@b.c"}},
	})
	if err == nil {
		t.Fatal("want error for field not in allowed list")
	}
}

func TestBuildSELECTLimitClamping(t *testing.T) {
	// Validate applies clamping: <=0 → default 20; >100 → 100; offset <0 → 0.
	q := Query{Limit: 1000, Offset: -3}
	if err := q.Validate([]string{"name"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Limit != maxLimit || q.Offset != 0 {
		t.Errorf("want limit=%d offset=0, got limit=%d offset=%d", maxLimit, q.Limit, q.Offset)
	}

	q2 := Query{}
	if err := q2.Validate([]string{"name"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q2.Limit != defaultLimit {
		t.Errorf("want default limit=%d, got %d", defaultLimit, q2.Limit)
	}

	// Values travel as args, never in the SQL text.
	sql, args, err := BuildSELECT("licenses", []string{"name"}, Query{Limit: 1000, Offset: -3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(sql, "1000") || strings.Contains(sql, "-3") {
		t.Errorf("clamped values leaked into SQL text: %s", sql)
	}
	if args[len(args)-2] != maxLimit || args[len(args)-1] != 0 {
		t.Errorf("want clamped limit/offset args, got %v", args)
	}
}

func TestBuildSELECTValueNeverConcatenated(t *testing.T) {
	sql, args, err := BuildSELECT("licenses", []string{"name"}, Query{
		Filters: []Filter{{Field: "name", Op: Eq, Value: "x'); DROP TABLE auditors; --"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The hostile value must appear only as a placeholder (in args), never in SQL.
	if strings.Contains(sql, "DROP") || strings.Contains(sql, ";") {
		t.Errorf("hostile value leaked into SQL text: %s", sql)
	}
	// len = 2 (tenant/org) + 1 filter + limit + offset.
	if len(args) != 5 || args[2] != "x'); DROP TABLE auditors; --" {
		t.Errorf("hostile value not preserved as arg: %v", args)
	}
}

func TestBuildSELECTFiltersUniqueAndNoValueSQL(t *testing.T) {
	sql, args, err := BuildSELECT("licenses", []string{"name", "age"}, Query{
		Filters: []Filter{
			{Field: "name", Op: Eq, Value: "alice"},
			{Field: "age", Op: Gte, Value: "21"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only $n placeholders are permitted — no raw values in SQL.
	if strings.Contains(sql, "alice") || strings.Contains(sql, "21") {
		t.Errorf("filter values leaked into SQL: %s", sql)
	}
	if args[2] != "alice" || args[3] != "21" {
		t.Errorf("filter values wrong in args: %v", args)
	}
}

func TestBuildSELECTRoundTripFromParser(t *testing.T) {
	q, err := ParseQuery(url.Values{
		"name":  {"eq.bob"},
		"order": {"created_at.desc"},
		"limit": {"30"},
	})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	sql, args, err := BuildSELECT("licenses", []string{"name", "created_at"}, q)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}
	if !strings.Contains(sql, "AND name = $3") || !strings.Contains(sql, "ORDER BY created_at DESC") {
		t.Errorf("round-trip SQL wrong: %s", sql)
	}
	if args[2] != "bob" || args[len(args)-2] != 30 {
		t.Errorf("round-trip args wrong: %v", args)
	}
}

func TestBuildSELECTUnknownParamsIgnoredByParser(t *testing.T) {
	q, err := ParseQuery(url.Values{
		"name":   {"eq.ada"},
		"token":  {"something"},
		"cursor": {"abc.def"},
	})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(q.Filters) != 1 {
		t.Fatalf("want 1 filter, got %d", len(q.Filters))
	}
	if _, _, err := BuildSELECT("licenses", []string{"name"}, q); err != nil {
		t.Fatalf("build error: %v", err)
	}
}
