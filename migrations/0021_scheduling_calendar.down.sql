-- Migration 0021: Scheduling Calendar with Drag-Drop + Competency Gating
-- Down migration (reverses up migration)

BEGIN;

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_competency_status ON technician_competency;
DROP FUNCTION IF EXISTS update_technician_competency_status();

-- Drop RLS policies
DROP POLICY IF EXISTS scheduling_rule_org_isolation ON scheduling_rule;
DROP POLICY IF EXISTS scheduling_rule_tenant_isolation ON scheduling_rule;

DROP POLICY IF EXISTS technician_competency_org_isolation ON technician_competency;
DROP POLICY IF EXISTS technician_competency_tenant_isolation ON technician_competency;

DROP POLICY IF EXISTS schedule_calendar_entry_org_isolation ON schedule_calendar_entry;
DROP POLICY IF EXISTS schedule_calendar_entry_tenant_isolation ON schedule_calendar_entry;

-- Disable RLS
ALTER TABLE scheduling_rule DISABLE ROW LEVEL SECURITY;
ALTER TABLE technician_competency DISABLE ROW LEVEL SECURITY;
ALTER TABLE schedule_calendar_entry DISABLE ROW LEVEL SECURITY;

-- Drop indexes
DROP INDEX IF EXISTS idx_scheduling_rule_tenant_org_enabled;

DROP INDEX IF EXISTS idx_technician_competency_expires;
DROP INDEX IF EXISTS idx_technician_competency_tenant_org;
DROP INDEX IF EXISTS idx_technician_competency_technician_equipment;

DROP INDEX IF EXISTS idx_schedule_calendar_entry_status;
DROP INDEX IF EXISTS idx_schedule_calendar_entry_assignment;
DROP INDEX IF EXISTS idx_schedule_calendar_entry_work_order;
DROP INDEX IF EXISTS idx_schedule_calendar_entry_tenant_org_date;
DROP INDEX IF EXISTS idx_schedule_calendar_entry_technician_date;

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS scheduling_rule;
DROP TABLE IF EXISTS technician_competency;
DROP TABLE IF EXISTS schedule_calendar_entry;

COMMIT;