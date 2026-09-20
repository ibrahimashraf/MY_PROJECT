package chaos_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestAdversarialMultiTenantRLSPenetrationSuite(t *testing.T) {
	dbURL := "postgres://postgres:postgres_local_test_password@localhost:15432/integin_migration_test?sslmode=disable"
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Skip("database not available")
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("db ping failed: %v", err)
	}

	const (
		victimTenant    = "victim-corp"
		victimOrg       = "victim-org"
		hackerTenant    = "hacker-corp"
		hackerOrg       = "hacker-org"
		testWorkOrderID = "wo-victim-secret-1"
	)

	// Clean up previous fixtures
	_, _ = db.ExecContext(ctx, "DELETE FROM work_order WHERE id = $1", testWorkOrderID)

	// 1. Seed Victim Data inside an isolated transaction using application role
	seedTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin seed tx: %v", err)
	}
	if _, err := seedTx.ExecContext(ctx, `SET ROLE integin_test_runtime`); err != nil {
		seedTx.Rollback()
		t.Fatalf("failed to set role integin_test_runtime: %v", err)
	}
	_, err = seedTx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, victimTenant, victimOrg)
	if err != nil {
		seedTx.Rollback()
		t.Fatalf("failed to set victim tenant context: %v", err)
	}
	_, err = seedTx.ExecContext(ctx, `
		INSERT INTO work_order (
			id, tenant_id, organization_id, client_id, job_number, 
			request_state, execution_state, commercial_state, certificate_state, revision, created_by, updated_by
		) VALUES (
			$1, $2, $3, 'client-v', 'JOB-V1', 'accepted', 'in_progress', 'ready_for_office_confirmation', 'pending_validation', 1, 'user-v', 'user-v'
		)
	`, testWorkOrderID, victimTenant, victimOrg)
	if err != nil {
		seedTx.Rollback()
		t.Fatalf("failed to insert victim work order: %v", err)
	}
	if err := seedTx.Commit(); err != nil {
		t.Fatalf("failed to commit victim seed tx: %v", err)
	}

	// Clean up at the end
	t.Cleanup(func() {
		cleanupTx, _ := db.BeginTx(context.Background(), nil)
		if cleanupTx != nil {
			_, _ = cleanupTx.ExecContext(context.Background(), `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, victimTenant, victimOrg)
			_, _ = cleanupTx.ExecContext(context.Background(), "DELETE FROM work_order WHERE id = $1", testWorkOrderID)
			_ = cleanupTx.Commit()
		}
	})

	t.Run("Attack 1: Unauthenticated Session Reads 0 Rows", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()

		if _, err := tx.ExecContext(ctx, "SET ROLE integin_test_runtime"); err != nil {
			t.Fatal(err)
		}

		var count int
		err = tx.QueryRowContext(ctx, "SELECT count(*) FROM work_order WHERE id = $1", testWorkOrderID).Scan(&count)
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		if count != 0 {
			t.Fatalf("SECURITY BREACH: unauthenticated query saw %d victim rows!", count)
		}
		t.Log("PASS: Unauthenticated session returned 0 rows under strict RLS")
	})

	t.Run("Attack 2: Empty String GUC Poisoning Reads 0 Rows", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()

		if _, err := tx.ExecContext(ctx, "SET ROLE integin_test_runtime"); err != nil {
			t.Fatal(err)
		}

		// Attempt to bypass with empty strings
		_, err = tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', '', true), set_config('integin.organization_id', '', true)`)
		if err != nil {
			t.Fatal(err)
		}

		var count int
		err = tx.QueryRowContext(ctx, "SELECT count(*) FROM work_order WHERE id = $1", testWorkOrderID).Scan(&count)
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		if count != 0 {
			t.Fatalf("SECURITY BREACH: empty string poison attack saw %d victim rows!", count)
		}
		t.Log("PASS: Empty-string GUC poisoning returned 0 rows")
	})

	t.Run("Attack 3: Cross-Tenant Adversary Reads 0 Rows", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()

		if _, err := tx.ExecContext(ctx, "SET ROLE integin_test_runtime"); err != nil {
			t.Fatal(err)
		}

		// Authenticate as Hacker
		_, err = tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, hackerTenant, hackerOrg)
		if err != nil {
			t.Fatal(err)
		}

		var count int
		err = tx.QueryRowContext(ctx, "SELECT count(*) FROM work_order WHERE id = $1", testWorkOrderID).Scan(&count)
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		if count != 0 {
			t.Fatalf("SECURITY BREACH: hacker saw %d victim rows!", count)
		}
		t.Log("PASS: Cross-tenant adversary saw 0 rows")
	})

	t.Run("Attack 4: Forged Tenant ID Write Rejected by WITH CHECK", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()

		if _, err := tx.ExecContext(ctx, "SET ROLE integin_test_runtime"); err != nil {
			t.Fatal(err)
		}

		// Hacker is authenticated as Hacker
		_, err = tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, hackerTenant, hackerOrg)
		if err != nil {
			t.Fatal(err)
		}

		// Hacker attempts to inject a work order into victim's tenant
		_, err = tx.ExecContext(ctx, `
			INSERT INTO work_order (
				id, tenant_id, organization_id, client_id, job_number, 
				request_state, execution_state, commercial_state, certificate_state, revision, created_by, updated_by
			) VALUES (
				'wo-trojan-1', $1, $2, 'client-h', 'JOB-TROJAN', 'accepted', 'in_progress', 'ready_for_office_confirmation', 'pending_validation', 1, 'hacker', 'hacker'
			)
		`, victimTenant, victimOrg)

		if err == nil {
			t.Fatal("SECURITY BREACH: PostgreSQL permitted cross-tenant forged insert!")
		}
		t.Logf("PASS: Forged cross-tenant insert hard-rejected by RLS WITH CHECK constraint: %v", err)
	})

	t.Run("Attack 5: Legitimate Tenant Read Passes Correctly", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()

		if _, err := tx.ExecContext(ctx, "SET ROLE integin_test_runtime"); err != nil {
			t.Fatal(err)
		}

		// Authenticate as Victim
		_, err = tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)`, victimTenant, victimOrg)
		if err != nil {
			t.Fatal(err)
		}

		var count int
		err = tx.QueryRowContext(ctx, "SELECT count(*) FROM work_order WHERE id = $1", testWorkOrderID).Scan(&count)
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		if count != 1 {
			t.Fatalf("Legitimate tenant could not read own work order! got count = %d", count)
		}
		t.Log("PASS: Legitimate tenant read successfully returned 1 row")
	})
}


