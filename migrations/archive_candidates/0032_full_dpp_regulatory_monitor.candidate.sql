-- Migration 0032: Full DPP 4 Pillars + Regulatory Monitor + CTRN Concept
-- Compatible with existing schema (TEXT IDs, integin.* settings, FORCE RLS)
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.

BEGIN;

-- product_passport_dpp: Digital Product Passport with 4 pillars (Assignment, Update, Use, Disposal)
CREATE TABLE IF NOT EXISTS product_passport_dpp (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    serial_number TEXT,
    batch_number TEXT,
    manufacturer_id TEXT,
    dpp_status TEXT NOT NULL DEFAULT 'ASSIGNMENT'
        CHECK (dpp_status IN ('ASSIGNMENT','UPDATING','IMMUTABLE','DISPOSAL')),
    dpp_version INTEGER NOT NULL DEFAULT 1,
    dpp_sha256 BYTEA CHECK (dpp_sha256 IS NULL OR octet_length(dpp_sha256) = 32),
    assignment_payload JSONB,
    update_payload JSONB,
    use_payload JSONB,
    disposal_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, asset_id)
);

-- regulatory_monitor: Track regulatory changes across jurisdictions
CREATE TABLE IF NOT EXISTS regulatory_monitor (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    source TEXT NOT NULL
        CHECK (source IN ('EUR_LEX','RCRA','JAPAN_BASIC','ESPR','OTHER')),
    regulation_code TEXT NOT NULL,
    title TEXT NOT NULL,
    effective_date DATE,
    summary TEXT,
    impact_assessment TEXT,
    status TEXT NOT NULL DEFAULT 'MONITORING'
        CHECK (status IN ('MONITORING','ACTION_REQUIRED','COMPLIANT')),
    last_checked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    checked_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- compliance_action: Actions required to address regulatory changes
CREATE TABLE IF NOT EXISTS compliance_action (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    monitor_id TEXT NOT NULL,
    action_type TEXT NOT NULL
        CHECK (action_type IN ('GAP_ANALYSIS','POLICY_UPDATE','SYSTEM_CHANGE','TRAINING')),
    status TEXT NOT NULL DEFAULT 'TODO'
        CHECK (status IN ('TODO','IN_PROGRESS','DONE')),
    assignee TEXT,
    due_date DATE,
    completed_at TIMESTAMPTZ,
    evidence_refs JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_product_passport_dpp_sha256 ON product_passport_dpp (tenant_id, organization_id, dpp_sha256);
CREATE INDEX IF NOT EXISTS idx_product_passport_dpp_tenant_org ON product_passport_dpp (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_product_passport_dpp_asset ON product_passport_dpp (tenant_id, organization_id, asset_id);
CREATE INDEX IF NOT EXISTS idx_product_passport_dpp_status ON product_passport_dpp (tenant_id, organization_id, dpp_status);

CREATE INDEX IF NOT EXISTS idx_regulatory_monitor_tenant_org ON regulatory_monitor (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_regulatory_monitor_source ON regulatory_monitor (tenant_id, organization_id, source);
CREATE INDEX IF NOT EXISTS idx_regulatory_monitor_status ON regulatory_monitor (tenant_id, organization_id, status);
CREATE INDEX IF NOT EXISTS idx_regulatory_monitor_effective ON regulatory_monitor (tenant_id, organization_id, effective_date);

CREATE INDEX IF NOT EXISTS idx_compliance_action_tenant_org ON compliance_action (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_compliance_action_monitor ON compliance_action (tenant_id, organization_id, monitor_id);
CREATE INDEX IF NOT EXISTS idx_compliance_action_status ON compliance_action (tenant_id, organization_id, status);
CREATE INDEX IF NOT EXISTS idx_compliance_action_assignee ON compliance_action (tenant_id, organization_id, assignee);
CREATE INDEX IF NOT EXISTS idx_compliance_action_due ON compliance_action (tenant_id, organization_id, due_date);

-- RLS
ALTER TABLE product_passport_dpp ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_passport_dpp FORCE ROW LEVEL SECURITY;
ALTER TABLE regulatory_monitor ENABLE ROW LEVEL SECURITY;
ALTER TABLE regulatory_monitor FORCE ROW LEVEL SECURITY;
ALTER TABLE compliance_action ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_action FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS product_passport_dpp_tenant_isolation ON product_passport_dpp;
CREATE POLICY product_passport_dpp_tenant_isolation ON product_passport_dpp
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

DROP POLICY IF EXISTS regulatory_monitor_tenant_isolation ON regulatory_monitor;
CREATE POLICY regulatory_monitor_tenant_isolation ON regulatory_monitor
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

DROP POLICY IF EXISTS compliance_action_tenant_isolation ON compliance_action;
CREATE POLICY compliance_action_tenant_isolation ON compliance_action
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

-- Trigger: prevent UPDATE when dpp_status = 'IMMUTABLE'
CREATE OR REPLACE FUNCTION prevent_immutable_dpp_update()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.dpp_status = 'IMMUTABLE' THEN
        RAISE EXCEPTION 'Cannot modify DPP in IMMUTABLE status';
    END IF;
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

DROP TRIGGER IF EXISTS trg_prevent_immutable_dpp_update ON product_passport_dpp;
CREATE TRIGGER trg_prevent_immutable_dpp_update
    BEFORE UPDATE ON product_passport_dpp
    FOR EACH ROW
    EXECUTE FUNCTION prevent_immutable_dpp_update();

-- Trigger: prevent DELETE when dpp_status = 'IMMUTABLE'
CREATE OR REPLACE FUNCTION prevent_immutable_dpp_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.dpp_status = 'IMMUTABLE' THEN
        RAISE EXCEPTION 'Cannot delete DPP in IMMUTABLE status';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

DROP TRIGGER IF EXISTS trg_prevent_immutable_dpp_delete ON product_passport_dpp;
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
$$ LANGUAGE plpgsql SECURITY DEFINER;

DROP TRIGGER IF EXISTS trg_product_passport_dpp_updated_at ON product_passport_dpp;
CREATE TRIGGER trg_product_passport_dpp_updated_at
    BEFORE UPDATE ON product_passport_dpp
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trg_regulatory_monitor_updated_at ON regulatory_monitor;
CREATE TRIGGER trg_regulatory_monitor_updated_at
    BEFORE UPDATE ON regulatory_monitor
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trg_compliance_action_updated_at ON compliance_action;
CREATE TRIGGER trg_compliance_action_updated_at
    BEFORE UPDATE ON compliance_action
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMIT;