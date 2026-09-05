-- INTEGIN Hardening: River HOT Updates, Heap Fillfactor & Retention Indexing
-- Addresses Trap #1 and Trap #2 discovered during deep architectural audit.

BEGIN;

-- 1. DROP GLOBAL STATE INDEX
-- The global index on (state, scheduled_at) forces an index write for EVERY state transition
-- (available -> running -> completed), completely defeating PostgreSQL Heap-Only Tuples (HOT).
DROP INDEX IF EXISTS river_job_state_and_scheduled_at;

-- 2. CREATE PARTIAL ACTIVE STATE INDEX
-- Only index active jobs. Once a job reaches completed/cancelled/discarded, it drops out of the index,
-- eliminating index writes on completion and keeping the active working set ultra-compact in memory.
CREATE INDEX IF NOT EXISTS river_job_active_state_idx
    ON river_job (state, scheduled_at)
    WHERE state IN ('available', 'running', 'retryable');

-- 3. ENABLE HEAP-ONLY TUPLES (HOT) VIA FILLFACTOR
-- By default fillfactor is 100%, meaning pages have no headroom for row versions created by updates.
-- Setting fillfactor to 75 reserves 25% free space per page for in-place updates.
ALTER TABLE river_job SET (fillfactor = 75);

-- 4. FAST RETENTION PRUNE INDEX
-- Enables the retention worker to efficiently locate and purge completed jobs older than retention threshold
-- without full table scans or locking.
CREATE INDEX IF NOT EXISTS river_job_prune_idx
    ON river_job (finalized_at)
    WHERE state IN ('completed', 'cancelled', 'discarded');

COMMIT;
