# INTEGIN Phase 1 Findings

## Workspace

The attached specification defines a six-phase INTEGIN implementation. The user explicitly requested Phase 1. The connected Windows project directory is `C:\MY PROJECT`, mounted in the sandbox as `/mnt/desktop/MY PROJECT`, and was empty before initialization.

## Phase 1 Requirements

The shared kernel must provide:

- `integin_domain_events`: immutable domain event definitions carrying `tenant_id`.
- `integin_shared_types`: enums, error codes, tenant context, response types, and calibration status.
- Notification category definitions.
- Feature flag definitions.
- Calibration record schema.
- Event serialization/deserialization tests.

## Design Constraints

Business logic must remain outside the shared contracts. Events represent immutable facts and must include enough metadata for tenant isolation, correlation, causation, actor attribution, and schema versioning. The implementation will avoid third-party Go dependencies in Phase 1.

## Source Basis

The implementation is based on the user-provided attachments `INTEGINPARTI.md` through `INTEGINPARTV.md` and `integin-build-instruction-prompt.md`, especially Part II Shared Kernel sections, Part IV Phase 1 deliverables, and Part V event catalog.

## Revision Session

The connected desktop terminal sidecar disconnected while verifying the newly installed Go toolchain. I will continue editing the mounted project files directly and validate from a sandbox copy; the desktop session may need reconnecting before the user can run commands locally.

## Phase 2 Handoff — Core Engines

Phase 1 shared contracts are ready for consumption by deterministic domain engines. The first Phase 2 package should be `internal/domain/inspection`, using the shared event envelope and shared enums without mutating them. The inspection engine must support the documented lifecycle, discipline-neutral findings, severity-based verdict computation, immutable revision history, and returned domain events for state changes.

The remaining Phase 2 packages are template composition and expression evaluation, certificate lifecycle with separation-of-duties enforcement, and deterministic data completeness with an advisory-only AI hook. Acceptance tests must remain dependency-free and must prove that prior inspection revisions and frozen template snapshots are not rewritten.

## Deployment Integration — Dual-Service Review

The connected desktop project root did not expose a separate UI or Python service tree during the direct inspection, so the next implementation will add both components. A sandbox mounted-path glob search was rejected by the environment; direct desktop inspection and project-file operations remain the supported path.

The Windows Python environment does not currently have `pytest` installed, so `python -m pytest ai_service -q` cannot run there until the optional Python dependencies are installed. The Python service remains testable through its documented `requirements.txt` setup; the failed lookup for a root `README.md` was an incorrect path assumption, not a project error.

## Stage 0 Work-Order Runtime Findings — 2026-08-21

The Work-Order repository is the SQL transaction owner. Its explicit `WithinTransaction` adapter invokes the service callback with the same repository so application composition cannot create a false nested-transaction boundary. This preserves the established atomic mutation behavior for tenant context, revision checks, canonical inspection validation, normalized submission writes, and state transitions.

The completed controlled HTTP proof reached the server mux, token-validation boundary, production PostgreSQL identity resolver, server-derived actor projection, role/capability authorizer, Work-Order service, canonical inspection validator, and PostgreSQL persistence. The proof accepted two completed open canonical inspection records, persisted normalized submission items, submitted both inspections, returned the same receipt on idempotent replay, and returned non-disclosing `403` to a resolved inspector from another organization.

The deterministic validator double is an intentional local-boundary test seam. It does not exercise discovery, JWKS retrieval, signature verification, issuer configuration, or a live external OIDC identity provider. Those remain a separate runtime gate.

## Non-Owner Read-Isolation Finding — 2026-08-21

The applied pilot database supports a non-owner canonical inspection read-isolation proof without granting privileges or changing policies. `integin_pilot_runtime` is non-superuser, does not bypass RLS, and has the required `SELECT` capability. Under `SET LOCAL ROLE`, forced RLS returned one expected record per matching organization and zero records when tenant/organization settings were unset. All temporary graphs were removed and post-run counts were zero.

## Controlled OIDC Validator-Composition Finding — 2026-08-21

The production validator successfully completed loopback discovery and JWKS retrieval, validated a generated RS256 bearer token, rejected both a tampered signature and expiry, and passed only issuer/subject identity to the local PostgreSQL membership resolver. The resulting Work-Order path was database-backed and cleaned its generated fixtures. This is stronger than a validator double but remains distinct from a real external identity-provider deployment.

