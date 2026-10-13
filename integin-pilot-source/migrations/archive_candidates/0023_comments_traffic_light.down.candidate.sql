-- Migration 0023: Predefined Comments Library + Traffic Light Color Coding
-- Down migration (reverses up migration)

BEGIN;

-- Drop RLS policies for inspection_comment
DROP POLICY IF EXISTS inspection_comment_org_isolation ON inspection_comment;
DROP POLICY IF EXISTS inspection_comment_tenant_isolation ON inspection_comment;

-- Drop RLS policies for comment_library
DROP POLICY IF EXISTS comment_library_org_isolation ON comment_library;
DROP POLICY IF EXISTS comment_library_tenant_isolation ON comment_library;

-- Disable RLS
ALTER TABLE inspection_comment DISABLE ROW LEVEL SECURITY;
ALTER TABLE comment_library DISABLE ROW LEVEL SECURITY;

-- Drop indexes for inspection_comment
DROP INDEX IF EXISTS idx_inspection_comment_library;
DROP INDEX IF EXISTS idx_inspection_comment_tenant_org;
DROP INDEX IF EXISTS idx_inspection_comment_inspection_question;

-- Drop indexes for comment_library
DROP INDEX IF EXISTS idx_comment_library_active;
DROP INDEX IF EXISTS idx_comment_library_tenant_org_equipment;
DROP INDEX IF EXISTS idx_comment_library_tenant_org_category;

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS inspection_comment;
DROP TABLE IF EXISTS comment_library;

COMMIT;