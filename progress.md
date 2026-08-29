# INTEGIN Phase 1 Progress

## Session Log

### 2026-08-13

- Reviewed the INTEGIN specification attachments and build prompt.
- Confirmed the requested work is Phase 1 — Shared Kernel.
- Inspected the connected project directory and confirmed it was empty.
- Created `task_plan.md`, `findings.md`, and `progress.md` for persistent planning.

## Current Status

Phase 1 planning and workspace initialization are complete. The Go module and shared-kernel source files have been created under `internal/shared`. A sandbox Go 1.22.2 toolchain was installed for validation because the connected Windows environment does not currently expose Go on PATH.

## Validation Log

The connected Windows test attempt reported that `go` is not recognized. The same source tree was copied into the sandbox and validated successfully with `go test ./...`; all packages compiled and the event tests passed. Phase 1 acceptance criteria are satisfied in the sandbox validation environment.

### Revision Session — 2026-08-13

- Restored the persistent plan and reviewed the existing Phase 1 deliverables.
- Revised `internal/shared/events` so event fields are private, JSON serialization is explicit, payloads are defensively copied, and event construction/deserialization validate event identity and tenant context.
- Added tests for immutable payload access, event identity, tenant context, UTC timestamp normalization, calibration expiry and validation, notification mandates, feature-flag scopes, and shared tenant context validation.
- Applied `gofmt` to all Go files.
- Validated the revised project with `go test ./...` in the sandbox: all packages passed.
- Copied the formatted, validated source tree back into `C:\MY PROJECT`.
- The connected desktop terminal sidecar disconnected during direct verification, so local Windows execution should be retried after reconnecting the desktop session.

## Final Revision Validation

The revised project was validated directly from the mounted workspace. `gofmt -l` reported no unformatted Go files, `go test ./...` passed for all packages, and `go vet ./...` completed without findings. The revised Phase 1 acceptance criteria are satisfied in the project source tree.

## Phase 1 Closure and Phase 2 Handoff — 2026-08-13

The persistent plan was reviewed and Phase 1 was confirmed complete. The task plan has been rewritten for the Phase 2 Core Engines handoff. Phase 2 will begin with the Inspection Engine, followed by template composition, certificate lifecycle and separation-of-duties checks, data completeness, and acceptance tests.

The single next action is to create `internal/domain/inspection` and implement its deterministic state machine, finding model, verdict calculation, revision handling, and unit tests using the Phase 1 shared contracts.

## Phase 2 — Inspection Engine

Implemented `internal/domain/inspection` with a deterministic inspection aggregate, lifecycle transition guards, discipline-neutral findings, severity-based verdict computation, immutable revision snapshots, separation-of-duties checks for review approval, defensive accessors, and tenant-scoped domain-event emission. Added tests for the full lifecycle, invalid transitions, verdict ordering, rejection and revision history, finding defaults, and accessor immutability.

The initial lifecycle test had an incorrect event index assertion; it was corrected to match the seven-event sequence. After correction, `gofmt`, `go test ./...`, and `go vet ./...` all passed against the mounted project.

## Phase 2 — Template Language

Implemented `internal/domain/template` with JSON-compatible template definitions, registry validation, recursive asset-tree composition, flat checklist snapshots, historical version isolation, cycle detection, and a deterministic expression evaluator supporting boolean logic, comparisons, parentheses, strings, booleans, numbers, and field identifiers. Tests passed for crane-hoist-hook composition, versioned snapshots, expression results, and invalid expressions.

## Phase 2 — Certificate Engine

Implemented `internal/domain/certificate` with controlled draft-to-approval-to-sign-to-issue transitions, authenticated actors, separation-of-duties enforcement, documented exception recording, expiry, revocation, supersession, immutable-style accessors, and tenant-scoped domain-event emission. Tests passed for the full lifecycle, role separation, exception handling, invalid transitions, revocation, expiry, and supersession.

## Phase 2 — Data Completeness and Acceptance

Implemented `internal/domain/completeness` with deterministic mandatory-field and nested-field checks, input non-mutation, and advisory suggestions forced to `blocking: false`. Added a cross-engine acceptance workflow covering template resolution, completeness, inspection close, and certificate issuance.

The complete Phase 2 suite passed with `go test ./...` and `go vet ./...` after formatting all new packages. A domain README was added to document package boundaries and validation commands.

## Phase 2 Final Handoff — 2026-08-13

Phase 2 Core Engines is complete. The project now includes Inspection, Template, Certificate, and Data Completeness engines plus a cross-engine acceptance workflow. The final mounted-project validation passed `gofmt`, `go test ./...`, and `go vet ./...`. The next planned phase is Phase 3 Supporting Engines: authorization, offline synchronization, and device trust.

## Phase 3 Plan Revision — 2026-08-13

Phases 1 and 2 were reviewed and confirmed complete with no unresolved implementation errors. The persistent plan has been rewritten for Phase 3 Supporting Engines. The next implementation action is the pure Authorization Engine, followed by Device Trust, Offline Sync, and Phase 3 acceptance validation.

## Phase 3 — Authorization Engine

Implemented `internal/domain/authorization` as a pure decision engine. It evaluates tenant and organization isolation, capabilities, resource scopes, workflow state, feature flags, environment policy, trusted offline devices, and offline authority validity, returning structured deterministic deny reasons. Tests passed for valid authorization, cross-tenant denial, capability and scope denial, workflow/feature/environment policy, and offline device authority checks.

## Phase 3 — Device Trust Engine

