-- Migration 0024: 3-Tier Escalation + Overdue Triggers
-- Fixed: TEXT IDs, no bad REFERENCES, combined RLS, FORCE RLS

BEGIN;

CREATE TABLE IF NOT EXISTS notification_template (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    code TEXT NOT NULL CHECK (char_length(code) BETWEEN 1 AND 64),
    subject TEXT NOT NULL CHECK (char_length(subject) BETWEEN 1 AND 255),
    body_template TEXT NOT NULL,
    channel TEXT NOT NULL CHECK (channel IN ('EMAIL', 'SMS', 'IN_APP')),
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, code)
);

CREATE TABLE IF NOT EXISTS escalation_rule (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 128),
    trigger_type TEXT NOT NULL CHECK (trigger_type IN ('PRE_DUE', 'DUE', 'OVERDUE')),
    days_offset INTEGER NOT NULL,
    target_role TEXT NOT NULL CHECK (target_role IN ('TECH', 'SUPERVISOR', 'OPS')),
    escalation_tier SMALLINT NOT NULL CHECK (escalation_tier IN (1, 2, 3)),
    template_id TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, name)
);

CREATE TABLE IF NOT EXISTS escalation_event (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    rule_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    assignment_id TEXT,
    tier SMALLINT NOT NULL CHECK (tier IN (1, 2, 3)),
    status TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'SENT', 'ACKNOWLEDGED', 'ESCALATED')),
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_notification_template_tenant_org
    ON notification_template (tenant_id, organization_id);

CREATE INDEX IF NOT EXISTS idx_escalation_rule_tenant_org_enabled
    ON escalation_rule (tenant_id, organization_id, enabled);
CREATE INDEX IF NOT EXISTS idx_escalation_rule_trigger_tier
    ON escalation_rule (trigger_type, escalation_tier);

CREATE INDEX IF NOT EXISTS idx_escalation_event_rule_inspection
    ON escalation_event (rule_id, inspection_id);
CREATE INDEX IF NOT EXISTS idx_escalation_event_tier_status
    ON escalation_event (tier, status);
CREATE INDEX IF NOT EXISTS idx_escalation_event_tenant_org
    ON escalation_event (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_escalation_event_triggered
    ON escalation_event (triggered_at);

ALTER TABLE notification_template ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_template FORCE ROW LEVEL SECURITY;
ALTER TABLE escalation_rule ENABLE ROW LEVEL SECURITY;
ALTER TABLE escalation_rule FORCE ROW LEVEL SECURITY;
ALTER TABLE escalation_event ENABLE ROW LEVEL SECURITY;
ALTER TABLE escalation_event FORCE ROW LEVEL SECURITY;

CREATE POLICY notification_template_isolation ON notification_template
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE POLICY escalation_rule_isolation ON escalation_rule
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE POLICY escalation_event_isolation ON escalation_event
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
