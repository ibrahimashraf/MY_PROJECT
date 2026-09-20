-- Rollback for 0018_timesheets_courses.
DROP INDEX IF EXISTS time_sheet_inspector_idx;
DROP INDEX IF EXISTS course_org_idx;
DROP TABLE IF EXISTS course_enrollment;
DROP TABLE IF EXISTS course;
DROP TABLE IF EXISTS time_sheet;