Implemented `internal/domain/device_trust` with pending/trusted/restricted/locked/revoked/retired lifecycle transitions, monotonic epochs, HMAC-signed offline authority packages, bounded validity, tenant/device/user binding, revocation-event emission, and epoch-based invalidation. Tests passed for lifecycle transitions, invalid transitions, authority signatures, expiry, revocation, and stale epochs.

## Phase 3 — Offline Sync Engine

Implemented `internal/domain/sync` with canonical payload hashing, HMAC transaction signatures, tenant/device/user identity validation, offline authority validation, sequence continuity, idempotent duplicate detection, explicit held gaps, conflicts, and security failures. Tests passed for applied and duplicate transactions, sequence gaps, tampering, conflicts, tenant mismatches, invalid signatures, and revoked devices.

## Phase 3 — Acceptance and Handoff

Added a cross-engine acceptance test linking authorization, trusted-device authority issuance, offline transaction application, device revocation, and subsequent authorization/sync denial. The mounted project passed the final Phase 3 validation: no unformatted Go files, `go test ./...` passed for all packages, and `go vet ./...` passed without findings.

## Phase 4 Plan Revision — 2026-08-13

Reviewed the uploaded `INTEGINPARTIV.md`. It confirms that Phase 4 is the infrastructure and event-sourcing backbone: PostgreSQL event store with partitioning, in-process event bus, aggregate replay repository, SMTP adapter, and MinIO object storage. The persistent plan has been rewritten with these deliverables, non-negotiables, acceptance criteria, and a dependency-light implementation order.

The next action is to create `internal/eventstore` and `internal/eventbus`, starting with append-only optimistic-concurrency semantics and append-before-publish tests.

## Phase 4 — Event Store and Event Bus

Implemented `internal/eventstore` with an append-only interface, in-memory implementation, PostgreSQL adapter seam using `database/sql`, optimistic concurrency, tenant-scoped event persistence, and deterministic replay ordering. Implemented `internal/eventbus` with in-process subscriptions and an append-before-publish committer. Tests passed for replay, stale-version rejection, aggregate identity validation, subscriber delivery, and authoritative store-before-publish behavior.

## Phase 4 — Repository and Storage Adapters

Implemented `internal/repository` with a generic aggregate replay repository over the authoritative event store. Implemented `internal/storage` with defensive in-memory object storage and a MinIO-compatible HTTP adapter using explicit object keys, content metadata, request signing, health checks, and path-traversal validation. Tests passed for aggregate replay, object immutability, deletion, MinIO request construction, and invalid keys.

## Phase 4 — SMTP Provider and Configuration

Implemented `internal/email` with validated SMTP configuration, standard-library SMTP sending, explicit queued/sending/sent/failed delivery states, captured failure reasons, injectable sender and health-check seams, and message validation. Tests passed for configuration errors, successful delivery, failed delivery, health failure, and invalid messages.

## Phase 4 — Migrations and Final Validation

Added the partitioned PostgreSQL event-log migration and rollback migration, including tenant and replay indexes, plus a dependency-free migration contract test. Added `internal/INFRASTRUCTURE.md` documenting adapter boundaries, runtime service configuration, and optional integration-test rules.

Final mounted-project validation passed with no unformatted Go files, `go test ./...`, and `go vet ./...`. All existing Phase 1–3 packages and new Phase 4 packages passed.

## Phase 4 Final Handoff — 2026-08-13

Phase 4 Infrastructure is complete. The project now includes the event store, event bus, aggregate replay repository, MinIO-compatible storage adapter, SMTP provider, PostgreSQL event-log migration and rollback, migration contract tests, and infrastructure documentation. Final validation passed with no unformatted Go files, `go test ./...`, and `go vet ./...`. The next planned phase is Phase 5 Platform Engines.

## Phase 5 Plan Revision — 2026-08-13

Phases 1–4 were reviewed and confirmed complete with no unresolved implementation blockers. The persistent plan has been rewritten for Phase 5 Platform Engines, including the uploaded blueprint requirements for notifications, audit, environment/release, planning, reporting, integrations, Standards Vault, feature flags, QR verification, calibration, and Feature Console.

The next action is to implement `internal/platform/notification` with preference resolution, mandatory-category handling, in-app notification records, email-delivery queue states, retry scheduling, and dependency-free tests.

## Phase 5 — Notification Engine

Implemented `internal/platform/notification` with preference precedence across user, role, and tenant levels, mandatory-category bypass, always-created in-app notifications, optional email delivery records, queued/sending/sent/failed/retry/given-up transitions, exponential retry delays, attempt counts, and explicit errors. Tests passed for preference resolution, mandatory categories, in-app creation, email queueing, successful sends, retries, and give-up behavior.

## Phase 5 — Audit, Environment, Release, and Planning

Implemented `internal/platform/audit` with tenant-scoped append-only records and defensive context copies. Implemented `internal/platform/environment` with Testing/Live release states, selected-tenant promotion, health checks, explicit failure, rollback, and recovery points. Implemented `internal/platform/planning` with work-order transitions, tenant/organization/scope checks, active-user validation, and competency requirements. All package tests passed.

## Phase 5 — Reporting, Integrations, and Standards Vault

Implemented `internal/platform/reporting` with CSV rendering and a certificate PDF renderer seam, `internal/platform/integration` with adapter health and idempotent execution, and `internal/platform/standards_vault` with tenant-aware metadata, clause search, retrieval, and object-storage delegation. Tests passed for exports, adapter health/idempotency, tenant isolation, clause search, and downloads.

