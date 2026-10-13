-- Migration 0074 Down Script: Rollback Time-Bucket Partitioning & CQRS Replication Outbox

DROP TABLE IF EXISTS cqrs_replication_outbox CASCADE;
DROP TABLE IF EXISTS sensor_telemetry_stream CASCADE;
