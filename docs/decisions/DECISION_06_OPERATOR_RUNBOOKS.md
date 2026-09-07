# Decision 06 — Operator Runbooks and Incident Control

**Status:** Draft for approval  
**Decision owner:** Platform Operator  
**Required approvers:** Security Operator; Product/Safety Authority  
**Scope:** Minimum operational response procedures for a controlled INTEGIN production pilot.

## Decision

INTEGIN will not depend on informal knowledge for routine operations or incidents. Each safety, security, identity, evidence, backup, or release event has a concise controlled runbook with a named decision authority, immediate containment steps, verification, communications, recovery criteria, and post-incident record. Runbooks are versioned alongside the platform documentation, but no runbook contains secrets or private key material.

## Roles and escalation

| Role | Primary responsibility | May not do alone |
|---|---|---|
| Platform Operator | Runtime health, deployment, backup, restore, observability, incident coordination | Change signing-key policy, approve own exceptional authority change, alter tenant workflow data directly. |
| Security Operator | Identity, credentials, containment, key/secret response, access review | Alter business workflow/evidence records outside Go application services. |
| Product/Safety Authority | Safety/workflow/evidence/retention risk decision and pilot reopening approval | Directly administer infrastructure without Platform/Security control. |
| Engineering Change Owner | Code/configuration change, tests, contract compatibility, release evidence | Self-approve an authority/evidence-impacting change. |
| Tenant Administrator | Tenant membership and eligible device approval within assigned organization | Bypass RLS, replace server authority, inspect other tenant state. |

For a severity-one event, the Platform Operator coordinates containment, the Security Operator owns credential/identity decisions, and the Product/Safety Authority decides whether field operations or pilot traffic must be paused. A single urgent containment action is permitted when delay would worsen exposure, but it must be recorded and reviewed immediately afterward.

## Required runbooks

| Runbook | First containment action | Reopen/close condition |
|---|---|---|
| Service outage / degraded sync | Establish incident, preserve logs/release reference, stop nonessential changes, assess offline-authority impact | Readiness, tenant RLS, sync/evidence smoke test, and actual RTO record verified. |
| PostgreSQL failure or suspected corruption | Freeze writes if integrity is uncertain; preserve evidence; start isolated recovery path | Restored state, RLS, role, devices/authorities/receipts, and audit checks pass. |
| RustFS evidence mismatch or loss | Stop accepting affected evidence class if digest/receipt integrity is uncertain; retain diagnostic metadata | Fresh-volume restore and manifest/digest verification pass; exception decision recorded. |
| Lost/compromised field device | Lock/revoke device, increment authority epoch, capture reason/time, assess recent receipts | New device re-enrolled; old authority rejected; affected work reviewed according to policy. |
| Suspected credential/key compromise | Contain affected identity/path, rotate/revoke as required, preserve evidence, assess blast radius | New key/credential validated, exposed sessions/authority contained, historic verification preserved. |
| Keycloak/OIDC outage or token anomaly | Deny new login/enrollment/authority issuance; do not fail open; assess bounded offline operation | Issuer/JWKS/session checks healthy; local authorization remains correct; incident evidence recorded. |
| OpenBao issue | Block new secret issuance/rotation where unsafe; use approved recovery only | Secret access policy, audit, and service credentials restored without exposing values. |
| Failed backup or restore | Mark backup objective at risk, investigate immediately, create a new verified backup | Isolated restore succeeds with target checksums/manifests; missed RPO is reported. |
| Unsafe release or migration | Stop rollout, use approved rollback/forward plan, preserve artifact/configuration evidence | Contract, RLS, authority, and accepted data verified under an approved release state. |
| Suspected tenant isolation breach | Contain access path, preserve audit/telemetry, notify Security Operator and Product/Safety Authority | Scope assessed, defect corrected/tested, affected organizations handled through approved communication and legal/privacy process. |
| Advisory-service malfunction | Disable or isolate advisory feature if needed; preserve no primary workflow change | `blocking=false` and no-primary-mutation tests pass; advisory history reviewed as appropriate. |

## Standard runbook structure

Every detailed runbook will contain the following sections: trigger and severity; immediate safety statement; authority and contact roles; prerequisites/access; containment; evidence to preserve; decision points; exact commands or controlled console actions; verification; communication template; recovery/reopen criteria; post-incident review; and links to relevant backups, release records, or key registries. Commands must use placeholders and secret references only—never live credentials.

## Incident severity

| Severity | Definition | Target response posture |
|---|---|---|
| S1 | Confirmed or likely tenant breach, signing/key compromise, evidence-integrity uncertainty, unrecoverable authority defect, or broad service unavailability | Immediate containment, named incident roles, Product/Safety Authority decision on pause/reopen, formal review. |
| S2 | Material degradation, failed backup, identity outage, repeated sync/evidence failure, or a potentially contained security event | Same-day investigation and recovery; escalate to S1 if integrity/isolation becomes uncertain. |
| S3 | Limited functional issue without evidence/authority/isolation impact | Planned correction with release record and trend monitoring. |

## Operational acceptance criteria

| Test | Required outcome |
|---|---|
| Tabletop exercise | Roles can coordinate a lost device, failed backup, and identity outage using the runbooks without improvising security authority. |
| Live-safe drill | A non-production revoke/recovery/restore scenario completes and records the expected audit, notification, and reopening evidence. |
| Access review | Named operator roles and break-glass path are valid, MFA protected, and do not use shared credentials. |
| Communication exercise | An incident statement distinguishes confirmed facts, impact, action, owner, next update, and data that must not be disclosed. |

## Related records

- [Decision 01 — Production Operating Model](./DECISION_01_PRODUCTION_OPERATING_MODEL.md)
- [Decision 02 — Threat Model](./DECISION_02_THREAT_MODEL.md)
- [Decision 04 — Recovery Objectives](./DECISION_04_RECOVERY_OBJECTIVES.md)
- [Decision 05 — Supply-Chain and Release Integrity Policy](./DECISION_05_SUPPLY_CHAIN_RELEASE_POLICY.md)
