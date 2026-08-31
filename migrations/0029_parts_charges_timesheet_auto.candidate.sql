-- Migration 0029: Parts Catalog, Service Charges, Timesheet Auto-Capture
-- Compatible with existing schema (TEXT IDs, integin.* settings, FORCE RLS)
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.

BEGIN;

CREATE TYPE charge_type AS ENUM ('LABOUR','PARTS','TRAVEL','MILEAGE','OTHER');
CREATE TYPE timesheet_auto_status AS ENUM ('AUTO','APPROVED','REJECTED');

CREATE TABLE IF NOT EXISTS parts_catalog (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    code TEXT NOT NULL CHECK (char_length(code) BETWEEN 1 AND 100),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 255),
    description TEXT,
    unit_price NUMERIC(12,2) NOT NULL CHECK (unit_price >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, code)
);

CREATE TABLE IF NOT EXISTS service_charge (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT,
    inspection_id TEXT,
    charge_type charge_type NOT NULL,
    description TEXT NOT NULL,
    quantity NUMERIC(10,3) NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit_price NUMERIC(12,2) NOT NULL CHECK (unit_price >= 0),
    total_price NUMERIC(12,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    tax_rate NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (tax_rate >= 0),
    tax_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS timesheet_auto_capture (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    assignment_id TEXT NOT NULL,
    technician_id TEXT NOT NULL,
    check_in_at TIMESTAMPTZ NOT NULL,
    check_out_at TIMESTAMPTZ,
    auto_calculated_hours NUMERIC(5,2) CHECK (auto_calculated_hours IS NULL OR auto_calculated_hours >= 0),
    manual_override_hours NUMERIC(5,2) CHECK (manual_override_hours IS NULL OR manual_override_hours >= 0),
    status timesheet_auto_status NOT NULL DEFAULT 'AUTO',
    approved_by TEXT,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, work_order_id, assignment_id, technician_id, check_in_at)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_parts_catalog_tenant_org ON parts_catalog (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_parts_catalog_code ON parts_catalog (tenant_id, organization_id, code);
CREATE INDEX IF NOT EXISTS idx_parts_catalog_active ON parts_catalog (tenant_id, organization_id, is_active);

CREATE INDEX IF NOT EXISTS idx_service_charge_tenant_org ON service_charge (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_service_charge_work_order ON service_charge (tenant_id, organization_id, work_order_id);
CREATE INDEX IF NOT EXISTS idx_service_charge_inspection ON service_charge (tenant_id, organization_id, inspection_id);
CREATE INDEX IF NOT EXISTS idx_service_charge_type ON service_charge (tenant_id, organization_id, charge_type);

CREATE INDEX IF NOT EXISTS idx_timesheet_auto_tenant_org ON timesheet_auto_capture (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_timesheet_auto_work_order_tech ON timesheet_auto_capture (tenant_id, organization_id, work_order_id, technician_id);
CREATE INDEX IF NOT EXISTS idx_timesheet_auto_assignment ON timesheet_auto_capture (tenant_id, organization_id, assignment_id);
CREATE INDEX IF NOT EXISTS idx_timesheet_auto_status ON timesheet_auto_capture (tenant_id, organization_id, status);
CREATE INDEX IF NOT EXISTS idx_timesheet_auto_check_in ON timesheet_auto_capture (tenant_id, organization_id, check_in_at);

-- RLS
ALTER TABLE parts_catalog ENABLE ROW LEVEL SECURITY;
ALTER TABLE parts_catalog FORCE ROW LEVEL SECURITY;
ALTER TABLE service_charge ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_charge FORCE ROW LEVEL SECURITY;
ALTER TABLE timesheet_auto_capture ENABLE ROW LEVEL SECURITY;
ALTER TABLE timesheet_auto_capture FORCE ROW LEVEL SECURITY;

CREATE POLICY parts_catalog_tenant_isolation ON parts_catalog
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE POLICY service_charge_tenant_isolation ON service_charge
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE POLICY timesheet_auto_capture_tenant_isolation ON timesheet_auto_capture
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;