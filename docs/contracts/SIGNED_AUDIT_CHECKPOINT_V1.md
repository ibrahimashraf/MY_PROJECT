# INTEGIN Signed Audit Checkpoint v1 — Canonical Input Contract

> **Status:** Canonical-root baseline only. This artifact defines deterministic checkpoint input and SHA-256 root verification. It does not sign a root, retrieve a key, persist a checkpoint, expose an endpoint, or modify audit/event data.

## Purpose and boundary

INTEGIN needs portable tamper-evidence without creating a second workflow or audit authority. The v1 checkpoint contract creates a reproducible root over allowlisted audit metadata. It uses one tenant and one environment per checkpoint, contiguous external sequence bounds, a previous-root reference, and SHA-256 digests of record context rather than raw contexts or event payloads.

| Concern | v1 rule |
|---|---|
| Authority | Go/PostgreSQL remains authoritative for audit records and workflow. A checkpoint is integrity evidence, not a write authority. |
| Scope | Every entry must match the checkpoint tenant and environment. Mixed-scope checkpoints are rejected. |
| Ordering | Entries are sorted by sequence before hashing. Sequence bounds must be contiguous; gaps and duplicates are rejected. |
| Privacy | Raw audit context and event payload are excluded. `context_sha256` protects their reference without exporting contents. |
| Chaining | The first checkpoint references `GENESIS`; later checkpoints reference a prior 64-character lowercase SHA-256 root. |
| Integrity | `root_sha256` covers fixed-field canonical JSON excluding itself. Any metadata/context-digest change invalidates verification. |
| Signing | Deliberately deferred. A root is not a signature, attestation, or storage proof. |

## Canonicalization

`internal/auditcheckpoint` validates fields, tenant/environment scope, SHA-256 context digest format, unique record IDs, unique contiguous sequences, and previous-root format. It copies and sorts entries by sequence, encodes the fixed structure with UTC RFC3339 nanosecond timestamps, and hashes that byte representation with SHA-256. It neither queries a ledger nor persists a root.

> Verification failure is read-only evidence. It must never silently repair, delete, reorder, or mutate authoritative audit or event records.

## Artifacts

| Artifact | Purpose |
|---|---|
| `internal/auditcheckpoint/checkpoint.go` | Pure canonical payload, `Seal`, and `Verify` contract. |
| `internal/auditcheckpoint/checkpoint_test.go` | Deterministic ordering, tamper, scope, sequence, and context-digest tests. |
| `contracts/audit_checkpoint_v1.schema.json` | Portable JSON Schema 2020-12. |
| `contracts/audit_checkpoint_v1_test.go` | Required-field schema guard. |

## Deferred signing and persistence

The next checkpoint phase must introduce all of the following together: a dedicated audit-attestation key **reference** (not key material in source); approved secret-path and workload policy; explicit signature algorithm/key ID; PostgreSQL checkpoint metadata under RLS; immutable RustFS artifact; key rotation/retirement procedure; a read-only independent verifier; and recovery exercises. No current INTEGIN service depends on this contract.

## References

- [Production-Trust Baseline Roadmap](PRODUCTION_TRUST_BASELINE_ROADMAP.md)
- [Platform Governance Backlog](PLATFORM_GOVERNANCE_BACKLOG.md)
- [Evidence Export Manifest v1](EVIDENCE_EXPORT_MANIFEST_V1.md)
- [Production Secret Inventory and OpenBao Recovery Design](PRODUCTION_SECRET_INVENTORY_AND_OPENBAO_RECOVERY_DESIGN.md)
