-- INTEGIN Background Queue: Rollback for High-Throughput River Scaling (10,000 jobs/sec)
BEGIN;

DROP INDEX IF EXISTS river_job_available_idx;
DROP INDEX IF EXISTS river_job_unique_active_idx;
DROP INDEX IF EXISTS river_job_queue_priority_idx;

ALTER TABLE river_job RESET (
    autovacuum_vacuum_scale_factor,
    autovacuum_vacuum_cost_limit
);

COMMIT;