## Field Matrix Toolchain Finding — 2026-08-21

The Windows Flutter batch launcher silently exits in the connected sidecar, but the SDK itself is usable through its bundled Dart runtime and `flutter_tools.snapshot`. The Field live provisioning test runs with a process-only standard `PROGRAMFILES(X86)` value and safely skips because all required `INTEGIN_LIVE_*` endpoints are absent. No runtime fixture traffic was sent and no workstation configuration was changed.

## Field Matrix Fixture Provisioning Runtime Finding — 2026-08-21

The existing loopback acceptance server already supplied the controlled Field endpoints. A Flutter test run through the bundled snapshot provisioned a generated device, applied signed sync, returned `APPLIED` then `DUPLICATE` for the same evidence, and passed. A receipt was written before provisioning; its cleanup harness removed only receipt-named database and object-store artifacts, verified their absence, and removed the receipt. The first harness verification had an argument-binding defect, which was repaired and followed by a complete successful rerun.

## Disposable Keycloak Provider Finding — 2026-08-21

A genuine Keycloak `26.7.1` access token passed through production discovery, JWKS, RS256 validation, issuer/audience/authorized-party checking, local identity resolution, and Work-Order canonical persistence. A tampered version of the same token received `401`. The temporary realm, client, user, container, and every `it-keycloak-*` pilot fixture were removed. The only remaining Keycloak work is hosted production deployment and operational governance, not protocol integration.

## Repository Hygiene Finding — 2026-08-21

The substantial safely removable workspace output was Flutter build output (about 102 MB) plus standard Python test/interpreter caches. `.dart_tool` and the local verification virtual environment are also reproducible but were retained to preserve active test readiness. The cleanup guard stopped on `.pytest_cache` until its standard cache structure was confirmed, then completed only the reviewed removals. Protected evidence, source, migrations, backups, and planning records remained intact.

## Stage A Recovery Drill Inventory — 2026-08-21

The active pilot services are `integin-pilot-postgres` on loopback port `15432` and `integin-pilot-rustfs` on loopback port `19000`; separate active non-pilot services also exist on `5432` and `9000`. The recovery drill must never restore over any of these active services. Two verified pilot PostgreSQL custom-format backups are present: `integin-pilot-pre-0008-20260821-094050.dump` and `integin-pilot-pre-0009-20260821-104416.dump`.

The runbook requires both a PostgreSQL backup/restore drill and an object-plus-manifest RustFS restore drill. The verification must preserve tenant ownership, evidence IDs, ciphertext bytes, digest values, inspection relationships, and audit records. An isolated destination container pair with fresh volumes and no published non-loopback port must be used.

The current evidence handler writes ciphertext objects and metadata to the storage boundary but does not itself persist an evidence row, inspection relationship, or audit row in PostgreSQL. The restore drill must therefore state two separate proofs: object-plus-metadata manifest restoration through RustFS, and relational/audit restoration from the PostgreSQL backup. It must not falsely claim a database evidence-relationship round trip from the handler alone.

The S3 storage adapter currently sends `X-Amz-Meta-*` headers on `Put`, but its `Get` method returns an empty metadata map rather than parsing response metadata headers. A restore drill can preserve metadata by using a manifest captured at backup time, but cannot independently verify restored metadata through the present storage abstraction. The next bounded prerequisite is metadata-read support plus unit coverage before any claim of a full object-plus-metadata restore proof.

The dated acceptance recovery archive was created on 2026-08-15 and contains the base device/authority/sync schema only. It predates the Stage 0 Work-Order and canonical inspection migrations, so it is useful as historical recovery evidence but cannot prove current Stage 0 restoration. The new drill must seed a generated, namespaced Stage 0 relational graph and evidence object in the pilot, capture a fresh PostgreSQL dump and RustFS snapshot, remove the source fixture, restore into fresh targets, and verify the snapshot alone retains the graph/object/metadata contract.

## Stage A Current-Schema Recovery Finding — 2026-08-21

The controlled recovery drill passed after fresh fixture seeding and backup. PostgreSQL restoration preserved the generated Work-Order containment graph and audit state event; forced RLS remained effective for a non-superuser runtime role after restore. The object proof restored ciphertext and all expected metadata through the repaired S3 adapter. The original source fixture/object, destination containers, and destination volumes were removed. The historical recovery archive remains distinct because it predates the current schema.

