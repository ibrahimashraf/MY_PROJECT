-- INTEGIN Ticket 03 D7-4 rollback: remove signed submission hardening from work_order_operation.
-- Reverses migrations/0043_work_order_signed_submission.sql.
-- Keeps base 0009 persistence and 0010 RLS baseline intact; removes only D7-4 additive columns
-- and the explicit replay-guard index created by this slice. Disposable isolated
-- rollback only; never pilot.

DROP POLICY IF EXISTS work_order_operation_tenant_organization_isolation ON work_order_operation;

-- Restore baseline 0010 RLS for work_order_operation so rollback leaves tenant isolation intact
-- (0010 remains the baseline; a full RLS rollback would use 0010 down).
ALTER TABLE work_order_operation ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_operation FORCE ROW LEVEL SECURITY;
CREATE POLICY work_order_operation_tenant_organization_isolation ON work_order_operation
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

-- Grants: restore 0010 least-privilege grants for this table (idempotent).
REVOKE ALL ON TABLE work_order_operation FROM PUBLIC, integin_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE work_order_operation TO integin_runtime;
GRANT USAGE ON SCHEMA public TO integin_runtime;

-- Drop replay-guard index that was explicitly created by D7-4.
-- Preserve 0009's work_order_operation_idempotency_idx (base replay guard) and the
-- UNIQUE (tenant_id, organization_id, operation_id) constraint.
DROP INDEX IF EXISTS work_order_operation_operation_id_idx;

-- Drop payload-hash hex check added by D7-4.
ALTER TABLE work_order_operation DROP CONSTRAINT IF EXISTS work_order_operation_payload_hash_hex_ck;

-- Drop signed-payload columns added by D7-4.
ALTER TABLE work_order_operation DROP COLUMN IF EXISTS signing_key_id;
ALTER TABLE work_order_operation DROP COLUMN IF EXISTS signature;
ALTER TABLE work_order_operation DROP COLUMN IF EXISTS payload_hash;
