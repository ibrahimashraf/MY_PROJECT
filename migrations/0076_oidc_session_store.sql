-- Identity-scoped OIDC session store backing SessionLifecycleManager so
-- multi-instance restarts retain active session state. A session is keyed by
-- its opaque bearer token, which is itself the isolation boundary: tenant/org
-- RLS is intentionally not applied (subject-bound table) because a session
-- must remain resolvable across the manager's lifecycle checks regardless of
-- the tenant an operator may have last scoped.

BEGIN;

CREATE TABLE IF NOT EXISTS oidc_session_store (
    session_id     TEXT        NOT NULL,
    subject        TEXT        NOT NULL,
    issued_at      TIMESTAMPTZ NOT NULL,
    last_active_at TIMESTAMPTZ NOT NULL,
    revoked_at     TIMESTAMPTZ,
    expires_at     TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (session_id),
    CHECK (btrim(session_id) <> ''),
    CHECK (btrim(subject) <> ''),
    CHECK (expires_at > issued_at)
);

-- Enables PurgeExpired sweeps without scanning the whole table.
CREATE INDEX IF NOT EXISTS oidc_session_store_expiry_idx
    ON oidc_session_store (expires_at);

GRANT SELECT, INSERT, UPDATE, DELETE ON oidc_session_store TO integin_test_runtime;

COMMIT;
