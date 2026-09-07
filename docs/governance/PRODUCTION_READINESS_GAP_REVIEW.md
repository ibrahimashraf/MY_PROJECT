# INTEGIN Production-Readiness Gap Review

**Status:** Planning assessment — no live runtime change authorized  
**Date:** 2026-08-15  
**Basis:** The completed live Flutter/Go acceptance run, PostgreSQL/RustFS recovery drills, and the current production-trust design set.

## 1. Executive assessment

INTEGIN has crossed an important boundary: the core technical proposition has been demonstrated with a real Flutter client, server-authoritative Ed25519 device trust, durable PostgreSQL state, RustFS evidence objects, and isolated recovery drills. The remaining work is not a rewrite. It is the disciplined conversion of a functioning integration platform into an operable, defensible production service.

The most important gaps are **operational ownership and recovery objectives**, **cryptographic/signing-key governance**, **software supply-chain control**, **formal threat/abuse-case analysis**, and a **production operating environment decision**. These are more valuable than adding another application framework, workflow engine, database, authorization graph, queue, service mesh, or AI capability.

> **The next improvement is not more architecture. It is proving that the existing authority architecture can be securely released, operated, recovered, investigated, and improved by named people under real failure conditions.**

## 2. Current strengths

| Proven or designed capability | Why it matters |
|---|---|
| Server-authoritative sync and evidence acceptance from Flutter | The field client does not define final workflow state. |
| Ed25519 device keys, authority packages, receipt behavior, and duplicate handling | Offline work has a cryptographic and replay-aware trust model. |
| PostgreSQL RLS and runtime-role validation | Tenant isolation has a database backstop, not only application checks. |
| RustFS object-contract validation and fresh-volume restore drill | Evidence storage behavior and recovery have been exercised rather than assumed. |
| Binary-safe PostgreSQL backup/isolated restore drill | Durable business state can be recovered without corrupting binary backup data. |
| Controlled local provisioning clearly marked non-production | A development convenience is not being misrepresented as a production control. |
| Production enrollment and identity designs | The correct next security boundary has been specified before implementation. |
| Production-trust roadmap | New components have acceptance gates and no component is allowed to replace Go/PostgreSQL authority. |

## 3. Priority classification

### 3.1 Must complete before a controlled production pilot

| Gap | Why it is a release blocker | Minimum satisfactory outcome | Accountable owner |
|---|---|---|---|
| **Production hosting and network decision** | The current Windows/Docker runtime is an integration environment, not a production operations model. | Document hosting ownership, supported OS/runtime, HTTPS edge, DNS, firewall/segmentation, patch window, admin access, monitoring network path, and failure-domain assumptions. | Platform Operator |
| **Threat model and abuse-case review** | Enrollment, offline authority, evidence, export, tenant access, and operator actions are high-value attack paths. | Versioned threat model covering external attacker, lost/compromised device, malicious tenant user/admin, compromised service credential, supply-chain compromise, and operator error; each high risk has an owner/control/test. | Security Operator |
| **Signing-key and cryptographic lifecycle policy** | Device authority, audit checkpoints, export verification, and server identity cannot rely on informal secret handling. | Separate key purposes; versioned key IDs; generation/storage/rotation/revocation/compromise procedures; dual-control recovery; retention of verification keys; no private key in source, mobile app, logs, or backup plaintext. | Security Operator |
| **Recovery objectives and exercises** | A successful restore drill does not yet define acceptable business interruption or data-loss limits. | Approved RTO/RPO per PostgreSQL, RustFS, Keycloak, OpenBao, and audit/export artifacts; quarterly isolated restore plan; named decision maker for declaring recovery. | Platform Operator + Product/Safety Authority |
| **Supply-chain/release integrity** | Industrial assurance software must be able to explain exactly what was released and from which dependencies. | Pinned container images by digest, dependency-lock enforcement, SBOM generation, vulnerability review workflow, signed/tagged release artifact, protected release process, and rollback evidence. | Engineering Change Owner |
| **OpenAPI v1, canonical signed vectors, and compatibility policy** | Go/Flutter protocols can otherwise drift while the system still appears locally functional. | Versioned OpenAPI and error semantics; canonical signature/authority/enrollment vectors verified in Go and Flutter; compatibility/deprecation rule. | Engineering Change Owner |
| **Production enrollment approval** | The local bridge must never leak into shared environments. | Approved enrollment design; implementation tests for proof, approval, RLS, separation of duties, revocation, recovery, and outage fail-closed behavior. | Security Operator + Product/Safety Authority |
| **Operator runbooks and incident classification** | A secure design cannot be operated safely through memory or informal chat. | Runbooks for credential rotation, compromised device, enrollment rejection, authority revocation, evidence-object mismatch, failed backup, restore, identity outage, and release rollback. | Platform Operator |

