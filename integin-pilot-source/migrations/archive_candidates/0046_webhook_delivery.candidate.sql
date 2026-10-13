-- Migration 0046: Webhook Delivery Tracking
-- TEXT IDs, combined RLS, FORCE RLS

BEGIN;

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    short_link_code TEXT NOT NULL REFERENCES short_links(code) ON DELETE CASCADE,
    payload JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'delivered', 'failed', 'dead_letter')) DEFAULT 'pending',
    attempt INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 6,
    next_retry_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_tenant_org ON webhook_deliveries (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_short_link_code ON webhook_deliveries (tenant_id, organization_id, short_link_code);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON webhook_deliveries (tenant_id, organization_id, status);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_next_retry_at ON webhook_deliveries (next_retry_at) WHERE next_retry_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_created_at ON webhook_deliveries (tenant_id, organization_id, created_at DESC);

ALTER TABLE webhook_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhook_deliveries FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS webhook_deliveries_isolation ON webhook_deliveries;
CREATE POLICY webhook_deliveries_isolation ON webhook_deliveries
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE OR REPLACE FUNCTION update_webhook_delivery_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql VOLATILE;

DROP TRIGGER IF EXISTS trigger_update_webhook_delivery_updated_at ON webhook_deliveries;
CREATE TRIGGER trigger_update_webhook_delivery_updated_at
    BEFORE UPDATE ON webhook_deliveries
    FOR EACH ROW
    EXECUTE FUNCTION update_webhook_delivery_updated_at();

GRANT SELECT, INSERT, UPDATE, DELETE ON webhook_deliveries TO integin_runtime;

COMMIT;
