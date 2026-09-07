# Decision 05 — Supply-Chain and Release Integrity Policy

**Status:** Draft for approval  
**Decision owner:** Engineering Change Owner  
**Required approvers:** Security Operator; Platform Operator; Product/Safety Authority for safety-relevant changes  
**Scope:** Source, dependency, build, test, release, rollback, and evidence controls for the controlled pilot.

## Decision

Every pilot release must be identifiable, reproducible to a defined degree, approved according to risk, tested against the authoritative contracts, and recoverable by rollback or documented forward remediation. No production change is released from an uncommitted workstation state, floating dependency/image tag, or unreviewed private credential.

The policy applies to Go authority code, Flutter field application, Python advisory service, Keycloak/OpenBao/observability configuration, PostgreSQL migrations, RustFS configuration, OpenAPI schemas, evidence export/checkpoint verifier, and deployment manifests.

## Required release evidence

| Evidence item | Requirement |
|---|---|
| Source revision | Immutable commit identifier and reviewed change description. |
| Build inputs | Language/toolchain version, dependency lock files, pinned container image digests, configuration template version, and migration set. |
| SBOM and dependency review | Generate a software bill of materials for released application/container artifacts; review known critical vulnerabilities and license changes before approval. |
| Test record | Unit, integration, RLS, contract/test-vector, signature, backup/restore-relevant, and applicable Flutter acceptance results. |
| Security/configuration review | Confirmation that no secret, private key, unsafe development flag, wildcard callback, public data port, or local provisioning bridge is included. |
| Approval | Named approvers based on change class; Safety Authority approval for authority scope, workflow, evidence, retention, or certificate impacts. |
| Deployment record | Environment, image/artifact digest, schema version, time, operator/job identity, and outcome. |
| Rollback/forward plan | Last known good artifact/schema compatibility, trigger, decision owner, and data-migration constraints. |

## Change classes

| Class | Examples | Minimum approval and testing |
|---|---|---|
| Routine non-authoritative | Copy, dashboard, UI presentation that cannot alter workflow/evidence decisions | Engineering Change Owner; automated build/test; documented release. |
| Functional | Go/Flutter feature, API addition, advisory UI/service behavior | Engineering Change Owner + Platform Operator; relevant integration/contract tests; rollback plan. |
| Security/identity | Keycloak, OpenBao, token validation, secrets, key lifecycle, network exposure | Security Operator + Platform Operator + Engineering Change Owner; security tests and recovery impact review. |
| Authority/evidence/safety | Device enrollment, authority scope, workflow acceptance, evidence/digest/export/retention, RLS | Product/Safety Authority + Security Operator + Engineering Change Owner; adversarial/compatibility/recovery tests and explicit release hold point. |
| Emergency containment | Credential compromise, active vulnerability, exposed service, data integrity incident | Authorized incident lead may act immediately to contain; retrospective review and release record within one business day. |

## Build and deployment rules

1. Dependencies are locked and reviewed. Container images are referenced by immutable digest, never `latest`.
2. CI/build environments receive only the minimum secret scope needed; release secrets are not available to ordinary development jobs.
3. Database migrations must be forward-compatible or have a documented safe recovery plan. Destructive migrations require a backup/restore rehearsal and Product/Safety Authority approval where authoritative data is affected.
4. Every protocol-affecting change updates OpenAPI/version semantics and canonical signed test vectors before client rollout.
5. Flutter releases must declare the compatible server/protocol range and preserve an upgrade/recovery path for offline queues and authority packages.
6. The AI advisory service retains `blocking=false`; a release must not introduce a route that allows AI output to mutate or gate primary workflow state.
7. A pilot deployment starts with a non-production rehearsal and a pre/post deployment health/contract verification record.

## Rollback position

Application binaries/configuration can roll back only when database and protocol compatibility are explicitly confirmed. When a migration makes rollback unsafe, the approved path is a forward remediation or restore from a pre-change backup—not an undocumented manual database edit. A release must state this before deployment.

## Acceptance criteria

| Test | Required outcome |
|---|---|
| Provenance exercise | Independent reviewer maps a running artifact to commit, SBOM, image digest, configuration version, migration list, tests, approvals, and operator. |
| Unsafe-flag scan | Released production configuration contains no local provisioning flag, no wildcard origin/callback, and no source-controlled secret. |
| Rollback rehearsal | A compatible functional release returns to the prior known-good artifact without losing verified authoritative state. |
| Migration rehearsal | A representative non-production dataset migrates, validates, restores/forwards as documented, and preserves RLS/audit properties. |
| Dependency response drill | A newly identified critical dependency issue can be triaged, contained, patched/released, and documented through the defined process. |

## Related records

- [Decision 04 — Recovery Objectives](./DECISION_04_RECOVERY_OBJECTIVES.md)
- [Decision 06 — Operator Runbooks](./DECISION_06_OPERATOR_RUNBOOKS.md)
- [Production-Readiness Gap Review](./PRODUCTION_READINESS_GAP_REVIEW.md)
