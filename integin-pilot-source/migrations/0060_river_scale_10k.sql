-- integin Background Queue: High-Throughput River Scaling (10,000 jobs/sec)
-- Builds on 0059_river_job_queue.sql.
-- Optimizes index structures and engine parameters for extreme throughput.

BEGIN;

-- 1. High-Performance Partial B-Tree for active job worker polling:
-- Prevents workers from scanning millions of completed/discarded/cancelled historical jobs.
CREATE INDEX IF NOT EXISTS river_job_available_idx
ON river_job (scheduled_at, priority)
WHERE state = 'available';

-- 2. Lean Partial Unique Index for Active Job Idempotency:
-- Enforces deduplication on unique_key only across active/in-flight jobs without bloating history.
CREATE UNIQUE INDEX IF NOT EXISTS river_job_unique_active_idx
ON river_job (unique_key)
WHERE state IN ('available', 'running', 'retryable');

-- 3. Dedicated Index for Queue-Specific Polling & Prioritization:
CREATE INDEX IF NOT EXISTS river_job_queue_priority_idx
ON river_job (queue, priority, scheduled_at)
WHERE state = 'available';

-- 4. Tune table storage parameters for high-churn queue MVCC efficiency:
-- More aggressive autovacuuming prevents dead tuple accumulation at 10,000 writes/sec.
ALTER TABLE river_job SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_vacuum_cost_limit = 2000
);

COMMIT;
