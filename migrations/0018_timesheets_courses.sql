-- INTEGIN time sheets / courses — operational optional, behind feature flags FlagTimeSheets/FlagCourses.
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.
BEGIN;

CREATE TABLE IF NOT EXISTS time_sheet (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    inspector_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    inspection_id TEXT,
    hours NUMERIC(5,2) NOT NULL CHECK (hours > 0 AND hours <= 24),
    occurred_on DATE NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (tenant_id, organization_id, work_order_id) REFERENCES work_order(tenant_id, organization_id, id) ON DELETE CASCADE DEFERRABLE
);

CREATE TABLE IF NOT EXISTS course (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 2 AND 200),
    instructor_id TEXT,
    status TEXT NOT NULL CHECK (status IN ('DRAFT','ACTIVE','RETIRED')) DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS course_enrollment (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    course_id TEXT NOT NULL REFERENCES course(id) ON DELETE RESTRICT,
    inspector_id TEXT NOT NULL,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL CHECK (status IN ('ENROLLED','COMPLETED','CANCELLED')) DEFAULT 'ENROLLED',
    UNIQUE (tenant_id, organization_id, course_id, inspector_id)
);

CREATE INDEX IF NOT EXISTS time_sheet_inspector_idx ON time_sheet (tenant_id, organization_id, inspector_id, occurred_on DESC);
CREATE INDEX IF NOT EXISTS course_org_idx ON course (tenant_id, organization_id, status);

ALTER TABLE time_sheet ENABLE ROW LEVEL SECURITY;
ALTER TABLE time_sheet FORCE ROW LEVEL SECURITY;
ALTER TABLE course ENABLE ROW LEVEL SECURITY;
ALTER TABLE course FORCE ROW LEVEL SECURITY;
ALTER TABLE course_enrollment ENABLE ROW LEVEL SECURITY;
ALTER TABLE course_enrollment FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS time_sheet_tenant_isolation ON time_sheet;
CREATE POLICY time_sheet_tenant_isolation ON time_sheet USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
DROP POLICY IF EXISTS course_tenant_isolation ON course;
CREATE POLICY course_tenant_isolation ON course USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
DROP POLICY IF EXISTS course_enrollment_tenant_isolation ON course_enrollment;
CREATE POLICY course_enrollment_tenant_isolation ON course_enrollment USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
