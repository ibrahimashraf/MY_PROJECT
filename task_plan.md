# INTEGIN — Dual-Service Implementation

## Goal
Complete the optional INTEGIN dual-service delivery: validate and checkpoint the Quiet Signal advisor console, implement and test the Python FastAPI advisory service, and preserve the advisory-only boundary across every integration point.

## Phase Status

| Phase | Status | Scope |
|---|---|---|
| 1. Shared Kernel | complete | Immutable tenant-scoped events, shared types, response definitions, notifications, feature flags, calibration records, tests, formatting, and vet validation. |
| 2. Core Engines | complete | Inspection, template, certificate, data completeness, cross-engine acceptance tests, formatting, and vet validation. |
| 3. Supporting Engines | complete | Authorization, device trust, offline sync, cross-engine acceptance tests, formatting, and vet validation. |
| 4. Infrastructure | complete | Event store, event bus, aggregate replay repository, MinIO storage, SMTP provider, migrations, documentation, tests, and vet validation. |
| 5. Platform Engines | complete | Notifications, audit, environment/release, planning, reporting, integrations, Standards Vault, feature flags, QR verification, calibration, Feature Console, acceptance tests, formatting, and vet validation. |
| 6. AI Advisory Layer | complete | Advisory contracts, AI-free zones, seven monitoring lenses, reasoning traces, regulation analysis, conflict resolution, client requests, canary metadata, acceptance tests, formatting, and vet validation. |
| 7. Go AI-Service Integration | complete | Versioned HTTP contract, guarded zones, timeouts, health checks, tenant validation, response normalization, configuration, and mock validation. |
| 8. Secondary Advisor UI | in_progress | Quiet Signal React/TypeScript console with evidence-linked insights, reasoning traces, health/canary metadata, and advisory safeguards. |
| 9. Python Advisory Service | complete | FastAPI `/healthz` and `/v1/advisory`, allowed-zone enforcement, deterministic monitoring-lens summaries, response metadata, and `blocking=false`. |
| 10. Cross-Service Handoff | in_progress | Final Go validation, persistent records, UI checkpoint, and delivery of the working console and service instructions. |

## Current Phase

Phase 8 — Secondary Advisor UI validation and checkpoint preparation.

## Acceptance Criteria

1. The advisor UI renders at desktop and mobile sizes, exposes evidence-linked signals and reasoning traces, and clearly communicates that insights cannot change primary INTEGIN workflows.
2. The UI passes TypeScript validation and production build validation, with only documented non-blocking asset/chunk warnings.
3. The Python service accepts only `MONITORING` and `REGULATION` zones, rejects all AI-free zones, and returns the v1 contract with confidence, rationale, evidence references, provider metadata, limitations, and `blocking=false`.
4. The Python service remains deterministic and does not mutate INTEGIN state or receive secrets, private prompts, or primary decision controls.
5. Go tests and vet remain passing after the dual-service additions.
6. A single pre-delivery UI checkpoint is created before presenting the console to the user.

## Deployment-Integration Acceptance Criteria

1. The AI service client communicates through an explicit versioned request/response contract and never receives AI-free-zone requests.
2. Provider timeouts, non-2xx responses, malformed responses, and unavailable services become explicit advisory errors; primary workflow state remains unchanged.
3. Provider responses are normalized through the existing advisory contract, including `blocking=false`, confidence, rationale, evidence references, model metadata, and limitations.
4. Secondary advisor view models expose only approved advisory fields and never expose secrets, hidden prompts, private credentials, or primary decision mutation hooks.
5. Health checks and canary metadata express baseline, candidate, health, traffic percentage, and rollback readiness.
6. Integration tests are opt-in or use local mocks; default `go test ./...` remains dependency-free.
7. Deployment configuration is documented without embedding credentials in source control.

## Non-Negotiables

AI remains advisory only. The integration client cannot set verdicts, approve inspections, issue certificates, validate calibration, authorize requests, accept sync transactions, or expand public QR projections. Service failure must degrade advisory panels gracefully without blocking or mutating primary workflows.

## Design Decisions

| Decision | Rationale |
|---|---|
| Use a versioned HTTP contract with an injectable transport. | It supports the Python service while preserving deterministic Go tests and allowing future provider changes. |
| Normalize all provider output through `internal/advisory`. | The existing contract already enforces confidence, rationale, evidence, metadata, and `blocking=false`. |
| Keep secondary advisor data separate from primary domain aggregates. | A deployment failure or stale insight must never alter authoritative business state. |
| Require explicit environment configuration for live integration. | Credentials and service endpoints must remain outside source code and default tests. |

