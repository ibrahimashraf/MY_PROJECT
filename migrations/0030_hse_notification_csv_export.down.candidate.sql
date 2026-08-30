-- 0030_hse_notification_csv_export.down.candidate.sql
-- Reverse migration: drop HSE Notification + CSV Export tables

DROP INDEX IF EXISTS idx_export_template_export_type;
DROP TABLE IF EXISTS export_template;

DROP INDEX IF EXISTS idx_csv_export_job_status_requested_by;
DROP TABLE IF EXISTS csv_export_job;

DROP INDEX IF EXISTS idx_hse_notification_inspection_id;
DROP TABLE IF EXISTS hse_notification;