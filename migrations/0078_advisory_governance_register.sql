-- Migration 0078: Advisory Governance Register (P2).
-- Global reference tables for approved AI models and prompt versions, plus
-- tenant-scoped append-only audit trail and inspector feedback. Global
-- reference tables carry public SELECT for validation; tenant tables carry
-- hardened NULLIF session-GUC RLS (FORCE enabled).
--
-- AI-free zones (VERDICT, CERTIFICATE, ...) are never listed in allowed_zones;
-- enforcement is both data-driven (allowed_zones) and defended in
-- internal/advisory/register.go.

BEGIN;

CREATE TABLE IF NOT EXISTS advisory_model_registry (
    id            TEXT        PRIMARY KEY,
    provider      TEXT        NOT NULL,
    model_name    TEXT        NOT NULL,
    version       TEXT        NOT NULL,
    status        TEXT        NOT NULL DEFAULT 'APPROVED'
                  CHECK (status IN ('APPROVED', 'DEPRECATED')),
    max_tokens    INT         NOT NULL DEFAULT 4096,
    allowed_zones TEXT[]      NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (btrim(id) <> ''),
    CHECK (btrim(provider) <> ''),
    CHECK (btrim(model_name) <> ''),
    CHECK (btrim(version) <> ''),
    CHECK (max_tokens > 0)
);

CREATE TABLE IF NOT EXISTS advisory_prompt_registry (
    id                 TEXT        PRIMARY KEY,
    version            TEXT        NOT NULL,
    system_prompt_hash TEXT        NOT NULL,
    input_schema_hash  TEXT        NOT NULL,
    status             TEXT        NOT NULL DEFAULT 'ACTIVE'
                       CHECK (status IN ('ACTIVE', 'SUPERSEDED')),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (btrim(id) <> ''),
    CHECK (btrim(version) <> ''),
    CHECK (btrim(system_prompt_hash) <> ''),
    CHECK (btrim(input_schema_hash) <> '')
);

CREATE TABLE IF NOT EXISTS advisory_audit_trail (
    id             TEXT             PRIMARY KEY,
    tenant_id      TEXT             NOT NULL,
    insight_id     TEXT             NOT NULL,
    model_id       TEXT             NOT NULL,
    prompt_version TEXT             NOT NULL,
    zone           TEXT             NOT NULL,
    lens           TEXT             NOT NULL,
    evidence_refs  TEXT[]           NOT NULL DEFAULT '{}',
    confidence     DOUBLE PRECISION NOT NULL,
    blocking       BOOLEAN          NOT NULL DEFAULT FALSE
                   CHECK (blocking = FALSE),
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT now(),
    CHECK (btrim(id) <> ''),
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(insight_id) <> ''),
    CHECK (btrim(model_id) <> ''),
    CHECK (btrim(prompt_version) <> ''),
    CHECK (btrim(zone) <> ''),
    CHECK (btrim(lens) <> ''),
    CHECK (confidence >= 0 AND confidence <= 1)
);

CREATE TABLE IF NOT EXISTS advisory_feedback (
    id            TEXT        PRIMARY KEY,
    tenant_id     TEXT        NOT NULL,
    audit_id      TEXT        NOT NULL,
    inspector_id  TEXT        NOT NULL,
    disposition   TEXT        NOT NULL
                  CHECK (disposition IN ('ACCEPTED', 'REJECTED', 'IGNORED', 'CORRECTED')),
    notes         TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (btrim(id) <> ''),
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(audit_id) <> ''),
    CHECK (btrim(inspector_id) <> '')
);

CREATE INDEX IF NOT EXISTS advisory_audit_trail_tenant_insight_idx
    ON advisory_audit_trail (tenant_id, insight_id);
CREATE INDEX IF NOT EXISTS advisory_feedback_tenant_audit_idx
    ON advisory_feedback (tenant_id, audit_id);

ALTER TABLE advisory_audit_trail ENABLE ROW LEVEL SECURITY;
ALTER TABLE advisory_audit_trail FORCE ROW LEVEL SECURITY;
ALTER TABLE advisory_feedback ENABLE ROW LEVEL SECURITY;
ALTER TABLE advisory_feedback FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS advisory_audit_trail_tenant_isolation ON advisory_audit_trail;
CREATE POLICY advisory_audit_trail_tenant_isolation ON advisory_audit_trail
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    );

DROP POLICY IF EXISTS advisory_feedback_tenant_isolation ON advisory_feedback;
CREATE POLICY advisory_feedback_tenant_isolation ON advisory_feedback
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    );

GRANT SELECT ON advisory_model_registry TO integin_test_runtime;
GRANT SELECT ON advisory_prompt_registry TO integin_test_runtime;
GRANT SELECT, INSERT ON advisory_audit_trail TO integin_test_runtime;
GRANT SELECT, INSERT ON advisory_feedback TO integin_test_runtime;

COMMIT;