## Phase 1–6 Closure

Phases 1–6 are complete with no unresolved implementation blockers. The mounted project contains the full deterministic INTEGIN core, platform engines, advisory contracts, monitoring, regulation/conflict/client-request flows, canary metadata, and passing `gofmt`, `go test ./...`, and `go vet ./...` validation.

## Errors Encountered

| Error | Attempt | Resolution |
|---|---:|---|
| Unix commands were initially used against the Windows project path. | 1 | Switched to PowerShell-compatible inspection and mounted-file operations. |
| Desktop terminal sidecar disconnected during verification. | 1 | Validated the mounted source directly from the sandbox and recorded reconnect requirement. |
| Inspection lifecycle test expected the submitted event at the wrong index. | 1 | Corrected the assertion; all Phase 2 tests passed afterward. |
| Initial SMTP test stub used an incorrect callback signature. | 1 | Replaced it with a clean `smtp.Auth`-compatible sender test; the email package then passed. |
| Phase 6 conflict merger declared an unused `baseOK` variable. | 1 | Removed the unused variable before rerunning the suite. |
| Initial conflict test fixture did not contain genuine concurrent conflicts. | 1 | Updated serial and status to diverge on both sides; the suite then passed. |
| A targeted edit used pre-gofmt test text. | 1 | Read the formatted file and applied a line-accurate edit. |
| A final-status edit targeted a duplicate error row. | 1 | Re-read the plan, removed duplicate rows, and applied final status edits separately. |

Deployment integration is complete. The optional Python AI service and secondary advisor UI can now be integrated through the documented contract without changing the advisory-only primary decision boundary.

## Dual-Service Validation Notes

The UI screenshot review described the design as strong and aligned with Quiet Signal. `pnpm check` and `pnpm build` pass. The CSS import ordering warning was fixed. Vite still reports that the generated `/manus-storage` hero asset remains runtime-resolved and that the JavaScript bundle is above the default 500 kB advisory threshold; neither blocks the build.

The Python service tests pass in the sandbox: 4 passed. The connected Windows Python environment does not currently have `pytest`, so local Windows test execution requires installing `ai_service/requirements.txt` first. Go validation was started on Windows but did not produce output within three 60-second waits; it must be confirmed or reported as an environment-timeout limitation before final delivery.

## Errors Encountered — Dual-Service Session

| Error | Attempt | Resolution |
|---|---:|---|
| Windows Python environment lacked `pytest`. | 1 | Validated the service in the sandbox after installing pytest; documented the Windows dependency setup. |
| CSS font import appeared after stylesheet rules. | 1 | Moved the font import before Tailwind imports; the warning is gone. |
| Windows Go validation produced no output after three 60-second waits. | 1 | Inspected the terminal and retained the process for diagnosis; final status remains pending. |
| Mounted-file read used an invalid negative range. | 1 | Re-read the file with a valid range and logged the issue. |
| An incorrect root `README.md` path was queried. | 1 | Used the existing `ai_service/README.md` and planning files instead. |

## Next Step

Confirm the Windows Go validation status without repeating the timed-out wait, then create the single pre-delivery UI checkpoint and deliver the UI/service handoff.

## Stage 0 Work-Order Continuation — 2026-08-21

### Goal

Complete and evidence the server-authoritative Work-Order Stage 0 foundation without claiming external identity-provider, certificate-issuance, Field-app, or production deployment proof that has not been executed.

| Phase | Status | Evidence |
|---|---|---|
| Domain, persistence, canonical inspection, authorization, and authenticated transport foundation | complete | Existing controlled PostgreSQL and handler contract suites; applied pilot migrations `0005`, `0008`, and `0009`. |
| Server composition and transaction-boundary decision | complete | Repository-owned transaction adapter, optional dependency construction, server mux test, OpenAPI route record, and full regression. |
| Authenticated HTTP-to-database runtime proof | complete | `TestAuthenticatedPartialSubmissionHTTPPostgresIntegration` returned `200`, `200`, and `403`, asserted canonical writes and cleanup, and left zero namespaced pilot fixtures. |
| Evidence reconciliation | complete | Ledger and dedicated runtime evidence document updated with proven limits and remaining gates. |
| Real identity-provider proof, non-owner read-isolation proof, and Flutter Matrix Fixture Provisioning | pending | These are independent gates and must not be inferred from the completed local HTTP proof. |

