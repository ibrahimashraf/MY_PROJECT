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
