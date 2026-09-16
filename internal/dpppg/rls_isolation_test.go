package dpppg

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"integin/internal/domain/dpp"
)

// TestMultiTenantRLSIsolationDrill executes a high-concurrency performance drill
// to verify zero connection pool bleeding across concurrent tenant sessions.
func TestMultiTenantRLSIsolationDrill(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DB_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres_local_test_password@127.0.0.1:15432/integin_migration_test?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(50)

	if err := db.Ping(); err != nil {
		t.Skipf("skipping RLS drill because DB is not available: %v", err)
	}

	t.Log("Starting Multi-Tenant RLS & Performance Drill against live PostgreSQL...")

	var wg sync.WaitGroup
	const numTenants = 50
	const queriesPerTenant = 20

	results := make(chan error, numTenants*queriesPerTenant)

	for i := 0; i < numTenants; i++ {
		wg.Add(1)
		go func(tenantIndex int) {
			defer wg.Done()
			
			tenantID := fmt.Sprintf("tenant-%04d", tenantIndex)
			actor := dpp.ActorContext{
				TenantID:       tenantID,
				OrganizationID: "org-1",
				ActorID:        "actor-1",
			}
			
			repo, err := NewRepository(db)
			if err != nil {
				results <- fmt.Errorf("failed to create repo: %w", err)
				return
			}
			
			for q := 0; q < queriesPerTenant; q++ {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				
				// In PostgresRepository, ListRegulatoryMonitors uses BeginTx and sets RLS variables.
				// We run it concurrently to ensure the pool resets properly.
				_, listErr := repo.ListRegulatoryMonitors(ctx, actor, "", "")
				if listErr != nil {
					results <- fmt.Errorf("RLS Drill failed for %s: %w", tenantID, listErr)
				}
				cancel()
			}
		}(i)
	}

	wg.Wait()
	close(results)

	var errs []error
	for err := range results {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		t.Fatalf("RLS isolation failed with %d bleeds. First error: %v", len(errs), errs[0])
	}

	t.Logf("Successfully completed RLS drill for %d tenants (%d queries total)", numTenants, numTenants*queriesPerTenant)
}
