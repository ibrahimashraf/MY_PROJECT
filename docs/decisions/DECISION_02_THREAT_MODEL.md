# Decision 02 — Controlled-Pilot Threat Model

**Status:** Draft for approval  
**Decision owner:** Security Operator  
**Required approvers:** Product/Safety Authority; Platform Operator; Engineering Change Owner  
**Scope:** Security and abuse-case boundaries for the controlled production pilot.

## Decision

INTEGIN will use a **fail-closed, server-authoritative trust model**. A valid user login, device key, OIDC token, signed client message, tenant identifier, or AI advisory result is never independently sufficient to authorize a safety-relevant mutation. The Go authority must validate the entire applicable context: issuer/subject, active local membership, tenant and organization, role/capability, device state, authority scope/epoch/expiry, workflow state, and database RLS context.

The threat model prioritizes confidentiality of evidence and keys, integrity of device-signed work and approvals, tenant isolation, recoverability, and the ability to investigate an operator or service action. Availability is important, but it must not be achieved by failing open or accepting unverified field mutations.

## Protected assets

| Asset | Required property | Authority |
|---|---|---|
| Tenant/organization state | Isolation and correct authorization | Go + PostgreSQL RLS |
| Device private Ed25519 key | Non-exportability and device-local secrecy | Flutter/platform secure storage |
| Device public-key/trust state | Accurate lifecycle, no unauthorized registration/reuse | Go + PostgreSQL |
| Offline authority package | Bound scope, epoch, expiry, and server signature | Go authority |
| Evidence object and digest metadata | Confidentiality, immutability/tamper detection, traceability | Go + PostgreSQL + RustFS |
| OIDC identity/session | Authentic issuer/subject, session/MFA assurance | Keycloak + Go token validation |
| Signing and encryption keys | Separation, rotation, revocation, verifiability | Decision 03 + OpenBao/approved key boundary |
| Audit and export artifacts | Complete, attributable, independently verifiable | Go + PostgreSQL + RustFS + verifier |
| Release artifact and configuration | Reproducibility, provenance, rollback capability | Decision 05 |

## Threat scenarios and controls

| Threat scenario | Primary controls | Required test/evidence |
|---|---|---|
| Attacker submits cross-tenant IDs or uses an altered token claim | Server-side membership resolution, transaction-scoped RLS, object-key derivation, audience/issuer validation | Cross-tenant API, repository, object, enrollment, export, and admin tests fail closed. |
| Field device is lost, stolen, or compromised | Device lifecycle, authority epoch, bounded authority duration, remote revoke, replacement enrollment | Post-revocation signed sync is rejected; replacement preserves audit linkage. |
| User enrolls an attacker-controlled key | OIDC binding, short single-use challenge, Ed25519 proof, approver authorization, separation of duties | Subject/organization/key swap, replay, self-approval, and expired proof tests fail. |
| Tenant administrator abuses authority | Local capability checks, separation of duties for high-risk/self actions, explicit reason, immutable audit | Insufficient-role and self-approval attempts fail; approval record identifies actor/reason/policy. |
| OIDC/Keycloak compromise or outage | Separate identity service, short sessions, issuer/JWKS checks, local authorization, bounded existing offline authority | Invalid/stale token fails; outage prevents new login/enrollment/authority issue and never grants access. |
| OpenBao or runtime credential compromise | Segregated paths/identities, lease/revocation, audit, rotation, no secrets in source/logs | One workload cannot read another path; credential rotation and containment drill pass. |
| Evidence object is modified, deleted, or substituted | Server-derived object identity, digest/receipt linkage, export manifest, restore verification, signed checkpoints | Altered object/digest/checkpoint is detected by verifier. |
| Malicious/defective release | Pinned/digested artifacts, protected release, SBOM, change approval, rollback | Artifact can be traced to source/test/migration/approval; rollback rehearsal passes. |
| Telemetry leaks evidence, keys, or personal data | Allowlisted structured fields, redaction, scanner tests, access controls | Synthetic secret/token/evidence fixture is absent from logs, traces, metrics, and support bundles. |
| AI influences a primary decision | `blocking=false`, advisory-only API contract, UI boundary, audit of advisory provenance | A failed or malicious advisory response cannot mutate workflow state or prevent an authorized server action. |
| Physical/admin host compromise | Segmented network, named MFA access, patching, backup encryption, incident/runbook procedures | Access review and recovery/containment exercise complete. |

## Explicit non-goals

The controlled pilot does not claim protection against every nation-state or physical compromise scenario. It does claim that high-impact common failures and abuse attempts have explicit controls, detection/recovery paths, and tests. A feature without a defined authority, threat treatment, and recovery path does not enter the pilot.

## Security decisions

1. Client input is descriptive until server validation proves otherwise.
2. OIDC authenticates; it does not replace local tenant/workflow authorization.
3. The local provisioning bridge remains development-only and disabled everywhere shared.
4. AI remains advisory, non-blocking, and unable to mutate primary state.
5. Evidence verification must survive restoration and be possible without a production write credential.
6. Security events are retained and investigated without copying secrets or evidence contents into telemetry.

## Review triggers

Revisit this model for a new external integration, new evidence class, device-attestation rollout, customer data residency requirement, administrator incident, production security event, or change to offline authority scope/duration.

## Related records

- [Decision 03 — Cryptographic Key Lifecycle](./DECISION_03_CRYPTOGRAPHIC_KEY_LIFECYCLE.md)
- [Decision 06 — Operator Runbooks](./DECISION_06_OPERATOR_RUNBOOKS.md)
- [Production Device Enrollment Specification](./PRODUCTION_DEVICE_ENROLLMENT_SPEC.md)
