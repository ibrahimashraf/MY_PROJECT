-- Migration 0023: Predefined Comments Library + Traffic Light Color Coding
-- Fixed: TEXT IDs, no bad REFERENCES, combined RLS, FORCE RLS

BEGIN;

CREATE TABLE IF NOT EXISTS comment_library (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('DEFECT', 'OBSERVATION', 'RECOMMENDATION')),
    equipment_type_id TEXT,
    code TEXT NOT NULL CHECK (char_length(code) BETWEEN 1 AND 64),
    text TEXT NOT NULL,
    traffic_light TEXT NOT NULL DEFAULT 'GREEN'
        CHECK (traffic_light IN ('GREEN', 'AMBER', 'RED')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, category, code)
);

CREATE TABLE IF NOT EXISTS inspection_comment (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    question_code TEXT NOT NULL CHECK (char_length(question_code) BETWEEN 1 AND 64),
    comment_library_id TEXT,
    custom_text TEXT,
    traffic_light TEXT NOT NULL DEFAULT 'GREEN'
        CHECK (traffic_light IN ('GREEN', 'AMBER', 'RED')),
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_comment_library_tenant_org_category
    ON comment_library (tenant_id, organization_id, category);
CREATE INDEX IF NOT EXISTS idx_comment_library_tenant_org_equipment
    ON comment_library (tenant_id, organization_id, equipment_type_id);
CREATE INDEX IF NOT EXISTS idx_comment_library_active
    ON comment_library (tenant_id, organization_id, is_active);

CREATE INDEX IF NOT EXISTS idx_inspection_comment_inspection_question
    ON inspection_comment (inspection_id, question_code);
CREATE INDEX IF NOT EXISTS idx_inspection_comment_tenant_org
    ON inspection_comment (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_inspection_comment_library
    ON inspection_comment (comment_library_id);

ALTER TABLE comment_library ENABLE ROW LEVEL SECURITY;
ALTER TABLE comment_library FORCE ROW LEVEL SECURITY;
ALTER TABLE inspection_comment ENABLE ROW LEVEL SECURITY;
ALTER TABLE inspection_comment FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS comment_library_isolation ON comment_library;
CREATE POLICY comment_library_isolation ON comment_library
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

DROP POLICY IF EXISTS inspection_comment_isolation ON inspection_comment;
CREATE POLICY inspection_comment_isolation ON inspection_comment
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
