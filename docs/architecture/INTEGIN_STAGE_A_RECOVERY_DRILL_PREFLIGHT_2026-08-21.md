# INTEGIN Stage A PostgreSQL and RustFS Recovery Drill Preflight

## Scope

The drill uses a newly generated `it-recovery-*` fixture in the applied pilot database and one generated ciphertext evidence object in the pilot RustFS bucket. It creates a fresh custom-format PostgreSQL backup after the fixture is present, then restores into a new disposable PostgreSQL container. It separately backs up the evidence object through the S3 storage boundary and restores it into a new disposable RustFS container.

| Boundary | Source | Restore target | Prohibited action |
|---|---|---|---|
| PostgreSQL | `integin-pilot-postgres` on `127.0.0.1:15432` | Fresh `postgres:18` container on loopback-only `127.0.0.1:25432` with a new volume | No restore over pilot or non-pilot database. |
| Object storage | `integin-pilot-rustfs` on `127.0.0.1:19000` | Fresh `rustfs/rustfs:1.0.0-rc.1` container on loopback-only `127.0.0.1:29100` with a new volume | No copy into a non-pilot or active bucket. |
| Fixture authority | One generated tenant, organization, actor, work order, scope item, assignment, inspection, and state event | Restored graph is read/verified only | No user, client, certificate, or production data. |

## Acceptance criteria

The source backup must contain the current Work-Order fixture. The restored database must contain exactly one fixture row for each containment and audit record. A non-superuser, `NOBYPASSRLS` role must see the inspection in the matching tenant/organization context and zero rows after an organization mismatch.

The evidence manifest must preserve key, content type, plaintext digest, ciphertext digest, and the five stored metadata values. The restored object must return the same ciphertext digest and metadata through the storage abstraction. The source fixture and source evidence object must be deleted after backup. The disposable target containers and their volumes must be deleted after verification.

## Known limits

The prior 2026-08-15 recovery archive predates the Work-Order schema and is not used for the current completion claim. This drill verifies generated current-schema relational/audit data plus S3 object and metadata recovery; it does not establish a production deployment restoration procedure or a full raw RustFS volume disaster recovery claim.
