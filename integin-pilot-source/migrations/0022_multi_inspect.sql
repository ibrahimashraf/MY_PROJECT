-- Migration 0022: Multi-Inspect Rack Workflow
-- Fixed: TEXT IDs, no bad REFERENCES, combined RLS, FORCE RLS

BEGIN;

CREATE TABLE IF NOT EXISTS multi_inspect_batch (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT,
    assignment_id TEXT,
    batch_code TEXT NOT NULL CHECK (char_length(batch_code) BETWEEN 1 AND 64),
    status TEXT NOT NULL DEFAULT 'DRAFT'
        CHECK (status IN ('DRAFT', 'ACTIVE', 'COMPLETED', 'ABORTED')),
    total_items INTEGER NOT NULL DEFAULT 0,
    completed_items INTEGER NOT NULL DEFAULT 0,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, batch_code)
);

CREATE TABLE IF NOT EXISTS multi_inspect_item (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    batch_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    inspection_id TEXT,
    sequence_num INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'IN_PROGRESS', 'PASS', 'FAIL', 'SKIPPED')),
    decision_at TIMESTAMPTZ,
    decided_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (batch_id, sequence_num)
);

CREATE TABLE IF NOT EXISTS inspect_template_preset (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    equipment_type_id TEXT NOT NULL,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 255),
    pass_fail_mode BOOLEAN NOT NULL DEFAULT FALSE,
    component_checks JSONB NOT NULL DEFAULT '[]',
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, equipment_type_id, name)
);

CREATE INDEX IF NOT EXISTS idx_multi_inspect_batch_tenant_org_status
    ON multi_inspect_batch (tenant_id, organization_id, status);
CREATE INDEX IF NOT EXISTS idx_multi_inspect_batch_work_order
    ON multi_inspect_batch (work_order_id);
CREATE INDEX IF NOT EXISTS idx_multi_inspect_batch_assignment
    ON multi_inspect_batch (assignment_id);

CREATE INDEX IF NOT EXISTS idx_multi_inspect_item_batch_sequence
    ON multi_inspect_item (batch_id, sequence_num);
CREATE INDEX IF NOT EXISTS idx_multi_inspect_item_asset_lookup
    ON multi_inspect_item (asset_id);
CREATE INDEX IF NOT EXISTS idx_multi_inspect_item_inspection
    ON multi_inspect_item (inspection_id);
CREATE INDEX IF NOT EXISTS idx_multi_inspect_item_tenant_org_status
    ON multi_inspect_item (tenant_id, organization_id, status);

CREATE INDEX IF NOT EXISTS idx_inspect_template_preset_tenant_org_equipment
    ON inspect_template_preset (tenant_id, organization_id, equipment_type_id);

ALTER TABLE multi_inspect_batch ENABLE ROW LEVEL SECURITY;
ALTER TABLE multi_inspect_batch FORCE ROW LEVEL SECURITY;
ALTER TABLE multi_inspect_item ENABLE ROW LEVEL SECURITY;
ALTER TABLE multi_inspect_item FORCE ROW LEVEL SECURITY;
ALTER TABLE inspect_template_preset ENABLE ROW LEVEL SECURITY;
ALTER TABLE inspect_template_preset FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS multi_inspect_batch_isolation ON multi_inspect_batch;
CREATE POLICY multi_inspect_batch_isolation ON multi_inspect_batch
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

DROP POLICY IF EXISTS multi_inspect_item_isolation ON multi_inspect_item;
CREATE POLICY multi_inspect_item_isolation ON multi_inspect_item
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

DROP POLICY IF EXISTS inspect_template_preset_isolation ON inspect_template_preset;
CREATE POLICY inspect_template_preset_isolation ON inspect_template_preset
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

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

DROP TRIGGER IF EXISTS trigger_update_batch_completed_count ON multi_inspect_item;
CREATE TRIGGER trigger_update_batch_completed_count
    AFTER INSERT OR UPDATE OR DELETE ON multi_inspect_item
    FOR EACH ROW
    EXECUTE FUNCTION update_multi_inspect_batch_completed_count();

COMMIT;
