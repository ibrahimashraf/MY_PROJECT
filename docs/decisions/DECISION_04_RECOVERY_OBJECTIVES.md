# Decision 04 — Recovery Objectives, Backup, and Restore Policy

**Status:** Draft for approval  
**Decision owner:** Platform Operator  
**Required approvers:** Product/Safety Authority; Security Operator  
**Scope:** Recovery objectives and evidence for the controlled production pilot.

## Decision

INTEGIN will operate as a recoverable single-site pilot. Recovery is measured by the time required to restore an approved service and by the maximum approved loss of accepted server-authoritative data. The proposed objectives below are **pilot targets, not claims of current capability**. Each target becomes binding only after the corresponding backup frequency, retention, restoration drill, and operator procedure prove it.

| Service/data set | Proposed pilot RTO | Proposed pilot RPO | Rationale |
|---|---:|---:|---|
| Go authority/API | 4 hours | Not applicable to stateless binaries; configuration/release must be reproducible | Restoring authoritative service availability is urgent; accepted state lives elsewhere. |
| INTEGIN PostgreSQL | 4 hours | 1 hour | Contains tenant state, device trust, receipt/audit/workflow records; losing more than a bounded hour requires formal Safety Authority decision. |
| RustFS evidence objects and metadata-linked archive | 8 hours | 1 hour | Evidence availability matters, but integrity and exact manifest matching matter more than unsafe speed. |
| Keycloak identity database/configuration | 4 hours | 24 hours | Identity configuration changes less frequently; a loss must not silently create/alter INTEGIN membership. |
| OpenBao configuration and secret recovery material | 4 hours | 24 hours | Service recovery cannot depend on an unrecoverable secret store; normal operations use leases/rotation, not secret history. |
| Audit checkpoint/export artifacts | 8 hours | 24 hours | Independent verification must remain possible after the main service returns. |
| Observability configuration | 24 hours | 24 hours | Important for operations but must not block authoritative field work. |

These targets require revision if a pilot customer, contract, regulator, or Safety Authority needs a shorter interruption/data-loss limit. A target without a successful drill is a desired service level, not an accepted capability.

## Backup policy

PostgreSQL uses binary-safe native-format dumps and, before RPO claims are made, an approved schedule that proves the one-hour target. Streaming/WAL-based recovery may be introduced if it is necessary to meet the agreed RPO; it is not presumed to exist. RustFS backup archives contain an explicit SHA-256 manifest, are created internally without unsafe host-shell binary redirection, and are restored only into a fresh isolated volume for verification. Identity and secrets backups are separately encrypted, access controlled, and exercised without using production credentials in development.

Every backup record must include source environment, time, component version, schema/migration level, manifest or checksum, retention expiry, encryption/key reference, storage location class, operator/job identity, and verification outcome. A backup that merely completes is not “good” until a restore verifies its intended integrity property.

## Recovery order

1. Declare the incident and freeze nonessential changes; preserve relevant logs and release references.
2. Establish a clean/replacement host and secured operations access.
3. Restore OpenBao/key access through its approved dual-control procedure.
4. Restore PostgreSQL into an isolated target and verify schema, RLS, runtime-role boundaries, and row-level state.
5. Restore RustFS into fresh storage; compare archive/object manifests and digests.
6. Restore Keycloak/configuration as required; rotate/revoke sessions and keys if the incident demands it.
7. Deploy the approved Go release/configuration and perform readiness checks.
8. Run independent export/checkpoint verification and sample tenant-safe end-to-end acceptance.
9. Obtain Platform Operator and Product/Safety Authority approval before reopening pilot traffic.
10. Record actual RTO/RPO, exceptions, root cause, and follow-up controls.

## Recovery acceptance criteria

| Drill | Required outcome |
|---|---|
| PostgreSQL isolated restore | Expected tables, migrations, runtime role, RLS state, devices/authorities/receipts, and tenant controls validate. |
| RustFS fresh-volume restore | Archive manifest matches exactly; no restore overwrites the source production volume. |
| Identity/secrets recovery | Authorized test login and runtime-secret retrieval work; no membership/authority is granted merely by restore. |
| Authority/revocation check | Revoked device and expired/old authority remain rejected after restoration. |
| Independent evidence check | Export manifest/checkpoint verifier accepts clean recovered data and rejects an altered fixture. |
| Recovery measurement | Actual start/end times and maximum accepted data timestamp are recorded against RTO/RPO targets. |

## Review triggers

Revisit objectives after the first pilot drill, material data-volume growth, a customer obligation, a production incident, identity/secrets deployment, or a stable RustFS/object-storage decision.

## Related records

- [Decision 01 — Production Operating Model](./DECISION_01_PRODUCTION_OPERATING_MODEL.md)
- [Decision 03 — Cryptographic Key Lifecycle](./DECISION_03_CRYPTOGRAPHIC_KEY_LIFECYCLE.md)
- [Decision 06 — Operator Runbooks](./DECISION_06_OPERATOR_RUNBOOKS.md)
