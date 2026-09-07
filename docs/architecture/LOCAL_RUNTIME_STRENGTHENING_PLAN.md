# INTEGIN Local Runtime-Strengthening Plan

**Status:** Draft execution plan — no runtime change authorized by this document  
**Purpose:** Make the current local INTEGIN capability stronger without sacrificing the already proven Flutter/Go/PostgreSQL/RustFS acceptance environment.

## 1. Decision

The correct way to strengthen the local runtime is **not** to install every new component into the existing acceptance environment at once. The correct way is to create a separate, isolated **local pilot clone**, prove the additions there, and promote one validated capability at a time.

The existing runtime remains the **control environment**. It has demonstrated server-authoritative field sync and evidence handling, durable PostgreSQL state, RustFS object behavior, and binary-safe recovery. The cloned runtime becomes the **change environment** for Keycloak, OpenBao, OIDC validation, production enrollment, observability, and later signed checkpoints.

> **A local clone makes the platform stronger because it adds repeatability, comparison, rollback, and safety—not because it adds more containers.**

## 2. Target local topology

| Concern | Current acceptance environment | Proposed local pilot clone |
|---|---|---|
| Purpose | Preserve proven acceptance/recovery control | Rehearse production-trust changes safely |
| Directory | `C:\INTEGIN-RUNTIME` | `C:\INTEGIN-PILOT` |
| Go server | Existing provision-enabled acceptance binary on `127.0.0.1:8080` | Separate binary/config on `127.0.0.1:18080` |
| PostgreSQL | Existing `integin-postgres`, standard local port `5432` | Separate `integin-pilot-postgres` data volume, loopback-only `15432` or Docker-network-only access |
| RustFS | Existing `integin-rustfs` data/log volumes, ports `9000/9001` | Separate `integin-pilot-rustfs` data/log volumes, loopback-only `19000/19001` or Docker-network-only access |
| Identity | Not yet present | Separate Keycloak service and a separate cloned PostgreSQL identity database, loopback-only `18081` during rehearsal |
| Secrets | Existing private development environment file | Isolated OpenBao rehearsal service and separate non-production recovery material, loopback-only `18200` |
| Networking | Existing Docker configuration | Dedicated `integin-pilot-net`; no pilot data service publicly exposed |
| Data | Existing validated local integration data | Synthetic/controlled clone only; no automatic copy or shared volume |

The pilot clone must use different container names, named volumes, network, configuration file, database role/passwords, bucket names/prefixes, key IDs, OIDC realm/client IDs, and service ports. It must never mount, write to, or restore over an existing acceptance volume.

## 3. Build sequence

### Stage A — Freeze a recoverable control point

Before building the clone, create a dated acceptance-runtime baseline: Go binary/source revision, container image digests, environment-template revision, migration version, RustFS bucket/object manifest summary, PostgreSQL catalog/backup reference, and the completed Flutter acceptance evidence. This is an inventory, not a production deployment.

No current container is stopped, recreated, renamed, reconfigured, or given a new secret in this stage. The goal is to ensure there is a known good state if later experimentation must be compared or discarded.

### Stage B — Build independent state boundaries

