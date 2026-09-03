-- INTEGIN Work-Order Field Package: authenticated QR/NFC entry with digest-only token storage.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.
-- Builds on 0053 asset_entitlement, 0052 form_version.

BEGIN;

-- qr_nfc_entry: opaque, unguessable tokens stored only as SHA-256 digest.
-- Links physical asset tags to server-authoritative entitlements.
CREATE TABLE qr_nfc_entry (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    entitlement_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    entry_type TEXT NOT NULL CHECK (entry_type IN ('QR', 'NFC')),
    token_digest TEXT NOT NULL,
    token_version BIGINT NOT NULL DEFAULT 1 CHECK (token_version > 0),
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'REVOKED', 'EXPIRED')),
    issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revoked_by TEXT,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, token_digest),
    FOREIGN KEY (tenant_id, organization_id, entitlement_id)
        REFERENCES asset_entitlement (tenant_id, organization_id, id) ON DELETE RESTRICT,
    CHECK (
        (status = 'ACTIVE' AND revoked_at IS NULL AND revoked_by IS NULL)
        OR (status = 'REVOKED' AND revoked_at IS NOT NULL AND revoked_by IS NOT NULL)
        OR (status = 'EXPIRED')
    )
);

-- qr_nfc_entry_log: privacy-preserving access log (no raw tokens, no asset details).
CREATE TABLE qr_nfc_entry_log (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    entry_id TEXT NOT NULL,
    entry_type TEXT NOT NULL,
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    accessor_id TEXT NOT NULL,
    accessor_ip TEXT,
    outcome TEXT NOT NULL CHECK (outcome IN ('SUCCESS', 'DENIED', 'EXPIRED', 'REVOKED')),
    PRIMARY KEY (tenant_id, organization_id, id),
    FOREIGN KEY (tenant_id, organization_id, entry_id)
        REFERENCES qr_nfc_entry (tenant_id, organization_id, id) ON DELETE RESTRICT
);

-- Indexes for lookup patterns.
CREATE INDEX qr_nfc_entry_entitlement_idx
    ON qr_nfc_entry (tenant_id, organization_id, entitlement_id, status);

CREATE INDEX qr_nfc_entry_asset_idx
    ON qr_nfc_entry (tenant_id, organization_id, asset_id, status);

CREATE INDEX qr_nfc_entry_status_idx
    ON qr_nfc_entry (tenant_id, organization_id, status, expires_at);

CREATE INDEX qr_nfc_entry_log_entry_idx
    ON qr_nfc_entry_log (tenant_id, organization_id, entry_id, accessed_at);

CREATE INDEX qr_nfc_entry_log_accessor_idx
    ON qr_nfc_entry_log (tenant_id, organization_id, accessor_id, accessed_at);

-- Forced Row Level Security: tenant + organization isolation.
ALTER TABLE qr_nfc_entry ENABLE ROW LEVEL SECURITY;
ALTER TABLE qr_nfc_entry FORCE ROW LEVEL SECURITY;
ALTER TABLE qr_nfc_entry_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE qr_nfc_entry_log FORCE ROW LEVEL SECURITY;

CREATE POLICY qr_nfc_entry_tenant_organization_isolation ON qr_nfc_entry
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

CREATE POLICY qr_nfc_entry_log_tenant_organization_isolation ON qr_nfc_entry_log
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

COMMIT;
