-- Down migration for 0083_report_engine_persistence_and_rls
BEGIN;

DROP TABLE IF EXISTS generated_report CASCADE;
DROP TABLE IF EXISTS report_config CASCADE;

COMMIT;