## Remaining Stage A Capability Inventory — 2026-08-21

`internal/exportmanifest` provides a well-validated, deterministic, checksum-sealed evidence-export manifest contract, but it is intentionally pure: it has no database, storage, HTTP, workflow, or authority dependency. It cannot independently enumerate authoritative evidence, check current object bytes, or persist an approved export record. The current `internal/monitoring` package is an in-memory advisory-trace component and is unrelated to runtime health/readiness or evidence retention.

The next dependency-complete Stage A requirement is therefore an authoritative evidence-metadata and export-projection boundary: metadata must be durably tenant-scoped, bind an inspection and ciphertext object to both digests, be reconciled against storage on export, and then feed the already-tested export-manifest contract. Runtime-operations readiness remains a separate later gate.

The selected evidence-retention slice will keep bytes in RustFS and introduce a forced-RLS relational `evidence_metadata` index attached by composite containment to canonical inspections. The existing export-manifest package remains a pure sealed projection. The existing evidence upload handler is not sufficient authority for metadata registration because it accepts tenant and organization in the payload and does not persist a relational record; the new application boundary must receive server-derived scope from a validated mutation path.

## Stage A Evidence Retention and Verified Export Finding — 2026-08-21

The implemented `evidence_metadata` index binds a canonical inspection, tenant/organization scope, immutable contained object key, byte count, plaintext/ciphertext digests, authority/signature provenance, explicit encryption algorithm/key reference, and privacy/retention fields. The additive encryption columns were necessary because `exportmanifest.EvidenceRecord` requires those values; representing them as an assumed default or borrowing signature metadata would have been a false claim. The `0010` and `0011` candidates and rollback candidates passed isolated restored-copy review before pilot application with retained immediate pre-apply backups.

The verified projection treats the relational index as a selector, not as proof of current bytes. It retrieves every candidate object from RustFS again and rejects object-key, content-type, byte-count, or ciphertext SHA-256 discrepancies before calling `Seal`. A single manifest is deliberately rejected when its evidence records have incompatible privacy/retention/hold/redaction policy sets; this prevents collapsing per-record policy into a false global export claim.

The first integration run exposed a meaningful test-boundary issue: the pilot owner role is superuser and can bypass RLS, so an unfiltered repository list query observed a cross-organization row even though the table is forced-RLS. The repository now has explicit actor-derived tenant/organization predicates as defense in depth, and the controlled proof separately assumes non-owner `integin_pilot_runtime` to demonstrate actual forced-RLS behavior of `1` matching row and `0` mismatched rows. This does not replace a future application-role connection policy; it prevents the privileged test connection from silently weakening the repository boundary.

## Stage A Runtime Operations Finding — 2026-08-21

The server already had transport-level timeouts and preliminary middleware. The completed gate made the policy explicit and testable: liveness has no dependency probe; readiness has a bounded dependency context and non-disclosing `503`; correlations must be safe bounded identifiers; logs use a path without query data; and known `Content-Length` excess is rejected as `413` before route handling. `http.MaxBytesReader` remains the streaming protection mechanism, so handlers that read bodies must surface its read error rather than treat it as a domain error.

The live proof used only a generated loopback candidate with disabled evidence storage, not the active pilot server. It shows operational boundary behavior, not production observability, downstream service availability, distributed tracing, load behavior, or a readiness statement for all deployment environments.

## Stage A Authenticated Evidence Registration Finding — 2026-08-21

The legacy evidence upload handler is a storage compatibility surface, not an authority-safe registration surface, because it receives tenant and organization fields. The new route establishes an explicit separation: authenticated membership derives scope; the application service derives the contained key and registration owner; the object store is re-read; and the PostgreSQL adapter confirms assignment scope in the same RLS-scoped transaction as immutable insertion. This avoids both client scope claims and a check-then-insert assignment race.

The route retains client-supplied device, authority, signature, encryption, classification, and retention fields as metadata claims. It verifies object bytes but does not yet verify an offline authority package or signature binding; recording such claims must not be misrepresented as signature validation. The successful proof is limited to online OIDC/membership-derived registration.

## Certificate versus Export Sequence Finding — 2026-08-21