## Phase 5 — Feature Flags, QR, Calibration, and Feature Console

Implemented `internal/platform/featureflag` with request-time override precedence and expiry, `internal/platform/qrverification` with narrow public projection, Testing markers, authenticated technical access, and rate limits, `internal/platform/calibration` with expiry hooks, submission blocking, and supersession, and `internal/platform/featureconsole` with module metadata, defensive copies, versions, and controlled update paths. All package tests passed.

## Phase 5 — Acceptance and Final Validation

Added a cross-engine platform acceptance test linking feature flags, calibration submission eligibility, notification queueing, QR verification, and Feature Console update metadata. The mounted project passed final validation with no unformatted Go files, `go test ./...`, and `go vet ./...`. All Phase 1–5 packages passed, including the new platform acceptance package.

## Phase 6 Plan Revision — 2026-08-13

Phases 1–5 were reviewed and confirmed complete with no unresolved implementation blockers. The persistent plan has been rewritten for the AI Advisory Layer and Enhancements. The implementation will begin with advisory insight contracts, strict `blocking=false` enforcement, AI-free-zone policy guards, and injectable provider seams.

## Phase 6 — Advisory Contracts and Monitoring

Implemented `internal/advisory` with advisory insight contracts, confidence/rationale validation, evidence references, provider metadata, forced `blocking=false`, injectable providers, and explicit AI-free-zone rejection. Implemented `internal/monitoring` with seven independent lenses, tenant-aware persistence, evidence-linked reasoning traces, model/prompt metadata, limitations, and advisory-only repository enforcement. Tests passed for AI-free zones, unsafe metadata rejection, all seven lenses, traces, and tenant isolation.

## Phase 6 — Regulation, Conflicts, Client Requests, and Validation

Implemented `internal/phase6` with deterministic regulation-change impact analysis, advisory-only affected-template findings, field-level three-way merge with preserved human conflicts, tenant-isolated client inspection request lifecycle, and a cross-engine acceptance workflow. Implemented `internal/canary` with candidate/baseline metadata, health-gated traffic, and rollback readiness.

Final mounted-project validation passed with no unformatted Go files, `go test ./...`, and `go vet ./...`. The full Phase 1–6 suite passed, including AI-free-zone, monitoring, regulation, conflict, client-request, and canary tests.

## Phase 6 Final Handoff — 2026-08-13

Phase 6 AI Advisory Layer and Enhancements is complete. The project includes advisory contracts with AI-free zones, seven monitoring lenses, evidence-linked reasoning traces, regulation impact analysis, field-level conflict preservation, client inspection requests, and canary deployment metadata. Final validation passed with no unformatted Go files, `go test ./...`, and `go vet ./...`. The remaining deployment work is optional integration of the Python AI service and secondary advisor UI, without changing the advisory-only primary decision boundary.

## Deployment Integration Plan — 2026-08-13

Phases 1–6 were reviewed and confirmed complete. The persistent plan now covers optional Python AI-service integration, secondary advisor presentation, opt-in integration validation, health/canary checks, and deployment documentation. The next action is to create `internal/aiintegration` with a versioned advisory HTTP contract, injectable transport, timeout handling, response normalization, health checks, and dependency-free mock tests.

## Deployment Integration — AI-Service Contract

Implemented `internal/aiintegration` with a versioned `/v1/advisory` HTTP contract, injectable HTTP client, API-key header support, timeouts, health checks, tenant validation, malformed-response errors, AI-free-zone rejection, and normalization through the existing advisory contract. Tests passed with local mock HTTP servers and no external service dependency.

## Deployment Integration — Presentation and Opt-In Validation

Implemented `internal/advisorview` with approved-field secondary-panel view models, tenant filtering, evidence links, limitations, blocking enforcement, and deployment metadata. Implemented `internal/deployconfig` with disabled-by-default configuration loading, endpoint/API-key separation, timeout validation, and explicit enablement requirements. Regression tests and `go vet ./...` passed across the full project.

## Deployment Integration Final Validation — 2026-08-13

Added `AI_DEPLOYMENT.md` documenting service topology, configuration variables, contract/security rules, rollout/rollback, canary operation, and validation commands. Final validation passed with no unformatted Go files, `go test ./...`, and `go vet ./...`. The AI-service integration remains opt-in and advisory-only.

## Dual-Service Delivery — 2026-08-13

Validated the Quiet Signal advisor console with a full-page desktop screenshot and an independent style review; the review found the design aligned with the chosen direction. `pnpm check` and `pnpm build` pass. The font import ordering warning was fixed; Vite still reports only runtime resolution of the generated hero asset and a non-blocking bundle-size advisory.

Implemented `ai_service/main.py`, `ai_service/requirements.txt`, `ai_service/test_main.py`, and `ai_service/README.md`. The service exposes `/healthz` and `/v1/advisory`, accepts only `MONITORING` and `REGULATION`, rejects the AI-free zones, returns evidence/rationale/provider metadata, and forces `blocking=false`. Sandbox validation passed with 4 tests.

The connected Windows environment does not have pytest installed, so local Python tests require installing `ai_service/requirements.txt`. The Windows Go suite was started but did not return output within three 60-second waits; this remains a validation-environment limitation to report unless the process completes before checkpointing.

## Work-Order Stage 0 — Server Composition and HTTP Runtime Proof — 2026-08-21

