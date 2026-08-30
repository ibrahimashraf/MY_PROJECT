-- Migration 0024: 3-Tier Escalation + Overdue Triggers
-- Up migration

BEGIN;

-- notification_template table
CREATE TABLE notification_template (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    code VARCHAR(64) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    body_template TEXT NOT NULL,
    channel VARCHAR(10) NOT NULL
        CHECK (channel IN ('EMAIL', 'SMS', 'IN_APP')),
    created_by UUID NOT NULL REFERENCES app_user(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, org_id, code)
);

-- escalation_rule table
CREATE TABLE escalation_rule (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    trigger_type VARCHAR(10) NOT NULL
        CHECK (trigger_type IN ('PRE_DUE', 'DUE', 'OVERDUE')),
    days_offset INTEGER NOT NULL,
    target_role VARCHAR(12) NOT NULL
        CHECK (target_role IN ('TECH', 'SUPERVISOR', 'OPS')),
    escalation_tier SMALLINT NOT NULL
        CHECK (escalation_tier IN (1, 2, 3)),
    template_id UUID NOT NULL REFERENCES notification_template(id) ON DELETE RESTRICT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL REFERENCES app_user(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, org_id, name)
);

-- escalation_event table
CREATE TABLE escalation_event (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    rule_id UUID NOT NULL REFERENCES escalation_rule(id) ON DELETE CASCADE,
    inspection_id UUID NOT NULL REFERENCES inspection(id) ON DELETE CASCADE,
    assignment_id UUID NOT NULL REFERENCES inspection_assignment(id) ON DELETE CASCADE,
    tier SMALLINT NOT NULL
        CHECK (tier IN (1, 2, 3)),
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'SENT', 'ACKNOWLEDGED', 'ESCALATED')),
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID REFERENCES app_user(id) ON DELETE SET NULL
);

-- Indexes for notification_template
CREATE INDEX idx_notification_template_tenant_org
    ON notification_template (tenant_id, org_id);

-- Indexes for escalation_rule
CREATE INDEX idx_escalation_rule_tenant_org_enabled
    ON escalation_rule (tenant_id, org_id, enabled)
    WHERE enabled = TRUE;

CREATE INDEX idx_escalation_rule_trigger_tier
    ON escalation_rule (trigger_type, escalation_tier);

-- Indexes for escalation_event (for cron queries)
CREATE INDEX idx_escalation_event_rule_inspection
    ON escalation_event (rule_id, inspection_id);

CREATE INDEX idx_escalation_event_tier_status
    ON escalation_event (tier, status);

CREATE INDEX idx_escalation_event_tenant_org
    ON escalation_event (tenant_id, org_id);

CREATE INDEX idx_escalation_event_triggered
    ON escalation_event (triggered_at);

-- Enable RLS on all tables (FORCE RLS)
ALTER TABLE notification_template ENABLE ROW LEVEL SECURITY;
ALTER TABLE escalation_rule ENABLE ROW LEVEL SECURITY;
ALTER TABLE escalation_event ENABLE ROW LEVEL SECURITY;

-- RLS Policies for notification_template
CREATE POLICY notification_template_tenant_isolation
    ON notification_template
    FOR ALL
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY notification_template_org_isolation
    ON notification_template
    FOR ALL
    USING (org_id = current_org_id())
    WITH CHECK (org_id = current_org_id());

-- RLS Policies for escalation_rule
CREATE POLICY escalation_rule_tenant_isolation
    ON escalation_rule
    FOR ALL
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY escalation_rule_org_isolation
    ON escalation_rule
    FOR ALL
    USING (org_id = current_org_id())
    WITH CHECK (org_id = current_org_id());

-- RLS Policies for escalation_event
CREATE POLICY escalation_event_tenant_isolation
    ON escalation_event
    FOR ALL
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY escalation_event_org_isolation
    ON escalation_event
    FOR ALL
    USING (org_id = current_org_id())
    WITH CHECK (org_id = current_org_id());

COMMIT;