### Errors Encountered

| Error | Resolution |
|---|---|
| Initial runtime test created an assignment-scope row without required tenant/organization columns. | Added the tenant-scoped columns and values to the fixture insert. |
| Initial cross-organization request surfaced as `400` from a hidden `sql.ErrNoRows`. | Mapped tenant-hidden records to the same non-disclosing `403 authorization_failed` response as explicit authorization denial. |
| Initial cleanup loop passed two arguments to one-parameter SQL statements. | Replaced it with a statement/argument dependency-order table. |
| Early failed test runs left only namespaced test artifacts. | Inventoried, dependency-cleaned, and then added explicit in-test absence assertions; final inventory was zero work orders, inspections, and identity subjects. |

### Next Step

The controlled non-owner canonical inspection read-isolation, controlled OIDC validator-composition, Field Matrix Fixture Provisioning, and disposable Keycloak provider proofs are complete. The Field proof ran against the existing loopback-only acceptance service with deterministic receipt-driven cleanup. The Keycloak proof used a disposable generated realm/client/user and verified discovery, JWKS, signed-token validation, derived local authority, `401` tamper rejection, `200` submission, and provider/pilot cleanup. Remaining identity work is production-hosted Keycloak deployment and operations, which is deliberately out of scope until a production environment is introduced.

## Repository Hygiene — 2026-08-21

Selective cleanup completed before the next roadmap milestone. Removed only reviewed generated Flutter build and Python cache directories; retained `.dart_tool`, the active Python verification environment, IDE preferences, source, migrations, evidence, backups, and planning records. `git diff --check`, `go test ./...`, and `go vet ./...` passed. The next action is to resume the dependency-ordered post-Stage 0 roadmap inventory.

## Stage A Recovery Drill — 2026-08-21

Completed the selected Stage A operational-trust gate. Added S3 metadata retrieval, a manifest-driven recovery command, and a current-schema generated-fixture drill. The successful run created a fresh pilot dump and evidence-object backup, restored into separate loopback-only PostgreSQL and RustFS containers, verified relational/audit graph restoration, RLS matching/mismatch behavior, ciphertext and metadata restoration, source cleanup, and target disposal. `go test ./...` and `go vet ./...` passed. Next roadmap work should select the remaining Stage A runtime-operations or evidence-retention gate without reusing this proof as a production DR claim.

## Stage A Evidence Retention and Verified Export — 2026-08-21

### Status

Complete. The authoritative metadata registry, verified projection, and controlled pilot proof are complete. The next independent Stage A work is runtime operations: health and readiness endpoints, correlation identifiers, structured logs, and request limits.

### Evidence

The pilot migrations `0010_evidence_metadata.candidate.sql` and `0011_evidence_metadata_encryption_export.candidate.sql`, along with their rollback candidates, were backed up, applied to disposable restored copies, reviewed, and only then applied to the pilot after fresh verified backups. Metadata registration is tenant/organization scoped, immutable on replay, and contains explicit encryption provenance required by the canonical manifest. The projection re-reads each storage object, validates object identity, content type, byte count, and SHA-256 before sealing. `TestPostgresRustFSVerifiedExportIntegration` passed with same-tenant sealing, direct non-owner RLS `1/0` organization checks, missing-object rejection, same-length digest-mismatch rejection, and cleanup. The repository originally relied on forced RLS alone for list queries; explicit actor-derived predicates were added after the privileged local owner test role demonstrated that runtime test identities must not be treated as an isolation boundary. Full `go test ./...` and `go vet ./...` passed.

The first combined final-regression command routed concurrent Go output through a PowerShell pipeline and ended with an ambiguous failure status even though its controlled export test passed. The direct unpiped reruns then passed `go test -count=1 ./...` and `go vet ./...`. A subsequent root-directory hygiene review removed exactly six 106–118-byte accidental checksum-marker files created by earlier PowerShell output handling; no dump or source file was removed, and five verified `0010`/`0011` pre-apply backups remain under `operations\acceptance\backups`.

### Next Step

Define and implement the Stage A runtime-operations gate without coupling it to the evidence registry: operational health/readiness semantics, correlation-ID propagation, structured redacted logging, and bounded request-size/timeout behavior, each with focused negative and runtime evidence.

## Stage A Runtime Operations — 2026-08-21

### Status

