-- 0032_full_dpp_regulatory_monitor.candidate.sql
-- DPP 4 Pillars + Regulatory Monitor + CTRN Concept

-- product_passport_dpp: Digital Product Passport with 4 pillars (Assignment, Update, Use, Disposal)
CREATE TABLE product_passport_dpp (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    asset_id UUID NOT NULL,
    serial_number TEXT,
    batch_number TEXT,
    manufacturer_id UUID,
    dpp_status TEXT NOT NULL DEFAULT 'ASSIGNMENT'
        CHECK (dpp_status IN ('ASSIGNMENT', 'UPDATING', 'IMMUTABLE', 'DISPOSAL')),
    dpp_version INTEGER NOT NULL DEFAULT 1,
    dpp_sha256 BYTEA,
    assignment_payload JSONB,
    update_payload JSONB,
    use_payload JSONB,
    disposal_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, org_id, asset_id)
);

-- Force RLS on product_passport_dpp
ALTER TABLE product_passport_dpp ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_passport_dpp FORCE ROW LEVEL SECURITY;

-- Index for integrity verification on dpp_sha256
CREATE INDEX idx_product_passport_dpp_sha256 ON product_passport_dpp (dpp_sha256);
CREATE INDEX idx_product_passport_dpp_tenant_org ON product_passport_dpp (tenant_id, org_id);
CREATE INDEX idx_product_passport_dpp_asset ON product_passport_dpp (asset_id);
CREATE INDEX idx_product_passport_dpp_status ON product_passport_dpp (dpp_status);

-- regulatory_monitor: Track regulatory changes across jurisdictions
CREATE TABLE regulatory_monitor (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    source TEXT NOT NULL
        CHECK (source IN ('EUR_LEX', 'RCRA', 'JAPAN_BASIC', 'ESPR', 'OTHER')),
    regulation_code TEXT NOT NULL,
    title TEXT NOT NULL,
    effective_date DATE,
    summary TEXT,
    impact_assessment TEXT,
    status TEXT NOT NULL DEFAULT 'MONITORING'
        CHECK (status IN ('MONITORING', 'ACTION_REQUIRED', 'COMPLIANT')),
    last_checked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    checked_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Force RLS on regulatory_monitor
ALTER TABLE regulatory_monitor ENABLE ROW LEVEL SECURITY;
ALTER TABLE regulatory_monitor FORCE ROW LEVEL SECURITY;

CREATE INDEX idx_regulatory_monitor_tenant_org ON regulatory_monitor (tenant_id, org_id);
CREATE INDEX idx_regulatory_monitor_source ON regulatory_monitor (source);
CREATE INDEX idx_regulatory_monitor_status ON regulatory_monitor (status);
CREATE INDEX idx_regulatory_monitor_effective ON regulatory_monitor (effective_date);

-- compliance_action: Actions required to address regulatory changes
CREATE TABLE compliance_action (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    monitor_id UUID NOT NULL REFERENCES regulatory_monitor(id) ON DELETE CASCADE,
    action_type TEXT NOT NULL
        CHECK (action_type IN ('GAP_ANALYSIS', 'POLICY_UPDATE', 'SYSTEM_CHANGE', 'TRAINING')),
    status TEXT NOT NULL DEFAULT 'TODO'
        CHECK (status IN ('TODO', 'IN_PROGRESS', 'DONE')),
    assignee UUID,
    due_date DATE,
    completed_at TIMESTAMPTZ,
    evidence_refs JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Force RLS on compliance_action
ALTER TABLE compliance_action ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_action FORCE ROW LEVEL SECURITY;

CREATE INDEX idx_compliance_action_tenant_org ON compliance_action (tenant_id, org_id);
CREATE INDEX idx_compliance_action_monitor ON compliance_action (monitor_id);
CREATE INDEX idx_compliance_action_status ON compliance_action (status);
CREATE INDEX idx_compliance_action_assignee ON compliance_action (assignee);
CREATE INDEX idx_compliance_action_due ON compliance_action (due_date);

-- Trigger to prevent updates to product_passport_dpp when status is IMMUTABLE
CREATE OR REPLACE FUNCTION prevent_immutable_dpp_update()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.dpp_status = 'IMMUTABLE' THEN
        RAISE EXCEPTION 'Cannot modify DPP in IMMUTABLE status';
    END IF;
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_immutable_dpp_update
BEFORE UPDATE ON product_passport_dpp
FOR EACH ROW
EXECUTE FUNCTION prevent_immutable_dpp_update();

-- Trigger to prevent deletion of product_passport_dpp when status is IMMUTABLE
CREATE OR REPLACE FUNCTION prevent_immutable_dpp_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.dpp_status = 'IMMUTABLE' THEN
        RAISE EXCEPTION 'Cannot delete DPP in IMMUTABLE status';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_immutable_dpp_delete
BEFORE DELETE ON product_passport_dpp
FOR EACH ROW
EXECUTE FUNCTION prevent_immutable_dpp_delete();

-- Updated_at triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_product_passport_dpp_updated_at
BEFORE UPDATE ON product_passport_dpp
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_regulatory_monitor_updated_at
BEFORE UPDATE ON regulatory_monitor
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_compliance_action_updated_at
BEFORE UPDATE ON compliance_action
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();