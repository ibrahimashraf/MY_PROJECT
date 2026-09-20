DO $$BEGIN IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'integin_resolve_identity_membership') THEN REVOKE ALL ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM integin_runtime; END IF; END$$;
DROP FUNCTION IF EXISTS integin_resolve_identity_membership(TEXT, TEXT);
DROP TABLE IF EXISTS identity_membership_capability;
DROP TABLE IF EXISTS identity_membership;
DROP TABLE IF EXISTS identity_subject;
