package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestWorkOrderRLSMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0010b_work_order_rls.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	tables := []string{
		"work_order",
		"work_order_scope_item",
		"work_order_assignment",
		"work_order_assignment_scope",
		"inspection_record",
		"work_order_submission_segment",
		"work_order_submission_item",
		"work_order_operation",
		"work_order_state_event",
		"work_order_provisional_record",
	}
	for _, table := range tables {
		for _, fragment := range []string{
			"ALTER TABLE " + table + " ENABLE ROW LEVEL SECURITY",
			"ALTER TABLE " + table + " FORCE ROW LEVEL SECURITY",
			"CREATE POLICY " + table + "_tenant_organization_isolation",
		} {
			if !strings.Contains(text, fragment) {
				t.Fatalf("RLS migration missing %q", fragment)
			}
		}
	}
	if !strings.Contains(text, "current_setting('integin.tenant_id', true)") {
		t.Fatal("RLS migration must bind policies to the transaction-local tenant")
	}
	if !strings.Contains(text, "current_setting('integin.organization_id', true)") {
		t.Fatal("RLS migration must bind policies to the transaction-local organization")
	}
	if !strings.Contains(text, "GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE") {
		t.Fatal("RLS migration must grant work-order DML to integin_runtime")
	}
	if !strings.Contains(text, "REVOKE ALL ON TABLE") {
		t.Fatal("RLS migration must revoke broad public/runtime table privileges before granting least privilege")
	}
	if _, err := os.Stat("0010b_work_order_rls.down.sql"); err != nil {
		t.Fatal(err)
	}
}

func TestWorkOrderRLSMigrationHasWriteChecks(t *testing.T) {
	sql, err := os.ReadFile("0010b_work_order_rls.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	if strings.Count(text, "WITH CHECK") < 10 {
		t.Fatalf("expected write checks for all ten work-order tables, got %d", strings.Count(text, "WITH CHECK"))
	}
}

func TestWorkOrderRLSDMigrationRollbackContract(t *testing.T) {
	sql, err := os.ReadFile("0010b_work_order_rls.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{
		"DROP POLICY IF EXISTS work_order_tenant_organization_isolation ON work_order",
		"ALTER TABLE work_order NO FORCE ROW LEVEL SECURITY",
		"ALTER TABLE work_order DISABLE ROW LEVEL SECURITY",
		"REVOKE ALL ON TABLE work_order",
		"REVOKE ALL ON TABLE",
		"work_order_provisional_record FROM integin_runtime",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("RLS rollback missing %q", fragment)
		}
	}
}
