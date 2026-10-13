BEGIN;

DROP POLICY IF EXISTS device_enrollment_requests_tenant_isolation ON device_enrollment_requests;
ALTER TABLE device_enrollment_requests DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS device_enrollment_requests;

COMMIT;