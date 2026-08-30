-- Migration 0029: Parts catalog, service charges, timesheet auto-capture
-- Up migration

CREATE TYPE charge_type AS ENUM ('LABOUR', 'PARTS', 'TRAVEL', 'MILEAGE', 'OTHER');
CREATE TYPE timesheet_auto_status AS ENUM ('AUTO', 'APPROVED', 'REJECTED');

CREATE TABLE parts_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    code VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    unit_price NUMERIC(12,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, org_id, code)
);

CREATE TABLE service_charge (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    work_order_id UUID,
    inspection_id UUID,
    charge_type charge_type NOT NULL,
    description TEXT NOT NULL,
    quantity NUMERIC(10,3) NOT NULL DEFAULT 1,
    unit_price NUMERIC(12,2) NOT NULL,
    total_price NUMERIC(12,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    tax_rate NUMERIC(5,2) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(12,2) GENERATED ALWAYS AS (total_price * tax_rate / 100) STORED,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE timesheet_auto_capture (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    work_order_id UUID NOT NULL,
    assignment_id UUID NOT NULL,
    technician_id UUID NOT NULL,
    check_in_at TIMESTAMPTZ NOT NULL,
    check_out_at TIMESTAMPTZ,
    auto_calculated_hours NUMERIC(5,2),
    manual_override_hours NUMERIC(5,2),
    status timesheet_auto_status NOT NULL DEFAULT 'AUTO',
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, org_id, work_order_id, assignment_id, technician_id, check_in_at)
);

ALTER TABLE parts_catalog ENABLE ROW LEVEL SECURITY;
ALTER TABLE parts_catalog FORCE ROW LEVEL SECURITY;

ALTER TABLE service_charge ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_charge FORCE ROW LEVEL SECURITY;

ALTER TABLE timesheet_auto_capture ENABLE ROW LEVEL SECURITY;
ALTER TABLE timesheet_auto_capture FORCE ROW LEVEL SECURITY;

CREATE INDEX idx_parts_catalog_tenant_org ON parts_catalog (tenant_id, org_id);
CREATE INDEX idx_parts_catalog_code ON parts_catalog (tenant_id, org_id, code);
CREATE INDEX idx_parts_catalog_active ON parts_catalog (tenant_id, org_id, is_active);

CREATE INDEX idx_service_charge_tenant_org ON service_charge (tenant_id, org_id);
CREATE INDEX idx_service_charge_work_order ON service_charge (work_order_id);
CREATE INDEX idx_service_charge_inspection ON service_charge (inspection_id);
CREATE INDEX idx_service_charge_type ON service_charge (charge_type);

CREATE INDEX idx_timesheet_auto_tenant_org ON timesheet_auto_capture (tenant_id, org_id);
CREATE INDEX idx_timesheet_auto_work_order_tech ON timesheet_auto_capture (work_order_id, technician_id);
CREATE INDEX idx_timesheet_auto_assignment ON timesheet_auto_capture (assignment_id);
CREATE INDEX idx_timesheet_auto_status ON timesheet_auto_capture (status);
CREATE INDEX idx_timesheet_auto_check_in ON timesheet_auto_capture (check_in_at);

CREATE POLICY parts_catalog_tenant_isolation ON parts_catalog
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY service_charge_tenant_isolation ON service_charge
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY timesheet_auto_capture_tenant_isolation ON timesheet_auto_capture
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);