DROP POLICY IF EXISTS event_log_default_tenant_isolation ON event_log_default;
DROP POLICY IF EXISTS event_log_tenant_isolation ON event_log;
ALTER TABLE event_log_default DISABLE ROW LEVEL SECURITY;
ALTER TABLE event_log DISABLE ROW LEVEL SECURITY;