### 3.2 Should complete before broad multi-organization rollout

| Gap | Value | Minimum satisfactory outcome | Accountable owner |
|---|---|---|---|
| **Tenant-safe observability** | Enables early detection and evidence-based response without leaking sensitive field data. | OTel/Prometheus/Loki/Grafana baseline, correlation IDs, redaction rules, dashboard ownership, alert routing, retention rules, and synthetic failure alerts. | Platform Operator |
| **Signed audit checkpoints and independent export verification** | Strengthens customer/regulator confidence and recovery assurance. | Versioned checkpoint algorithm/test vectors, signed checkpoint artifacts, manifest v1, and a read-only verifier that detects changed data/objects. | Engineering Change Owner |
| **Performance, capacity, and degraded-network testing** | Field work is often performed in weak connectivity and burst-sync conditions. | Measured throughput/latency targets, offline queue/backlog test, evidence-upload retry test, sync conflict test, PostgreSQL/RustFS capacity baseline, and stated pilot limits. | Engineering Change Owner |
| **Data governance and retention policy** | Evidence, certificates, operator records, and identity profile data have different retention and legal-hold requirements. | Data inventory, classification, retention/deletion/legal-hold rules, export boundary, privacy/access request procedure, and tested selective retrieval/deletion where permitted. | Product/Safety Authority + Security Operator |
| **Accessibility and bilingual localization baseline** | A field system must be usable under real industrial conditions and the product has a bilingual positioning goal. | Keyboard/screen-reader baseline for operations UI, high-contrast and field readability review, English/Arabic localization model, RTL test plan, date/number/time-zone rules, and terminology glossary. | Product/Safety Authority + Engineering Change Owner |
| **User onboarding, competency, and support model** | Human authority must be explicit, not assumed from a login. | Role onboarding, evidence/authority training, competence assignment/review process, support escalation, and a paper/fallback procedure with controlled reconciliation. | Product/Safety Authority |
| **Migration and import assurance** | Customers will likely bring equipment, certificates, and inspection history from legacy sources. | Staged import contract, source-to-target mapping, dry-run/reconciliation reports, quarantine path, approval of exceptional rows, and immutable source manifest. | Engineering Change Owner |

### 3.3 Deliberately defer until a measured trigger exists

| Deferred item | Reconsider only when | Reason to defer now |
|---|---|---|
| Kubernetes, service mesh, multi-region active-active runtime | Measured availability, scale, or deployment-frequency needs exceed a managed VM/container model. | More control planes and failure modes would distract from first-production operating discipline. |
| Kafka, Redis, event streaming platform | Measured async throughput, replay, integration, or fan-out requirements cannot be met with the modular monolith/outbox design. | Adds a new durability and recovery authority. |
| External authorization graph or policy engine | The tested Go authorization engine cannot express a documented relationship policy need. | Duplicates tenant/workflow authority and RLS. |
| Blockchain or separate immutable database | Signed audit checkpoints and read-only export verification fail a specific regulator/customer requirement. | Adds a second record/recovery story without solving current gaps. |
| step-ca / workload mTLS | Multiple independently deployed services create demonstrated certificate-management need. | Does not replace Ed25519 field-device signing and is not a current bottleneck. |
| SIEM, SOAR, full DLP suite | Structured telemetry, incident volume, or formal regulatory scope requires it. | First establish high-quality, redacted telemetry and runbooks. |
| AI expansion or automated decisioning | Advisory quality, governance, evidence, and human-review processes are proven. | The existing non-blocking advisory boundary is correct and must not be weakened. |