Complete. The bounded operational HTTP gate is now independently evidenced; it does not expand any Work-Order, identity, evidence, certificate, or export authority.

### Evidence

`internal/server` now provides dependency-free liveness, deadline-bounded readiness, cache suppression on operational replies, safe caller-or-generated correlation IDs, no-query structured request logs, and deterministic `413` rejection for known oversized request bodies while retaining streaming limits. Focused tests covered success and dependency/readiness deadline failure, correlation behavior, log redaction, known and streaming limits. A disposable loopback candidate on `127.0.0.1:18181` returned health/readiness `200/200`, echoed a supplied correlation ID, redacted a query token in captured logs, returned `413` before sync dispatch for an oversized body, and removed its temporary binary on stop. Direct unpiped `go test -count=1 ./...` and `go vet ./...` passed. The combined regression/hygiene invocations were separately stopped after all useful Go validation completed because Git’s non-fatal CRLF-warning stream stalled the connected terminal; a prior direct `git diff --check` had passed after the preceding evidence milestone.

### Next Step

Select the next dependency-ready server-authoritative milestone. Candidate work should be evaluated against the verified foundations rather than bundled into runtime operations: evidence metadata ingress composition, audited export approval persistence, certificate issuance workflow, or another bounded Stage A operational capability.

## Stage A Authenticated Evidence Metadata Registration — 2026-08-21

### Status

Complete. The server now has a bounded metadata-registration ingress that can bind an already-stored object to an assigned inspector’s canonical inspection without accepting scope or actor authority from its request body.

### Evidence

Added `internal/evidenceregistration`, `internal/evidencehttp`, atomic `evidencepg.RegisterForActiveAssignment`, optional server composition, server entry-point wiring, and OpenAPI documentation. The local PostgreSQL/RustFS proof passed derived-scope registration, unknown authority-field rejection before mutation, immutable duplicate replay, missing-object conflict, cross-organization assignment denial, dependency-ordered cleanup, and zero namespaced metadata residue. The full direct Go test and vet gates passed. The legacy `/evidence` object-upload route remains a separate compatibility boundary and was neither promoted nor relied upon as authoritative metadata registration.

### Next Step

The next dependency-ready authority boundary is either audited export approval persistence—so verified manifests can become an accountable release artifact—or the broader certificate issuance lifecycle. Neither should be conflated with registration, storage, or manifest verification.

## Certificate versus Export Sequence Review — 2026-08-21

The bounded critical-boundary review is complete. It recommends **certificate issuance first** in the normal INTEGIN lifecycle: Work Order, canonical inspection, verified evidence, review/approval, certificate issuance, QR/client verification, then optional export approval and external transmission. Export approval is not removed; it remains a separate release ledger for a sealed manifest and is required before any later external disclosure. A future external authority adapter may require a pre-recognition evidence export before that authority accepts a certificate, but this is an adapter-specific exception and does not reverse the general local certificate-first implementation order. No certificate, export approval, release endpoint, or external submission was created by the review. Owner confirmation is required before implementing either authority mutation.

## High-Level Operating Model Review — 2026-08-21

Completed the requested higher-level review across field execution, inspection facts, evidence, certification, verifier access, disclosure, external recognition, commercial handoff, renewal, and governance. The model is intentionally multi-track: Work Order and field execution establish scope; evidence establishes support; certificate issuance represents the official conclusion; QR verification exposes a safe public projection; export approval controls evidence disclosure; office closeout supports commercial handoff; and external recognition remains an optional adapter policy. The normal sequence is certificate-first, followed by QR/client verification, office/commercial closeout, renewal operations, optional evidence export approval, and only then external transmission where configured. The detailed review also defines senior self-issue, independent review, external-recognition, evidence/no-photo, offline onboarding, multi-inspector, and correction/supersession exception policies. It records that neither a certificate nor an export grants the other authority. Next implementation remains blocked only for explicit owner policy confirmation of the certificate approval profile; no release authority was created by this review.

## Attached Project-Inspection Conversation Audit — 2026-08-21

Reviewed the user-supplied separate-agent conversation as historical engineering input, not source of truth. Its product operating-model observations are already reconciled in the high-volume and integrated operating-model records. Its generic Go/PostgreSQL advice supports the present modular Go/PostgreSQL/RustFS direction but does not justify new infrastructure. Direct current-source checks confirmed that the old sync held/replay, pre-auth receipt lookup, unknown-signature-algorithm, and non-atomic held-persistence concerns remain corrective candidates; the Work-Order custom PostgreSQL-array helper remains without special-character proof. The attachment also correctly identified a missing binding-key registry as a prerequisite for any certificate designer. The adjusted roadmap tracks three separate future slices: offline-sync durability correction, certificate-template binding registry, then certificate authority lifecycle. No authority mutation was implemented by this audit.

