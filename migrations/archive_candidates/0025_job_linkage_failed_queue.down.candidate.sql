BEGIN;
DROP POLICY IF EXISTS failed_inspection_queue_tenant_isolation ON failed_inspection_queue;
DROP POLICY IF EXISTS job_linkage_config_tenant_isolation ON job_linkage_config;
ALTER TABLE failed_inspection_queue DISABLE ROW LEVEL SECURITY;
ALTER TABLE job_linkage_config DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_failed_inspection_queue_review_status;
DROP TABLE IF EXISTS failed_inspection_queue;
DROP TABLE IF EXISTS job_linkage_config;
COMMIT;