REVOKE EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM integin_pilot_runtime;
DROP FUNCTION IF EXISTS integin_resolve_identity_membership(TEXT, TEXT);
DROP TABLE IF EXISTS identity_membership_capability;
DROP TABLE IF EXISTS identity_membership;
DROP TABLE IF EXISTS identity_subject;