Implemented `internal/workorderpg/transaction_runner.go` as a documented repository-owned transaction adapter and added focused tests. Implemented `internal/server/workorder_composition.go`, injected the optional handler through `server.Dependencies`, mounted `POST /work-orders/partial-submissions` only when composed, and constructed it from `cmd/integin-server` only after OIDC and PostgreSQL identity dependencies initialize. The endpoint is included in the OpenAPI contract.

Added `TestAuthenticatedPartialSubmissionHTTPPostgresIntegration`. The controlled pilot test uses unique `it-http-*` fixture identities, a deterministic token validator, the production local identity resolver, derived-authority service composition, canonical inspection persistence, and the real server mux. It proved an authorized `200`, idempotent replay `200` with an equivalent receipt, canonical submitted inspection/item writes, and cross-organization `403` without Work-Order disclosure. Explicit in-test cleanup plus a post-run inventory confirmed zero namespaced Work-Order, inspection, and identity fixtures.

Final validation on the connected source workspace passed `go test ./...` and `go vet ./...`. The evidence ledger and dedicated runtime evidence document distinguish the proven local path from still-unproven real external OIDC validation, non-owner durable read isolation, and Flutter Matrix Fixture Provisioning.

## Stage 0 — Non-Owner Canonical Inspection Read Isolation — 2026-08-21

Completed the independent applied-pilot RLS read-isolation gate with two generated `it-rls-read-*` graphs under one tenant and separate organizations. A transaction assumed `integin_pilot_runtime`, which is neither superuser nor `BYPASSRLS`, and observed `A=1`, `B=1`, and `UNSET=0` as contexts changed. Owner-side cleanup removed all graph rows; post-run inventory returned zero namespaced work orders and inspections. The full `go test ./...` regression suite then passed.

## Stage 0 — Controlled OIDC Validator and Work-Order Composition — 2026-08-21

Added `TestSignedOIDCPartialSubmissionHTTPPostgresIntegration` with a local loopback discovery/JWKS issuer and generated RSA signing key. The production OIDC validator rejected a tampered signature and expired token with `401`, then accepted a valid RS256 bearer token and completed local membership resolution, canonical inspection partial submission, and receipt persistence with `200`. The pilot database inventory returned zero `it-oidc-http-*` work-order, inspection, and identity rows after cleanup. Full `go test ./...` and `go vet ./...` passed.

## Field Matrix Fixture Provisioning Preflight — 2026-08-21

Diagnosed the connected Flutter runner. `flutter.bat` exits silently in the sidecar, while bundled Dart and the Flutter tools snapshot work. The snapshot-run Field acceptance test initially required a standard `PROGRAMFILES(X86)` process environment value; with that temporary value it exited `0` and skipped exactly as designed because all three live endpoint definitions are absent. No Flutter SDK, PATH, workstation environment, endpoints, fixtures, or runtime service was changed.

## Stage 0 — Field Matrix Fixture Provisioning Runtime Proof — 2026-08-21

Discovered and used the existing loopback-only `integin-server-provision` acceptance runtime. Added a receipt-first Field test convention and `integin-field-acceptance-cleanup` command so generated device, authority, sync, and evidence identities can be removed deterministically. The final acceptance run passed provisioning, signed sync, `APPLIED` evidence, duplicate-safe evidence replay, and cleanup. Pilot Work-Order fixture namespaces and the Field receipt were then verified absent. Full `go test ./...` and `go vet ./...` passed after the related HTTP authority hardening and Field changes.

## Stage 0 — Disposable Keycloak Provider Proof — 2026-08-21

Added `TestKeycloakPartialSubmissionHTTPPostgresIntegration`, an environment-gated test that uses a genuine Keycloak token and production OIDC validator rather than an issuer double. A temporary loopback Keycloak `26.7.1` server with generated realm/client/user initially exposed a Keycloak account-completeness policy; the user profile was corrected without retaining the failed realm or fixtures. The final proof passed tampered-token `401`, valid-token `200`, canonical submission persistence, pilot cleanup, disposable realm cleanup, and container removal. The full `go test ./...` and `go vet ./...` suite passed afterward.

## Repository Hygiene — 2026-08-21

Completed selective workspace cleanup. Removed `field_app/build/`, `ai_service/.pytest_cache/`, and `ai_service/__pycache__/` only. The first guard declined to remove pytest cache due to its nested ignore rule; after verifying the standard cache shape, removal succeeded. All protected paths remained present, and `git diff --check`, `go test ./...`, and `go vet ./...` passed without recreating Flutter build output.

## Stage A Recovery Drill Inventory — 2026-08-21

Started the next roadmap gate: PostgreSQL and RustFS recovery verification. Confirmed the pilot source services are active and must not be altered; recorded the two verified custom-format pilot backups and the runbook’s requirement for isolated restore targets plus object/manifest, tenant, digest, relationship, and audit checks. No restore, database mutation, object copy, or service restart has occurred.

## Stage A Current-Schema PostgreSQL and RustFS Recovery Drill — 2026-08-21

Added `cmd/integin-recovery-drill` and S3 metadata-read coverage. The first controlled attempt stopped at an incorrect in-container dump connection; the manifest-scoped source cleanup then exposed and corrected a SQL argument-binding defect. The final drill seeded an `it-recovery-*` current-schema graph and evidence object, backed up the source, restored into fresh loopback-only PostgreSQL/RustFS containers, verified relational rows, audit state event, RLS matching/mismatch, ciphertext digest, and metadata, then removed source fixture/object and target containers/volumes. The successful dump checksum and manifest are retained under the acceptance backup directory. Full `go test ./...` and `go vet ./...` passed.

## Stage A Evidence Retention and Verified Export — 2026-08-21

