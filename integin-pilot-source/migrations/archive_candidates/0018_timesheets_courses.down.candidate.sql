BEGIN;
DROP POLICY IF EXISTS course_enrollment_tenant_isolation ON course_enrollment;
DROP POLICY IF EXISTS course_tenant_isolation ON course;
DROP POLICY IF EXISTS time_sheet_tenant_isolation ON time_sheet;
DROP TABLE IF EXISTS course_enrollment;
DROP TABLE IF EXISTS course;
DROP TABLE IF EXISTS time_sheet;
COMMIT;
