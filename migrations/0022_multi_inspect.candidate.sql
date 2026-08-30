-- Migration 0022: Multi-Inspect Rack Workflow
-- Up migration

BEGIN;

-- multi_inspect_batch table
CREATE TABLE multi_inspect_batch (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    work_order_id UUID REFERENCES work_order(id) ON DELETE SET NULL,
    assignment_id UUID REFERENCES assignment(id) ON DELETE SET NULL,
    batch_code VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT'
        CHECK (status IN ('DRAFT', 'ACTIVE', 'COMPLETED', 'ABORTED')),
    total_items INTEGER NOT NULL DEFAULT 0,
    completed_items INTEGER NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES app_user(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, org_id, batch_code)
);

-- multi_inspect_item table
CREATE TABLE multi_inspect_item (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    batch_id UUID NOT NULL REFERENCES multi_inspect_batch(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES asset(id) ON DELETE CASCADE,
    inspection_id UUID REFERENCES inspection(id) ON DELETE SET NULL,
    sequence_num INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'IN_PROGRESS', 'PASS', 'FAIL', 'SKIPPED')),
    decision_at TIMESTAMPTZ,
    decided_by UUID REFERENCES app_user(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (batch_id, sequence_num)
);

-- inspect_template_preset table
CREATE TABLE inspect_template_preset (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    equipment_type_id UUID NOT NULL REFERENCES equipment_type(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    pass_fail_mode BOOLEAN NOT NULL DEFAULT FALSE,
    component_checks JSONB NOT NULL DEFAULT '[]',
    created_by UUID NOT NULL REFERENCES app_user(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, org_id, equipment_type_id, name)
);

-- Indexes for multi_inspect_batch
CREATE INDEX idx_multi_inspect_batch_tenant_org_status
    ON multi_inspect_batch (tenant_id, org_id, status);

CREATE INDEX idx_multi_inspect_batch_work_order
    ON multi_inspect_batch (work_order_id);

CREATE INDEX idx_multi_inspect_batch_assignment
    ON multi_inspect_batch (assignment_id);

-- Indexes for multi_inspect_item
CREATE INDEX idx_multi_inspect_item_batch_sequence
    ON multi_inspect_item (batch_id, sequence_num);

CREATE INDEX idx_multi_inspect_item_asset_lookup
    ON multi_inspect_item (asset_id);

CREATE INDEX idx_multi_inspect_item_inspection
    ON multi_inspect_item (inspection_id);

CREATE INDEX idx_multi_inspect_item_tenant_org_status
    ON multi_inspect_item (tenant_id, org_id, status);

-- Indexes for inspect_template_preset
CREATE INDEX idx_inspect_template_preset_tenant_org_equipment
    ON inspect_template_preset (tenant_id, org_id, equipment_type_id);

-- Enable RLS on all tables
ALTER TABLE multi_inspect_batch ENABLE ROW LEVEL SECURITY;
ALTER TABLE multi_inspect_item ENABLE ROW LEVEL SECURITY;
ALTER TABLE inspect_template_preset ENABLE ROW LEVEL SECURITY;

-- RLS Policies for multi_inspect_batch
CREATE POLICY multi_inspect_batch_tenant_isolation
    ON multi_inspect_batch
    FOR ALL
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY multi_inspect_batch_org_isolation
    ON multi_inspect_batch
    FOR ALL
    USING (org_id = current_org_id())
    WITH CHECK (org_id = current_org_id());

-- RLS Policies for multi_inspect_item
CREATE POLICY multi_inspect_item_tenant_isolation
    ON multi_inspect_item
    FOR ALL
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY multi_inspect_item_org_isolation
    ON multi_inspect_item
    FOR ALL
    USING (org_id = current_org_id())
    WITH CHECK (org_id = current_org_id());

-- RLS Policies for inspect_template_preset
CREATE POLICY inspect_template_preset_tenant_isolation
    ON inspect_template_preset
    FOR ALL
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY inspect_template_preset_org_isolation
    ON inspect_template_preset
    FOR ALL
    USING (org_id = current_org_id())
    WITH CHECK (org_id = current_org_id());

-- Function to maintain batch.completed_items count
CREATE OR REPLACE FUNCTION update_multi_inspect_batch_completed_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status IN ('PASS', 'FAIL', 'SKIPPED') THEN
            UPDATE multi_inspect_batch
            SET completed_items = completed_items + 1
            WHERE id = NEW.batch_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.status NOT IN ('PASS', 'FAIL', 'SKIPPED') AND NEW.status IN ('PASS', 'FAIL', 'SKIPPED') THEN
            UPDATE multi_inspect_batch
            SET completed_items = completed_items + 1
            WHERE id = NEW.batch_id;
        ELSIF OLD.status IN ('PASS', 'FAIL', 'SKIPPED') AND NEW.status NOT IN ('PASS', 'FAIL', 'SKIPPED') THEN
            UPDATE multi_inspect_batch
            SET completed_items = completed_items - 1
            WHERE id = NEW.batch_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.status IN ('PASS', 'FAIL', 'SKIPPED') THEN
            UPDATE multi_inspect_batch
            SET completed_items = completed_items - 1
            WHERE id = OLD.batch_id;
        END IF;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to maintain batch.completed_items count
CREATE TRIGGER trigger_update_batch_completed_count
    AFTER INSERT OR UPDATE OR DELETE ON multi_inspect_item
    FOR EACH ROW
    EXECUTE FUNCTION update_multi_inspect_batch_completed_count();

COMMIT;