Added `internal/domain/evidence`, `internal/evidencepg`, and `internal/evidenceexport`. Applied reviewed pilot migrations `0010` for the forced-RLS relational metadata index and `0011` for manifest-required explicit encryption provenance, retaining separate immediate pre-apply backups and isolated restored-copy up/down review evidence. The export projection lists actor-scoped metadata, re-fetches each RustFS object, rejects missing/key/content-type/byte-count/digest failures and mixed privacy sets, then seals/verifies the existing pure manifest contract.

The first PostgreSQL/RustFS integration attempt exposed that the local owner is a superuser and bypasses RLS; it returned a cross-organization row when a repository list query depended solely on RLS. The repair added explicit actor-derived tenant/organization predicates to list queries. The final `TestPostgresRustFSVerifiedExportIntegration` passed same-tenant sealing, repository cross-organization zero, direct `integin_pilot_runtime` RLS `1/0`, missing object rejection, same-length ciphertext digest mismatch rejection, and dependency-ordered object/database cleanup. Post-run `it-evidence-*` metadata inventory was zero. Full `go test ./...` and `go vet ./...` passed.

The first combined final regression run used a PowerShell output pipeline and returned an ambiguous nonzero result despite the controlled export test passing. Direct unpiped `go test -count=1 ./...` and `go vet ./...` both passed afterward. A review then removed exactly six tiny accidental checksum-marker text artifacts from the source root after confirming their approved binary backups remained in `operations\acceptance\backups`; five evidence migration pre-apply dumps are retained.

## Stage A Runtime Operations — 2026-08-21

Defined the operational HTTP contract and implemented bounded readiness context, no-store operational replies, caller correlation grammar validation with generated fallback, redacted path-only structured request logs, and deterministic known-body `413` rejection before dispatch. Focused `internal/server` tests passed liveness, readiness success/failure/deadline, correlation behavior, redacted logging, and known/streaming body limits. A disposable loopback candidate passed health/readiness `200/200`, correlation echo, query redaction, and oversized `413`; it was terminated and the temporary binary removed. Direct unpiped full `go test -count=1 ./...` and full `go vet ./...` passed. A combined command was stopped only after its test/vet results were complete because the terminal stalled while Git emitted pre-existing non-fatal CRLF conversion warnings.

## Stage A Authenticated Evidence Metadata Registration — 2026-08-21

Added a separate OIDC/membership-derived `/evidence/metadata-registrations` boundary, application service, atomic assignment-scoped repository method, optional composition, and OpenAPI contract. The first server-route test correctly failed because OpenAPI route completeness was intentionally enforced; the contract and test inventory were then updated before focused checks passed. The controlled PostgreSQL/RustFS test passed forged authority-field rejection, applied registration, duplicate replay, missing-object conflict, cross-organization assignment denial, object cleanup, and zero database residue. `go test -count=1 ./...` and `go vet ./...` passed after the changes.

## Certificate versus Export Sequence Review — 2026-08-21

Completed a bounded critical-boundary review after the user requested delegation and debate. The available capability check correctly selected owner-only handling rather than an independent mutation crew. The review considered field/client workflow, assurance/disclosure control, and future external integration. It recommends certificate-first for the normal lifecycle, with separately approved evidence export afterward and a future per-adapter pre-recognition export exception. The recommendation and residual policy gaps are retained in the review and structured decision-record documents; no authority path was implemented or released.

## High-Level Operating Model Review — 2026-08-21

Completed the owner-requested whole-model review, including official public context from ISO/IEC 17020:2026, SAAC, and ZATCA. The review does not claim accreditation, tax applicability, compliance, or external acceptance. It reconciles the normal INTEGIN lifecycle and configurable exception profiles, defines the hard separation between inspection, certificate, evidence export, external recognition, commercial handoff, and reminders, and sequences the roadmap: certificate lifecycle, QR projection, Work Execution Summary, renewal queue, export approval ledger, commercial adapter, then external authority adapters. No production or authority mutation was made.

## Attached Project-Inspection Conversation Audit — 2026-08-21

Parsed the supplied historical `project-inspection.json`, extracted its 18 user-authored messages, and treated its agent output as untrusted historical analysis. It mostly confirmed existing product direction and documented governance practice. Current source rechecks independently confirmed that the sync processor still performs receipt lookup before device/signature validation, treats held receipts as duplicate by payload hash alone, falls through unknown signature algorithms to HMAC, and persists held receipt/envelope separately. The current Work-Order array parameter helper also lacks quoted/backslash proof. The audit added tracked corrective tasks and elevated the previously discussed certificate-template binding-key registry to the first sub-foundation before certificate rendering. No code or runtime behavior changed.

## Offline-Sync Durability Correction — 2026-08-21

Implemented the defined sync durability correction in `internal/domain/sync` and `internal/syncstate`. Initial focused compilation passed. The first live proof was delayed because Docker Desktop and the pilot PostgreSQL container had stopped; the engine was restored and the container restart showed no OOM or recorded database error. A first controlled proof exposed an early-return bug in `SaveReceipt` that skipped held promotion; this was repaired. A second proof exposed global fixed transaction-ID collisions from prior failed fixtures; IDs were namespaced. The test then passed, but external residue inspection found cleanup defects caused by closed database ordering and parameterized multi-command SQL; those test-harness defects were repaired, prior namespaced residue was removed in dependency order, and the final live proof passed with zero residue. Full unpiped `go test -count=1 ./...` and `go vet ./...` passed.

