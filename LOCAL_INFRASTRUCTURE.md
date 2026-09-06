# INTEGIN Local Infrastructure Contract

This document defines the local development sequence for the live integration gate. It is intentionally credential-free. Use a private environment file derived from `INTEGIN_LOCAL_ENV.example`; never place database passwords, RustFS secrets, signing keys, or authority material in source control.

## Start order

Install and verify Docker Desktop with WSL 2, PostgreSQL (port `15432`), and RustFS. Start PostgreSQL with a persistent data volume and create a dedicated `integin_dev` database plus an application role (`integin_runtime`). Start RustFS with persistent data and log volumes, expose its S3 endpoint on local port `19000`, restrict the administration console on `19001`, and create the `integin-pilot-evidence` bucket with dedicated access credentials (see `WORKSPACE.md` for live environment variables).

The exact RustFS binary or container command must follow the official release instructions for the installed package. Do not substitute `latest`, and record the binary checksum or container digest in the integration record. RustFS is an integration-test dependency here, not a production approval.

## Migration order

From the repository root, apply the migrations in ascending order using the same database URL that will be supplied to the Go server:

```powershell
$env:PGPASSWORD = "<database-password>"
psql "$env:INTEGIN_DB_URL" -f migrations/0001_event_log.sql
psql "$env:INTEGIN_DB_URL" -f migrations/0002_device_trust_sync.sql
psql "$env:INTEGIN_DB_URL" -f migrations/0003_event_log_tenant_rls.sql
Remove-Item Env:PGPASSWORD
```

Verify that `event_log`, `device_registry`, `authority_package`, `sync_device_state`, `sync_receipt`, and `sync_held_transaction` exist. Verify RLS is enabled on the event-log parent/default partition and the five trust/sync tables, and that a transaction-local `integin.tenant_id` is required for tenant-scoped reads and writes. Bootstrap only test tenant, trusted-device, and active-authority records through an explicit local setup procedure. Use a non-owner, non-superuser runtime database role for `INTEGIN_DB_URL`; the bootstrap database role is not the application runtime identity.

## Start and readiness gate

Export the values in `INTEGIN_LOCAL_ENV.example`, start `cmd/integin-server`, and wait for both endpoints:

```powershell
Invoke-WebRequest http://127.0.0.1:18080/healthz
Invoke-WebRequest http://127.0.0.1:18080/readyz
```

`/healthz` proves the process is alive. `/readyz` must remain the gate for PostgreSQL and RustFS dependency readiness. Do not run the Flutter-to-Go acceptance matrix if readiness is not successful.

## Stop conditions

Stop and record the result if the tenant is missing, the device is not trusted, the authority is expired or bound to another organization/environment, RLS can be bypassed, the bucket is not reachable with SigV4, or encrypted evidence is stored without both plaintext and ciphertext digests. These are integration failures, not reasons to weaken the server-authoritative contract.
