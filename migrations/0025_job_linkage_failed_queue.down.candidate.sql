-- 0025_job_linkage_failed_queue.down.candidate.sql
-- Revert: Mandatory job linkage + failed suppression queue

DROP INDEX IF EXISTS idx_failed_inspection_queue_review_status;

DROP TABLE IF EXISTS failed_inspection_queue;

DROP TABLE IF EXISTS job_linkage_config;