## Certificate-Template Binding Registry — 2026-08-21

The pending resolver sources were formatted and compiled successfully with `go test ./internal/domain/certificatetemplate ./internal/certtemplatepg`. The first controlled resolver invocation could not reach `127.0.0.1:15432`; investigation showed that `integin-pilot-postgres` had exited with code `255`. The existing pilot container was started without changing its data or schema, `pg_isready` confirmed readiness, and the control proof was retried. `TestPostgresResolveCanonicalInspectionTemplateIntegration` then passed in `0.15s`, including canonical inspection-value resolution and cross-organization denial. The fixture inventory query confirmed `certificate_template_resolver_residue=0`. The direct repository gates subsequently passed: `go test -count=1 ./...` and `go vet ./...`. The registry gate is therefore complete; the next planned boundary is the certificate authority lifecycle, which remains unimplemented.

## Checklist Record Recovery — 2026-08-21

A transient mounted-filesystem transport interruption during checklist reconciliation emptied the parent `C:\MY PROJECT\todo.md`. The engineering source, applied migrations, pilot database, pilot object store, and architecture decision remained intact. The empty file was retained as `todo.md.empty-recovery-20260821-232800.bak`; a concise evidence-grounded checklist was rebuilt directly on the connected desktop from `task_plan.md`, `progress.md`, and `findings.md`. The recovered checklist retains the completed Stage 0/Stage A and certificate-template registry milestones and the next certificate lifecycle, Work-Order special-character, and protected-boundary tasks.

## Certificate Authority Lifecycle Design — 2026-08-21

Completed the authority-sensitive lifecycle design before persistence implementation. The design reconciles the existing pure certificate aggregate with the owner-approved dual profile: independent review remains the default, while senior self-issue is an explicit, policy-scoped, actor-derived exception with durable audit evidence. It defines canonical eligibility as an `APPROVED` and `FINALIZED` inspection revision; immutable template/value snapshots; transaction-scoped per-tenant/organization numbering; template/inspection-type validity; truthful revocation/supersession; and a narrow QR-safe public projection. It records that the current binding registry lacks asset serial/description, test scope, evidence sufficiency, signatory display identity, and rendering authority, so the lifecycle slice will not claim or expose them. A complex/deep architecture-and-testing review was selected; no independent persistent reviewers are available, so the owner completed bounded architecture/authority and verification/negative-case passes.

## Certificate Lifecycle Draft-Persistence Foundation — 2026-08-21

Created and completed the first certificate-lifecycle persistence gate. The `0013` candidate and rollback were first exercised against a disposable PostgreSQL restored from a fresh pilot dump. The initial binary dump attempt failed because PowerShell byte encoding rejects text-pipeline content; the recovery wrote the dump inside the container and copied it as a binary file. The first disposable restore then failed because the backup grants reference `integin_pilot_runtime`; the review container was recreated with only that required non-login runtime role, restoring successfully. The up/down review verified five lifecycle tables and five forced-RLS relations, and the disposable container was removed. The candidate was then applied to pilot only after the verified pre-apply backup was retained.

Implemented the certificate-authority domain, atomic draft-service contract, PostgreSQL draft repository, and a controlled integration test. The pilot proof passed approved/finalized canonical inspection selection, approved template/policy binding, immutable draft and audit persistence, explicit cross-organization denial, cleanup, and zero namespaced residue. Full Go regression and vet passed. Review/sign transition helpers and issuance helpers are source-compiled but still need focused integration evidence; no certificate has been issued and no public verifier endpoint exists.

## Certificate Issuance, Revocation, and QR-Safe Projection Proof — 2026-08-21

Extended `TestPostgresCreateCertificateDraftIntegration` to pass the independent draft/submission/review/sign/issue path. It proved approved policy/template gating, transactional number allocation, snapshot persistence, a 32-byte public token digest, duplicate issuance rejection, correct-token repository projection, wrong-token non-disclosure, revocation, repeat-revocation rejection, public `REVOKED` status, cleanup, and zero namespaced residue. A malformed regex insertion initially corrupted the test tail by treating newline whitespace as part of its marker; the tail was rebuilt from the last complete audit-query line. A subsequent assertion incorrectly queried only `ISSUED` after revocation; it was corrected to accept the expected issued-or-revoked state before rerun. Final `go test -count=1 ./...` and `go vet ./...` passed.

## Complete Certificate Lifecycle Persistence Proof — 2026-08-21

The lifecycle proof now includes both owner-approved profiles and all current terminal/corrective states. After revocation made the original inspection revision eligible for a new active record, the proof enabled the approved self-issue policy, issued a senior self-issue certificate, raised the canonical inspection revision, issued a same-asset replacement, superseded the prior certificate atomically, then expired the replacement after its stored expiry. Public projections returned truthful `REVOKED`, `SUPERSEDED`, and `EXPIRED` states using only digest-matched tokens. The final fixture cleanup count was zero and final repository regression/vet passed.

## Certificate HTTP Transport Foundation — 2026-08-21

Reviewed existing evidence HTTP/OIDC/membership patterns and found that OIDC validates authentication only while local membership resolves tenant/organization/actor/capabilities. Added a certificate handler and local actor resolver that keep authority server-derived, reject unknown JSON fields, set `Cache-Control: no-store`, and test missing authentication, forged authority-shaped draft fields, and server-derived actor propagation. The initial handler deliberately supports only draft, submit, review, and sign; it is not yet server-composed. Full regression/vet passed.

## Certificate Mutation HTTP Source Proof — 2026-08-22