The normal business artifact and the disclosure artifact are separate. Certificate-first controls the official client/site-facing inspection outcome after review; export approval controls the later disclosure of a defined sealed evidence package to a recipient for a purpose. The correct general lifecycle is certificate-first, followed by optional export approval. A future external authority adapter may require export approval before it transmits or recognizes a package, but that external submission requirement is a policy of the adapter and does not turn export approval into a universal certificate prerequisite.

## High-Level Operating Model Source Findings — 2026-08-21

The ISO public overview for withdrawn ISO/IEC 17020:2012 states that the standard addresses competence of inspection bodies and the impartiality and consistency of inspection activities; ISO identifies ISO/IEC 17020:2026 as its replacement. This supports treating reviewer/signatory separation as a configurable impartiality safeguard, not as a universal assertion that every inspection must use two different people. The detailed current standard and any accreditation requirements have not been assessed. Source: https://www.iso.org/standard/52994.html.

ZATCA’s official e-invoicing overview distinguishes structured electronic invoices from scanned paper copies and describes a generation phase plus an integration phase in which the electronic solution connects with ZATCA systems and generates invoices in the required format. This supports keeping commercial invoicing as a distinct future integration/handoff track after operational completion; it does not determine the owner’s tax applicability, taxpayer wave, or INTEGIN compliance. Source: https://zatca.gov.sa/en/E-Invoicing/Introduction/Pages/What-is-e-invoicing.aspx.

ISO’s current public page identifies ISO/IEC 17020:2026 as published in March 2026 and states that it specifies requirements for competence, impartiality, and consistent operation of inspection bodies. It describes inspection as examination and determination of conformity against requirements from standards, regulations, contracts, or internal specifications across an item lifecycle. This supports a configurable rule-driven INTEGIN model that distinguishes inspection facts, certificate approval, and external recognition; it does not establish that any INTEGIN workflow satisfies the standard. Source: https://www.iso.org/standard/17020.

SAAC’s official overview states that it grants accreditation, qualifies assessors, and monitors conformity-assessment bodies’ compliance with approved accreditation requirements; its public services include inspection-body and certification-body accreditation. This supports treating Saudi accreditation as a separate organizational/assessment relationship and treating any future INTEGIN adapter as a controlled support tool, not proof of accreditation or regulator acceptance. Source: https://saac.gov.sa/en/home/.

## Attached Project-Inspection Conversation Audit — 2026-08-21

The user supplied a historical separate-agent conversation containing a project inspection and generic Go/PostgreSQL architecture advice. It adds no new certificate, evidence-disclosure, commercial, or external-authority policy; its product workflow observations align with existing high-volume and integrated operating-model records. Current source directly rechecked `internal/domain/sync/sync.go`: durable receipt deduplication still occurs before device/signature checks; duplicate handling ignores a receipt outcome; unsupported signature-algorithm strings still take the HMAC branch; and a held receipt plus held envelope are persisted in separate calls. These historical claims therefore remain current corrective candidates rather than closed risks. The custom Work-Order PostgreSQL array literal helper also remains without targeted special-character proof.

The attached conversation’s “Binding Key Registry before certificate designer” recommendation is valid: no current source match exists for a binding-key registry. It should become the first sub-foundation of the certificate authority lifecycle, while the sync held/replay correction remains a separate offline-durability workstream. The full comparison is recorded in `docs/architecture/INTEGIN_ATTACHED_PROJECT_INSPECTION_AUDIT_2026-08-21.md`.

## Offline-Sync Durability Correction — 2026-08-21

The confirmed historical sync risks were corrected independently of Work-Order and evidence work. A durable held record is now a resumable authorized recovery state, not a duplicate. Receipt inspection occurs only after device, authority, algorithm, and signature validation; `Ed25519` and legacy `HMAC-SHA256` are exact allow-listed values. PostgreSQL held receipt/envelope persistence and held-to-applied promotion are transaction-scoped. The controlled test demonstrated that a trigger-induced held-envelope insert failure rolls the receipt back as well. The first integration fixture used `defer db.Close` before `t.Cleanup` and a parameterized multi-command statement, which prevented its cleanup; the test now registers close as the earlier cleanup callback and executes four dependency-ordered delete statements. Its rerun left zero `it-sync-durability-*` rows.

## Certificate-Template Binding Registry Finding — 2026-08-21

