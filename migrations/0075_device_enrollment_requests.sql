-- Device enrollment request lifecycle: PENDING -> APPROVED / REJECTED.
-- Tenant/org composite-isolated persistence for the device trust enrollment
-- flow (internal/platform/devicetrust), using the hardened NULLIF session-GUC
-- isolation pattern shared with 0071/0073.

BEGIN;

CREATE TABLE IF NOT EXISTS device_enrollment_requests (
    request_id        TEXT        NOT NULL,
    tenant_id         TEXT        NOT NULL,
    organization_id   TEXT        NOT NULL,
    user_id           TEXT        NOT NULL,
    device_id         TEXT        NOT NULL,
    public_key        TEXT        NOT NULL,
    nonce             TEXT        NOT NULL,
    signature         TEXT        NOT NULL,
    attestation       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    status            TEXT        NOT NULL DEFAULT 'PENDING'
                      CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
    requested_at      TIMESTAMPTZ NOT NULL,
    approved_by       TEXT        NOT NULL DEFAULT '',
    rejected_by       TEXT        NOT NULL DEFAULT '',
    rejection_reason  TEXT        NOT NULL DEFAULT '',
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, request_id),
    CHECK (btrim(request_id) <> ''),
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(organization_id) <> ''),
    CHECK (btrim(user_id) <> ''),
    CHECK (btrim(device_id) <> ''),
    CHECK (btrim(public_key) <> ''),
    CHECK (btrim(nonce) <> ''),
    CHECK (btrim(signature) <> '')
);

CREATE INDEX IF NOT EXISTS device_enrollment_requests_device_idx
    ON device_enrollment_requests (tenant_id, organization_id, device_id);

CREATE INDEX IF NOT EXISTS device_enrollment_requests_status_idx
    ON device_enrollment_requests (tenant_id, organization_id, status);

ALTER TABLE device_enrollment_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE device_enrollment_requests FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS device_enrollment_requests_tenant_isolation ON device_enrollment_requests;
CREATE POLICY device_enrollment_requests_tenant_isolation ON device_enrollment_requests
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );

GRANT SELECT, INSERT, UPDATE ON device_enrollment_requests TO integin_test_runtime;

COMMIT;