Built internal/certificatehttp for the full source-level mutation route shape and rebuilt it after a disconnected partial edit. Focused tests prove no-store missing-auth rejection, forged tenant field rejection, server-derived draft actor propagation, successful one-time issuance token response, and rejection of a non-empty issue body. Full go test -count=1 ./... and go vet ./... passed. The handler is not yet server-composed or live-runtime exercised.

## Public Verifier Recovery — 2026-08-22

A public verifier package was drafted but could not be completed because repeated desktop-sidecar interruptions prevented helper writes. The incomplete, untested package was removed rather than retained in a failing state. The repository was restored to the last passing certificate transport source and go test -count=1 ./... plus go vet ./... passed. Public verifier HTTP transport remains an explicit unchecked item; the existing digest-backed repository projection remains the only verified public-verification foundation.

## Public Certificate Verifier Source Proof — 2026-08-22

Completed the previously deferred public verifier using the existing digest-backed projection only. Tests prove bounded projection/no-store response, invalid/unknown token indistinguishability, and limiter-before-lookup. Full repository regression and vet passed. No public server route or production-runtime claim is made.

## Certificate Runtime Composition and Negative Probe — 2026-08-22

Mounted optional certificate routes in server.NewMux, wired public repository projection with database presence and mutation routes under the existing enabled-OIDC branch, and documented both prefixes in OpenAPI. Mux composition test and full regression/vet passed. The read-only candidate negative probe passed for public invalid/unknown tokens and method rejection; OIDC-disabled mutation composition returned 404 as designed. Candidate cleanup and zero it-certificate-runtime-probe certificate/inspection rows were verified.

## Controlled Signed-OIDC Certificate Mutation Proof — 2026-08-22

Added and passed TestSignedOIDCCertificateTransportUsesLocalMembership. It exercises the composed server mux with a local RS256 discovery/JWKS issuer, real OIDC validation, and migration-owned local identity resolution. Verified 401 tampering, 403 unknown subject, 204 authorized server-derived actor propagation, 403 zero-capability membership denial, cleanup, and full regression/vet.

2026-08-22: Certificate public transport composition proof completed. Added TestPublicVerifierComposesThroughServerMux: a valid bounded projection is served as 200 no-store through server.NewMux and the immediate second request is 429 no-store with only one verifier lookup. Actual issued-token digest resolution remains proven in certificatepg integration. Full go test -count=1 ./... and go vet ./... passed; queried lifecycle/OIDC residue was 0|0|0. The first cross-layer DB test attempt created a Go import cycle and was removed cleanly before final regression. Shared rate control, external release, QR/PDF output, and canonical serial/description/type/test-scope projection remain unproven.

2026-08-22: Completed follow-on certificate-verification design inventory. Recorded canonical data gaps (serial and description absent; asset type only in work-order scope; inspection type not persisted in inspection_record; no canonical public test-scope summary), drafted the tenant-safe immutable public-binding contract, documented PDF/QR prerequisites, and published the gated eight-step implementation sequence. No schema, route, renderer, QR artifact, public release, or external integration was added.

2026-08-22: Completed 0014 canonical public-binding slice. Disposable up/down review passed; fresh pre-apply dump saved at tmp\integin-pilot-pre-0014-20260822-094737.dump; 0014 applied to pilot. Issuance now snapshots required policy-approved canonical asset and inspection public-scope values and includes them in the snapshot digest. Integration proved snapshot persists serial/description/type/inspection scope and survives later mutable asset changes. Namespaced residue was 0|0|0|0|0; full go test -count=1 ./... and go vet ./... passed. Public HTTP projection, QR/PDF output, public deployment, and shared rate/abuse controls remain unimplemented.

2026-08-22: Completed immutable snapshot-backed public verifier expansion. VerifyPublic joins only certificate_snapshot.public_binding_snapshot, and the handler conditionally allow-lists serial, description, type, inspection type, and structured test scope while preserving the original public fields, no-store, 404 indistinguishability, and limiter behavior. Focused tests, full go test -count=1 ./..., go vet ./..., and zero fixture residue (0|0|0|0) passed. QR/PDF rendering, shared rate/abuse control, public release, and external integrations remain unimplemented.

2026-08-22: Implemented internal/certificaterender fixed-cell PDF and high-recovery QR capability with go-pdf/fpdf v0.9.0 and skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e. Renderer validates template geometry and bounded text policies, rejects insecure QR origins/overflow, returns byte digest and non-sensitive metadata, and uploads through storage.Store. Focused renderer tests plus full go test -count=1 ./... and go vet ./... passed. It is not yet connected to actual issuance snapshots/tokens, PostgreSQL artifact metadata, authenticated download, public release, or shared rate/abuse controls.

2026-08-22: Owner-authorized Go upgrade completed. The active toolchain is go1.26.5 windows/amd64 and go.mod now declares go 1.26.5. go mod tidy, focused certificate/renderer tests, full go test -count=1 ./..., and go vet ./... passed. The Codeberg fpdf v0.12.0 release line is official and currently active, but the project remains on frozen github.com/go-pdf/fpdf v0.9.0 until the owner explicitly approves a module-path switch; the QR dependency is likewise frozen.

2026-08-22: Completed technology update audit excluding frozen PDF/QR dependencies. Current: Go 1.26.5, PostgreSQL 18.6, Keycloak 26.7.1, Docker 29.7.2, JWT v5.3.1. Recommended separately controlled updates: OpenBao 2.6.0 to 2.6.2 due upstream security fixes; lib/pq 1.10.9 to 1.12.3 with integration proof; RustFS rc.1 to newer rc only after backup/recovery/S3 compatibility rehearsal. PostgreSQL uses a broad postgres:18 tag and should later be exact minor/digest pinned. No version, image, data, config, or frozen dependency was changed by this audit.

