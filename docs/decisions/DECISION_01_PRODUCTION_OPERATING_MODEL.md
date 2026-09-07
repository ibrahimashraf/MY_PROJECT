# Decision 01 — Production Operating Model

**Status:** Draft for approval  
**Decision owner:** Platform Operator  
**Required approvers:** Security Operator; Product/Safety Authority  
**Scope:** Controlled production pilot hosting and network operating model.

## Decision

The current Windows Docker runtime remains an **integration and recovery-drill environment**. A controlled production pilot will run on a **dedicated, organization-controlled Linux LTS server or virtual machine**, separate from developer workstations and separate from local test data. The exact hosting provider or physical location is intentionally not chosen in this record; the controls below apply to an on-premises server, a private virtual machine, or an approved infrastructure provider.

The pilot will use a deliberately simple modular-monolith topology: Go authority service, PostgreSQL, RustFS, Keycloak, OpenBao, reverse proxy/TLS edge, and later the observability stack. Docker Compose may be used for the single-host pilot if every image is pinned by digest, persistent data is on named/encrypted volumes or managed storage, and configuration is stored outside source control. Horizontal scaling, Kubernetes, a service mesh, and multi-region active-active operation are not pilot requirements.

> **A pilot may have one documented host failure domain. It may not have one undocumented operator, one unbacked-up credential, or one untested recovery path.**

## Required topology and network boundaries

| Zone | Components | Access rule |
|---|---|---|
| Public edge | DNS, HTTPS reverse proxy, user-facing Go/OIDC routes | Only TCP 443 is exposed externally. TLS certificates, headers, rate limits, request size limits, and access logs are controlled at this boundary. |
| Application network | Go authority, Keycloak, OpenBao, OTel Collector | Not directly internet reachable. Service-to-service access is allowlisted by destination, port, and service identity/credential. |
| Data network | PostgreSQL, RustFS | Not directly internet reachable. Access is restricted to required application, migration, backup, verifier, and operator paths. |
| Operations network | Backup destination, Grafana, controlled administration access | Restricted to named operator identities with MFA and auditable access. No routine administrative action uses a shared account. |

PostgreSQL port **5432** remains standard inside the controlled network. RustFS administrative and S3 ports must not be exposed to the public internet. The production Go service must reject a configuration in which `INTEGIN_LOCAL_PROVISIONING_ENABLED=true`; the local bridge and production enrollment are mutually exclusive.

## Environment separation

| Environment | Purpose | Permitted data | Prohibited behavior |
|---|---|---|---|
| Development | Local coding and unit/integration tests | Synthetic or explicitly approved non-production data | Customer data, production credentials, production signing keys |
| Pilot non-production | OIDC/OpenBao/enrollment/release rehearsals | Synthetic and controlled pilot fixtures | Public access, unapproved customer evidence, production authority keys |
| Controlled production pilot | Limited approved organizations and field users | Approved pilot data under the retention and export policy | Experimental schema changes, local provisioning bridge, unreviewed images |

Each environment has independent databases, object buckets/prefixes, Keycloak realm, OpenBao paths, signing keys, TLS certificates, client IDs, and backup destinations. A production database, object archive, or identity export is never restored into development without a documented approved sanitization process.

## Host and service operation controls

The pilot host must have supported operating-system updates, encrypted storage where the platform permits it, a documented patch cadence, least-privilege service accounts, time synchronization, host health monitoring, firewall rules, and a recovery console method that is independent of the normal web path. Remote operator access requires named accounts, MFA at the access layer, and audit logging. Break-glass access is time-bounded, two-person approved where practical, and recorded.

Every image must be pinned by immutable digest. Every persistent volume must have an owner, path/volume identifier, backup method, recovery procedure, and retention treatment. Container `latest` tags are prohibited in pilot and production deployments. The RustFS `1.0.0-rc.1` approval remains limited to development/integration testing; a controlled production pilot requires a separately approved stable object-storage release or a documented risk acceptance.

## Pilot availability position

The controlled pilot is a **single-site, recoverable service**, not an active-active high-availability platform. A host failure can cause a temporary interruption while the documented recovery procedure restores the service. This is acceptable only if the RTO/RPO in Decision 04 are approved and field authority durations/procedures permit a bounded offline operating period. No team may claim continuous availability until its measured architecture and drill results support that claim.

## Acceptance criteria

| Test | Required outcome |
|---|---|
| External port scan | Only approved edge routes are visible; PostgreSQL, RustFS, Keycloak administration, and OpenBao are not publicly reachable. |
| Configuration validation | Production process refuses local-provisioning enablement and rejects missing/duplicate environment identifiers. |
| Credential boundary test | Go, Keycloak, RustFS, PostgreSQL, and OpenBao runtime identities can access only their approved resources. |
| Host loss rehearsal | A replacement host restores the approved pilot service within the approved RTO using documented procedures. |
| Operator access review | Every production administrative account is named, MFA protected, and recoverable without shared credentials. |

## Approval and revisiting

This decision becomes effective only after the selected hosting location, named operators, backup destination, external DNS/TLS ownership, and pilot organization limits are attached as a non-secret appendix. Revisit it before broad rollout, a second site, an external integration, or any decision to process data subject to stronger residency/availability requirements.

## Related records

- [Decision 02 — Threat Model](./DECISION_02_THREAT_MODEL.md)
- [Decision 04 — Recovery Objectives](./DECISION_04_RECOVERY_OBJECTIVES.md)
- [Production-Trust Baseline Roadmap](./PRODUCTION_TRUST_BASELINE_ROADMAP.md)
