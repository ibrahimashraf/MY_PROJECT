-- Migration 0024: 3-Tier Escalation + Overdue Triggers
-- Down migration (reverses up migration)

BEGIN;

-- Drop RLS policies for escalation_event
DROP POLICY IF EXISTS escalation_event_org_isolation ON escalation_event;
DROP POLICY IF EXISTS escalation_event_tenant_isolation ON escalation_event;

-- Drop RLS policies for escalation_rule
DROP POLICY IF EXISTS escalation_rule_org_isolation ON escalation_rule;
DROP POLICY IF EXISTS escalation_rule_tenant_isolation ON escalation_rule;

-- Drop RLS policies for notification_template
DROP POLICY IF EXISTS notification_template_org_isolation ON notification_template;
DROP POLICY IF EXISTS notification_template_tenant_isolation ON notification_template;

-- Disable RLS
ALTER TABLE escalation_event DISABLE ROW LEVEL SECURITY;
ALTER TABLE escalation_rule DISABLE ROW LEVEL SECURITY;
ALTER TABLE notification_template DISABLE ROW LEVEL SECURITY;

-- Drop indexes for escalation_event
DROP INDEX IF EXISTS idx_escalation_event_triggered;
DROP INDEX IF EXISTS idx_escalation_event_tenant_org;
DROP INDEX IF EXISTS idx_escalation_event_tier_status;
DROP INDEX IF EXISTS idx_escalation_event_rule_inspection;

-- Drop indexes for escalation_rule
DROP INDEX IF EXISTS idx_escalation_rule_trigger_tier;
DROP INDEX IF EXISTS idx_escalation_rule_tenant_org_enabled;

-- Drop indexes for notification_template
DROP INDEX IF EXISTS idx_notification_template_tenant_org;

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS escalation_event;
DROP TABLE IF EXISTS escalation_rule;
DROP TABLE IF EXISTS notification_template;

COMMIT;