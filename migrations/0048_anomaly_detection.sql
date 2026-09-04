-- Migration 0048: Anomaly Detection for Short Links
-- TEXT IDs, combined RLS, FORCE RLS

BEGIN;

CREATE TABLE anomaly_rules (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL CHECK (type IN ('geo', 'frequency', 'pattern')),
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_anomaly_rules_tenant_org ON anomaly_rules (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_anomaly_rules_tenant_enabled ON anomaly_rules (tenant_id, organization_id, enabled) WHERE enabled = true;

ALTER TABLE anomaly_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE anomaly_rules FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS anomaly_rules_isolation ON anomaly_rules;
CREATE POLICY anomaly_rules_isolation ON anomaly_rules
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE OR REPLACE FUNCTION update_anomaly_rule_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql VOLATILE;

DROP TRIGGER IF EXISTS trigger_update_anomaly_rule_updated_at ON anomaly_rules;
CREATE TRIGGER trigger_update_anomaly_rule_updated_at
    BEFORE UPDATE ON anomaly_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_anomaly_rule_updated_at();

CREATE TABLE anomaly_alerts (
    id TEXT PRIMARY KEY,
    rule_id TEXT NOT NULL REFERENCES anomaly_rules(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    short_link_code TEXT NOT NULL REFERENCES short_links(code) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('geo', 'frequency', 'pattern')),
    status TEXT NOT NULL CHECK (status IN ('firing', 'acknowledged', 'resolved')) DEFAULT 'firing',
    message TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}',
    fired_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by TEXT,
    resolved_at TIMESTAMPTZ,
    resolved_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_anomaly_alerts_tenant_org ON anomaly_alerts (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_anomaly_alerts_rule ON anomaly_alerts (rule_id);
CREATE INDEX IF NOT EXISTS idx_anomaly_alerts_short_link ON anomaly_alerts (short_link_code);
CREATE INDEX IF NOT EXISTS idx_anomaly_alerts_status ON anomaly_alerts (tenant_id, organization_id, status);
CREATE INDEX IF NOT EXISTS idx_anomaly_alerts_type ON anomaly_alerts (tenant_id, organization_id, type);
CREATE INDEX IF NOT EXISTS idx_anomaly_alerts_fired_at ON anomaly_alerts (tenant_id, organization_id, fired_at DESC);
CREATE INDEX IF NOT EXISTS idx_anomaly_alerts_dedup ON anomaly_alerts (rule_id, short_link_code, fired_at DESC);

ALTER TABLE anomaly_alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE anomaly_alerts FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS anomaly_alerts_isolation ON anomaly_alerts;
CREATE POLICY anomaly_alerts_isolation ON anomaly_alerts
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE OR REPLACE FUNCTION update_anomaly_alert_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql VOLATILE;

DROP TRIGGER IF EXISTS trigger_update_anomaly_alert_updated_at ON anomaly_alerts;
CREATE TRIGGER trigger_update_anomaly_alert_updated_at
    BEFORE UPDATE ON anomaly_alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_anomaly_alert_updated_at();

GRANT SELECT, INSERT, UPDATE, DELETE ON anomaly_rules TO integin_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON anomaly_alerts TO integin_runtime;

COMMIT;