## 4. Recommended next 90-day sequence

The sequence below is intentionally execution-light at first. It reduces the chance that new infrastructure overtakes operating discipline.

| Window | Focus | Observable exit condition |
|---|---|---|
| **Days 0–15** | Approve production operating model, threat model, cryptographic policy, RTO/RPO, owners, and runbook index. Freeze local provisioning as development-only. | The accountable people can name the authoritative components, recovery objectives, highest risks, and escalation process. |
| **Days 16–35** | Publish OpenAPI v1/canonical vectors; create supply-chain/release record; complete secret inventory and OpenBao recovery design; build a non-production identity environment. | A clean environment can reproduce the exact protocol tests and trace every credential/artifact to an owner. |
| **Days 36–60** | Implement disabled-by-default Go OIDC/JWKS validation and local subject mapping; add production-enrollment migrations/tests; complete Keycloak and OpenBao recovery exercises. | Invalid tokens, cross-tenant attempts, expired challenges, self-approval, and revoked authority fail closed. |
| **Days 61–75** | Enable non-production Flutter OIDC/enrollment acceptance; perform degraded-network and sync/evidence load tests; finalize pilot capacity limits. | A pilot-like field journey succeeds, then demonstrably fails safely under revocation/outage/weak-network cases. |
| **Days 76–90** | Add safe observability, release record, signed audit-checkpoint prototype, export manifest, and independent recovery/export verification. | An independent reviewer can assess a release, restore evidence, and detect altered export data without write access. |

## 5. Fitness tests that matter most

The following tests are more valuable than additional framework work because each proves a critical claim the platform makes.

| Claim | Fitness test |
|---|---|
| A tenant cannot access another tenant’s state. | Run cross-tenant API, repository, RLS, evidence-object, export-manifest, enrollment, and administration attempts under valid authenticated sessions. |
| A field device cannot extend its own authority. | Alter client scope/duration/tenant values and replay old authority; verify server-derived scope/epoch/expiry rejects every attempt. |
| A compromised device can be contained. | Revoke device, force authority epoch change, attempt offline sync, verify rejection, replacement enrollment, and audit linkage. |
| An identity outage does not fail open. | Make OIDC unavailable; allow only bounded previously issued offline authority where designed; deny new login/enrollment/authority issue. |
| A release can be explained and reversed. | Trace a deployed artifact to source revision, dependency digest, schema migration, approval, test record, and rollback point. |
| Evidence remains verifiable after recovery. | Restore PostgreSQL and RustFS separately into isolation, validate export manifest/digests/checkpoints, and record deviations. |
| Operations do not expose sensitive material. | Scan structured logs/traces/metrics and support bundles for secrets, tokens, private keys, raw evidence, and disallowed personal data. |

## 6. Decision

**Do not add more core technologies now.** Complete the must-do pilot blockers, then the broad-rollout improvements in the stated sequence. The desired future platform is already coherent: Go/PostgreSQL remains authoritative; Keycloak supports identity; OpenBao secures operational secrets; RustFS holds evidence; Flutter signs field actions; and observability/checkpoint/verifier components provide operational and independent assurance.

The next best action is to convert the first row of the 90-day sequence into named decision records and test artifacts—not to change the currently working local runtime.

## Related documents

- [Production-Trust Baseline Roadmap](./PRODUCTION_TRUST_BASELINE_ROADMAP.md)
- [Production Device Enrollment Specification](./PRODUCTION_DEVICE_ENROLLMENT_SPEC.md)
- [Identity and Authorization Design](./IDENTITY_AUTHORIZATION_DESIGN.md)
- [Platform Governance Backlog](./PLATFORM_GOVERNANCE_BACKLOG.md)
- [Architecture Evolution Roadmap](./ARCHITECTURE_EVOLUTION_ROADMAP.md)
