# INTEGIN Platform Governance Backlog

**Status:** Active foundation backlog  
**Date:** 2026-08-15  
**Scope:** Governance work required to evolve the current self-hosted industrial assurance platform without weakening the Go authority, PostgreSQL tenant isolation, RustFS evidence contract, Flutter offline boundary, or advisory-only AI rule.

## Governing principle

> **Every new capability must have one accountable owner, one authoritative write path, explicit tenant/organization/environment context, an auditable approval record, and a recovery procedure before it can affect a customer workflow.**

The current Go modular monolith remains the only authority for workflow acceptance, certificate controls, device trust, authority packages, signed sync, and primary audit state. PostgreSQL remains the durable authority and RustFS remains an S3-compatible encrypted-evidence store. The local provisioning bridge completed in Phase 17 is a development/integration mechanism only; it is not a production enrollment design.

## Governance roles

| Role | Accountable decision | Must not do |
|---|---|---|
| Product and Safety Authority | Procedure version, inspection policy, certificate/revocation policy, safety release approval | Delegate a safety or certification decision to AI or a client device |
| Tenant Administrator | Organization membership, user roles, device-enrollment request approval | Directly bypass a signature, RLS boundary, authority epoch, or retention rule |
| Platform Operator | Runtime deployment, backup execution, recovery drill, monitoring response | Alter tenant business records to resolve an operational incident |
| Security Operator | Identity-provider configuration, key-rotation policy, incident containment, audit access | Export evidence or private keys without approved procedure and audit record |
| Advisory Reviewer | Review non-blocking Python/AI signals and request human follow-up | Mutate inspection verdicts, certificates, authorities, or sync outcomes |
| Engineering Change Owner | Contract versioning, migration safety, test evidence, rollback plan | Merge an authoritative-path change without fitness-test evidence |

## Prioritized implementation backlog

| Priority | Work item | Owner | Required outcome | Exit evidence |
|---|---|---|---|---|
| P0 | Replace local-only provisioning with production device enrollment | Security Operator + Go authority | Authenticated, approval-based device registration with public-key proof of possession, scoped authority issue, revocation, and audit record | Cross-tenant rejection, duplicate-key handling, approval/revocation tests, no private-key transport |
| P0 | Introduce self-hosted OIDC and organization-aware authorization | Security Operator | Human identities, MFA policy, service accounts, role claims, session lifecycle, and tenant membership projection | Unauthorized/cross-tenant access tests, break-glass procedure, backup of identity configuration |
| P0 | Define evidence export manifest v1 | Product and Safety Authority + Platform Operator | A tenant-scoped manifest maps each evidence ID to object key, encryption metadata, plaintext/ciphertext digests, inspection/certificate linkage, and export timestamp | Rust verifier can validate a sample export without database write access |
| P0 | Publish OpenAPI-derived API contract v1 | Engineering Change Owner | Versioned request/response schemas, sync/evidence outcomes, error codes, deprecation rules, and generated client compatibility checks | Contract CI check and Flutter/TypeScript compatibility test |
| P1 | Add observability foundation | Platform Operator | OpenTelemetry-compatible traces, structured logs, metrics, dashboards, and alerts with no evidence bytes or secrets | Trace a test transaction across sync/evidence; alert on readiness, held backlog, authority expiry, and backup failure |
| P1 | Establish formal release and migration gates | Engineering Change Owner | Versioned migration approval, rollback declaration, fixture replay, tenant/RLS test, evidence digest test, and recovery check | Release record links artifact checksum, migration list, test report, and rollback plan |
| P1 | Define retention, legal hold, and export access policy | Product and Safety Authority + Security Operator | Configurable tenant retention schedule, immutable legal hold behavior, export approvals, and deletion evidence | Policy acceptance tests and audit records for hold/export/delete actions |
| P2 | Build TypeScript operations workbench | Product + Operations | Least-privilege views for device/authority lifecycle, held sync review, audit search, evidence metadata, and reports | No direct DB access; API authorization and privacy tests |
| P2 | Build independent Rust export verifier | Engineering Change Owner | Read-only verification of manifest, signed envelope, authority package, and ciphertext digest | Repeatable verifier attestation for a recovery export |
| P2 | Implement advisory governance register | Advisory Reviewer | Model/version inventory, prompt/data constraints, evidence references, feedback loop, and `blocking=false` enforcement | Audit view proves no advisory operation has primary write authority |

## Production device-enrollment design gate

The completed local bridge is intentionally constrained to loopback local integration. It must be disabled in every production deployment. The production successor must be introduced only when all conditions below are met.