## Offline-Sync Durability Correction — 2026-08-21

Complete. The processor now authenticates before receipt lookup, accepts only exact supported algorithms, treats matching held receipts as resumable, and prevents in-memory held state from getting ahead of durable storage. The PostgreSQL adapter now atomically persists held receipt/envelope state and atomically promotes a held receipt to applied while advancing the sequence and deleting the envelope. Focused tests and the controlled pilot PostgreSQL proof passed; the proof also injected a held-row failure and verified no partial durable state. The first integration harness cleanup used a closed database handle and parameterized multi-statement execution; both test-only defects were corrected, the test was rerun, and the post-run `it-sync-durability-*` inventory was zero. Full `go test -count=1 ./...` and `go vet ./...` passed. The next work is the independent certificate-template binding registry, not certificate issuance itself.

## Certificate-Template Binding Registry — 2026-08-21

### Status

Complete. The pre-certificate registry is now a separately evidenced, tenant/organization-scoped foundation. It stores immutable template definitions and bounded layout cells, restricts binding to the approved `inspection_record` catalog, resolves only approved templates against canonical inspection facts, and fails closed when a required value cannot be rendered within its declared bounds.

### Evidence

Migration `0012_certificate_template_binding_registry.candidate.sql` was previously backed up, reviewed on a disposable copy, and applied to the pilot with forced RLS. Focused formatting and package validation passed with `go test ./internal/domain/certificatetemplate ./internal/certtemplatepg`. The controlled `TestPostgresResolveCanonicalInspectionTemplateIntegration` passed against the restored pilot PostgreSQL service, proving canonical inspection-value resolution and cross-organization denial. A post-run inventory verified zero `it-certificate-template-resolver-*` records across `certificate_template` and `inspection_record`. Direct unpiped `go test -count=1 ./...` and `go vet ./...` then passed.

### Limitation and Next Step

The current catalog deliberately exposes only the nine canonical `inspection_record` fields. It does not yet resolve asset details, findings, conditional sections, photos, repeating regions, certificate identifiers, signatures, expiry, revocation, or public QR projections. The next authority boundary is the certificate lifecycle: canonical eligibility, review/self-issue policy enforcement, immutable template snapshotting, certificate numbering/status/expiry, revocation, supersession, and a QR-safe public projection. No certificate authority mutation has yet been implemented or proven by this registry gate.

### Record-Recovery Note

The parent checklist `C:\MY PROJECT\todo.md` was unintentionally emptied during a mounted-filesystem transport interruption while this result was being recorded. No source, migration, backup, container, or database state was affected. The checklist was reconstructed from the durable task plan, progress, and findings records, and the empty pre-recovery file was retained as `todo.md.empty-recovery-20260821-232800.bak`.

## Certificate Authority Lifecycle Design — 2026-08-21

The authority-safe design and acceptance matrix are complete in `docs\architecture\INTEGIN_CERTIFICATE_AUTHORITY_LIFECYCLE_DESIGN_2026-08-21.md`. It adopts independent review by default and a server-derived, policy-scoped senior self-issue exception; requires an approved/finalized canonical inspection revision, approved template, immutable issuance snapshot, per-scope number allocation, template/inspection-type validity policy, truthful revocation/supersession, and a QR-safe projection. The design expressly excludes unsupported asset/test/evidence claims and creates no certificate mutation, schema application, renderer, public endpoint, or external recognition action. Next work is the controlled lifecycle migration and persistence implementation.

## Certificate Lifecycle Draft-Persistence Foundation — 2026-08-21

Implemented the first authority-safe persistence slice. Candidate migration `0013_certificate_authority_lifecycle` and its rollback passed an isolated restored-copy up/down review with five created/forced-RLS tables, then were applied to the pilot after a fresh verified pre-apply dump. `internal/domain/certificateauthority` now enforces canonical approved/finalized inspection eligibility, independent review by default, a policy/capability/reason-bound senior self-issue exception, and controlled in-memory transitions. `internal/certificatepg` owns one transaction for canonical inspection/policy/template reads, server-derived scope, immutable draft persistence, and `DRAFT_CREATED` audit insertion. `TestPostgresCreateCertificateDraftIntegration` passed same-scope draft creation, persisted audit, cross-organization denial, and zero namespaced residue. Full `go test -count=1 ./...` and `go vet ./...` passed before the current in-progress issue adapter work.

