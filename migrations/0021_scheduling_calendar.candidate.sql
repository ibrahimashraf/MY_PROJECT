-- Migration 0021: Scheduling Calendar with Drag-Drop + Competency Gating
-- Up migration

BEGIN;

-- schedule_calendar_entry table
CREATE TABLE schedule_calendar_entry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    work_order_id UUID REFERENCES work_order(id) ON DELETE SET NULL,
    assignment_id UUID REFERENCES assignment(id) ON DELETE SET NULL,
    technician_id UUID NOT NULL REFERENCES technician(id) ON DELETE CASCADE,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'SCHEDULED'
        CHECK (status IN ('SCHEDULED', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED')),
    competency_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID NOT NULL REFERENCES app_user(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT valid_time_range CHECK (end_at > start_at)
);

-- technician_competency table
CREATE TABLE technician_competency (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    technician_id UUID NOT NULL REFERENCES technician(id) ON DELETE CASCADE,
    equipment_type_id UUID NOT NULL REFERENCES equipment_type(id) ON DELETE CASCADE,
    certification_ref VARCHAR(255),
    expires_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'CURRENT'
        CHECK (status IN ('CURRENT', 'EXPIRED')),
    verified_by UUID REFERENCES app_user(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, org_id, technician_id, equipment_type_id)
);

-- scheduling_rule table
CREATE TABLE scheduling_rule (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    rule_type VARCHAR(20) NOT NULL
        CHECK (rule_type IN ('OVERLAP', 'COMPETENCY', 'INTERVAL')),
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for schedule_calendar_entry
CREATE INDEX idx_schedule_calendar_entry_technician_date
    ON schedule_calendar_entry (technician_id, start_at, end_at);

CREATE INDEX idx_schedule_calendar_entry_tenant_org_date
    ON schedule_calendar_entry (tenant_id, org_id, start_at);

CREATE INDEX idx_schedule_calendar_entry_work_order
    ON schedule_calendar_entry (work_order_id);

CREATE INDEX idx_schedule_calendar_entry_assignment
    ON schedule_calendar_entry (assignment_id);

CREATE INDEX idx_schedule_calendar_entry_status
    ON schedule_calendar_entry (status);

-- Indexes for technician_competency
CREATE INDEX idx_technician_competency_technician_equipment
    ON technician_competency (technician_id, equipment_type_id);

CREATE INDEX idx_technician_competency_tenant_org
    ON technician_competency (tenant_id, org_id);

CREATE INDEX idx_technician_competency_expires
    ON technician_competency (expires_at);

-- Indexes for scheduling_rule
CREATE INDEX idx_scheduling_rule_tenant_org_enabled
    ON scheduling_rule (tenant_id, org_id, enabled);

-- Enable RLS on all tables
ALTER TABLE schedule_calendar_entry ENABLE ROW LEVEL SECURITY;
ALTER TABLE technician_competency ENABLE ROW LEVEL SECURITY;
ALTER TABLE scheduling_rule ENABLE ROW LEVEL SECURITY;

-- RLS Policies for schedule_calendar_entry
CREATE POLICY schedule_calendar_entry_tenant_isolation
    ON schedule_calendar_entry
    FOR ALL
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY schedule_calendar_entry_org_isolation
    ON schedule_calendar_entry
    FOR ALL
    USING (org_id = current_org_id())
    WITH CHECK (org_id = current_org_id());

-- RLS Policies for technician_competency
CREATE POLICY technician_competency_tenant_isolation
    ON technician_competency
    FOR ALL
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY technician_competency_org_isolation
    ON technician_competency
    FOR ALL
    USING (org_id = current_org_id())
    WITH CHECK (org_id = current_org_id());

-- RLS Policies for scheduling_rule
CREATE POLICY scheduling_rule_tenant_isolation
    ON scheduling_rule
    FOR ALL
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY scheduling_rule_org_isolation
    ON scheduling_rule
    FOR ALL
    USING (org_id = current_org_id())
    WITH CHECK (org_id = current_org_id());

-- Function to update competency status based on expiry
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
$$ LANGUAGE plpgsql;

-- Trigger to auto-update competency status
CREATE TRIGGER trigger_update_competency_status
    BEFORE INSERT OR UPDATE ON technician_competency
    FOR EACH ROW
    EXECUTE FUNCTION update_technician_competency_status();

COMMIT;