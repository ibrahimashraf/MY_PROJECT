-- Bootstrap anchor for the frozen pre-rename history.
--
-- Migrations 0001-0084 were authored when the application role was
-- integin_test_runtime, and GRANT statements name it. History is frozen, so those
-- files cannot be rewritten; instead this bootstrap ensures the role name
-- exists (NOLOGIN — it is a grant target only, never a login identity)
-- before any later file references it. The live application role is
-- integin_runtime; nothing may log in as integin_test_runtime.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'integin_test_runtime') THEN
    CREATE ROLE integin_test_runtime WITH NOLOGIN;
  END IF;
END $$;