The source now also contains compiling review/sign transition helpers and a compiling issuance coordinator with approved template/canonical inspection snapshot preparation, scoped number allocation, and hashed random public token generation. These later transitions have not yet received controlled integration coverage; expiry, revocation, supersession, public verifier projection, HTTP/OIDC composition, document rendering, and external recognition remain unproven and must not be claimed.

## Certificate Issuance and QR-Safe Projection Proof — 2026-08-21

The controlled pilot lifecycle test now proves the independent path from `DRAFT` through `PENDING_REVIEW`, `APPROVED`, `SIGNED`, and `ISSUED`; an approved template and policy are re-read and locked at issuance; a per-scope number is allocated transactionally; immutable template/canonical inspection snapshots plus SHA-256 digest are persisted; only a SHA-256 digest of an unguessable 32-byte public token is stored; duplicate issuance is rejected; correct-token public projection is narrow; wrong-token lookup returns no projection; and revocation produces a truthful public `REVOKED` state. The post-test namespaced lifecycle residue count is zero and final repository regression/vet passed.

Still unproven: PostgreSQL senior-self-issue path, expiry, supersession (method exists but no controlled proof), externally callable HTTP/OIDC transport, renderer/PDF and QR image generation, public portal HTTP headers/rate limits, full asset serial/description/test-scope projection, and external recognition.

## Complete Certificate Lifecycle Persistence Proof — 2026-08-21

The server-authoritative certificate lifecycle foundation is now proven at PostgreSQL repository level. `TestPostgresCreateCertificateDraftIntegration` covers the independent default path; governed senior self-issue under approved policy/capability/reason; policy/template eligibility; number allocation; immutable snapshots; hashed public tokens; correct and wrong token projections; revocation; supersession to a replacement on later inspection revision for the same asset; expiry; repeat-action rejection; cross-organization draft denial; dependency-ordered cleanup; and zero residue. Final repository `go test -count=1 ./...` and `go vet ./...` passed.

The next certificate boundary is transport and presentation, not lifecycle authority: authenticated mutation HTTP/OIDC composition; a no-store public verifier endpoint with rate/abuse controls; PDF/QR rendering; and richer public fields only after authoritative asset/test sources exist. No external recognition, renderer, public portal, or HTTP route is proven by the repository proof.

## Certificate HTTP Transport Foundation — 2026-08-21

Created `docs/architecture/INTEGIN_CERTIFICATE_TRANSPORT_DECISION_2026-08-21.md`, `internal/certificatehttp`, and membership-derived actor mapping. The handler currently proves no-store replies, bearer-token validation boundary, local actor derivation interface, unknown-field rejection, and server-derived actor propagation for draft creation, submit, review, and sign operations. Full repository regression and vet passed. It is not yet composed into `cmd/integin-server`; issue/revoke/supersede/expire routes, public verifier HTTP transport, live OIDC/membership runtime proof, rate/abuse control, and document rendering remain separate work.

## Certificate Mutation HTTP Source Proof — 2026-08-22

internal/certificatehttp now carries the source-level routes for draft, submit, review, sign, issue, revoke, supersede, and expire. The handler validates injected OIDC authentication, derives scope/capabilities through injected local membership, rejects authority-shaped/unknown draft fields, uses no-store replies, and emits the raw public token only on a successful issue response. Focus tests and full Go regression/vet passed. It is not yet mounted in cmd/integin-server or proven against a live OIDC/membership runtime; public verifier HTTP remains unimplemented.

## Public Certificate Verifier Source Proof — 2026-08-22

internal/certificatepublichttp implements GET /verify/certificates/{token} as a source-level handler over the digest-backed certificatepg.VerifyPublic projection. It accepts bounded URL-safe tokens, sends no-store replies, makes invalid and unknown tokens both 404, returns only certificate number/status/issued/expiry/asset ID, and rate-limits by normalized remote address before lookup. Focused tests and full regression/vet passed. It is not yet mounted in the server, externally rate-limited, or live-runtime exercised.

## Certificate Runtime Composition and Negative Probe — 2026-08-22

