-- Apply tenant isolation to both the event-log partitioned parent and its
-- default partition. The runtime role must be a non-owner, non-superuser role.

ALTER TABLE event_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_log FORCE ROW LEVEL SECURITY;
ALTER TABLE event_log_default ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_log_default FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = current_schema() AND tablename = 'event_log' AND policyname = 'event_log_tenant_isolation') THEN
        CREATE POLICY event_log_tenant_isolation ON event_log
            USING (tenant_id = current_setting('integin.tenant_id', true))
            WITH CHECK (tenant_id = current_setting('integin.tenant_id', true));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = current_schema() AND tablename = 'event_log_default' AND policyname = 'event_log_default_tenant_isolation') THEN
        CREATE POLICY event_log_default_tenant_isolation ON event_log_default
            USING (tenant_id = current_setting('integin.tenant_id', true))
            WITH CHECK (tenant_id = current_setting('integin.tenant_id', true));
    END IF;
END $$;
