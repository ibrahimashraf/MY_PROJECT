-- Migration 0023: Predefined Comments Library + Traffic Light Color Coding
-- Up migration

BEGIN;

-- comment_library table
CREATE TABLE comment_library (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    category VARCHAR(20) NOT NULL
        CHECK (category IN ('DEFECT', 'OBSERVATION', 'RECOMMENDATION')),
    equipment_type_id UUID REFERENCES equipment_type(id) ON DELETE SET NULL,
    code VARCHAR(64) NOT NULL,
    text TEXT NOT NULL,
    traffic_light VARCHAR(10) NOT NULL DEFAULT 'GREEN'
        CHECK (traffic_light IN ('GREEN', 'AMBER', 'RED')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES app_user(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, org_id, category, code)
);

-- inspection_comment table
CREATE TABLE inspection_comment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    inspection_id UUID NOT NULL REFERENCES inspection(id) ON DELETE CASCADE,
    question_code VARCHAR(64) NOT NULL,
    comment_library_id UUID REFERENCES comment_library(id) ON DELETE SET NULL,
    custom_text TEXT,
    traffic_light VARCHAR(10) NOT NULL DEFAULT 'GREEN'
        CHECK (traffic_light IN ('GREEN', 'AMBER', 'RED')),
    created_by UUID NOT NULL REFERENCES app_user(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for comment_library
CREATE INDEX idx_comment_library_tenant_org_category
    ON comment_library (tenant_id, org_id, category);

CREATE INDEX idx_comment_library_tenant_org_equipment
    ON comment_library (tenant_id, org_id, equipment_type_id);

CREATE INDEX idx_comment_library_active
    ON comment_library (tenant_id, org_id, is_active)
    WHERE is_active = TRUE;

-- Indexes for inspection_comment
CREATE INDEX idx_inspection_comment_inspection_question
    ON inspection_comment (inspection_id, question_code);

CREATE INDEX idx_inspection_comment_tenant_org
    ON inspection_comment (tenant_id, org_id);

CREATE INDEX idx_inspection_comment_library
    ON inspection_comment (comment_library_id);

-- Enable RLS on all tables (FORCE RLS)
ALTER TABLE comment_library ENABLE ROW LEVEL SECURITY;
ALTER TABLE inspection_comment ENABLE ROW LEVEL SECURITY;

-- RLS Policies for comment_library
CREATE POLICY comment_library_tenant_isolation
    ON comment_library
    FOR ALL
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY comment_library_org_isolation
    ON comment_library
    FOR ALL
    USING (org_id = current_org_id())
    WITH CHECK (org_id = current_org_id());

-- RLS Policies for inspection_comment
CREATE POLICY inspection_comment_tenant_isolation
    ON inspection_comment
    FOR ALL
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY inspection_comment_org_isolation
    ON inspection_comment
    FOR ALL
    USING (org_id = current_org_id())
    WITH CHECK (org_id = current_org_id());

COMMIT;