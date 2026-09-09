package queue_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river"

	"integin/internal/queue"
)

func TestQueueBatchIngestionContract(t *testing.T) {
	// Verify that the queue package compiles and satisfies the batch insertion interface contract
	var q *queue.Queue
	if q != nil {
		ctx := context.Background()
		var params []river.InsertManyParams
		_ = q.InsertManyTx(ctx, nil, params)
	}
}

func TestQueuePruneCompletedIntegration(t *testing.T) {
	dbURL := "postgres://postgres:postgres_local_test_password@localhost:15432/integin_migration_test?sslmode=disable"
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Skip("skipping integration test; db not available")
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping integration test; db ping failed: %v", err)
	}

	q, err := queue.NewQueue(context.Background(), db, river.NewWorkers())
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}

	// Insert test jobs eligible for pruning
	_, err = db.ExecContext(ctx, `
		INSERT INTO river_job (args, kind, state, finalized_at, scheduled_at)
		VALUES 
			('{}'::jsonb, 'test_prune', 'completed'::river_job_state, now() - interval '2 hours', now()),
			('{}'::jsonb, 'test_prune', 'cancelled'::river_job_state, now() - interval '2 hours', now()),
			('{}'::jsonb, 'test_prune', 'discarded'::river_job_state, now() - interval '2 hours', now())
	`)
	if err != nil {
		t.Fatalf("failed to seed completed jobs: %v", err)
	}

	pruned, err := q.PruneCompleted(context.Background(), 1*time.Hour, 1000)
	if err != nil {
		t.Fatalf("PruneCompleted failed: %v", err)
	}
	if pruned == 0 {
		t.Errorf("expected pruned > 0, got %d", pruned)
	}
	t.Logf("successfully pruned %d completed jobs", pruned)
}

// testCertificateWorker implements the worker interface for CertificateRenderJobArgs
type testCertificateWorker struct {
	river.WorkerDefaults[queue.CertificateRenderJobArgs]
	executed chan string
}

func (w *testCertificateWorker) Work(ctx context.Context, job *river.Job[queue.CertificateRenderJobArgs]) error {
	if w.executed != nil {
		select {
		case w.executed <- job.Args.CertificateID:
		default:
		}
	}
	return nil
}

// TestRiverAtomicTransactionRollbackAndCommitIntegration validates that:
// 1. If a database transaction containing an InsertTx is rolled back, NO job is scheduled.
// 2. If the transaction commits, the job is persisted and scheduled atomically.
func TestRiverAtomicTransactionRollbackAndCommitIntegration(t *testing.T) {
	dbURL := "postgres://postgres:postgres_local_test_password@localhost:15432/integin_migration_test?sslmode=disable"
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Skip("skipping integration test; db not available")
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping integration test; db ping failed: %v", err)
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, &testCertificateWorker{})
	q, err := queue.NewQueue(ctx, db, workers)
	if err != nil {
		t.Fatalf("failed to initialize queue: %v", err)
	}

	uniqueCertID := "cert-rollback-test-" + time.Now().Format("20060102150405.000000000")
	jobArgs := queue.CertificateRenderJobArgs{
		TenantID:       "tenant-test",
		OrganizationID: "org-test",
		CertificateID:  uniqueCertID,
		InspectionID:   "insp-test",
		ActorUserID:    "user-test",
	}

	// 1. Transactional Rollback: verify no job exists
	tx1, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin tx1: %v", err)
	}
	if err := q.InsertTx(ctx, tx1, jobArgs); err != nil {
		_ = tx1.Rollback()
		t.Fatalf("failed to insert tx1: %v", err)
	}
	_ = tx1.Rollback()

	var count1 int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM river_job WHERE args->>'certificate_id' = $1", uniqueCertID).Scan(&count1)
	if err != nil {
		t.Fatalf("query job count after rollback: %v", err)
	}
	if count1 != 0 {
		t.Fatalf("ATOMICITY VIOLATION: job appeared in river_job after transaction rollback! count=%d", count1)
	}
	t.Log("PASS: Transaction rollback successfully discarded job.")

	// 2. Transactional Commit: verify job is persisted
	tx2, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin tx2: %v", err)
	}
	if err := q.InsertTx(ctx, tx2, jobArgs); err != nil {
		_ = tx2.Rollback()
		t.Fatalf("failed to insert tx2: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatalf("failed to commit tx2: %v", err)
	}

	var count2 int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM river_job WHERE args->>'certificate_id' = $1", uniqueCertID).Scan(&count2)
	if err != nil {
		t.Fatalf("query job count after commit: %v", err)
	}
	if count2 != 1 {
		t.Fatalf("COMMIT FAILURE: expected 1 job in river_job, got %d", count2)
	}
	t.Log("PASS: Transaction commit atomically persisted job.")

	// Cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM river_job WHERE args->>'certificate_id' = $1", uniqueCertID)
}

// TestRiverWorkerExecutionThroughPgCatIntegration tests that a River worker pool
// can connect through the PgCat pooler (port 6432), pick up a job, process it,
// and transition its state to completed without statement collisions.
func TestRiverWorkerExecutionThroughPgCatIntegration(t *testing.T) {
	pgcatURL := "postgres://postgres:postgres_local_test_password@127.0.0.1:6432/integin_migration_test?sslmode=disable&default_query_exec_mode=exec"
	db, err := sql.Open("pgx", pgcatURL)
	if err != nil {
		t.Skip("skipping PgCat worker test; db open failed")
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping PgCat worker test; ping failed: %v", err)
	}

	executedCh := make(chan string, 1)
	workers := river.NewWorkers()
	river.AddWorker(workers, &testCertificateWorker{executed: executedCh})

	q, err := queue.NewQueue(ctx, db, workers)
	if err != nil {
		t.Fatalf("failed to initialize queue via PgCat: %v", err)
	}

	uniqueCertID := "cert-pgcat-worker-" + time.Now().Format("20060102150405.000000000")
	jobArgs := queue.CertificateRenderJobArgs{
		TenantID:       "tenant-pgcat",
		OrganizationID: "org-pgcat",
		CertificateID:  uniqueCertID,
		InspectionID:   "insp-pgcat",
		ActorUserID:    "user-pgcat",
	}

	// Start River client
	if err := q.Start(ctx); err != nil {
		t.Fatalf("failed to start river queue worker: %v", err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		_ = q.Stop(stopCtx)
	}()

	// Enqueue job within a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := q.InsertTx(ctx, tx, jobArgs); err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert tx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	// Wait for worker execution
	select {
	case id := <-executedCh:
		if id != uniqueCertID {
			t.Fatalf("unexpected certificate id executed: got %s, want %s", id, uniqueCertID)
		}
		t.Logf("PASS: Worker successfully processed job %s via PgCat!", id)
	case <-time.After(5 * time.Second):
		t.Fatal("TIMED OUT: Worker failed to process job within 5 seconds")
	}

	// Wait for River engine to finalize job to 'completed'
	var state string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		err = db.QueryRowContext(ctx, "SELECT state::text FROM river_job WHERE args->>'certificate_id' = $1", uniqueCertID).Scan(&state)
		if err == nil && state == "completed" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if state != "completed" {
		t.Fatalf("expected job state 'completed', got %q", state)
	}
	t.Log("PASS: Job state transitioned to completed.")

	// Cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM river_job WHERE args->>'certificate_id' = $1", uniqueCertID)
}