The certificate-template registry is now proven as a pre-issuance layout/binding foundation, not as certificate authority. `certificate_template` and `certificate_template_cell` are forced-RLS, tenant/organization-scoped immutable records. Their resolver obtains values only from the canonical `inspection_record` fields currently present in the approved nine-key catalog and requires template approval before resolution. The controlled pilot test `TestPostgresResolveCanonicalInspectionTemplateIntegration` passed canonical-value resolution and cross-organization denial, after the stopped local PostgreSQL container was restored unchanged and confirmed ready. The test’s namespaced `certificate_template` and `inspection_record` rows were absent afterward; direct full `go test -count=1 ./...` and `go vet ./...` passed. This does not establish inspection eligibility, reviewer/signatory authorization, certificate number allocation, issuance snapshotting, validity, revocation, supersession, rendered document generation, or public QR behavior; those remain the separate certificate lifecycle boundary.

## Certificate Lifecycle Draft-Persistence Finding — 2026-08-21

A certificate record cannot safely be created from client authority or an unchecked template. The implemented repository therefore opens a transaction, sets RLS scope from the server-derived actor, locks and reads the canonical inspection under explicit tenant/organization predicates, requires `APPROVED` plus `FINALIZED`, joins policy to an `APPROVED` certificate template, derives the profile decision through domain validation, and inserts the draft and `DRAFT_CREATED` audit record before commit. The candidate schema permits a draft before number allocation, then requires certificate number/issuer/issue time/expiry on issued and corrective terminal states. The review identified that later status actions need their own controlled integration evidence; a compiling method is not equivalent to proof.

## Certificate Issuance and QR-Safe Projection Finding — 2026-08-21

Issuance must be an atomic authority transition, not a renderer action. The proof now shows that a signed certificate locks its approved policy/template and canonical inspection facts, allocates its per-scope number, writes immutable snapshots, stores only a token digest, writes audit evidence, and becomes issued in one transaction. The verifier resolves only the raw token digest and returns certificate number, truthful status, issue/expiry time, and current asset ID; it does not expose evidence, snapshots, tenant/organization, actor identifiers, audit history, or the raw token. This is a repository-level public projection, not yet a public HTTP portal.

## Complete Certificate Lifecycle Persistence Finding — 2026-08-21

The certificate authority layer can now distinguish a default independent review from a policy-bounded senior self-issue without accepting profile authority from a client. It contains all current state transitions in RLS-scoped transactions, locks canonical inspection/policy/template records at issuance, persists immutable issuance facts, and never stores the raw public token. Supersession requires an issued same-asset replacement in the same tenant/organization and is atomic through transaction rollback. Expiry is an explicit guarded transition; no scheduled expiry processor is implemented yet.

## Certificate HTTP Transport Finding — 2026-08-21

OIDC claims cannot safely be treated as complete lifecycle authority on their own: the existing validator deliberately authenticates issuer identity while local identity membership supplies the actor scope and capability list. The certificate transport foundation therefore depends on an explicit OIDC-principal-to-local-membership resolver and does not accept any authority-shaped body field. A compiled handler is not a live transport proof until it is composed with the configured validator and resolver and exercised at runtime.

## Certificate Mutation HTTP Finding — 2026-08-22

OIDC authenticates the caller but does not itself establish local certificate authority. The certificate handler therefore obtains actor, tenant, organization, and capabilities only from local membership resolution and never accepts them in a request. The raw public token is an issuance secret: it may be disclosed once in the successful authenticated issue response, never in audit records, public projections, or retrieval responses.

## Public Verifier Recovery Finding — 2026-08-22

A partially written public HTTP endpoint is worse than no endpoint because it can expose unaudited response fields or create inconsistent availability. The incomplete package was removed and no public route is claimed. The verified database projection remains constrained to certificate number, status, issue time, expiry time, and asset ID.

## Public Certificate Verifier Finding — 2026-08-22

The public handler does not manufacture the owner-required full serial, description, or inspection test scope because their approved canonical sources are not yet present. It exposes only the five fields already selected by the repository projection. The handler requires server composition and a deployment-grade shared limiter before a public production endpoint can be claimed.

## Certificate Runtime Composition Finding — 2026-08-22

The mutation endpoint is intentionally absent, not anonymously reachable, whenever OIDC is disabled. Its 404 in the OIDC-disabled candidate is a configuration fail-closed proof, not a proof of authenticated authorization. Public verification is mounted with database availability but is still limited by a process-local limiter; deployment-grade shared rate enforcement and a valid issued-token runtime result remain separate prerequisites for public release.

