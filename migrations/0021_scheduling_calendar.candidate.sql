-- Migration 0021: Scheduling Calendar with Drag-Drop + Competency Gating
-- Compatible with existing schema (TEXT IDs, integin.* settings, FORCE RLS)
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.

BEGIN;

-- schedule_calendar_entry table
CREATE TABLE IF NOT EXISTS schedule_calendar_entry (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT,
    assignment_id TEXT,
    technician_id TEXT NOT NULL,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'SCHEDULED'
        CHECK (status IN ('SCHEDULED','IN_PROGRESS','COMPLETED','CANCELLED')),
    competency_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT valid_time_range CHECK (end_at > start_at)
);

-- technician_competency table
CREATE TABLE IF NOT EXISTS technician_competency (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    technician_id TEXT NOT NULL,
    equipment_type_id TEXT NOT NULL,
    certification_ref TEXT,
    expires_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'CURRENT'
        CHECK (status IN ('CURRENT','EXPIRED')),
    verified_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, technician_id, equipment_type_id)
);

-- scheduling_rule table
CREATE TABLE IF NOT EXISTS scheduling_rule (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    rule_type TEXT NOT NULL
        CHECK (rule_type IN ('OVERLAP','COMPETENCY','INTERVAL')),
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_schedule_calendar_entry_technician_date
    ON schedule_calendar_entry (tenant_id, organization_id, technician_id, start_at, end_at);

CREATE INDEX IF NOT EXISTS idx_schedule_calendar_entry_tenant_org_date
    ON schedule_calendar_entry (tenant_id, organization_id, start_at);

CREATE INDEX IF NOT EXISTS idx_schedule_calendar_entry_work_order
    ON schedule_calendar_entry (tenant_id, organization_id, work_order_id);

CREATE INDEX IF NOT EXISTS idx_schedule_calendar_entry_assignment
    ON schedule_calendar_entry (tenant_id, organization_id, assignment_id);

CREATE INDEX IF NOT EXISTS idx_schedule_calendar_entry_status
    ON schedule_calendar_entry (tenant_id, organization_id, status);

CREATE INDEX IF NOT EXISTS idx_technician_competency_technician_equipment
    ON technician_competency (tenant_id, organization_id, technician_id, equipment_type_id);

CREATE INDEX IF NOT EXISTS idx_technician_competency_tenant_org
    ON technician_competency (tenant_id, organization_id);

CREATE INDEX IF NOT EXISTS idx_technician_competency_expires
    ON technician_competency (tenant_id, organization_id, expires_at);

CREATE INDEX IF NOT EXISTS idx_scheduling_rule_tenant_org_enabled
    ON scheduling_rule (tenant_id, organization_id, enabled);

-- RLS
ALTER TABLE schedule_calendar_entry ENABLE ROW LEVEL SECURITY;
ALTER TABLE schedule_calendar_entry FORCE ROW LEVEL SECURITY;
ALTER TABLE technician_competency ENABLE ROW LEVEL SECURITY;
ALTER TABLE technician_competency FORCE ROW LEVEL SECURITY;
ALTER TABLE scheduling_rule ENABLE ROW LEVEL SECURITY;
ALTER TABLE scheduling_rule FORCE ROW LEVEL SECURITY;

CREATE POLICY schedule_calendar_entry_tenant_isolation
    ON schedule_calendar_entry
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE POLICY technician_competency_tenant_isolation
    ON technician_competency
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE POLICY scheduling_rule_tenant_isolation
    ON scheduling_rule
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

-- Trigger function for competency auto-expiry
CREATE OR REPLACE FUNCTION update_technician_competency_status()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.expires_at IS NOT NULL AND NEW.expires_at < now() THEN
        NEW.status := 'EXPIRED';
    ELSE
        NEW.status := 'CURRENT';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

DROP TRIGGER IF EXISTS trigger_update_competency_status ON technician_competency;
CREATE TRIGGER trigger_update_competency_status
    BEFORE INSERT OR UPDATE ON technician_competency
    FOR EACH ROW
    EXECUTE FUNCTION update_technician_competency_status();

COMMIT;