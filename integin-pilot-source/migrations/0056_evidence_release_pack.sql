-- integin Work-Order Field Package: evidence pack and release-pack composition.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.
-- Builds on 0010 evidence_metadata (evidence index), 0052 evidence_policy (form definition).

BEGIN;

-- evidence_pack: a composed, sealed collection of evidence bound to an inspection.
-- Manifest of included evidence is stored as JSONB (append-only at seal time).
CREATE TABLE IF NOT EXISTS evidence_pack (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    pack_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'SEALED', 'RELEASED', 'SUPERSEDED')),
    evidence_manifest JSONB NOT NULL DEFAULT '[]'::jsonb,
    evidence_count BIGINT NOT NULL DEFAULT 0 CHECK (evidence_count >= 0),
    classification TEXT NOT NULL,
    retention_reference TEXT NOT NULL,
    hold_state TEXT NOT NULL,
    redaction_policy_ref TEXT NOT NULL,
    pack_checksum TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, pack_name),
    FOREIGN KEY (tenant_id, organization_id, inspection_id)
        REFERENCES inspection_record (tenant_id, organization_id, id) ON DELETE RESTRICT,
    CHECK (
        (status = 'SEALED' AND evidence_count > 0 AND pack_checksum != '')
        OR (status != 'SEALED')
    ),
    CHECK (
        (status = 'SEALED' AND jsonb_array_length(evidence_manifest) = evidence_count)
        OR (status != 'SEALED')
    )
);

-- release_pack: recipient-purpose approval record tracking a sealed pack's release.
CREATE TABLE IF NOT EXISTS release_pack (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    pack_id TEXT NOT NULL,
    recipient TEXT NOT NULL,
    purpose TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'APPROVED', 'RELEASED', 'SUPERSEDED')),
    classification TEXT NOT NULL,
    retention_reference TEXT NOT NULL,
    hold_state TEXT NOT NULL,
    redaction_policy_ref TEXT NOT NULL,
    approved_by TEXT,
    approved_at TIMESTAMPTZ,
    released_by TEXT,
    released_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, pack_id, recipient, purpose),
    FOREIGN KEY (tenant_id, organization_id, pack_id)
        REFERENCES evidence_pack (tenant_id, organization_id, id) ON DELETE RESTRICT,
    CHECK (
        (status = 'APPROVED' AND approved_by IS NOT NULL AND approved_at IS NOT NULL)
        OR (status != 'APPROVED')
    ),
    CHECK (
        (status = 'RELEASED' AND released_by IS NOT NULL AND released_at IS NOT NULL)
        OR (status != 'RELEASED')
    )
);

-- Indexes for lookup patterns.
CREATE INDEX evidence_pack_inspection_idx
    ON evidence_pack (tenant_id, organization_id, inspection_id, status);

CREATE INDEX evidence_pack_status_idx
    ON evidence_pack (tenant_id, organization_id, status, updated_at DESC);

CREATE INDEX release_pack_pack_idx
    ON release_pack (tenant_id, organization_id, pack_id, status);

CREATE INDEX release_pack_recipient_idx
    ON release_pack (tenant_id, organization_id, recipient, status);

CREATE INDEX release_pack_status_idx
    ON release_pack (tenant_id, organization_id, status, created_at DESC);

-- Forced Row Level Security: tenant + organization isolation.
ALTER TABLE evidence_pack ENABLE ROW LEVEL SECURITY;
ALTER TABLE evidence_pack FORCE ROW LEVEL SECURITY;
ALTER TABLE release_pack ENABLE ROW LEVEL SECURITY;
ALTER TABLE release_pack FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS evidence_pack_tenant_organization_isolation ON evidence_pack;
CREATE POLICY evidence_pack_tenant_organization_isolation ON evidence_pack
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

DROP POLICY IF EXISTS release_pack_tenant_organization_isolation ON release_pack;
CREATE POLICY release_pack_tenant_organization_isolation ON release_pack
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

COMMIT;