## Reusable Controlled OIDC Runtime Fixture — 2026-08-22

The existing work-order signed-OIDC PostgreSQL integration test supplies a reusable local RS256 issuer/JWKS server, registered local identity membership, and dependency-ordered cleanup pattern. It is the appropriate controlled fixture basis for certificate mutation runtime proof; it avoids treating OIDC claims as tenant or capability authority and does not require a persistent third-party IdP.

## Certificate Runtime Identity Boundary — 2026-08-22

The certificate actor resolver uses only the migration-owned integin_resolve_identity_membership(issuer, subject) function. It resolves exactly one local actor, tenant, organization, role, and capability set; unknown and ambiguous subjects fail. A signed-OIDC proof must insert and remove a local membership fixture rather than place authority in token claims.

## 2026-08-22 — Public certificate composition limit
2026-08-22: Certificate public transport composition proof completed. Added TestPublicVerifierComposesThroughServerMux: a valid bounded projection is served as 200 no-store through server.NewMux and the immediate second request is 429 no-store with only one verifier lookup. Actual issued-token digest resolution remains proven in certificatepg integration. Full go test -count=1 ./... and go vet ./... passed; queried lifecycle/OIDC residue was 0|0|0. The first cross-layer DB test attempt created a Go import cycle and was removed cleanly before final regression. Shared rate control, external release, QR/PDF output, and canonical serial/description/type/test-scope projection remain unproven.

## 2026-08-22 — Canonical public verification data boundary
2026-08-22: Completed follow-on certificate-verification design inventory. Recorded canonical data gaps (serial and description absent; asset type only in work-order scope; inspection type not persisted in inspection_record; no canonical public test-scope summary), drafted the tenant-safe immutable public-binding contract, documented PDF/QR prerequisites, and published the gated eight-step implementation sequence. No schema, route, renderer, QR artifact, public release, or external integration was added.

## 2026-08-22 — Immutable certificate public-binding snapshot
2026-08-22: Completed 0014 canonical public-binding slice. Disposable up/down review passed; fresh pre-apply dump saved at tmp\integin-pilot-pre-0014-20260822-094737.dump; 0014 applied to pilot. Issuance now snapshots required policy-approved canonical asset and inspection public-scope values and includes them in the snapshot digest. Integration proved snapshot persists serial/description/type/inspection scope and survives later mutable asset changes. Namespaced residue was 0|0|0|0|0; full go test -count=1 ./... and go vet ./... passed. Public HTTP projection, QR/PDF output, public deployment, and shared rate/abuse controls remain unimplemented.

## 2026-08-22 — Public verifier reads immutable snapshot only
2026-08-22: Completed immutable snapshot-backed public verifier expansion. VerifyPublic joins only certificate_snapshot.public_binding_snapshot, and the handler conditionally allow-lists serial, description, type, inspection type, and structured test scope while preserving the original public fields, no-store, 404 indistinguishability, and limiter behavior. Focused tests, full go test -count=1 ./..., go vet ./..., and zero fixture residue (0|0|0|0) passed. QR/PDF rendering, shared rate/abuse control, public release, and external integrations remain unimplemented.

## 2026-08-22 — Fixed-cell renderer capability
2026-08-22: Implemented internal/certificaterender fixed-cell PDF and high-recovery QR capability with go-pdf/fpdf v0.9.0 and skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e. Renderer validates template geometry and bounded text policies, rejects insecure QR origins/overflow, returns byte digest and non-sensitive metadata, and uploads through storage.Store. Focused renderer tests plus full go test -count=1 ./... and go vet ./... passed. It is not yet connected to actual issuance snapshots/tokens, PostgreSQL artifact metadata, authenticated download, public release, or shared rate/abuse controls.

## 2026-08-22 — Go toolchain upgrade and dependency freeze
2026-08-22: Owner-authorized Go upgrade completed. The active toolchain is go1.26.5 windows/amd64 and go.mod now declares go 1.26.5. go mod tidy, focused certificate/renderer tests, full go test -count=1 ./..., and go vet ./... passed. The Codeberg fpdf v0.12.0 release line is official and currently active, but the project remains on frozen github.com/go-pdf/fpdf v0.9.0 until the owner explicitly approves a module-path switch; the QR dependency is likewise frozen.

