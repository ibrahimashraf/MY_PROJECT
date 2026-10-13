package queryengine

import (
	"errors"
	"strings"
	"testing"
)

func TestSchemaRegistryRejectsUnmappedTable(t *testing.T) {
	guard, err := NewSchemaRegistry(map[string][]string{"licenses": {"name", "tier"}})
	if err != nil {
		t.Fatalf("NewSchemaRegistry: %v", err)
	}
	if _, _, err := guard.BuildSELECT("black_market", Query{}); !errors.Is(err, ErrUnknownTable) {
		t.Errorf("unmapped table must fail before building, got %v", err)
	}
}

func TestSchemaRegistryStrictColumnRejection(t *testing.T) {
	guard, err := NewSchemaRegistry(map[string][]string{"licenses": {"name", "tier"}})
	if err != nil {
		t.Fatalf("NewSchemaRegistry: %v", err)
	}

	if _, _, err := guard.BuildSELECT("licenses", Query{
		Filters: []Filter{{Field: "expires_at", Op: Eq, Value: "nope"}},
	}); err == nil || !strings.Contains(err.Error(), "not in allowed list") {
		t.Errorf("column outside whitelist must be rejected, got %v", err)
	}

	if _, _, err := guard.BuildSELECT("licenses", Query{
		Sorts: []Sort{{Field: "secret", Desc: true}},
	}); err == nil {
		t.Errorf("sort on unregistered column must be rejected")
	}

	// ALLOWED fields pass and keep the mandatory tenant guard.
	sql, args, err := guard.BuildSELECT("licenses", Query{
		Filters: []Filter{{Field: "tier", Op: Eq, Value: "ENTERPRISE"}},
		Sorts:   []Sort{{Field: "name", Desc: false}},
	})
	if err != nil {
		t.Fatalf("valid query failed: %v", err)
	}
	if !strings.Contains(sql, "WHERE tenant_id = $1 AND organization_id = $2") {
		t.Errorf("tenant guard missing: %s", sql)
	}
	if len(args[:2]) != 2 {
		t.Errorf("expected tenant/org placeholders first, got %v", args)
	}
	if strings.Contains(sql, "ENTERPRISE") {
		t.Errorf("filter value must never be inlined: %s", sql)
	}
}

func TestSchemaRegistryRegisterValidation(t *testing.T) {
	if _, err := NewSchemaRegistry(map[string][]string{"Bad Table": {"a"}}); err == nil {
		t.Error("invalid table name must be rejected")
	}
	guard, _ := NewSchemaRegistry(nil)
	if err := guard.Register("licenses", []string{"name; DROP TABLE x"}); err == nil {
		t.Error("injected column name must be rejected")
	}
	if err := guard.Register("licenses", nil); err == nil {
		t.Error("table with no columns must be rejected")
	}
	if err := guard.Register("licenses", []string{"a", "a"}); err == nil {
		t.Error("duplicate column must be rejected")
	}
}

func TestSchemaRegistryColumnDeterminism(t *testing.T) {
	guard, err := NewSchemaRegistry(map[string][]string{
		"certs": {"id", "status", "issuer"},
	})
	if err != nil {
		t.Fatalf("NewSchemaRegistry: %v", err)
	}
	cols, ok := guard.Columns("certs")
	if !ok || len(cols) != 3 || cols[0] != "id" {
		t.Errorf("unexpected columns: %v (ok=%v)", cols, ok)
	}
	if _, ok := guard.Columns("nope"); ok {
		t.Error("unknown table must not report columns")
	}
	if got := guard.Tables(); len(got) != 1 || got[0] != "certs" {
		t.Errorf("unexpected Tables(): %v", got)
	}
}