The server mux now has optional /certificates/ and /verify/certificates/ mounts, documented in OpenAPI, with main composition creating the public handler only with a database repository and the mutation handler only inside the enabled OIDC branch after validator and local identity resolver construction. Mux-route tests, full regression, and vet passed. A disposable local candidate with OIDC disabled and a disposable tenant scope returned health 200; malformed and unknown public verification tokens both 404 no-store; public POST 405 no-store; and the mutation path 404 because OIDC-disabled composition intentionally left it absent. Candidate cleanup succeeded and pilot counts for the disposable tenant were zero.

This does not prove an issued public certificate response, 429 rate-limit behavior in a live candidate, shared/deployment-grade rate control, OIDC-enabled valid or invalid mutation requests, local membership authorization in runtime, or production external availability.

## Pending Enabled-OIDC Certificate Runtime Proof

Reuse the local RS256 discovery/JWKS issuer and membership fixture from internal/workorderhttp/http_oidc_postgres_integration_test.go. The certificate proof must create a unique it-certificate-http-* membership, canonical approved inspection/template/policy fixture, and a runtime mux containing certificatehttp.Handler backed by certificatepg.Repository plus identity.PostgresResolver. It must show: tampered token 401; unknown membership 403; valid signed token with certificate.prepare creates only the server-derived tenant/organization draft; missing capability is rejected; fixture cleanup leaves zero rows in certificate, identity, inspection, and work-order tables. A separate issued-token runtime check remains required for public 200 projection and live 429 behavior.

## Controlled Signed-OIDC Certificate Mutation Proof — 2026-08-22

TestSignedOIDCCertificateTransportUsesLocalMembership passed against pilot PostgreSQL. Through server.NewMux and the actual OIDC validator/local PostgreSQL resolver, it proved tampered RS256 token rejection (401), signed unknown-subject rejection (403), signed local membership with certificate.prepare reaching the lifecycle with server-derived tenant, organization, actor, and capability values (204), and a signed local membership with no certificate capabilities rejected before lifecycle execution (403). The deterministic, namespaced identity fixtures cleaned up; a check confirmed zero subject-it-certificate-http-* and denied-it-certificate-http-* records. Full go test -count=1 ./... and go vet ./... then passed.

## Certificate public transport proof closure
2026-08-22: Certificate public transport composition proof completed. Added TestPublicVerifierComposesThroughServerMux: a valid bounded projection is served as 200 no-store through server.NewMux and the immediate second request is 429 no-store with only one verifier lookup. Actual issued-token digest resolution remains proven in certificatepg integration. Full go test -count=1 ./... and go vet ./... passed; queried lifecycle/OIDC residue was 0|0|0. The first cross-layer DB test attempt created a Go import cycle and was removed cleanly before final regression. Shared rate control, external release, QR/PDF output, and canonical serial/description/type/test-scope projection remain unproven.
Next: plan separately for a canonical equipment-public-binding and renderer/QR milestone; do not claim production public release.

## Next certificate verification milestone design
2026-08-22: Completed follow-on certificate-verification design inventory. Recorded canonical data gaps (serial and description absent; asset type only in work-order scope; inspection type not persisted in inspection_record; no canonical public test-scope summary), drafted the tenant-safe immutable public-binding contract, documented PDF/QR prerequisites, and published the gated eight-step implementation sequence. No schema, route, renderer, QR artifact, public release, or external integration was added.
Next: begin with canonical asset and inspection-public-scope model design; do not expand public projection first.

## 0014 certificate public-binding slice
2026-08-22: Completed 0014 canonical public-binding slice. Disposable up/down review passed; fresh pre-apply dump saved at tmp\integin-pilot-pre-0014-20260822-094737.dump; 0014 applied to pilot. Issuance now snapshots required policy-approved canonical asset and inspection public-scope values and includes them in the snapshot digest. Integration proved snapshot persists serial/description/type/inspection scope and survives later mutable asset changes. Namespaced residue was 0|0|0|0|0; full go test -count=1 ./... and go vet ./... passed. Public HTTP projection, QR/PDF output, public deployment, and shared rate/abuse controls remain unimplemented.
Next: separately design and prove a policy allow-listed public-response expansion from immutable snapshot data before any QR/PDF renderer.