## 2026-08-22 — Technology update posture
2026-08-22: Completed technology update audit excluding frozen PDF/QR dependencies. Current: Go 1.26.5, PostgreSQL 18.6, Keycloak 26.7.1, Docker 29.7.2, JWT v5.3.1. Recommended separately controlled updates: OpenBao 2.6.0 to 2.6.2 due upstream security fixes; lib/pq 1.10.9 to 1.12.3 with integration proof; RustFS rc.1 to newer rc only after backup/recovery/S3 compatibility rehearsal. PostgreSQL uses a broad postgres:18 tag and should later be exact minor/digest pinned. No version, image, data, config, or frozen dependency was changed by this audit.

## Performance baseline
2026-08-22: Performance baseline complete; see INTEGIN_PERFORMANCE_BASELINE_RESULTS_2026-08-22.md. No evidence supports domain/RLS/certificate/sync optimization.

## 2026-08-22 — Bottleneck comparison
2026-08-22: Repeated five-sample local-pilot bottleneck comparison complete. Certificate lifecycle with real DB verifier checks: mean 1298.05 ms, range 1157.77-1415.80 ms. RustFS signed contract: mean 864.50 ms, range 794.65-942.51 ms. Sync recovery/atomicity: mean 922.52 ms, range 821.47-1203.38 ms. These are full test-path timings including Go setup, not endpoint latency. No reproducible protected-logic bottleneck was identified; DB-backed HTTP verifier remains unisolated. Namespaced residue zero; full regression/vet passed.

## 2026-08-22 — HTTP verifier performance and token-log redaction
2026-08-22: Added cleanup-backed real PostgreSQL HTTP public-verifier fixture. Five independent 20-request samples: request mean-of-means 3.113 ms, mean range 2.463-4.625 ms, worst sample p95 10.043 ms. All responses 200 no-store. Discovered shared request logger exposed path token; fixed by logging /verify/certificates/:token and added regression test. Fixture residue zero; full regression/vet passed.

## 2026-08-22 — PDF/QR library evidence
Official sources: https://pkg.go.dev/codeberg.org/go-pdf/fpdf documents Codeberg FPDF as MIT-licensed, standard-library-only PDF generation with text, fixed drawing/images, UTF-8/RTL support, templates and barcodes; Go 1.26.5 satisfies its Go 1.25 requirement. https://pkg.go.dev/github.com/boombuler/barcode documents QR generation, PNG rendering, scaling and QR error-correction use, with current v1.1.0 API material. Recommendation pending owner approval: Codeberg FPDF plus boombuler/barcode QR, replacing frozen github.com/go-pdf/fpdf v0.9.0 and skip2/go-qrcode.

## 2026-08-22 — Advanced renderer research sources
WeasyPrint official docs: BSD license, Python 3.10+, HTML/CSS-to-PDF, Pango/HarfBuzz/font dependencies, and security constraints for untrusted HTML/CSS (resource fetching, local file exposure, CPU/memory limits): https://doc.courtbouillon.org/weasyprint/stable/ and https://doc.courtbouillon.org/weasyprint/stable/first_steps.html. CSS Paged Media page-box/header/footer/pagination model: https://www.w3.org/TR/css-page-3/. Segno: pure Python, BSD-3-Clause, full QR forcing and error control, PNG/SVG/PDF serialization: https://github.com/heuer/segno. Awesome-PDF-derived QA candidates: pdfcpu Apache-2.0 Go validation/signature evidence: https://github.com/pdfcpu/pdfcpu; pikepdf MPL-2.0 low-level inspection: https://github.com/pikepdf/pikepdf; Pdfalyzer structural/YARA assessment: https://github.com/michelcrypt4d4mus/pdfalyzer. WeasyPrint final selection must be gated by Arabic/English shaping and bounded fixed-cell proof.

Renderer qualification finding: local Pango/WeasyPrint synthetic checks are encouraging but partial. In-process request denial is not proof that an isolated production worker cannot egress; container/firewall evidence remains mandatory before adoption.

Mixed-direction finding: Arabic paragraphs must retain RTL flow while embedded English phrases, identifiers, measurements, and dates are rendered as isolated LTR spans. The practical synthetic fixture behaved acceptably, but real production copy still requires the restricted compiler and approved font pack.