Create `integin-pilot-net`, pilot-only container names, pilot-only named volumes, `C:\INTEGIN-PILOT\` configuration, and loopback-only port bindings. Create empty pilot PostgreSQL/RustFS instances; apply migrations to the pilot database; configure pilot runtime roles; and create pilot storage buckets/prefixes through the existing Go-owned contract.

The clone uses only synthetic data or a specifically approved disposable fixture. It does not clone private credentials or live evidence. Any backup restore exercise is restored into a **new pilot volume**, following the already proven binary-safe and fresh-volume procedures.

### Stage C — Prove the unchanged core on the clone

Before adding identity or secrets services, prove that the pilot Go/RustFS/PostgreSQL stack satisfies the same basic checks as the control: migrations, RLS/runtime role, object upload/digest/receipt behavior, restart behavior, and a Flutter end-to-end field acceptance session. This distinguishes clone/environment errors from later Keycloak/OpenBao/OIDC issues.

### Stage D — Add OpenBao as a rehearsal boundary

Deploy OpenBao only inside the pilot network/loopback boundary with distinct non-production recovery material. Start with a minimal test policy and a synthetic service secret. Prove that the Go and Keycloak rehearsal services can retrieve only their assigned secret path and that unauthorized access, logging, recovery, and rotation behavior are understood.

OpenBao does not receive a field-device private key, customer evidence, or a production signing key. It does not replace the existing private local environment file until its recovery design and test pass.

### Stage E — Add Keycloak and disabled-by-default Go OIDC validation

Deploy a pilot-only Keycloak backed by its own pilot identity database. Configure one pilot realm and one Flutter/native client with Authorization Code + PKCE only. Implement Go OIDC discovery/JWKS validation behind a disabled-by-default feature flag, local `(issuer, sub)` mapping, and transaction-scoped organization/tenant resolution.

The existing acceptance route stays available in the control environment. The pilot must first prove negative cases: invalid issuer, wrong audience, expired token, non-PKCE client, cross-organization selection, absent membership, and missing MFA for privileged paths. No OIDC role/group claim alone can grant tenant access.

### Stage F — Implement production enrollment in the clone

Replace local provisioning in the pilot clone only with the approved production lifecycle: OIDC-bound request, server-issued five-minute single-use challenge, Ed25519 proof, local authorization/approval, server-derived scoped authority, device epoch, revocation, replacement, and audit records. The local provisioning bridge remains disabled in pilot production-mode testing.

The field app must generate and retain its private key locally. The test suite must demonstrate that authority scope/duration cannot be client-expanded, a challenge cannot be replayed, cross-tenant enrollment is rejected, self-approval is blocked where required, and post-revocation sync is rejected.

### Stage G — Add observability and audit verification last

Add OTel Collector/Prometheus/Loki/Grafana only after the authority flows are stable. Instrument allowlisted correlation IDs and outcome codes, never tokens, private keys, evidence bytes, full signed payloads, or raw personal profiles. Add signed audit checkpoints and export-manifest verification after the basic observability data-minimization tests pass.

## 4. Promotion gates to the acceptance runtime

No capability moves from the pilot clone to the acceptance runtime merely because it starts. Every promotion is a small, separate change with a new baseline backup and an explicit rollback decision.

| Capability | Pilot proof required | Acceptance-runtime promotion rule | Rollback boundary |
|---|---|---|---|
| OpenAPI/canonical vectors | Go/Flutter test vectors pass; compatibility recorded | Documentation/tests first; no runtime behavior change required | Revert versioned schema/test artifact only |
| OpenBao integration | Least privilege, rotation, recovery, audit, and outage tests pass | Replace only the specific server-secret reference after backup and recovery test | Restore prior private environment reference; revoke pilot-only credential |
| Keycloak/OIDC validation | Negative token, RLS, role, MFA, and recovery tests pass | Add feature flag default off, then enable only for a named pilot session | Disable flag and invalidate new sessions; existing local acceptance path remains untouched until cutover |
| Production enrollment | Challenge/proof/approval/revoke/replacement and Flutter acceptance pass | Deploy as a distinct endpoint/feature with local provisioning disabled in that mode | Disable endpoint; preserve audit/device records; do not reactivate a revoked device |
| Observability | Secret/evidence leak scans, dashboard/alert, and outage tests pass | Add sidecar/collector config with telemetry non-blocking | Remove exporter/config; workflow remains functional |
| Signed audit checkpoints | Deterministic vector, signature, export, altered-data detection, recovery test pass | Enable new checkpoint schedule without modifying existing audit events | Stop new checkpoint issuance; historic event/audit data remains intact |

Every acceptance-runtime promotion starts with: a verified PostgreSQL backup; RustFS manifest/archive reference; current binary/config/image digest; health/smoke checks; named decision owner; and an explicit stop/rollback trigger. Database-destructive changes are prohibited unless separately approved through Decision 05 and rehearsed in the pilot clone.

## 5. What this approach changes now—and what it does not

| Strengthens now | Does not change now |
|---|---|
| Repeatable local promotion path | Existing `integin-postgres` container/data |
| Clear isolation of experimental service state | Existing RustFS container/data/log volumes |
| Safer OIDC, secrets, enrollment, and telemetry experimentation | Existing Go acceptance server on `127.0.0.1:8080` |
| Measured recovery and rollback confidence | Existing Flutter acceptance app configuration |
| A local rehearsal of the production-trust architecture | Customer-facing brand/name, production hosting, or public exposure |

## 6. First executable local pilot slice

The first slice is intentionally modest and can be completed without touching the control runtime:

1. Record and back up the current acceptance baseline.
2. Create the pilot directory, Docker network, empty PostgreSQL/RustFS volumes, and loopback-only ports.
3. Apply the current migrations and reproduce the core Go/RustFS/PostgreSQL checks in the clone.
4. Create versioned OpenAPI v1 and canonical signature/enrollment test-vector fixtures.
5. Stop and compare results before introducing OpenBao or Keycloak.

Only after this slice passes should the pilot add OpenBao, then Keycloak/OIDC, then production enrollment, in that order. This preserves the ability to identify exactly which change improves or harms the system.

## 7. Decision

**Yes, strengthening the local runtime is valuable.** The stronger path is a two-runtime method: preserve the successful acceptance runtime as the control, build a fully isolated local pilot clone, prove each production-trust component there, and promote only validated, reversible slices. Directly changing the current runtime remains appropriate only after a given capability has passed the clone’s security, recovery, tenant-isolation, evidence, and rollback gates.

## Related documents

- [Production-Trust Baseline Roadmap](./PRODUCTION_TRUST_BASELINE_ROADMAP.md)
- [Production Device Enrollment Specification](./PRODUCTION_DEVICE_ENROLLMENT_SPEC.md)
- [Decision 01 — Production Operating Model](./DECISION_01_PRODUCTION_OPERATING_MODEL.md)
- [Decision 04 — Recovery Objectives](./DECISION_04_RECOVERY_OBJECTIVES.md)
- [Decision 05 — Supply-Chain and Release Integrity Policy](./DECISION_05_SUPPLY_CHAIN_RELEASE_POLICY.md)
