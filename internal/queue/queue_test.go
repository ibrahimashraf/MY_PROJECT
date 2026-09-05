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

