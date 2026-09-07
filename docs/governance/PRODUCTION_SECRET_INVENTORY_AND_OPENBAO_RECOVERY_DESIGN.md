# INTEGIN Production Secret Inventory and OpenBao Recovery Design

> **Status:** Design baseline. The inventory contains only names, classes, owners, references, and control rules. It contains **no secret values**. The isolated OpenBao pilot remains initialized, sealed, loopback-only, and unwired from INTEGIN.

## Purpose

This package turns the completed OpenBao rehearsal into a controlled production-design baseline without making OpenBao an application dependency. It names every planned secret class, identifies the responsible owner, assigns a future least-privilege OpenBao path, and defines rotation, revocation, recovery, audit, and availability expectations.

| Non-negotiable rule | Design consequence |
|---|---|
| Go/PostgreSQL remains the sole workflow and authorization authority. | OpenBao stores operational secret material only; it never stores workflow truth, evidence, tenant authorization, device trust, or offline authority. |
| Secret values must not enter source or operational evidence. | Inventory and verifier use references only. Values, tokens, unseal shares, private keys, and plaintext evidence are never logged or displayed. |
| Workloads are separated by responsibility. | Future runtime, Keycloak, advisory, notification, and audit-attestation classes use distinct paths and policies. |
| Recovery requires accountable separation of duties. | No individual, CI process, application process, or local recovery file is a sufficient production recovery mechanism. |
| Safety workflow remains server-authoritative. | A future secret refresh failure does not silently fall back to a shared environment file or mutate/reverse accepted workflow state. |

## Inventory model

The machine-readable inventory is `integin-pilot-source/contracts/production_secret_inventory_v1.json`. It is intentionally a policy map, not a secret store.

| Class | Future consumer | Owner | Required production path family |
|---|---|---|---|
| PostgreSQL runtime credential | INTEGIN Go authority | Platform Operator | `kv/production/integin/runtime/...` |
| RustFS/S3 credential | INTEGIN evidence adapter | Platform Operator | `kv/production/integin/runtime/...` |
| Sync verification material | INTEGIN sync authority | Security Operator | `kv/production/integin/runtime/...` |
| Keycloak database/admin bootstrap | Keycloak only | Security Operator | `kv/production/keycloak/...` |
| Advisory AI provider credential | Python advisory service only | Advisory Reviewer + Platform Operator | `kv/production/integin/advisory/...` |
| SMTP credential | Notification adapter only | Platform Operator | `kv/production/integin/notifications/...` |
| Audit-attestation key reference | Future checkpoint component only | Security Operator | `kv/production/integin/audit/...` |

Every class requires an accountable owner, rotation trigger, revocation action, recovery reference, audit requirement, and availability rule. A path may identify a class; it may never contain its value in this document, a log, source file, test fixture, chat transcript, or exported manifest.

## OpenBao recovery design

The pilot rehearsal proved a bounded local baseline: dedicated container/network/volumes, loopback binding, Shamir `3` shares with threshold `2`, audit file, least-privilege fixture test, restart persistence, and sealed outage isolation. It is intentionally not production deployment authorization.

### Production prerequisites

1. Shared or production use requires TLS before any client connection.
2. Named recovery holders and approved secure share handoff must replace one-machine custody.
3. Policy-per-workload paths must be version-controlled and reviewed; no workload can read an unrelated path.
4. Backup cadence, isolated restoration environment, recovery evidence, audit retention, and operator/reviewer roles must be approved.
5. A recovery exercise must validate policies and audit configuration without displaying values.
6. Credential rotation must have a workload-specific restart/reload path and an explicit rollback/incident procedure.

### Sealed and outage behavior

The current pilot stays sealed and is not connected to INTEGIN. Future application integration must fail closed for a **new** secret retrieval when OpenBao is unavailable. It must never silently read a shared/production `.env` fallback. A bounded, explicitly approved in-memory cache policy may govern an already-valid credential; nonessential refresh failure must not by itself reject an accepted field transaction or modify authoritative workflow state.

## Reference-only verification

`operations/pilot/verify-secret-boundary.ps1` validates only the inventory metadata, protected-directory existence, OpenBao container lifecycle and network **names**, and loopback sealed-status metadata. It does not read private files, inspect container environment variables, unseal OpenBao, create a secret, or connect OpenBao to INTEGIN.

## Explicitly deferred implementation

This package does not move any existing local credential, create an OpenBao policy, unseal OpenBao, introduce TLS, use dynamic credentials, or wire Keycloak/Go/Flutter/RustFS/AI to OpenBao. Those are separate controlled implementation phases after production identity/enrollment policy, recovery-share custody, TLS, workload identity, and release/recovery gates are approved.

## References

- [OpenBao Pilot Rehearsal Plan](OPENBAO_PILOT_REHEARSAL_PLAN.md)
- [Production-Trust Baseline Roadmap](PRODUCTION_TRUST_BASELINE_ROADMAP.md)
- [Platform Governance Backlog](PLATFORM_GOVERNANCE_BACKLOG.md)
- [Evidence Export Manifest v1](EVIDENCE_EXPORT_MANIFEST_V1.md)
