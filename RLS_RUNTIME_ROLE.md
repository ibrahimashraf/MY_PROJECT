# INTEGIN PostgreSQL Runtime Role

The PostgreSQL role created by `POSTGRES_USER` initializes the local database. It is a bootstrap role and must not be used as the INTEGIN runtime identity because superusers bypass row-level security. INTEGIN should connect with a separate non-owner, non-superuser login role that has no `BYPASSRLS` attribute. PostgreSQL documents that superusers and roles with `BYPASSRLS` bypass row-security policies; table owners can also bypass them unless row security is forced. [1]

For local integration, connect to the bootstrap role inside the Docker container and create the runtime role with a private password. Grant only schema usage and the required table/sequence permissions. Then set `INTEGIN_DB_URL` to the `INTEGIN_runtime` connection string from `INTEGIN_LOCAL_ENV.example`.

```sql
CREATE ROLE INTEGIN_runtime LOGIN PASSWORD '<private-local-password>' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
GRANT CONNECT ON DATABASE INTEGIN TO INTEGIN_runtime;
GRANT USAGE ON SCHEMA public TO INTEGIN_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO INTEGIN_runtime;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO INTEGIN_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE INTEGIN_app IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO INTEGIN_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE INTEGIN_app IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO INTEGIN_runtime;
```

The runtime role is a live-integration control, not a replacement for application checks. The Go repository continues to set `INTEGIN.tenant_id` transaction-locally, and RLS remains a second tenant-isolation boundary.

## References

[1]: https://www.postgresql.org/docs/current/ddl-rowsecurity.html "PostgreSQL documentation: Row Security Policies"
