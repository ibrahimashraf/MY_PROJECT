# INTEGIN Live Integration Runbook

This runbook is for the first live PostgreSQL/RustFS validation. Do not start production mode until Docker, PostgreSQL, and RustFS are reachable and a known backup location is available.

## Prerequisites

Start pinned PostgreSQL and RustFS services with persistent volumes. Create the INTEGIN database and a dedicated database role. Create the `integin-evidence` RustFS bucket and a dedicated S3 key restricted to that bucket. Keep passwords and keys in environment variables or a local secret manager.

For local RustFS testing, keep the S3 API on port `9000` and restrict the administration console on port `9001` to administrators. Use writable non-root data and log volumes, TLS or a trusted TLS reverse proxy, and a pinned release rather than `latest`.

## INTEGIN configuration

```text
INTEGIN_DB_URL=<postgresql connection string>
INTEGIN_TENANT_ID=<tenant id>
INTEGIN_EVIDENCE_STORE=rustfs
INTEGIN_S3_ENDPOINT=http://localhost:9000
INTEGIN_S3_BUCKET=integin-evidence
INTEGIN_S3_ACCESS_KEY=<dedicated access key>
INTEGIN_S3_SECRET_KEY=<dedicated secret key>
INTEGIN_S3_REGION=us-east-1
```

Apply migrations in order. Verify PostgreSQL connectivity, tenant RLS, at least one trusted device, and one active authority package before submitting a field transaction. Confirm `/healthz` and `/readyz`.

## Acceptance sequence

Submit a valid signed mutation and expect `APPLIED`. Submit it again and expect `DUPLICATE`. Submit a future sequence and expect `HELD`. Submit a stale sequence and expect `CONFLICT`. Submit an invalid signature, revoked device, expired authority, organization mismatch, or capability violation and expect `SECURITY_FAILURE`.

Upload encrypted evidence and verify separate plaintext/ciphertext digests. Repeat the upload and expect `DUPLICATE`. Alter ciphertext and expect `SECURITY_FAILURE`. Restart PostgreSQL, RustFS, and the Go service, then verify that receipts, sequences, evidence manifests, and encrypted objects remain recoverable.

## Recovery gate

Perform a PostgreSQL backup/restore drill and a RustFS object-plus-manifest restore drill before production promotion. The restore must preserve tenant ownership, evidence IDs, ciphertext bytes, digest values, inspection relationships, and audit records. Record the exact RustFS version, image digest or binary checksum, migration version, backup timestamp, and test results in the release record.

### Binary-safe PostgreSQL backup and isolated restore

Do **not** redirect a custom-format (`pg_dump -Fc`) archive through Windows PowerShell. Create the binary file inside the PostgreSQL container, verify it there with the matching `pg_restore`, and only then copy the artifact to the host backup location. For a restore drill, create a separate verification database such as `integin_restore_drill`; never restore over the active `integin` database.

```text
1. In the PostgreSQL container: pg_dump -Fc -f /tmp/integin-recovery.dump integin
2. In the same container: pg_restore -l /tmp/integin-recovery.dump
3. Copy the verified dump and table-of-contents file to the protected backup directory.
4. Create integin_restore_drill and restore the archive there with --no-owner --no-privileges.
5. Verify expected device, authority, receipt, held-transaction, and RLS-table counts.
```

The `--no-owner --no-privileges` isolated-drill option proves data/schema recovery without altering host roles or active grants. A real operational recovery must additionally recreate or validate the runtime role and grants according to `RLS_RUNTIME_ROLE.md`.

### RustFS object archive and manifest drill

Export two artifacts from the RustFS data container: an archive of `/data` and a sorted SHA-256 manifest for every file under `/data`. Restore the archive into a **new Docker volume**, regenerate the SHA-256 manifest there, and require an exact manifest match. Do not extract an archive over the active RustFS data volume.

After a production-style RustFS restore into a separately started service, also perform S3 `HEAD`/`GET` verification against evidence-manifest records to confirm the object API and digest binding, not only archive bytes.

### Completed local recovery drill — 2026-08-15

The local drill created a protected recovery set under `C:\INTEGIN-RUNTIME\backups\integin-recovery-20260815-031148`. The PostgreSQL custom dump and catalog were generated and verified inside the PostgreSQL container, then restored into the isolated `integin_restore_drill` database. The restored database reported six devices, four active authority records, five receipts, and RLS enabled on all six protected tables. The RustFS `/data` archive was restored into a fresh `integin-rustfs-restore-drill` Docker volume; the regenerated file manifest matched the source manifest exactly. The active `integin` database and `integin-rustfs-data` volume were not replaced, and the active server was restarted successfully after the drill.
