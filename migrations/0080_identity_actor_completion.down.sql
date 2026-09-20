-- Rollback for 0080_identity_actor_completion: reverts Ticket 02-01 completion
-- to the 0009 state. Restores least privilege: re-forward via 0080 re-grants.
DO $$BEGIN IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'integin_resolve_identity_membership') THEN REVOKE EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM integin_runtime; END IF; END$$;
REVOKE ALL ON TABLE identity_actor, identity_subject, identity_membership, identity_membership_capability FROM integin_runtime;
DROP FUNCTION IF EXISTS integin_resolve_identity_membership(TEXT, TEXT);
DROP INDEX IF EXISTS identity_membership_actor_idx;
ALTER TABLE IF EXISTS identity_membership DROP CONSTRAINT IF EXISTS identity_membership_actor_tenant_org_fk;
ALTER TABLE IF EXISTS identity_membership DROP CONSTRAINT IF EXISTS identity_membership_work_order_role_ck;
ALTER TABLE IF EXISTS identity_membership ALTER COLUMN actor_id DROP NOT NULL;
ALTER TABLE IF EXISTS identity_membership ALTER COLUMN work_order_role DROP NOT NULL;
DROP TABLE IF EXISTS identity_actor;
