-- Rename the identity resolver function integin_* -> integin_*.
--
-- The function body is a pure SELECT over identity_* tables with no GUC
-- references, so a plain ALTER FUNCTION ... RENAME is semantics-preserving.
-- Grants follow the function by OID; EXECUTE for integin_runtime is
-- re-asserted explicitly. Historical migration files are left frozen.

BEGIN;

ALTER FUNCTION public.integin_resolve_identity_membership(TEXT, TEXT)
  RENAME TO integin_resolve_identity_membership;

REVOKE ALL ON FUNCTION public.integin_resolve_identity_membership(TEXT, TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.integin_resolve_identity_membership(TEXT, TEXT) TO integin_runtime;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'public' AND p.proname = 'integin_resolve_identity_membership'
  ) THEN
    RAISE EXCEPTION '0086: legacy resolver function still present after rename';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'public' AND p.proname = 'integin_resolve_identity_membership'
  ) THEN
    RAISE EXCEPTION '0086: renamed resolver function missing after rename';
  END IF;
END $$;

COMMIT;
