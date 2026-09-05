-- INTEGIN Hardening: WAL Suppression & XID Freeze Protection at 30k jobs/sec
-- Builds on 0065_river_canonical_v047_schema_alignment.sql.

BEGIN;

-- 1. Configure river_job autovacuum for aggressive freeze and vacuuming
-- Prevents 32-bit transaction ID (XID) wraparound when generating 2.5B+ transactions/day.
ALTER TABLE river_job SET (
    autovacuum_vacuum_scale_factor = 0.01,
    autovacuum_vacuum_threshold = 1000,
    autovacuum_vacuum_cost_limit = 5000,
    autovacuum_vacuum_cost_delay = 2,
    autovacuum_freeze_max_age = 50000000,
    autovacuum_freeze_table_age = 10000000,
    autovacuum_freeze_min_age = 1000000
);

-- 2. Tune river_notification similarly for high-speed notify events
ALTER TABLE river_notification SET (
    autovacuum_vacuum_scale_factor = 0.01,
    autovacuum_vacuum_threshold = 500,
    autovacuum_vacuum_cost_limit = 5000,
    autovacuum_vacuum_cost_delay = 2
);

COMMIT;