| Requirement | Design rule |
|---|---|
| Human identity | Enrollment request is tied to an authenticated OIDC subject and tenant membership. |
| Device identity | The client generates Ed25519 material locally; the private key never leaves secure storage. |
| Proof of possession | The client signs a server challenge before the authority accepts its public key. |
| Approval | A tenant administrator or controlled workflow approves enrollment before trust changes from pending to trusted. |
| Authority package | The server creates scoped, expiring, epoch-bound authority only after approval. |
| Revocation | Revocation is tenant-scoped, has a reason and timestamp, invalidates authority, and is auditable. |
| Recovery | Lost-device, key-rotation, re-enrollment, and offline authority-expiry procedures are documented and tested. |

## Evidence export manifest v1

The first portable evidence export must include a signed or otherwise integrity-protected JSON manifest. The manifest must be tenant-scoped and contain only authorized data.

| Field group | Minimum fields |
|---|---|
| Export identity | `manifest_version`, `export_id`, `tenant_id`, `organization_id`, `created_at`, exporter identity, procedure version |
| Evidence identity | `evidence_id`, `object_key`, content type, capture timestamp, inspection/entity reference |
| Integrity | `plaintext_sha256`, `ciphertext_sha256`, ciphertext byte length, encryption algorithm/key reference without secret material |
| Authority linkage | device ID, authority ID/epoch, transaction/receipt IDs, signature algorithm and key ID where applicable |
| Recovery | object count, manifest checksum, missing-object list, restore verification result |
| Privacy | classification, retention/hold state, redaction policy reference, export approval record |

The Rust verifier must read this artifact and referenced object files without database credentials. A manifest or object verification failure must never silently modify the authoritative record.

## OpenAPI and compatibility governance

The Go authority must publish a versioned OpenAPI contract for every client-facing route. Additive fields are allowed within a supported version; breaking changes require a new versioned route or media type, a migration note, and compatibility tests for the Flutter field client and TypeScript operations UI.

| Change type | Required control |
|---|---|
| Add optional response field | Schema review and generated-client compatibility test |
| Add request field | Explicit default/validation behavior and client-version support note |
| Change canonical signed-envelope field | New canonical version, test vectors, replay suite, and migration gate |
| Change sync outcome semantics | Approval by Safety Authority, state migration/replay test, and operations runbook update |
| Deprecate endpoint | Published retirement date, telemetry for remaining callers, and documented replacement |

## Operational metrics and alerts

The first telemetry set should be small, tenant-safe, and operationally useful.

| Signal | Alert / review condition |
|---|---|
| `integin_sync_outcomes_total{outcome}` | Sudden rise in `SECURITY_FAILURE`, `CONFLICT`, or `HELD` |
| `integin_held_transactions` | Any aged held transaction beyond the agreed operational threshold |
| `integin_authority_expiry_seconds` | Active authority nearing expiry without replacement workflow |
| `integin_evidence_operations_total{outcome}` | Digest/security failure or repeated RustFS operation failure |
| `integin_backup_last_success_timestamp` | Missing or stale verified backup/recovery drill evidence |
| `integin_http_request_duration_seconds` | Readiness degradation, sync/evidence latency regression, or error-rate threshold |

Logs must retain correlation ID, route, tenant-safe outcome code, device/authority references where authorized, and duration. They must not retain S3 credentials, private keys, plaintext evidence, or complete sensitive payloads.

## Release and recovery gate

Every authority-path release must produce a release record containing the following evidence before promotion.

1. Source revision and build-artifact checksum.
2. Migration list and tested rollback/forward strategy.
3. Go, Flutter, contract, RLS, signed-sync, evidence-digest, and advisory-boundary test results.
4. Backup artifact location/checksum and most recent isolated restore evidence.
5. Dependency/version inventory, including PostgreSQL and RustFS version/image digest.
6. Operator and security approval for the assessed risk class.
7. Rollback trigger, owner, and explicit statement of what is not automatically reversible.

## Next 90-day sequencing

| Time window | Deliverable | Decision gate |
|---|---|---|
| 0–30 days | OIDC evaluation, production enrollment specification, OpenAPI v1 baseline, evidence-manifest schema | No production field rollout until device trust and identity flows have accountable owners |
| 31–60 days | OIDC integration, approval/revocation workflow, metrics/traces/dashboard, release record template | Tenant/identity/RLS and observability acceptance tests pass |
| 61–90 days | Evidence export implementation, Rust verifier prototype, TypeScript operations API/UI slice | Independent export verification and operator recovery exercise pass |

## References

[1]: ./ARCHITECTURE_EVOLUTION_ROADMAP.md "Technology ownership, safety contracts, and staged evolution"

[2]: ./LIVE_INTEGRATION_RUNBOOK.md "Local integration, acceptance, and recovery procedure"

[3]: ./RLS_RUNTIME_ROLE.md "Runtime PostgreSQL role and tenant isolation setup"
