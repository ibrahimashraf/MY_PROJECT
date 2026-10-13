-- Reassign all functions owned by integin_pilot_runtime
DO $$ DECLARE r RECORD; BEGIN
  FOR r IN SELECT oid::regprocedure::text as sig FROM pg_proc WHERE proowner = (SELECT oid FROM pg_roles WHERE rolname = 'integin_pilot_runtime') LOOP
    EXECUTE 'ALTER FUNCTION ' || r.sig || ' OWNER TO integin_pilot_owner';
  END LOOP;
END $$;

-- Revoke remaining privileges
REVOKE ALL ON DATABASE integin_pilot FROM integin_pilot_runtime;
REVOKE ALL ON SCHEMA public FROM integin_pilot_runtime;
REVOKE ALL ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM integin_pilot_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE integin_pilot_runtime REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE integin_pilot_runtime REVOKE ALL ON SEQUENCES FROM PUBLIC;

-- Drop the role (owned objects already reassigned)
DROP OWNED BY integin_pilot_runtime;
DROP ROLE IF EXISTS integin_pilot_runtime;
