package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river"

	"integin/internal/queue"
)

type benchWorker struct {
	river.WorkerDefaults[queue.WebhookDeliveryJobArgs]
}

func (b *benchWorker) Work(ctx context.Context, job *river.Job[queue.WebhookDeliveryJobArgs]) error {
	return nil
}

func main() {
	var (
		dbURL       string
		totalJobs   int
		batchSize   int
		concurrency int
		asyncCommit bool
	)

	flag.StringVar(&dbURL, "db", "postgres://postgres:postgres_local_test_password@localhost:15432/integin_migration_test?sslmode=disable", "PostgreSQL connection string")
	flag.IntVar(&totalJobs, "jobs", 50000, "Total number of jobs to enqueue")
	flag.IntVar(&batchSize, "batch", 200, "Batch size per transaction")
	flag.IntVar(&concurrency, "concurrency", 20, "Number of concurrent worker goroutines")
	flag.BoolVar(&asyncCommit, "async-commit", false, "Use asynchronous commit for transactional enqueue")
	flag.Parse()

	fmt.Printf("============================================================\n")
	fmt.Printf("RIVER 10K LOAD BENCHMARK HARNESS\n")
	fmt.Printf("Target: %s\n", dbURL)
	fmt.Printf("Jobs: %d | Batch Size: %d | Concurrency: %d | AsyncCommit: %v\n", totalJobs, batchSize, concurrency, asyncCommit)
	fmt.Printf("============================================================\n")

	if strings.Contains(dbURL, "6432") && !strings.Contains(dbURL, "default_query_exec_mode") {
		separator := "?"
		if strings.Contains(dbURL, "?") {
			separator = "&"
		}
		dbURL = dbURL + separator + "default_query_exec_mode=exec"
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(concurrency * 2)
	db.SetMaxIdleConns(concurrency)
	db.SetConnMaxLifetime(15 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}

	// Initialize Queue with registered worker for ingestion validation
	workers := river.NewWorkers()
	river.AddWorker(workers, &benchWorker{})
	q, err := queue.NewQueue(context.Background(), db, workers)
	if err != nil {
		log.Fatalf("failed to initialize queue: %v", err)
	}

	// Clean out previous test jobs if any
	_, _ = db.Exec(`DELETE FROM river_job WHERE kind = 'webhook_delivery_dispatch'`)

	totalBatches := (totalJobs + batchSize - 1) / batchSize
	batchJobs := make(chan int, totalBatches)
	for i := 0; i < totalBatches; i++ {
		curSize := batchSize
		if (i+1)*batchSize > totalJobs {
			curSize = totalJobs - (i * batchSize)
		}
		batchJobs <- curSize
	}
	close(batchJobs)

	var (
		enqueuedTotal int64
		latencies     []time.Duration
		latencyMu     sync.Mutex
		wg            sync.WaitGroup
	)

	startTime := time.Now()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			var localLatencies []time.Duration

			for size := range batchJobs {
				batch := make([]river.InsertManyParams, size)
				for j := 0; j < size; j++ {
					batch[j] = river.InsertManyParams{
						Args: queue.WebhookDeliveryJobArgs{
							DeliveryID: time.Now().UnixNano() + int64(j),
							TenantID:   "tenant-benchmark",
						},
					}
				}

				txStart := time.Now()
				tx, txErr := db.BeginTx(context.Background(), nil)
				if txErr != nil {
					log.Printf("worker %d: failed to begin tx: %v", workerID, txErr)
					continue
				}

				if asyncCommit {
					_, _ = tx.Exec(`SET LOCAL synchronous_commit = off`)
				}

				if insErr := q.InsertManyTx(context.Background(), tx, batch); insErr != nil {
					_ = tx.Rollback()
					log.Printf("worker %d: insert failed: %v", workerID, insErr)
					continue
				}

				if cErr := tx.Commit(); cErr != nil {
					log.Printf("worker %d: commit failed: %v", workerID, cErr)
					continue
				}

				localLatencies = append(localLatencies, time.Since(txStart))
				atomic.AddInt64(&enqueuedTotal, int64(size))
			}

			latencyMu.Lock()
			latencies = append(latencies, localLatencies...)
			latencyMu.Unlock()
		}(w)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Calculate metrics
	throughput := float64(enqueuedTotal) / duration.Seconds()

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var p50, p95, p99 time.Duration
	if len(latencies) > 0 {
		p50 = latencies[len(latencies)*50/100]
		p95 = latencies[len(latencies)*95/100]
		p99 = latencies[len(latencies)*99/100]
	}

	fmt.Printf("\n--- RESULTS ---\n")
	fmt.Printf("Total Enqueued:      %d jobs\n", enqueuedTotal)
	fmt.Printf("Elapsed Duration:    %v\n", duration)
	fmt.Printf("Throughput:          %.2f jobs/sec\n", throughput)
	fmt.Printf("Batch Latency (p50): %v\n", p50)
	fmt.Printf("Batch Latency (p95): %v\n", p95)
	fmt.Printf("Batch Latency (p99): %v\n", p99)

	// Check table stats
	var nLive, nDead int64
	_ = db.QueryRow(`SELECT n_live_tup, n_dead_tup FROM pg_stat_user_tables WHERE relname = 'river_job'`).Scan(&nLive, &nDead)
	fmt.Printf("river_job Table Stats: Live Tuples: %d | Dead Tuples: %d\n", nLive, nDead)

	// Prune test jobs using the high-speed prune worker
	pruned, pruneErr := q.PruneCompleted(context.Background(), 0, 100000)
	if pruneErr != nil {
		fmt.Printf("Prune Check: %v\n", pruneErr)
	} else {
		fmt.Printf("Prune Check: Successfully verified pruner on %d jobs\n", pruned)
	}
	fmt.Printf("============================================================\n")
}
