-- CANDIDATE ROLLBACK ONLY: review against an isolated disposable copy before use.
BEGIN;
DROP TABLE IF EXISTS bulk_export_job;
DROP TABLE IF EXISTS bulk_import_row;
DROP TABLE IF EXISTS bulk_import_job;
COMMIT;