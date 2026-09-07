# INTEGIN Retention, Legal-Hold, Redaction, and Export-Access Policy v1

> **Status:** Design baseline, not enforced. This policy is a tenant-safe governance contract. It does not create retention timers, delete records, alter legal-hold state, redact evidence, grant export access, read/write RustFS objects, migrate PostgreSQL, or change a running INTEGIN service.

## Policy objective

INTEGIN must preserve authoritative evidence and audit lineage while allowing accountable future rules for retention, legal hold, redaction, and approved portable export. The policy protects the existing authority model: Go/PostgreSQL decides local authorization and authoritative state; RustFS stores encrypted evidence; future independent verification stays read-only.

| Policy concern | v1 binding rule |
|---|---|
| Tenant isolation | A policy instance applies to exactly one tenant; organization context is required when the record is organization-bound. |
| Activation | Enforcement needs approval from the Product and Safety Authority and Platform Operator. This artifact is not an activation. |
| Retention | A future server-authoritative disposition workflow may act only after approved profile, hold, approval, audit, recovery, and exception preconditions are met. No duration is implied here. |
| Legal hold | An active hold blocks disposition and any transformation that would impair preservation of held source evidence. Hold release is a separate audited accountable act. |
| Redaction | The source evidence remains immutable. A redacted item is a separate derivative with lineage, digest, policy, approval, and audit references. |
| Export access | An export requires local Go/PostgreSQL authorization, tenant scope, an approval reference, and an evidence export manifest with policy and hold snapshot. Keycloak claims alone are insufficient. |
| Recovery | Recovery evidence preserves policy version, profile/hold/redaction/approval references, checksums, exceptions, operator, and reviewer references. |

## Authority and precedence

> The most restrictive applicable legal, safety, contractual, or tenant requirement governs until an accountable authority records a scoped decision. Automatic conflict resolution is prohibited.

This policy separates a **policy definition** from a future **policy enforcement workflow**. A client, advisory service, object-store lifecycle configuration, or background job must not delete or rewrite authoritative evidence independently. A policy conflict, legal hold, export request, redaction request, and disposition request need explicit accountable authority, reason/reference, scope, time window, and audit correlation.

## Future enforcement requirements

Before implementation, the accountable owners must approve retention profiles and durations; legal-hold authority and custody; redaction tools and derivative lineage; export purpose/scope/expiry; exception workflow; and recovery/audit evidence. The eventual implementation must provide tenant RLS, server-side capability checks, immutable audit records, idempotency, recovery tests, and independent export verification.

| Future action | Minimum non-negotiable control |
|---|---|
| Retention disposition | Local server authorization, approved profile, no active hold, auditable reason, reversible/controlled recovery handling, and no client-side delete path. |
| Legal-hold activation/release | Accountable authority, scoped subject, reason, timestamps, immutable audit record, and no automated bypass. |
| Redaction | Immutable source, separately digested derivative, lineage, approved tool/version, and safety/recovery review where relevant. |
| Export | Local membership/capability authorization, bounded approved purpose/scope/expiry, manifest policy snapshot, and read-only verifier compatibility. |
| Recovery drill | Source/restored checksums, hold/profile/redaction/approval snapshots, exceptions, operator, reviewer, and no write-on-verification-failure. |

## Contract and test artifacts

The machine-readable contract is `integin-pilot-source/contracts/retention_legal_hold_redaction_export_access_policy_v1.json`. Its Go guard test confirms tenant scope, non-enforcement status, legal-hold precedence, source immutability, export-manifest requirements, audit/recovery metadata, conflict-resolution prohibition, and declared implementation deferrals.

## Explicit deferrals

This version deliberately defers all stateful operations. It does not define actual retention durations or classification schedules; persist legal holds; issue export approvals; build redaction transformations; create export/disposition APIs; configure RustFS lifecycle; change PostgreSQL; or make acceptance/pilot runtime changes. These are separate authoritative-path implementation phases after the documented approval decisions.

## References

- [Platform Governance Backlog](PLATFORM_GOVERNANCE_BACKLOG.md)
- [Production-Trust Baseline Roadmap](PRODUCTION_TRUST_BASELINE_ROADMAP.md)
- [Evidence Export Manifest v1](EVIDENCE_EXPORT_MANIFEST_V1.md)
- [Production Secret Inventory and OpenBao Recovery Design](PRODUCTION_SECRET_INVENTORY_AND_OPENBAO_RECOVERY_DESIGN.md)
