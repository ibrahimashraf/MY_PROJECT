-- Down migration for 0086: restore the legacy resolver function name.
-- Grants follow the function by OID; EXECUTE is re-asserted explicitly.

BEGIN;

DO $$BEGIN IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace WHERE n.nspname = 'public' AND p.proname = 'integin_resolve_identity_membership') THEN ALTER FUNCTION public.integin_resolve_identity_membership(TEXT, TEXT) RENAME TO integin_resolve_identity_membership; END IF; END$$;

REVOKE ALL ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) TO integin_runtime;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'public' AND p.proname = 'integin_resolve_identity_membership'
  ) THEN
    RAISE EXCEPTION '0086 down: legacy resolver function missing after rollback';
  END IF;
END $$;

COMMIT;
