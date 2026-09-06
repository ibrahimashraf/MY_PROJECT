package chaos_test

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestPgCatTransactionPoolingAndZeroGUCLeakProof executes a high-concurrency proof of
// isolation against PgCat (port 6432) operating in transaction pooling mode.
//
// It verifies:
// 1. Transaction-local GUCs (set_config('integin.tenant_id', ..., true)) are completely discarded
//    upon COMMIT/ROLLBACK and never leak to another tenant reusing the same backend connection.
// 2. Client connections configured with default_query_exec_mode=exec or simple_protocol
//    operate without prepared statement collisions (SQLSTATE 26000) under high connection multiplexing.
// 3. Row-level security (RLS) under integin_runtime prevents cross-tenant reads even when
//    dozens of goroutines alternate rapid transactions across the shared PgCat pool.
func TestPgCatTransactionPoolingAndZeroGUCLeakProof(t *testing.T) {
	pgcatURL := os.Getenv("INTEGIN_PGCAT_URL")
	if pgcatURL == "" {
		pgcatURL = "postgres://postgres:postgres_local_test_password@127.0.0.1:6432/integin_migration_test?sslmode=disable&default_query_exec_mode=exec"
	}

	db, err := sql.Open("pgx", pgcatURL)
	if err != nil {
		t.Skipf("PgCat pooler not available: %v", err)
	}
	defer db.Close()

	// High pool size on client side to maximize multiplexing through PgCat's backend pool
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("PgCat ping failed at %s: %v", pgcatURL, err)
	}

	const (
		numTenants  = 10
		iterations  = 30
		concurrency = 15
	)

	// Fixture setup: Create tenant-specific rows under test
	type tenantFixture struct {
		tenantID    string
		orgID       string
		workOrderID string
	}

	fixtures := make([]tenantFixture, numTenants)
	for i := 0; i < numTenants; i++ {
		fixtures[i] = tenantFixture{
			tenantID:    fmt.Sprintf("pgcat-tenant-%d", i),
			orgID:       fmt.Sprintf("pgcat-org-%d", i),
			workOrderID: fmt.Sprintf("wo-pgcat-%d", i),
		}
	}

	// Clean up and seed fixtures
	for _, f := range fixtures {
		seedTx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("failed to begin seed tx: %v", err)
		}
		if _, err := seedTx.ExecContext(ctx, "SET LOCAL ROLE integin_runtime"); err != nil {
			seedTx.Rollback()
			t.Fatalf("set local role failed: %v", err)
		}
		if _, err := seedTx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", f.tenantID, f.orgID); err != nil {
			seedTx.Rollback()
			t.Fatalf("set_config failed: %v", err)
		}
		_, _ = seedTx.ExecContext(ctx, "DELETE FROM work_order WHERE id = $1", f.workOrderID)
		_, err = seedTx.ExecContext(ctx, `
			INSERT INTO work_order (
				id, tenant_id, organization_id, client_id, job_number,
				request_state, execution_state, commercial_state, certificate_state, revision, created_by, updated_by
			) VALUES (
				$1, $2, $3, 'client-p', 'JOB-PGCAT', 'accepted', 'in_progress', 'ready_for_office_confirmation', 'pending_validation', 1, 'seeder', 'seeder'
			)
		`, f.workOrderID, f.tenantID, f.orgID)
		if err != nil {
			seedTx.Rollback()
			t.Fatalf("seed work_order failed: %v", err)
		}
		if err := seedTx.Commit(); err != nil {
			t.Fatalf("commit seed tx failed: %v", err)
		}
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		for _, f := range fixtures {
			cleanTx, _ := db.BeginTx(cleanupCtx, nil)
			if cleanTx != nil {
				_, _ = cleanTx.ExecContext(cleanupCtx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", f.tenantID, f.orgID)
				_, _ = cleanTx.ExecContext(cleanupCtx, "DELETE FROM work_order WHERE id = $1", f.workOrderID)
				_ = cleanTx.Commit()
			}
		}
	})

	t.Logf("Seeded %d tenant fixtures. Launching %d concurrent workers executing %d transactions each...", numTenants, concurrency, iterations)

	var wg sync.WaitGroup
	errChan := make(chan error, concurrency*iterations*2)

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for it := 0; it < iterations; it++ {
				// Pick a tenant
				targetIdx := rng.Intn(numTenants)
				current := fixtures[targetIdx]

				// Step 1: Open a transaction and inspect GUC before setting it.
				// In transaction pooling mode, when a connection is checked out from the pool,
				// any session state from previous tenants on that backend connection MUST NOT be present.
				tx, err := db.BeginTx(ctx, nil)
				if err != nil {
					errChan <- fmt.Errorf("worker %d: begin tx: %w", workerID, err)
					return
				}

				var priorTenant, priorOrg sql.NullString
				if err := tx.QueryRowContext(ctx, "SELECT current_setting('integin.tenant_id', true), current_setting('integin.organization_id', true)").Scan(&priorTenant, &priorOrg); err != nil {
					tx.Rollback()
					errChan <- fmt.Errorf("worker %d: check initial GUC: %w", workerID, err)
					return
				}

				if (priorTenant.Valid && priorTenant.String != "") || (priorOrg.Valid && priorOrg.String != "") {
					tx.Rollback()
					errChan <- fmt.Errorf("FATAL GUC LEAK DETECTED! Worker %d saw dirty session GUCs on fresh tx: tenant=%q, org=%q", workerID, priorTenant.String, priorOrg.String)
					return
				}

				// Step 2: Switch to integin_runtime locally and set transaction-local GUC
				if _, err := tx.ExecContext(ctx, "SET LOCAL ROLE integin_runtime"); err != nil {
					tx.Rollback()
					errChan <- fmt.Errorf("worker %d: set local role: %w", workerID, err)
					return
				}
				if _, err := tx.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)", current.tenantID, current.orgID); err != nil {
					tx.Rollback()
					errChan <- fmt.Errorf("worker %d: set_config: %w", workerID, err)
					return
				}

				// Step 3: Query own work order under RLS
				var ownCount int
				if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM work_order WHERE id = $1", current.workOrderID).Scan(&ownCount); err != nil {
					tx.Rollback()
					errChan <- fmt.Errorf("worker %d: query own row: %w", workerID, err)
					return
				}
				if ownCount != 1 {
					tx.Rollback()
					errChan <- fmt.Errorf("worker %d: expected 1 own work order for %s, got %d", workerID, current.tenantID, ownCount)
					return
				}

				// Step 4: Query another tenant's work order (must return 0 under RLS)
				otherIdx := (targetIdx + 1) % numTenants
				other := fixtures[otherIdx]
				var foreignCount int
				if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM work_order WHERE id = $1", other.workOrderID).Scan(&foreignCount); err != nil {
					tx.Rollback()
					errChan <- fmt.Errorf("worker %d: query foreign row: %w", workerID, err)
					return
				}
				if foreignCount != 0 {
					tx.Rollback()
					errChan <- fmt.Errorf("SECURITY BREACH! Worker %d authenticated as %s saw foreign work order for %s!", workerID, current.tenantID, other.tenantID)
					return
				}

				// 50% commit, 50% rollback to exercise both transaction terminal states
				if rng.Intn(2) == 0 {
					if err := tx.Commit(); err != nil {
						errChan <- fmt.Errorf("worker %d: commit: %w", workerID, err)
						return
					}
				} else {
					if err := tx.Rollback(); err != nil {
						errChan <- fmt.Errorf("worker %d: rollback: %w", workerID, err)
						return
					}
				}
			}
		}(w)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("PgCat pooler test error: %v", e)
		}
		t.Fatalf("FAILED: Encountered %d errors under PgCat transaction pooling", len(errs))
	}

	t.Logf("SUCCESS: Executed %d transactions across %d concurrent workers through PgCat with ZERO GUC leaks and 100%% RLS isolation.", concurrency*iterations, concurrency)
}
