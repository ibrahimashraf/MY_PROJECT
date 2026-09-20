package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestWorkOrderPersistenceMigrationContract(t *testing.T) {
	sql, err := os.ReadFile("0009b_work_order_persistence.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{
		"CREATE TABLE IF NOT EXISTS work_order",
		"CREATE TABLE IF NOT EXISTS work_order_scope_item",
		"CREATE TABLE IF NOT EXISTS work_order_assignment",
		"CREATE TABLE IF NOT EXISTS work_order_assignment_scope",
		"CREATE TABLE IF NOT EXISTS inspection_record",
		"CREATE TABLE IF NOT EXISTS work_order_submission_segment",
		"CREATE TABLE IF NOT EXISTS work_order_submission_item",
		"CREATE TABLE IF NOT EXISTS work_order_operation",
		"CREATE TABLE IF NOT EXISTS work_order_state_event",
		"CREATE TABLE IF NOT EXISTS work_order_provisional_record",
		"tenant_id TEXT NOT NULL",
		"organization_id TEXT NOT NULL",
		"request_state TEXT NOT NULL CONSTRAINT",
		"execution_state TEXT NOT NULL CONSTRAINT",
		"commercial_state TEXT NOT NULL CONSTRAINT",
		"certificate_state TEXT NOT NULL CONSTRAINT",
		"revision BIGINT NOT NULL CHECK (revision > 0)",
		"work_order_operation_idempotency_idx",
		"work_order_provisional_record_local_idx",
		"CREATE UNIQUE INDEX IF NOT EXISTS work_order_provisional_record_local_idx",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("work-order migration missing %q", fragment)
		}
	}
	if _, err := os.Stat("0009b_work_order_persistence.down.sql"); err != nil {
		t.Fatal(err)
	}
}

func TestWorkOrderPersistenceMigrationStatesAndRelationsContract(t *testing.T) {
	sql, err := os.ReadFile("0009b_work_order_persistence.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{
		"work_order_request_state_ck",
		"work_order_execution_state_ck",
		"work_order_commercial_state_ck",
		"work_order_certificate_state_ck",
		"work_order_assignment_state_ck",
		"inspection_record_lifecycle_state_ck",
		"inspection_record_finalization_state_ck",
		"work_order_submission_segment_state_ck",
		"work_order_provisional_record_state_ck",
		"work_order_scope_item_order_fk",
		"work_order_assignment_scope_assignment_fk",
		"work_order_assignment_scope_item_fk",
		"work_order_operation_receipt_jsonb_ck",
		"work_order_state_event_operation_fk",
		"DEFERRABLE INITIALLY DEFERRED",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("work-order relationship/state contract missing %q", fragment)
		}
	}
}

func TestWorkOrderPersistenceMigrationRollbackContract(t *testing.T) {
	sql, err := os.ReadFile("0009b_work_order_persistence.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(sql)
	required := []string{
		"DROP TABLE IF EXISTS work_order_state_event",
		"DROP TABLE IF EXISTS work_order_operation",
		"DROP TABLE IF EXISTS work_order_submission_item",
		"DROP TABLE IF EXISTS work_order_submission_segment",
		"DROP TABLE IF EXISTS inspection_record",
		"DROP TABLE IF EXISTS work_order_assignment_scope",
		"DROP TABLE IF EXISTS work_order_assignment",
		"DROP TABLE IF EXISTS work_order_scope_item",
		"DROP TABLE IF EXISTS work_order_provisional_record",
		"DROP TABLE IF EXISTS work_order",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("work-order rollback missing %q", fragment)
		}
	}
}
