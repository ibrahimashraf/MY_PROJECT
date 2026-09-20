-- Rollback for 0009_identity_work_order_actor.
DO $$BEGIN IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'integin_resolve_identity_membership') THEN REVOKE ALL ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM integin_runtime; END IF; END$$;
DROP FUNCTION IF EXISTS integin_resolve_identity_membership(TEXT, TEXT);
ALTER TABLE IF EXISTS identity_membership DROP CONSTRAINT IF EXISTS identity_membership_actor_role_required;
ALTER TABLE IF EXISTS identity_membership DROP COLUMN IF EXISTS work_order_role;
ALTER TABLE IF EXISTS identity_membership DROP COLUMN IF EXISTS actor_id;