2026-08-22: Non-invasive performance baseline complete; see docs/architecture/INTEGIN_PERFORMANCE_BASELINE_RESULTS_2026-08-22.md. Full regression/vet passed; no domain/RLS/certificate/sync optimization justified.

2026-08-22: Controlled invasive baseline complete. Certificate lifecycle including real DB public-verifier repository assertions: 1422 ms. Signed local RustFS contract: 1337 ms. Sync recovery/atomicity: 937 ms. Namespaced DB residue zero; full regression/vet passed. No protected logic optimization justified. Detailed result draft was blocked only by mounted-path new-file creation; existing contract document remains present.

2026-08-22: Repeated five-sample local-pilot bottleneck comparison complete. Certificate lifecycle with real DB verifier checks: mean 1298.05 ms, range 1157.77-1415.80 ms. RustFS signed contract: mean 864.50 ms, range 794.65-942.51 ms. Sync recovery/atomicity: mean 922.52 ms, range 821.47-1203.38 ms. These are full test-path timings including Go setup, not endpoint latency. No reproducible protected-logic bottleneck was identified; DB-backed HTTP verifier remains unisolated. Namespaced residue zero; full regression/vet passed.

2026-08-22: Added cleanup-backed real PostgreSQL HTTP public-verifier fixture. Five independent 20-request samples: request mean-of-means 3.113 ms, mean range 2.463-4.625 ms, worst sample p95 10.043 ms. All responses 200 no-store. Discovered shared request logger exposed path token; fixed by logging /verify/certificates/:token and added regression test. Fixture residue zero; full regression/vet passed.

## Nemotron/OpenCode External Analysis Reconciliation — 2026-08-22
The owner-supplied free Nemotron/OpenCode conversation is external advisory input, not current-source evidence. Its visible manifest/OIDC/certificate/public-verifier/PDF-QR absence claims are materially outdated: controlled Stage 0 Field/OIDC evidence, certificate lifecycle/0013, immutable public bindings/0014, snapshot-backed verifier projection, internal renderer, zero-residue proof, and repeated full regression/vet records now exist. Its AI advisory-only, human-confirmation, deterministic risk/action/KPI, document-ingestion, and infrastructure-hardening ideas remain valid future design candidates, not defects in completed authority work. Its B*/R* alleged bugs require current-source reproduction before priority assignment. No task, priority, or authority change was adopted solely from the external analysis.

## Advanced Multilingual Renderer Qualification Foundation — 2026-08-22
The next renderer milestone is specification only: a restricted declarative template contract, fixed-certificate and flowing-report layout classes, a synthetic Arabic/English qualification corpus, and a bakeoff protocol. No renderer service, Python runtime, package, certificate authority path, public artifact, external endpoint, or customer content is added. Go/PostgreSQL remains authority; a future renderer receives only a tenant-bound immutable snapshot, approved template/assets/fonts, and a verifier payload.

2026-08-22: Sandbox-only WeasyPrint qualification extended corpus passed preliminary fixed-cell, 4-page flowing table, bidi punctuation/numeral, invalid-font/missing-glyph, unsafe-template, bounded-input, and in-process no-resource-request checks. Environment-level egress denial remains unproven; no candidate adoption, authority integration, artifact storage, or public delivery occurred.

2026-08-22: Added and visually reviewed a synthetic Arabic-first mixed-direction fixture: 'أنا أريد I want حماماً.' plus isolated identifier, measurement, date, zone, and punctuation fields. It rendered as one page with expected extraction anchors; evidence remains synthetic-only.

2026-08-22: Created INTEGIN_WAYFINDER_DECISION_MAP_2026-08-22.md. It classifies supplied audits as advisory, confirms the current uncommitted worktree as the immediate recoverability concern, and defines D-01 baseline preservation as the sole initial implementation frontier. No code, migration, runtime, or public action was taken.

2026-08-22: D-01 non-destructive worktree review completed at HEAD 249cd06. Inventory found 64 status entries; full Go tests and vet passed. Recommended archive-first, review-then-commit; no archive, staging, commit, migration, runtime change, or public action was performed.

2026-08-22: D-01 review completed at HEAD 249cd06. Inventory found 64 status entries; full Go tests and vet passed. Archive-first, review-then-commit is recommended. No archive, staging, commit, migration, runtime change, or public action was performed.

2026-08-22: Owner-approved archive created at C:\MY PROJECT\preservation-archives\integin-pilot-source-precommit-20260822-150512.tar.gz with SHA-256 F070E02DC330FC473D856E925802C752245CAAB9CB927556FB54D719C283E841. Manifest recorded HEAD and 64 status entries. Full archive-read and Git-metadata-exclusion checks passed; status remained 64; no repository mutation occurred.

2026-08-22: Completed non-destructive shared-file boundary review. cmd/integin-server, internal/server/http, OpenAPI, and module files span work-order, evidence, certificate, operational, and frozen-renderer concerns. Future preservation must use hunk-level review; no staging, commit, or source change occurred.

2026-08-22: Unit 1 review refined the original broad grouping. The exact future proposal is sync-domain plus sync-state durability only; identity belongs to work-order, storage retrieval to evidence, and renderer dependencies to the frozen prototype. Focused sync tests passed; no files were staged or committed.