## Immutable public response expansion
2026-08-22: Completed immutable snapshot-backed public verifier expansion. VerifyPublic joins only certificate_snapshot.public_binding_snapshot, and the handler conditionally allow-lists serial, description, type, inspection type, and structured test scope while preserving the original public fields, no-store, 404 indistinguishability, and limiter behavior. Focused tests, full go test -count=1 ./..., go vet ./..., and zero fixture residue (0|0|0|0) passed. QR/PDF rendering, shared rate/abuse control, public release, and external integrations remain unimplemented.
Next: plan fixed-cell certificate rendering and QR generation from issuance snapshots, with artifact storage and release controls.

## Certificate renderer capability
2026-08-22: Implemented internal/certificaterender fixed-cell PDF and high-recovery QR capability with go-pdf/fpdf v0.9.0 and skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e. Renderer validates template geometry and bounded text policies, rejects insecure QR origins/overflow, returns byte digest and non-sensitive metadata, and uploads through storage.Store. Focused renderer tests plus full go test -count=1 ./... and go vet ./... passed. It is not yet connected to actual issuance snapshots/tokens, PostgreSQL artifact metadata, authenticated download, public release, or shared rate/abuse controls.
Next: integrate renderer only through an authenticated issuance artifact workflow with PostgreSQL metadata and controlled RustFS evidence registration; retain public JSON-only route.

## Go 1.26.5 upgrade
2026-08-22: Owner-authorized Go upgrade completed. The active toolchain is go1.26.5 windows/amd64 and go.mod now declares go 1.26.5. go mod tidy, focused certificate/renderer tests, full go test -count=1 ./..., and go vet ./... passed. The Codeberg fpdf v0.12.0 release line is official and currently active, but the project remains on frozen github.com/go-pdf/fpdf v0.9.0 until the owner explicitly approves a module-path switch; the QR dependency is likewise frozen.
Next: obtain owner decision to switch the frozen PDF dependency to codeberg.org/go-pdf/fpdf or remove the renderer prototype.

## Technology update audit
2026-08-22: Completed technology update audit excluding frozen PDF/QR dependencies. Current: Go 1.26.5, PostgreSQL 18.6, Keycloak 26.7.1, Docker 29.7.2, JWT v5.3.1. Recommended separately controlled updates: OpenBao 2.6.0 to 2.6.2 due upstream security fixes; lib/pq 1.10.9 to 1.12.3 with integration proof; RustFS rc.1 to newer rc only after backup/recovery/S3 compatibility rehearsal. PostgreSQL uses a broad postgres:18 tag and should later be exact minor/digest pinned. No version, image, data, config, or frozen dependency was changed by this audit.

## Performance baseline
2026-08-22: Performance baseline complete; see INTEGIN_PERFORMANCE_BASELINE_RESULTS_2026-08-22.md. No evidence supports domain/RLS/certificate/sync optimization.

## Bottleneck comparison
2026-08-22: Repeated five-sample local-pilot bottleneck comparison complete. Certificate lifecycle with real DB verifier checks: mean 1298.05 ms, range 1157.77-1415.80 ms. RustFS signed contract: mean 864.50 ms, range 794.65-942.51 ms. Sync recovery/atomicity: mean 922.52 ms, range 821.47-1203.38 ms. These are full test-path timings including Go setup, not endpoint latency. No reproducible protected-logic bottleneck was identified; DB-backed HTTP verifier remains unisolated. Namespaced residue zero; full regression/vet passed.

## Isolated HTTP public-verifier measurement
2026-08-22: Added cleanup-backed real PostgreSQL HTTP public-verifier fixture. Five independent 20-request samples: request mean-of-means 3.113 ms, mean range 2.463-4.625 ms, worst sample p95 10.043 ms. All responses 200 no-store. Discovered shared request logger exposed path token; fixed by logging /verify/certificates/:token and added regression test. Fixture residue zero; full regression/vet passed.

## Renderer Qualification Continuation - 2026-08-22
Status: partial synthetic qualification evidence only; fixed cells, flowing reports, bidi punctuation, font preflight, restricted-input checks, and local bounds were exercised without project integration. Next: prove environment-level egress denial, consolidate the evidence pack, and seek owner approval before selecting any renderer or font pack.

Validation note: git diff --check was stopped after a terminal timeout while emitting pre-existing CRLF conversion warnings for unrelated files. It did not report a whitespace error before termination and did not change project state.

## Wayfinder map - 2026-08-22
Created a non-implementation decision map. Initial frontier: D-01, owner-approved reviewed preservation of the current working tree. D-02 legacy evidence-ingress posture, D-03 multilingual renderer selection, and D-05 schema audit remain separately gated decisions.
