DO $$BEGIN IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'integin_resolve_identity_membership') THEN REVOKE EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM integin_test_runtime; END IF; END$$;
DROP FUNCTION IF EXISTS integin_resolve_identity_membership(TEXT, TEXT);

ALTER TABLE IF EXISTS identity_membership
    DROP CONSTRAINT IF EXISTS identity_membership_work_order_role_ck;

ALTER TABLE IF EXISTS identity_membership
    DROP CONSTRAINT IF EXISTS identity_membership_actor_tenant_org_fk;

DROP INDEX IF EXISTS identity_membership_actor_idx;

ALTER TABLE IF EXISTS identity_membership
    DROP COLUMN IF EXISTS work_order_role,
    DROP COLUMN IF EXISTS actor_id;

DROP TABLE IF EXISTS identity_actor;

