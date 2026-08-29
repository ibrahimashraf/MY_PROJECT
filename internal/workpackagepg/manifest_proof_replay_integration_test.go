package workpackagepg

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"integin/internal/domain/workpackage"

	_ "github.com/lib/pq"
)

const manifestReplayIntegrationDSNEnv = "INTEGIN_MANIFEST_REPLAY_TEST_DSN"

func TestManifestProofReplayStorePostgresIntegration(t *testing.T) {
	dsn := os.Getenv(manifestReplayIntegrationDSNEnv)
	if dsn == "" {
		t.Skip("set INTEGIN_MANIFEST_REPLAY_TEST_DSN to run disposable PostgreSQL replay integration tests")
	}
	adminDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	defer adminDB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := adminDB.PingContext(ctx); err != nil {
		t.Fatalf("ping integration database: %v", err)
	}
	rolePassword := resetManifestReplayIntegrationDatabase(t, ctx, adminDB)

	role := "integin_manifest_replay_test"
	roleDSN := integrationRoleDSN(t, dsn, role, rolePassword)
	appDB, err := sql.Open("postgres", roleDSN)
	if err != nil {
		t.Fatalf("open scoped integration database: %v", err)
	}
	defer appDB.Close()
	if err := appDB.PingContext(ctx); err != nil {
		t.Fatalf("ping scoped integration database: %v", err)
	}
	repository, err := NewRepository(appDB)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	store := NewManifestProofReplayStore(repository)
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	t.Run("concurrent consume permits exactly one request", func(t *testing.T) {
		const attempts = 8
		results := make(chan error, attempts)
		var group sync.WaitGroup
		for range attempts {
			group.Add(1)
			go func() {
				defer group.Done()
				results <- store.Consume(ctx, "tenant-a", "org-a", "device-a", workpackage.ManifestProofPurpose, "concurrent-request", now.Add(time.Minute))
			}()
		}
		group.Wait()
		close(results)
		var succeeded, replayed int
		for result := range results {
			switch {
			case result == nil:
				succeeded++
			case errors.Is(result, workpackage.ErrManifestProofReplayAlreadyConsumed):
				replayed++
			default:
				t.Fatalf("unexpected concurrent consume result: %v", result)
			}
		}
		if succeeded != 1 || replayed != attempts-1 {
			t.Fatalf("want one success and %d replays, got %d success and %d replay", attempts-1, succeeded, replayed)
		}
	})

	t.Run("same device request is independent across tenant scope", func(t *testing.T) {
		if err := store.Consume(ctx, "tenant-b", "org-a", "device-a", workpackage.ManifestProofPurpose, "concurrent-request", now.Add(time.Minute)); err != nil {
			t.Fatalf("tenant-separated consume: %v", err)
		}
	})

	t.Run("expired matching row is replaced atomically", func(t *testing.T) {
		if _, err := adminDB.ExecContext(ctx, `
			INSERT INTO manifest_proof_replay (tenant_id, organization_id, device_id, purpose, request_id, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, "tenant-a", "org-a", "device-a", workpackage.ManifestProofPurpose, "expired-request", now.Add(-time.Minute)); err != nil {
			t.Fatalf("seed expired replay row: %v", err)
		}
		if err := store.Consume(ctx, "tenant-a", "org-a", "device-a", workpackage.ManifestProofPurpose, "expired-request", now.Add(time.Minute)); err != nil {
			t.Fatalf("consume after expired row: %v", err)
		}
	})

	t.Run("failed insert rolls back replay consumption", func(t *testing.T) {
		if _, err := adminDB.ExecContext(ctx, `
			CREATE OR REPLACE FUNCTION reject_manifest_replay_rollback() RETURNS trigger AS $$
			BEGIN
				IF NEW.request_id = 'rollback-request' THEN RAISE EXCEPTION 'test rollback'; END IF;
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;
			CREATE TRIGGER manifest_replay_rollback_trigger
			BEFORE INSERT ON manifest_proof_replay
			FOR EACH ROW EXECUTE FUNCTION reject_manifest_replay_rollback();
		`); err != nil {
			t.Fatalf("install rollback trigger: %v", err)
		}
		t.Cleanup(func() {
			_, _ = adminDB.ExecContext(context.Background(), "DROP TRIGGER IF EXISTS manifest_replay_rollback_trigger ON manifest_proof_replay; DROP FUNCTION IF EXISTS reject_manifest_replay_rollback();")
		})
		if err := store.Consume(ctx, "tenant-a", "org-a", "device-a", workpackage.ManifestProofPurpose, "rollback-request", now.Add(time.Minute)); err == nil {
			t.Fatal("expected rollback-trigger failure")
		}
		var count int
		if err := adminDB.QueryRowContext(ctx, `SELECT count(*) FROM manifest_proof_replay WHERE request_id = 'rollback-request'`).Scan(&count); err != nil {
			t.Fatalf("query rollback row: %v", err)
		}
		if count != 0 {
			t.Fatalf("rollback request persisted %d rows", count)
		}
	})

	t.Run("RLS hides all rows without a tenant scope", func(t *testing.T) {
		var count int
		if err := appDB.QueryRowContext(ctx, "SELECT count(*) FROM manifest_proof_replay").Scan(&count); err != nil {
			t.Fatalf("query unscoped replay rows: %v", err)
		}
		if count != 0 {
			t.Fatalf("unscoped RLS query exposed %d rows", count)
		}
	})
}

func resetManifestReplayIntegrationDatabase(t *testing.T, ctx context.Context, db *sql.DB) string {
	t.Helper()
	migrationPath := filepath.Join("..", "..", "migrations", "0007_manifest_proof_replay.sql")
	migration, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read replay migration: %v", err)
	}
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS manifest_proof_replay CASCADE; DROP ROLE IF EXISTS integin_manifest_replay_test;"); err != nil {
		t.Fatalf("reset disposable replay database: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(migration)); err != nil {
		t.Fatalf("apply disposable replay migration: %v", err)
	}
	rolePassword := randomIntegrationRolePassword(t)
	if _, err := db.ExecContext(ctx, `
            CREATE ROLE integin_manifest_replay_test LOGIN;
            GRANT SELECT, INSERT, DELETE ON manifest_proof_replay TO integin_manifest_replay_test;
    `); err != nil {
		t.Fatalf("configure disposable application role: %v", err)
	}
	if _, err := db.ExecContext(ctx, "ALTER ROLE integin_manifest_replay_test PASSWORD '"+rolePassword+"'"); err != nil {
		t.Fatalf("set disposable application role password: %v", err)
	}
	return rolePassword
}

func randomIntegrationRolePassword(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("generate disposable role password: %v", err)
	}
	return hex.EncodeToString(raw)
}

func integrationRoleDSN(t *testing.T, dsn, role, password string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		t.Fatalf("%s must be a PostgreSQL URL for disposable integration testing", manifestReplayIntegrationDSNEnv)
	}
	parsed.User = url.UserPassword(role, password)
	return parsed.String()
}

func TestManifestReplayIntegrationDSNHelper(t *testing.T) {
	got := integrationRoleDSN(t, "postgres://postgres@127.0.0.1:25432/postgres?sslmode=disable", "role-a", "role-password")
	const want = "postgres://role-a:role-password@127.0.0.1:25432/postgres?sslmode=disable"
	if got != want {
		t.Fatalf("integration role DSN mismatch: want %q got %q", want, got)
	}
}

func ExampleManifestProofReplayStore_disposableIntegration() {
	fmt.Println("set INTEGIN_MANIFEST_REPLAY_TEST_DSN only for a disposable PostgreSQL target")
	// Output: set INTEGIN_MANIFEST_REPLAY_TEST_DSN only for a disposable PostgreSQL target
}
