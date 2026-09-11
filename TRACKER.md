# 🚀 INTEGIN Master Sprint & Session Tracker

**Document Version:** 3.3.0 (Comprehensive Global Architecture & 12-Tier Reconciliation Standard)  
**Last Reconciled:** 2026-09-08  
**Master Architectural Authority:** [`docs/architecture/GLOBAL_ARCHITECTURE_PLAN.md`](./docs/architecture/GLOBAL_ARCHITECTURE_PLAN.md)  
**Governing Topology:** The Master 12-Tier Architecture (L0–L11), The 7 Sovereign Compliance Pillars (§ 3.5), & The 4-Phase Transition Roadmap (§ 7)  
**Historical Planning Archives (>600KB):** [`archive/planning-history-august-2026/`](./archive/planning-history-august-2026/)

---

## 1. 🌍 The Master 4-Phase Transition Roadmap ([GLOBAL_ARCHITECTURE_PLAN.md § 7](./docs/architecture/GLOBAL_ARCHITECTURE_PLAN.md#7-the-4-phase-master-transition--adoption-plan))

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                              GLOBAL TRANSITION ROADMAP                                 │
├──────────────────────┬──────────────────────┬───────────────────┬──────────────────────┤
│ Phase 1: Core PKI &  │ Phase 2: Hybrid      │ Phase 3: Hardware │ Phase 4: K8s Svc Mesh│
│ Dynamic Licensing    │ Standards Discovery  │ & Audit Ledger    │ & Global Verification│
│ (Sprint 1)           │ (Sprint 2)           │ (Sprint 3 - ACTIVE│ (Sprint 4)           │
├──────────────────────┼──────────────────────┼───────────────────┼──────────────────────┤
│ • pkg/domain & DIDs  │ • pkg/rulesengine    │ • Tool Registry   │ • Stateless Resolver │
│ • pkg/licensing      │ • pkg/standardsync   │ • Merkle-CRDT Log │ • Public Verifier App│
│ • CLI Token Issuer   │ • pkg/jurisdictions  │ • FIPS Enclave Att│ • Helm / K8s Matrix  │
│                      │ • idempotency cache  │ • Asset Passport  │ • Sovereign Appliance│
│ STATUS: COMPLETE ✅  │ STATUS: COMPLETE ✅  │ STATUS: ACTIVE 🚀 │ STATUS: PLANNED      │
└──────────────────────┴──────────────────────┴───────────────────┴──────────────────────┘
```

| Master Phase | 12-Tier Scope | Target Packages / Modules | Phase Status | Key Deliverables & Evidence |
|---|---|---|:---:|---|
| **Phase 1: Core PKI & Dynamic Licensing (Sprint 1)** | **L0** (Root PKI) | `pkg/domain`, `pkg/licensing`, `cmd/integin-cli` | **COMPLETE ✅** | Asymmetric Ed25519 license validation, W3C DIDs (`did:integin`), offline covenants passing. |
| **Phase 2: Hybrid Standards Discovery & Dynamic Engine (Sprint 2)** | **L1** (Standards/AST), **L2** (Jurisdictions), **L4** (Hierarchy) | `pkg/rulesengine`, `pkg/standardsync`, `pkg/jurisdictions`, `internal/idempotency`, `pkg/queryengine` | **COMPLETE ✅** | **All 5 deliverables (D2.1–D2.5) complete & verified:** CEL nanocell + D2.2 idempotency/DLQ + D2.3 standards discovery + D2.4 jurisdictions + D2.5 PostgREST query engine. Live matrix PASS, race detector PASS across all packages. |
| **Phase 3: Edge Tool Calibration & Bitemporal Merkle Audit Ledger (Sprint 3)** | **L6, L7, L8, L9, L10** (Hardware, Tools, Ledger) | `pkg/onboarding`, `pkg/ledger`, `field_app`, `packagemanifest`, `domain/asset` | **ACTIVE 🚀** | **D3.1–D3.5 complete & verified** (D3.2/D3.3 landed `dd091d8`, D3.4 landed `c9f6441`, D3.5 minimal wizard + CDP smoke landed `4edb1c4`/`dd9ddb5`): ISO 17020 § 6.2 calibration gating, Apple SE / Android StrongBox attestation, 64-byte Merkle-CRDT bitemporal ledger, W3C asset passport, executive onboarding wizard. **Frontier: D3.6 TUS media streamer.** |
| **Phase 4: Cloud-Native K8s Mesh & Universal QR Trust (Sprint 4)** | **L11** (Stateless Edge Trust) | `pkg/verification`, `tools/public-verifier`, `config/k8s`, `config/edge-appliance` | **PLANNED 🌐** | Zero-backend-cost browser WebCrypto QR verification (`verify.integin.com`), K8s Helm charts, air-gapped appliance stack. |

---

## 2. 🏛️ Master 12-Tier Worldwide Layer Topology ([GLOBAL_ARCHITECTURE_PLAN.md § 2, § 6](./docs/architecture/GLOBAL_ARCHITECTURE_PLAN.md#2-the-master-12-tier-worldwide-layer-topology))

The following matrix tracks the live implementation status, Go packages, and PostgreSQL database migrations (0001–0070+) for all 12 architectural tiers:

| Tier | Tier Classification & Name | Implementation Status | Active Go Internal Packages | Database Migrations Covered (0001–0070+) |
| :---: | :--- | :---: | :--- | :--- |
| **L0** | **Global Root PKI Authority & Asymmetric Licensing Engine** | **COMPLETE ✅** | `pkg/domain`, `pkg/licensing`, `licensehttp`, `licensepg`, `flaghttp`, `flagpg`, `platform`, `deployconfig` | `0034_license_entitlement`, `0035_feature_flag_overrides`, `0062_harden_all_remaining_rls`, `0066_wal_suppression_and_xid_freeze_safeties` |
| **L1** | **Hybrid Standards Discovery & Dynamic AST Calculation Engine** | **COMPLETE ✅** | `pkg/standardsync`, `pkg/rulesengine`, `inspectionhttp`, `advisorview`, `advisory`, `aiintegration` | `0023_comments_traffic_light`, `0048_anomaly_detection` |
| **L2** | **Tenant Legal Entity, Multi-Currency & Dynamic Jurisdiction Adapters** | **COMPLETE ✅** | `pkg/jurisdictions`, `internal/idempotency`, `identity`, `tenant`, `settingshttp`, `settingspg`, `middleware` | `0003_event_log_tenant_rls`, `0004_identity_subject_membership`, `0008_identity_actor_alignment`, `0033_configurable_settings_audit_export`, `0062_harden_all_remaining_rls`, `0067_async_tenant_purge_tombstones`, `0071_sync_idempotency_cache`, `0072_river_poison_quarantine` |
| **L3** | **Dynamic Discipline & Inspection Package Scoping Engine** | **COMPLETE ✅** | `traininghttp`, `trainingpg`, `equipment` | `0018_timesheets_courses`, `0032_full_dpp_regulatory_monitor` |
| **L4** | **Global Enterprise Hierarchy & Operational Work Orders** | **COMPLETE ✅** | `workorderhttp`, `workorderpg`, `workorderauth`, `domain/workorder`, `riverqueue` | `0005_work_order_foundation`, `0009_work_order_persistence`, `0010_work_order_rls`, `0012_work_order_handover`, `0019_hierarchical_register`, `0029_parts_charges_timesheet_auto`, `0050`–`0060` (River queue scale), `0063_fix_unindexed_foreign_keys`, `0064_river_hot_updates`, `0065_river_canonical_v047`, `0069_state_machine_and_sequence_bounds` |
| **L5** | **Dynamic Certificate Governance & Configurable 4-Eyes QA** | **COMPLETE ✅** | `certificatehttp`, `certificatepg`, `certificaterender`, `certtemplatepg` | `0012_certificate_template_binding_registry`, `0013_certificate_authority_lifecycle`, `0015_certificate_artifact_metadata`, `0024_escalation_overdue`, `0026_custom_docx_templates`, `0070_add_certificate_performance_indexes` |
| **L6** | **Dynamic Inspector Credentialing & Skill Matrix Verification** | **UPCOMING 📅** | `internal/domain/scheduling`, `internal/identity`, `pkg/onboarding` | `0021_scheduling_calendar` |
| **L7** | **Dynamic Tool Calibration & Traceability Registry (ISO 17020 § 6.2)** | **ACTIVE 🚀** | `internal/evidenceapi`, `internal/evidenceexport`, `internal/evidencehttp`, `internal/evidencepg`, `internal/evidenceregistration`, `internal/platform/calibration`, `internal/domain/evidence`, `internal/domain/evidencepack`, `migrations/0073_*` | `0010_evidence_metadata`, `0011_evidence_metadata_encryption_export`, `0016_evidence_question_link`, `0031_nfc_rfid_qr_tagging_photo_markup`, `0044_work_order_evidence`, `0073_tool_calibration_registry` |
| **L8** | **Universal FIPS 140-3 Hardware Tablet Attestation (Enclave/StrongBox)** | **ACTIVE 🚀** | `pkg/onboarding` (WorkPackageManifest, DeviceTrustRecord, SignedInspectionReceipt), `field_app/` | `0002_device_trust_sync`, `0006_work_package_assignment_context`, `0007_manifest_proof_replay`, `0022_multi_inspect`, `0043_work_order_signed_submission`, `0068_mobile_cryptographic_hash_chain` |
| **L9** | **W3C Decentralized Asset Passport & Technical Quarantine Lifecycle** | **ACTIVE 🚀** | `pkg/domain` (models.go, did.go) | `0017_product_passport_geo`, `0020_bulk_import_export`, `0025_job_linkage_failed_queue` |
| **L10** | **Bitemporal Merkle-CRDT Tamper-Proof Audit Ledger (Forensic Blackbox)** | **ACTIVE 🚀** | `internal/domain/auditlog` (hashchain, repository, types), `searchhttp`, `searchpg` | `0001_event_log`, `0036_full_text_search`, `0037_search_backfill`, `0038_immutable_audit_log`, `0061_kill_gin_and_dark_hardening`, `0068_mobile_cryptographic_hash_chain` |
| **L11** | **Edge Zero-Knowledge QR Trust Gateway & Dynamic Multi-Regulator Sync** | **PLANNED 🌐** | `internal/certificatepublichttp`, `internal/shortlinkhttp`, `internal/shortlinkpg`, `internal/shortlinksvc`, `internal/analyticshttp`, `internal/analyticspg`, `internal/reportshandler`, `internal/reportspg` | `0014_certificate_public_bindings`, `0027_client_portal_domains_acls`, `0028_integrations_xero_m365_api`, `0030_hse_notification_csv_export`, `0039_short_links` ... `0047_short_link_hmac`, `0049_analytics_dashboard`, `0070_add_certificate_performance_indexes` |

---

## 3. ⚡ Sprint 2 (Hybrid Standards & Dynamic Rules Engine) — COMPLETE ✅

### Sprint 2 Deliverables Matrix & Technical Acceptance Gates:

#### Deliverable 2.1: Computational Nanocell (`pkg/rulesengine`) 🚀 *(IMMEDIATE FOCUS)*
*   [x] **Step 1: Add Google CEL Dependency**: ✅ `go get cel.dev/cel-go` resolved **v0.32.0** in `go.mod`. NOTE: canonical module path is `cel.dev/cel-go` (NOT `github.com/google/cel-go` — that path fails `go mod tidy` with a module-declared-path mismatch). Tracker SLA (v0.20+) satisfied.
*   [x] **Step 2: Implement Data Model (`pkg/rulesengine/schema.go`)**: ✅ DONE — `DynamicStandardDefinition`, `DynamicRule` (`RuleID`, `Description`, `Expression`, `Severity` `CRITICAL_QUARANTINE`/`WARNING`), `ChecklistSchema map[string]interface{}`, `ChecklistField`, `RuleResult`, `RuleSet`.
*   [x] **Step 3: Implement Ruleset Versioning (`pkg/rulesengine/versioning.go`)**: ✅ DONE — `DRAFT`→`ACTIVE`→`DEPRECATED`, immutable activation timestamps; illegal transitions rejected (`TestRuleSetVersioning`).
*   [x] **Step 4: Implement Evaluator Kernel (`pkg/rulesengine/evaluator.go`)**: ✅ DONE — `cel.ClearMacros()` strict env (non-Turing complete, `TestSandboxRejectsLoopInjection`), per-evaluation 4 MiB heap budget gate (`budgetCheck` charge-per-element walk, `TestSandboxBudgetExceeded`), typed `VarSet implements cel.Activation` directly (zero wrapper allocs), scalar bindings pre-adapted to CEL-native `ref.Val` at `Put`.
*   [x] **Step 5: Implement ASME B30.5 Mobile Crane Proof-Load Model**: ✅ DONE (`asme_b30_5.go`) — full ternary CEL formula + deployed tier-folded `B30_5GateExpression` (coefficient folded via pure-Go `B30_5ProofLoad` → 0-alloc device gate). Parity test `TestB30_5GateFoldingMatchesFullForm`.
*   [x] **Step 6: Implement ISO 4309 Wire-Rope Discard Criteria Model**: ✅ DONE (`iso_4309.go`) — `double(broken) >= 6.0*d` count rule + `(d_nom−d_meas)/d_nom > 0.07` reduction rule, both with folded gate forms. Boundary tests on both sides of each threshold.
*   [x] **Step 7: High-Performance Benchmark Suite (`evaluator_bench_test.go`)**: ✅ DONE — `BenchmarkB30_5HotPath` **271 ns/op, 0 allocs/op**; `BenchmarkISO4309HotPath` **227 ns/op, 0 allocs/op**; `BenchmarkRiggingHotPath` **46 ns/op, 0 allocs/op**. SLA `<50µs` + `0 allocs/op` MET (0-alloc achieved via compile-time coefficient folding + `ref.Val` pre-adaptation, verified with `-benchmem` + `-memprofile`; residual 1-2 allocs on the raw dynamic-form expressions come from CEL's `Double.Multiply` interface boxing and are eliminated by the folded gate).
*   [x] **Step 8: Security & Isolation Unit Tests (`evaluator_test.go`)**: ✅ DONE (**16/16 PASS**) — non-Turing termination (`TestSandboxTermination`), loop-macro rejection, malformed/non-bool/undeclared expression rejection, missing & unauthorized variable binding rejection, 4 MiB budget enforcement.
*   [x] **Step 9: Rigging Vector & Geotechnical Ground Bearing Pressure Engine (`rigging.go`, `geotech.go`)**: ✅ DONE — 2D sling tension `T = Load/(2·sinθ)` with `<30°` derating lockout, NaN/Inf/0-division guards; GBP `σ = P/A` with positive-area guard + FoS verdict.
*   **Verification Evidence (2026-09-07)**: `go vet ./...` CLEAN; `go test -count=1 ./...` PASS (0 failures, whole repo); `go test -race` BLOCKED on this host (no `gcc`, cgo unavailable; CGO_ENABLED=0 is the codebase standard per Hazard 40).

#### Deliverable 2.2: Stripe-Grade Global Idempotency (`internal/idempotency`)
*   [x] **Step 1: Database Migration (`migrations/0071_sync_idempotency_cache.sql`)** ✅ DONE — `sync_idempotency_cache` with `key_hash`, `tenant_id`, `organization_id`, `endpoint`, `request_hash`, `status`, `response_payload`, `http_status`, `created_at`, `expires_at`; composite PK `(tenant_id, organization_id, key_hash)`; TTL index + bounded BEFORE INSERT eviction trigger (`sync_idempotency_cache_evict_expired`, ≤1000 expired rows/sweep); 24h TTL enforced by `CHECK (expires_at > created_at)` and stored `expires_at = now()+24h`; hardened composite tenant RLS (`ENABLE` + `FORCE`, 0062-style NULLIF policy). Down migration included. Contract-tested in `migrations/migration_test.go`.
*   [x] **Step 2: Middleware Contract (`internal/idempotency/middleware.go`)** ✅ DONE — mandatory `Idempotency-Key` header (>400 on `idempotency_key_required`), key charset/length validation (8-128, `[A-Za-z0-9._-]`), atomic PostgreSQL `pg_advisory_xact_lock(hashtext(...))` + `SELECT ... FOR UPDATE` claim acquisition (key-held-through-commit linearizability), `X-Idempotent-Replay: true` replay marker, buffered request body hashing (`HashKey` = SHA-256 of `org\x00endpoint\x00key`; `HashRequest` = SHA-256 of body), scope resolvers for `/sync` and `/evidence`.
*   [x] **Step 3: Outcome Replay Engine** ✅ DONE (`postgres.go`, `status.go`) — structured outcomes cached verbatim (`APPLIED`, `DUPLICATE`, `HELD`, `CONFLICT`, `REJECTED`, `QUEUED`, `SECURITY_FAILURE`) derived from response body `outcome` field with HTTP-code fallback; replay is an in-memory response-buffer copy with **zero downstream DB execution** (SLA `<5ms` asserted in `TestMiddlewareReplaysCachedResponseWithoutDownstream`); 5xx responses are never cached (claim rolls back) so retries re-execute; reused key with different body ⇒ `409 Conflict`.
*   [x] **Step 4: Wire to Server Mux** ✅ DONE — wrapped `/sync` and `/evidence` in `internal/server/http.go` + `http_routes.go` (postgreSQL store built automatically from `dependencies.DB`; nil-safe for non-DB test hosts); `/evidence` only guarded when its handler is mounted.
*   [x] **Step 5: Concurrent Race & Chaos Tests** ✅ DONE — hermetic fake-store concurrency suite (`middleware_test.go`): 16-goroutine same-key same-payload ⇒ exactly 1 downstream execution + 15 identical replays; concurrent same-key different-payload ⇒ at least one 200 + at least one 409; plus gated PostgreSQL integration suite (`postgres_integration_test.go`, `INTEGIN_TEST_DATABASE_URL`): 16 concurrent claims ⇒ exactly 1 leader + 15 replays, cross-payload conflict, 5xx-not-cached re-execution, end-to-end middleware-over-Postgres exactly-once.
*   [x] **Step 6: River Poison-Pill Quarantine (DLQ)** ✅ DONE — `migrations/0072_river_poison_quarantine.sql` (`river_poison_quarantine`: job_id PK, kind, nullable tenant/org, args JSONB, error_text, stack_trace, attempt, original_state, quarantined_at; RLS `tenant_id IS NULL OR (tenant bound + optional org bound)`; GRANT to `integin_runtime`) + `internal/queue/poison_quarantine.go` worker interceptor that recovers fatal payload panics, records forensic evidence (panic text, stack trace, attempt, original state, encoded args, tenant scope), then returns `river.JobCancel(...)` so the job is cancelled (terminal) instead of stalling the queue. Attached to River via `queue.WithMiddleware`; wired in `cmd/integin-server/main.go`. Hermetic tests (`poison_quarantine_test.go`): panic contained, single record with full forensics, tenant scope preserved from args, ordinary errors pass through untouched, success unaffected.
*   **Verification Evidence (2026-09-07)**: `go vet ./...` CLEAN; `go test -count=1 ./...` PASS (0 failures, whole repo incl. `internal/idempotency`, `internal/queue`, `migrations`, D2.1 `pkg/rulesengine`); `gofmt -l` CLEAN on all changed files. **RACE DETECTOR UNBLOCKED (2026-09-07)**: MSYS2 MinGW-w64 toolchain installed (`C:\msys64\mingw64\bin`, GCC 16.1.0, CGO_ENABLED=1) — `go test -race -count=1 ./...` now runs and PASSES across the **entire repo** (0 failures, 0 data races), including the 16-goroutine idempotency concurrency tests and all live Postgres integration suites. **LIVE PostgreSQL verification**: ran the env-gated suites against the local `integin-dev-postgres` container (port 15432). All 78 migrations apply cleanly in order to a fresh ledger-tracked database (migration engine forced via direct `psql -v ON_ERROR_STOP=1` loop because `apply_migrations.ps1` aborts on psql's stderr NOTICEs under PowerShell 5.1 `$ErrorActionPreference=Stop`). With `INTEGIN_TEST_DATABASE_URL` pointed at the fully-migrated `integin_d2x_verify`, the **entire** `go test -count=1 ./...` suite passes (0 failures) — 15/15 `internal/idempotency` (incl. 4 Postgres integrations), plus all previously-gated suites (`syncstate`, `workorderhttp`, `workorderpg`, `certificatehttp`, `dpppg`, `qrnfcpg`, `chaos`, ...). Live run surfaced two real fixes: `advisoryKey` no longer sends a raw NUL-containing string to `hashtext()` (PostgreSQL `22021`), and the JSONB round-trip test expectation compares payloads semantically (PostgreSQL canonicalizes JSONB whitespace). NOTE: the older curated-subset `integin_dev` DB still has schema drift unrelated to D2.2 (missing `0043_work_order_signed_submission` `payload_hash`) — refresh it via the full migration stack before running gated suites against it.

#### Deliverable 2.3: Copyright-Safe Standards Discovery Engine (`pkg/standardsync`)
*   [x] **Step 1: Standards Metadata Model (`pkg/standardsync/schema.go`)** ✅ DONE — `StandardMetadataCard` (`StandardDID` `did:integin:standard:asme-b30-5-2021` slug form, `StandardBody`, `Code`, `RevisionYear`, `Title`, `ScopeAbstract`, `LifecycleState` `ACTIVE`/`SUPERSEDED`/`WITHDRAWN`/`DRAFT`, `ReplacesStandard`, `OfficialStoreURL`, `PublishedDate`, `ApplicableAssets`, `Tags`); `HybridSearchResult` (`QuerySummary`, `PrimaryMatches`, `DeprecatedMatches`, `SuggestedActions`, `Anchors`, `Metadata`); `Match`, `CodeAnchor`, `SearchMetadata`. `Validate()` enforces the citation-card contract (parseable DID that round-trips through `DID()`/`ParseDID()`, known lifecycle, https storefront URL — never `.pdf`/`/documents/`/`/files/`), and `Articles()` guarantees scope prose is never indexed as a feature. Tests: DID round-trip, citation-contract rejections (http/pdf/documents-path), lifecycle Binding() semantics, no-scope-leak.
*   [x] **Step 2: Hybrid AI Synthesizer & Abstract Search (`pkg/standardsync/search.go`, `index.go`)** ✅ DONE — deterministic 128-dim FNV hashed uni+bigram vectorizer (L2-normalized, architecture-stable, no map-order nondeterminism) + BM25-style lexical overlap; `Engine.Search` synthesizes a bounded answer over ingested metadata only, extracts and normalizes code anchors ("ASME B30.5", "b30.5", "B30.5-2021" all collapse), infers the owning body for body-less anchors against the index, pins the ACTIVE current revision as primary and routes superseded revisions to Deprecated, with auditable `SuggestedActions`. **`<30ms` SLA verified**: `BenchmarkEngineSearchWithin30ms` = **2.78ms/op on a 2,000-card universe** (i5-10310U). Determinism test: reordered insertion → identical ranking.
*   [x] **Step 3: Lifecycle Resolver (`pkg/standardsync/lifecycle.go`)** ✅ DONE — `Resolver` (immutable, concurrency-safe): `Current` (newest ACTIVE per body+code), `EffectiveAt` (publication-window resolution), `ReplacedBy`/`Predecessors` chain walks, `FormatReplaces` ("ISO 4309:2010 replaces ISO 4309:1990"), `VerifyBool` (sole gate for verdict injection), `LookupByCode` (body inference); `ValidateLifecycle` rejects two ACTIVE revisions of one series, orphaned SUPERSEDED cards, cross-series or same-vintage replacement ancestry. Errors: `ErrUnknownStandard`, `ErrNoActiveRevision`.
*   [x] **Step 4: Offline Edge Metadata Vector Index (`pkg/standardsync/local_cache.go`)** ✅ DONE — self-contained air-gapped cache: JSON envelope with `schema_version`, canonical SHA-256 checksum, sorted cards; `Store` writes atomically (temp + rename) for power-cut safety on ships/rigs; `OpenLocalCache` refuses corrupt or foreign-schema files. Round-trip, corruption, and foreign-version tests.
*   [x] **Step 5: Open Connectors for Global Standardization Bodies (`pkg/standardsync/connectors/`)** ✅ DONE — registry + adapters for **API, ASME, BSI, ISO, DIN, ASTM, DNV, IEC, LEEA, NFPA, AWS, JIS**: each pins official storefront root hosts and a code-pattern recognizer (`MatchesCode`/`CanonicalCode`). **Safe Harbor guard** (`ValidateOfficialStoreURL`): https-only, host-pinned, rejects `.pdf`/`.zip`/`/documents/`/`/files/` download paths; `SafeStoreURL` refuses unrecognized codes and never fabricates per-code deep links. Tested recognition samples per body (e.g. `DNV-ST-0377`, `NFPA 70E`, `JIS B 7516`) plus negative cases and guard rejections.
*   [x] **Step 6: 1-Click Execution Bridge (`pkg/standardsync/bridge.go`)** ✅ DONE — `BindToPackage` resolves DIDs, requires **ACTIVE** + `VerifyBool` (superseded/unresolved → `ErrNonBindingStandard`/`ErrUnresolvedStandard`), collapses duplicates, returns `ManifestBindingSet` with deterministic `sha256:` `BindingDigest` over package-hash + template-code + sorted DID list; `Verify` detects tampering; `MarshalJSON` emits citation-only card fields (no scope abstract/assets/tags). Non-invasive to the immutable `packagemanifest` signing grammar — binding digests attach alongside the existing manifest signature.
*   [x] **Step 7: Standards Ingest & AST Compiler Toolchain (`cmd/standards-compiler`)** ✅ DONE — CLI: `list-bodies`, `compile -in ... -out ... [-emit-cellist]`, `verify -cache ... [-query ...]`. Parses YAML (goccy/go-yaml) and JSON, validates citation contract + full lifecycle via resolver, builds the offline cache, and with `-emit-cellist` compiles every ACTIVE card into a **validated Google CEL gate** through the D2.1 `pkg/rulesengine` sandbox (compile → AST → evaluate round-trip) — e.g. `standard_did == "did:integin:standard:iso-4309-2010" && lifecycle == "ACTIVE"`.
*   **Verification Evidence (2026-09-07)**: `gofmt -l` CLEAN on all changed files; `go vet ./...` CLEAN; `go test -count=1 ./...` PASS (0 failures, whole repo including the new `pkg/standardsync`, `pkg/standardsync/connectors`, `cmd/standards-compiler`; D2.1 `pkg/rulesengine` 14/14, D2.2 idempotency/poison-quarantine still green). **Race**: `go test -race -count=1` PASS over `pkg/standardsync/...`, `pkg/rulesengine/...`, `internal/queue/...`, `internal/idempotency/...` (0 races). **Latency SLA**: 2.78ms/op @ 2,000-card index. **Live LOC demonstration**: `standards-compiler list-bodies` (12 bodies), `compile` of a 3-card YAML (2 ACTIVE + 1 SUPERSEDED with correct chain) → cache written; `verify -query "wire rope inspection discard criteria"` → OK, checksum verified, summary pins `did:integin:standard:iso-4309-2010` primary + 1990 deprecated; `-emit-cellist` printed 2 sandbox-validated CEL gates. Citation-only posture honored end to end (no payload ingestion path exists in the toolchain).

#### Deliverable 2.4: Dynamic Multi-Country Jurisdictions & Sovereign Adapters (`pkg/jurisdictions`)
*   [x] **Step 1: Jurisdiction Profile Schema (`pkg/jurisdictions/schema.go`)** ✅ DONE — Universal 195+ Country profile (ISO 3166-1 alpha-2/3, ISO 4217 currencies, bilingual locales). Named string-enum `LifecycleState` (`ACTIVE`/`BETA`/`PREVIEW`/`SUSPENDED`/`DEPRECATED`) and `PillarType` with all 7 sovereign compliance pillars (`Tax`, `SafetyRegulator`, `Accreditation`, `Certificate`, `Currency`, `Environmental`, `Labor`). `CountryProfile` carries `ISO2`/`ISO3`/`CurrencyCodes`, bilingual locales, `TaxAuthority` (name + ZATCA-style `TaxIDLabel`/`TaxIDRegexPattern`/`CommercialRegLabel`), `SafetyRegulator`, national accreditors, and `SubDivisionProfile` (emirates/states). Hand-written `Validate()` enforces the full jurisdiction contract (non-empty ISO2/ISO3, ISO 3166-1 normalization checks, currency list, regex compile + anchored `\A...\z`, subdivision names unique). `DID()` derives `did:integin:jurisdiction:<iso2>` (via the new `DIDTypeJurisdiction` + `GenerateJurisdictionDID` in `pkg/domain/did.go`). `HasPillar`, `LookupSubDivision`, sentinel errors. Tests: 29 PASS (validation, DID round-trip, pillar presence, subdivision lookup).
*   [x] **Step 2: Fast In-Memory Registry (`pkg/jurisdictions/registry.go`)** ✅ DONE — thread-safe `sync.RWMutex` registry with duplicate/not-active guards and sentinel errors `ErrDuplicateJurisdiction`/`ErrJurisdictionNotActive`/`ErrJurisdictionNotFound`; indexed by ISO2, ISO3, currency, and pillar; `ByCurrency`/`ByPillar` return value copies (no caller-mutable shared state). `seed.go` ships 8 priority sovereign profiles (🇸🇦 KSA, 🇦🇪 UAE, 🇺🇸 USA, 🇬🇧 UK, 🇩🇪 DE/EU, 🇸🇬 SG, 🇦🇺 AU, 🇳🇴 NO) with correct ZATCA 15-digit TRN regex (`^3\d{14}$`), VAT IDs, tax labels, regulators and sub-divisions. Tests: 19 PASS.
*   [x] **Step 3: Sovereign 7-Pillar Compliance Adapters (`pkg/jurisdictions/adapters/`)** ✅ DONE — `ComplianceAdapter` interface + `Result`/`Finding`/`ComplianceStatus`; individual adapters for **Tax** (ZATCA TRN/VAT validation), **SafetyRegulator** (OSHA plan/inspection), **Accreditation**, **Certificate**, **Currency**, **Environmental**, **Labor**; `CheckAll`/`Check` dispatch across every active jurisdiction. Deterministic, value-receiver reads, zero shared mutable state. Tests: all PASS incl. `TestCheckAllSeedJurisdictions` across the 8 seeded countries.
*   **Verification Evidence (2026-09-07)**: `gofmt -l` CLEAN on all changed D2.4 files (`pkg/jurisdictions/`, `pkg/domain/did.go`, `cmd/integin-live-matrix/*`); `go vet ./...` CLEAN; `go test -count=1 ./...` PASS across the whole repo (0 failures, incl. `pkg/jurisdictions` + `pkg/jurisdictions/adapters`); **Race**: `go test -race -count=1` PASS over `pkg/jurisdictions/...`, `pkg/domain`, `cmd/integin-live-matrix`. **Live matrix**: `integin-live-matrix seed` + `exercise` PASS end-to-end (sync: `APPLIED`→`DUPLICATE`→`HELD`→`CONFLICT`→`SECURITY_FAILURE`; evidence: `APPLIED`→`DUPLICATE`→`CONFLICT`→`SECURITY_FAILURE`), confirmed in the live Postgres `sync_idempotency_cache` (all outcomes recorded at 16:00:52 UTC with the expected HTTP statuses). Client updated so each logical write uses a Stripe-style distinct idempotency key, letting the sync/evidence processors produce their native duplicate/conflict verdicts instead of the middleware replay.

#### Deliverable 2.5: PostgREST-Inspired Dynamic Query Engine (`pkg/queryengine`)
*   [x] **Delegated to `opencode/big-pickle` via opencode-delegate, committed `6b13dcb` in `integin-pilot-source`** ✅ DONE — `schema.go`/`parser.go`/`builder.go` + 2 test files (23 tests PASS). Parameterized `$n` only, tenant guard `tenant_id=$1 AND organization_id=$2` always prepended, `[a-z_]+` identifier gate, limit 1..100 default 20. Gates: `go vet ./pkg/queryengine/...` CLEAN, `go test -count=1 ./pkg/queryengine/...` PASS, `gofmt` CLEAN. Known limit: duplicate filter per field rejected, so same-field range (gte+lte) needs follow-up.
*   [x] **Step 2: Same-field range follow-up (delegated `opencode/big-pickle`, 2026-09-07)** ✅ DONE — removed `seenOps` duplicate-field rejection in `pkg/queryengine/parser.go`; `gte+lte` on same field now allowed; tests order-independent + new `TestParseQuerySameFieldRange`. Gates: `go vet` CLEAN, `go test -count=1 ./...` PASS whole repo, `gofmt` CLEAN. Left uncommitted per AGENTS.md commit boundary.
*   [x] **Step 3: D2 review-findings closure (orchestrator verified, 2026-09-07)** ✅ DONE — F1: full `go test -count=1 ./...` 0 failures, `go vet` clean; 89-file `gofmt -w` scope creep REVERTED, kept only the 2-file `pkg/queryengine` diff (LF-normalized, `gofmt` clean). F2: SLAs re-measured live — B30.5 **546ns/0-alloc**, ISO4309 **207ns/0-alloc**, Rigging **51ns/0-alloc** (all <50µs ✓), EngineSearch **3.2ms** (<30ms ✓). F3: live-matrix RE-VERIFIED PASS 2026-09-07 23:43 UTC — `seed` PASS + `exercise` PASS (exit 0, clean) against live stack (postgres 15432, Keycloak 18180, RustFS 19000, `integin-server` on 18080). Fixes applied: server env gaps (`INTEGIN_SYNC_SECRET`, `INTEGIN_TENANT_ID`, OIDC client vars, RustFS store vars — `/evidence` mounts only when `EvidenceStore != nil` per `http_routes.go:49`); atomic seed→restart→exercise ordering (in-memory authority registry; re-seed after boot = "authority not registered"; retry after partial run = "key reused with different request"). Server binary left running on 18080. F4: committed by orchestrator. F5/F6: no redesign, dual-filter retained, no `BETWEEN`.

---

## 4. 📅 Upcoming Sprints Backlog (Sprint 3 & Sprint 4)

### Sprint 3: Edge Tool Calibration & Bitemporal Merkle Audit Ledger
*   [x] **3.1: ISO 17020 Section 6.2 Calibrated Tool Registry (`internal/shared/calibration/`, `internal/platform/calibration/`, `migrations/0073_*`)** — COMPLETE ✅ (2026-09-08, wiring landed `cc742ef`):
    *   Automatic calibration expiry gating: hard-block work order submission if inspection tool calibration has expired.
    *   Tamper-proof storage of tool serial numbers, calibration lab certificates, and uncertainty tolerances.
    *   🚀 **D3.1 progress (2026-09-08, COMMITTED `92cef08` + serial/identity wiring)**: persistent registry landed — `migrations/0073_tool_calibration_registry.sql` (+ `.down.sql`, contract-test registration) creates `tool_calibration_registry` (PK `tenant/org/id`, `next_due_date > calibration_date`, equipment/status indexes, FORCE RLS NULLIF policy, GRANT to `integin_runtime`); `internal/platform/calibration/postgres.go` adds `Store.Upsert` + `SubmissionBlocked` (expired-or-missing blocks, `pgtx.BeginScope`, `$n` only, hermetic fake-driver tests, no new deps). **Tool identity wiring**: `serial_number`, `lab_certificate_ref`, `uncertainty_tolerance` now flow end-to-end via `sharedcalibration.Record` → `Upsert` (12-param SQL) — tamper-proof storage claim verified. Gates: `gofmt` CLEAN, `go vet` CLEAN, full `go test -count=1 ./...` PASS (0 failures). Open: `SubmissionBlocked` vs `Service.SubmissionAllowed` stale-expired-row edge noted in relay report; live-DB migration apply + `integin-live-matrix` re-verify next on live stack.
    *   ✅ **D3.1 closure (2026-09-08, big-pickle)**: two open items resolved & verified.
        1. **Stale-expired-row edge fixed**: `Service.SubmissionAllowed` (service.go:80) previously returned "calibration is expired" on the FIRST expired ACTIVE row in a two-pass scan, blocking submission even when a newer unexpired ACTIVE row also existed for the same equipment — while the DB `Store.SubmissionBlocked` only checked for the ABSENCE of an unexpired ACTIVE row. Rewritten to a single-pass "allow iff any unexpired ACTIVE row exists", matching DB semantics. New test `TestCalibrationStaleExpiredRowDoesNotBlockWhenUnexpiredExists` covers expired+unexpired coexistence. All 8 calibration tests + full `go test -count=1 ./...` PASS (0 failures), `go vet` CLEAN, `gofmt` CLEAN on changed files (pre-existing `gofmt -l` hits in other packages left untouched).
        2. **Live-DB + matrix re-verify**: `0073_tool_calibration_registry.sql` applied to live `integin_dev` (container `integin-dev-postgres`:15432), table present with all 13 columns. RLS adversarial check live: cross-tenant INSERT as `integin_runtime` under `tenant-x/org-x` writing `tenant-y/org-y` row → **rejected (42501 row-level violation)**; scoped insert + read-back for the owning tenant/org → **allowed**. `integin-live-matrix` re-verified on live stack: SEED PASS → server restart (authority re-hydrated, healthz+readyz 200, 13 authorities loaded) → EXERCISE PASS, with live Postgres `sync_idempotency_cache` confirming full outcome matrix (sync: APPLIED/DUPLICATE/HELD/CONFLICT/SECURITY_FAILURE; evidence: same + REJECTED). Calibration table still empty post-cleanup (test row deleted). Note: the canonical `run-protected-pilot-matrix-v2.ps1` runner hardcodes the stale `C:\MY PROJECT` space-path (broken sparse dir), so the re-verify replicated its exact protected sequence manually with the same secrets in-memory (never printed).
    *   ✅ **Runner fix + full v2 matrix PASSED through canonical runner (2026-09-08, big-pickle)**: the `C:\MY PROJECT` space-path fix (35 pilot files) unblocked PRECHECK; a second latent PS 5.1 defect (blank `$process.ExitCode` with `Start-Process -RedirectStandardOutput`) made every bounded stage report FAILED despite the child running correctly — fixed by replacing `Start-Process` with a .NET `ProcessStartInfo` + `WaitForExit()` invocation. A follow-up pipe-hang (piped `ReadToEndAsync().Result` blocked when `launch-pilot-server.ps1` spawns a server inheriting the pipe) was fixed by in-child file redirection (`1>stderr 2>err`) with zero pipes. Also aligned `private/integin-secrets/integin-pilot.env` `INTEGIN_TENANT_ID` to `integin-integration-tenant` (the OIDC client's only ACTIVE DB membership; WORKSPACE.md:96 consensus) and inherited `INTEGIN_OIDC_*` from host (acceptance env normally supplies them). End-to-end: `PRECHECK PASSED → SEED PASSED → PILOT_STOP PASSED → PILOT_START PASSED → EXERCISE PASSED → MATRIX PASSED → CLEANUP COMPLETED`, live `sync_idempotency_cache` confirms APPLIED/CONFLICT/DUPLICATE/HELD/REJECTED/SECURITY_FAILURE. Same fix ported to `operations/acceptance/*` space-path refs (3 scripts). Deleted 5 stale pre-consolidation pilot DB scripts (`initialize-pilot-db`, `apply-pilot-identity-migration`, `prove-pilot-identity-negative-cases`, `verify-pilot-rls`, `create-pilot-env`) that targeted the defunct `integin-pilot-postgres`/`integin_pilot` stack and encoded the obsolete `integin_pilot_runtime` contract rejected by `migrations/identity_migration_test.go:47`. Closed same-day: `run-oidc-session-matrix.ps1` (which the earlier sweep had also missed on its `C:\MY PROJECT` path refs) retargeted AND FULL MATRIX PASSED through it (2026-09-08, big-pickle) from the defunct `integin-pilot-postgres`/`integin_pilot` stack to `integin-dev-postgres`/`integin_dev` — `Invoke-PilotSql` and the resolver-outage stop/start/ready loop now use the `postgres` superuser (local trust auth) as the owner-role identity-write path; all `identity_*` tables and `integin_resolve_identity_membership` verified present in `integin_dev`. Zero `integin-pilot-postgres`/`integin_pilot` refs remain; parse_errors=0. Live end-to-end PASS after three cache/k8-aware fixes the script predated: (1) Keycloak secrets file corrected to `keycloak-pilot-runtime.env` (was `keycloak-pilot-zip-runtime.env`; invalid_grant until fixed); (2) probe membership insert gains `actor_id`+`work_order_role`, matching live `integin_resolve_identity_membership` + dev-schema checks (no `identity_actor` table in integin_dev); (3) server wraps resolver in 60s `NewCachedResolver` (main.go:213) — the DB is mutated mid-run, so the enabled pilot is now cold-restarted (`-SkipBuild`) after the capability insert and again before the DB-outage check so each assert hits the DB, not the stale cache (this was the 403-on-authorized root cause). Full run: health/readiness 200, missing-bearer 401, invalid-bearer 401, unknown-local-subject 403, missing-local-capability 403, authorized-local-session 200, resolver-outage 503, session-after-database-recovery 200, `OIDC_PILOT_SESSION_MATRIX_PASSED`, disabled-pilot session-route 404, `OIDC_PILOT_SESSION_MATRIX_CLEANUP_AND_DISABLED_RESTORATION_VERIFIED`, err empty. Two runtime defects the first PASS exposed and fixed: (4) resolver-outage returned 200 until the cold-restart fix; (5) finally cleanup raced `docker start` with an immediate identity DELETE (probe row leaked, same class as the pipe-hangs) — finally now waits on `pg_isready` before identity cleanup; leaked row purged, DB now holds only the legitimate `integin-live-actor`. All changes uncommitted; commit pending orchestrator review.
*   [x] **3.2: Universal FIPS 140-3 Hardware Tablet Attestation (`pkg/onboarding/onboarding_engine.go`, `field_app/lib/workpackages/`)** — COMPLETE ✅ (2026-09-09, big-pickle, $0):
    *   Hardware cryptographic signing via Apple Secure Enclave & Android StrongBox KeyStore.
    *   Signed offline outbox with hardware attestation claims bound to inspector biometric identity.
    *   🚀 **D3.2 progress (2026-09-08, big-pickle `85591ca` + `a318946` + `39fac3d` + `0c601e1` + `3f3626d`, $0 no-account path)**: `AttestationClaim` (SECURE_ENCLAVE/STRONGBOX/SOFTWARE/NONE, opaque blob, biometric flag) + `VerifyClaim` policy (permissive default, opt-in RequireHardware/RequireBiometricBinding, unknown origins fail closed) wired into `ProcessDeviceEnrollment`; posture recorded on `DeviceTrustRecord`. Receipt side: `VerifyOfflineReceiptWithPolicy` delegates to existing crypto verification then enforces recorded posture (zero policy = legacy behavior). Offline Android chain verifier (`attestation_chain.go`, stdlib only): x509 chain to operator-provisioned Google roots (`LoadAttestationRoots` + `SetAttestationRoots`, nothing embedded from memory), KeyDescription checks (challenge==enrollment nonce, TEE/StrongBox level, GREEN boot, SIGN purpose), fail-closed enrollment with `AttestationVerified` audit flag. Offline Apple App Attest verifier (`apple_attest.go`, stdlib only, hand-rolled definite-length CBOR): x5c chain to operator-injected Apple root (`SetAppleAttestRoots`/`LoadAppleAttestRoots`), nonce `SHA256(authData||SHA256(challenge))` vs leaf 8.2 extension, rpIdHash vs SHA256(`SetEnrollmentExpectedAppID`); orchestrator-added hardening: definite lengths capped to unread buffer (crafted u64 length errors instead of panicking) + regression test. Dart dev simulator (`field_app/lib/security/attestation.dart`, no new deps): `SimulatedAttestationProvider` (fresh software Ed25519 per enrollment, const-only SOFTWARE origin — no code path can emit hardware origins), `buildEnrollmentSubmission` with byte-exact Go contract keys, `HardwareAttestationProvider` throwing native seam. Live round-trip proof (`field_app/tool/attestation_roundtrip.dart` + `pkg/onboarding/roundtrip_test.go`, `d20d1b2`): Go issues challenge → Dart signs → Go enrolls → asserts SOFTWARE/CLAIMED/unverified posture + tampered-nonce rejection; skips cleanly without Flutter. Gates: Go vet CLEAN, full `go test` PASS, race PASS; `flutter analyze` clean, `flutter test` 6/6 + regression files green. Open: operator provisions Google + Apple roots PEMs and ExpectedAppID at boot; device-side generation (Apple App ID $99/yr deferred); enrollment UI wiring of the simulator.
    *   ✅ **D3.2 closure (2026-09-09, big-pickle, $0, committed `0aa62c3`)**: boot provisioning `pkg/onboarding/provision.go` (env `INTEGIN_ATTEST_GOOGLE/APPLE_ROOTS_FILE`, `INTEGIN_APPLE_APP_ID`, fail-closed, all-or-nothing Apple pinning); loopback-gated `POST /enroll/challenge|submit` (`enroll_http.go`, `INTEGIN_PILOT_ENROLL_ENABLED`); Flutter `EnrollmentScreen` (idle/enrolling/success/error, SOFTWARE CLAIMED-unverified posture card, AppBar entry, default `:18080`) + 2 widget tests. Gates re-verified by orchestrator: vet clean, onboarding/server tests PASS, race PASS, analyze clean.
*   [x] **3.3: Bitemporal Merkle-CRDT Tamper-Proof Audit Ledger (`internal/domain/auditlog/`, `internal/auditlogpg/`)** — COMPLETE ✅ (2026-09-10, landed `dd091d8`):
    *   Sub-microsecond (<800ns write latency) 64-byte zero-allocation immutable event stream.
    *   Double-timeline recording: Transaction Time (when recorded) vs. Valid Time (when inspection occurred).
    *   🚀 **D3.3 progress (2026-09-10, big-pickle, $0)**: `Entry.ValidTime` + `ValidAt()` (zero→CreatedAt); `record.go` 64-byte canonical record, `BenchmarkAppendRecord` 443ns/op 0 allocs; migration `0081_audit_log_valid_time` (nullable valid_time + index, contract-registered); `auditlogpg` persists zero→NULL, `COALESCE(valid_time,created_at)` ValidFrom/To filters, NULL→zero mapping. Gates re-verified by orchestrator: vet clean, auditlog/auditlogpg/migrations tests PASS, race PASS. Committed `f1938b0` (hooks enabled).
*   [x] **3.4: W3C Decentralized Asset Passport & Technical Quarantine Lifecycle (`pkg/domain/models.go`, `pkg/domain/did.go`, `did:integin`)** — COMPLETE ✅ (2026-09-10, big-pickle, $0, landed `c9f6441`):
    *   Decentralized Identifier resolution (`did:integin:asset:<uuid>`).
    *   Autonomous safety quarantine: failed proof-load instantly locks asset state across all operational branches.
    *   🚀 **D3.4 progress**: `ResolveAssetDID` (parse→asset-type→lookup, typed errors) + proof-load verdict→`Quarantine()` (nil-safe) + tests. Gates re-verified by orchestrator: vet clean, domain tests PASS, race PASS, gofmt clean.
*   [x] **3.5: Universal Executive Onboarding & Physics Sandbox UI (`tools/onboarding-wizard/`)** — COMPLETE ✅ (2026-09-11, committed `4edb1c4` minimal 3-file static wizard; real-browser CDP smoke on Chrome Beta headless: valid `APEX-CRN-2026-00001` → preview shown + error hidden, `foo` → preview hidden + error shown, title correct, zero downloads):
    *   Web onboarding wizard (`index.html`, `style.css`, `app.js`) with regex token parsing and live certificate preview.
*   [ ] **3.6: TUS Chunked Resumable Media Streamer (`internal/storage/`) — `tus_handler.go` does not exist, create new**:
    *   Chunked 2MB upload protocol over weak offshore satellite VSAT with client-side AVIF/WebP downsampling.
*   [ ] **3.7: Offline Schema Drift & Version Negotiation (`internal/domain/sync/`) — `versioning.go` does not exist, create new**:
    *   `schema_epoch` handshake protocol allowing tablets offline for 30+ days to safely reconcile without data loss.
*   [ ] **3.8: RFC 3161 Courtroom Trusted Timestamping Authority — package does not exist, location TBD**:
    *   Embed RFC 3161 Timestamp Tokens (TST) in PDF/A-3b certificates to eliminate tablet backdating challenges.
*   [ ] **3.9: Mixed LTR/RTL Arabic/Latin PDF/A-3b Engine (`pkg/pdfrender/bidi.go`) — does not exist, create new**:
    *   HarfBuzz / ICU Unicode BiDi text shaping for certified bilingual Saudi (SASO/ZATCA) and UAE (ADNOC) certificates.
*   [ ] **3.10: ATEX Zone 1 Enclave PIN & Hardware Card Protocol (`field_app/lib/auth/`) — does not exist, create new**:
    *   Intrinsically safe tablet qualification with fallback enclave PIN and NFC smartcard tokens for greasy-glove field environments.
*   [ ] **3.11: Silent Duress PIN & Coercion Quarantine Protocol (`field_app/lib/auth/duress.dart`) — does not exist, create new**:
    *   Covert `STATE_COERCION_QUARANTINE` flagging protecting inspectors from physical coercion on isolated rigs.
*   [ ] **3.12: Forensic AI Inference Sealing (`internal/advisory/`) — `sealer.go` does not exist, create new**:
    *   Cryptographically hash model weights, prompts, and inference tensors into the Merkle ledger for judicial reproducibility.
*   [ ] **3.13: 2D Parametric Dynamic Blocks Lifting Simulator (`tools/lifting-simulator/2d/`) — does not exist, create new**:
    *   Interactive HTML5 Canvas & Flutter vector engine rendering plan/elevation views with reactive kinematic handles for major crane models (Liebherr, Tadano, Kato, Manitowoc).

### Sprint 4: Cloud-Native K8s Mesh & Universal QR Trust
*   [ ] **4.1: Stateless WebCrypto Browser Verifier (`tools/public-verifier/`, `pkg/verification`) — neither exists, create new**:
    *   Zero-backend-cost client-side public certificate verification via `#sig=...` URL fragment.
    *   Client-side Ed25519 signature validation and W3C DID document verification directly in browser WebCrypto API (`verify.integin.com`).
*   [ ] **4.2: Enterprise Kubernetes Helm Charts & Traefik Ingress (`config/k8s/helm/integin-platform/`, `terraform/`) — do not exist, create new**:
    *   High-availability pod auto-scaling (10,000 req/sec) with zero-downtime rolling upgrades.
*   [ ] **4.3: Air-Gapped Sovereign Edge Appliance Stack (`config/edge-appliance/`) — does not exist, create new**:
    *   Single-node offline container stack (`docker-compose.appliance.yml`) modeled after Coolify's Traefik dynamic labels.
*   [ ] **4.4: Dual-NVMe Air-Gapped Disaster Recovery (`deploy/edge-appliance/backup/`) — does not exist, create new**:
    *   Automated local `pgBackRest` WAL streaming to hot-swappable external rugged SSDs with $<60\text{s}$ rebuild script.
*   [ ] **4.5: Certificate Transparency Horizons (RFC 6962 Model, `pkg/verification/transparency.go`) — package does not exist, create new**:
    *   Public append-only Merkle transparency log preserving historical certificate validity across root CA rotations.
*   [ ] **4.6: Sovereign Cell-Based Multi-Region Sharding (`deploy/k8s/cells/`) — does not exist, create new**:
    *   Physical data plane pinning to sovereign regional cells (`cell-sa-central-01`, `cell-eu-west-01`) satisfying SDAIA and GDPR.
*   [ ] **4.7: Time-Bucket Table Partitioning & CQRS Replication (`migrations/0074_partitioning_and_cqrs.sql`) — 0073 already used by tool_calibration_registry**:
    *   Automated `pg_partman` weekly partitioning on append-heavy tables + PgCat read-replica routing eliminating XID wraparound.
*   [ ] **4.8: Dynamic Telemetric Sensor Jitter Verification (`pkg/rulesengine/jitter.go`) — does not exist, create new**:
    *   Harmonic micro-ripple frequency analysis and tool-to-enclave BLE pairing preventing counterfeit load cell spoofing.
*   [ ] **4.9: 3D WebGL Spatial Collision & 4D Temporal Tandem Lift Simulator (`tools/lifting-simulator/3d/`) — does not exist, create new**:
    *   Volumetric Three.js obstacle clearance, soil stress heatmaps, and time-stepped ($t_0 \rightarrow t_{\text{final}}$) dual-crane load-share simulation with 1-click execution binding.

---

## 5. 🔄 End-to-End Transaction Process Lifecycle ([GLOBAL_ARCHITECTURE_PLAN.md § 5](./docs/architecture/GLOBAL_ARCHITECTURE_PLAN.md#5-end-to-end-transaction-process))

```
1. Search & Discover (L1)  ──▶  2. 1-Click Manifest Binding (L4)  ──▶  3. Tablet Enclave Execution (L8)
   Natural-language query       Standard DIDs & formulas bound           Offline test executed; dynamic
   synthesizes ASME/ISO rules.   to Work Order package manifest.          CEL rules evaluate proof load.
                                                                                       │
4. Zero-Knowledge QR (L11) ◀──  5. 4-Eyes QA & Cert (L5)          ◀──  6. 64-Byte Merkle Append (L10)
   Browser verifies Ed25519     Immutable cert issued; public             Signed receipt synced; immutable
   sig with $0 backend cost.     binding snapshot generated.              Merkle-CRDT event appended.
```

---

## 6. 📐 Quantitative Quality Gates & Acceptance SLA Matrix ([GLOBAL_ARCHITECTURE_PLAN.md § 8](./docs/architecture/GLOBAL_ARCHITECTURE_PLAN.md#8-production-governance-quality-gates--acceptance-criteria))

| Technical Capability | Target Standard / Law | Quantitative Acceptance SLA | Verification Method |
|---|---|---|---|
| **Computational Nanocell Execution** | Google CEL Expression Sandbox | Strictly $< 50\mu\text{s}$ per evaluation; 0 memory escapes on hot path (`0 allocs/op`). | `go test -benchmem ./pkg/rulesengine/...` |
| **Sandbox Memory Ceiling** | Hard memory isolation | Strictly $\le 4\text{ MB}$ maximum heap allocation per execution nanocell. | Runtime memory profiler assertion |
| **Sandboxed I/O Isolation** | Zero-trust execution | Strictly 0 network calls, 0 disk I/O, 0 OS sub-processes during evaluation. | Sandboxed CEL environment AST check |
| **Standards Discovery Latency** | Copyright-Safe Metadata Index | Returns official standard metadata, active/superseded status in $< 30\text{ms}$. | Benchmark test suite |
| **Copyright Compliance** | Fair Use / Safe Harbor | 0 raw full-text paywalled PDFs stored; metadata citations & publisher deep-links only. | Storage & payload audit |
| **Idempotency Replay SLA** | Stripe-grade deduplication | Cached outcome replayed in $< 5\text{ms}$ with zero downstream database load. | HTTP concurrency benchmark |
| **Multi-Tenant RLS Penetration** | PostgreSQL 16 Table RLS | 100% hard-blocked (SQLSTATE 42501) across 5/5 adversarial penetration attack vectors. | `integin-live-matrix` security gate |
| **Audit Ledger Write Latency** | 64-byte Merkle-CRDT Stream | Sub-microsecond write latency ($< 800\text{ns}$); 100% ACID Monotonicity. | `go test -benchmem ./pkg/ledger/...` |
| **Test Quality Gate** | CI / Local Verification Gate | Every commit must pass `go test -count=1 ./...` (0 failures) and `go vet ./...` (0 errors). | Pre-commit validation gate |
| **Public QR Trust Cost** | Zero-Knowledge WebCrypto | Browser evaluates Ed25519 signature locally in client memory with $0 vendor compute. | Browser WebCrypto test |

---

## 7. 🔌 Active Runtime Infrastructure & Live Port Map

| Component | Container Name | Host Port | Target / Role / Credentials | Health Check |
|---|---|:---:|---|---|
| **PostgreSQL 16** | `integin-dev-postgres` | `15432` | DB: `integin_dev`, User: `integin_runtime`, Pass: `integin_live_run_2026` | `pg_isready -p 15432 -U integin_runtime` |
| **Keycloak IAM** | `integin-pilot-keycloak` | `18180` | App/Admin HTTP: `http://127.0.0.1:18180/admin/`, Token endpoint | `curl http://127.0.0.1:18180/realms/integin-pilot` |
| **Keycloak Metrics**| `integin-pilot-keycloak` | `19090` | Internal metrics & health only (`/health`, `/metrics`) | `curl http://127.0.0.1:19090/health` |
| **RustFS Object Store** | `integin-pilot-rustfs` | `19000` | S3 API endpoint, Bucket: `integin-pilot-evidence` | `curl http://127.0.0.1:19000/minio/health/live` |
| **RustFS Web Console** | `integin-pilot-rustfs` | `19001` | S3 Admin Web Console: `http://127.0.0.1:19001` | Browser navigation |
| **PgCat Pooler** | `integin-pgcat` | `6432` | Transaction connection pooler for high-throughput scaling | TCP connect |
| **integin-server** | Native Process | `18080` | Core API Server: `http://127.0.0.1:18080` | `curl http://127.0.0.1:18080/health` |

---

## 8. 🏛️ Governing Architectural References & Decisions

1.  **Master Global Architecture Plan**: [`docs/architecture/GLOBAL_ARCHITECTURE_PLAN.md`](./docs/architecture/GLOBAL_ARCHITECTURE_PLAN.md) — 12-Tier Worldwide Topology, Hybrid Intelligence Engine, and 4-Phase Transition.
2.  **Big Tech Silicon Valley Adoption Debate**: [`docs/architecture/INTEGIN_BIG_TECH_SILICON_VALLEY_ADOPTION_DEBATE_2026-09-07.md`](./docs/architecture/INTEGIN_BIG_TECH_SILICON_VALLEY_ADOPTION_DEBATE_2026-09-07.md) — Codification of Google CEL, Stripe Idempotency, 64-byte alignment, Apple Enclave, and WebCrypto.
3.  **Workspace Hygiene & Refactoring Governance**: [`docs/architecture/INTEGIN_WORKSPACE_HYGIENE_AND_REFACTORING_DEBATE_2026-09-07.md`](./docs/architecture/INTEGIN_WORKSPACE_HYGIENE_AND_REFACTORING_DEBATE_2026-09-07.md) — 4-Lens Governance Review resolving Items 6–12.
4.  **Master Plan & Sprint 2 Governance Debate**: [`docs/architecture/INTEGIN_MASTER_PLAN_AND_SPRINT_2_GOVERNANCE_DEBATE_2026-09-07.md`](./docs/architecture/INTEGIN_MASTER_PLAN_AND_SPRINT_2_GOVERNANCE_DEBATE_2026-09-07.md) — Multi-Lens Governance Review evaluating Google CEL, PostgreSQL idempotency, offline tablets, and copyright safe harbor.
5.  **Deep Architecture vs. Sprint 2 Execution Debate**: [`docs/architecture/INTEGIN_DEEP_ARCHITECTURE_VS_SPRINT_EXECUTION_DEBATE_2026-09-07.md`](./docs/architecture/INTEGIN_DEEP_ARCHITECTURE_VS_SPRINT_EXECUTION_DEBATE_2026-09-07.md) — Multi-Lens Governance Review establishing the integrated synthesis: formal deep specs followed by immediate Sprint 2 execution.
6.  **Deep Master 12-Tier Architecture Specification**: [`docs/architecture/GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md`](./docs/architecture/GLOBAL_ARCHITECTURE_DEEP_SPECIFICATION.md) — The aerospace/defense-grade specification detailing exact 64-byte Merkle structs, CEL grammar, 7-pillar sovereign schemas, and state machines.
7.  **Unplanned Infrastructure & Operational Blind Spots Debate**: [`docs/architecture/INTEGIN_UNPLANNED_INFRASTRUCTURE_AND_BLIND_SPOTS_DEBATE_2026-09-07.md`](./docs/architecture/INTEGIN_UNPLANNED_INFRASTRUCTURE_AND_BLIND_SPOTS_DEBATE_2026-09-07.md) — Multi-Lens Governance Review resolving the 8 industrial blind spots (offline schema drift, 2GB media choke, poison-pill DLQ, RFC 3161 timestamping, standards compiler).
8.  **User-Provided GitHub Reference Catalog**: [`docs/architecture/GITHUB_REPOSITORIES_REFERENCE.md`](./docs/architecture/GITHUB_REPOSITORIES_REFERENCE.md) — 24 cataloged reference repositories across tooling, scalability, edge computing, and security.
9.  **Deep Industrial Blind Spots & Visioned First-Principles**: [`docs/architecture/INTEGIN_DEEP_INDUSTRIAL_FORENSIC_AND_VISIONED_PRINCIPLES_2026-09-07.md`](./docs/architecture/INTEGIN_DEEP_INDUSTRIAL_FORENSIC_AND_VISIONED_PRINCIPLES_2026-09-07.md) — Complete 16-pillar blind spot defense and 6 first-principles physical trust engine.
10. **2D/3D/4D Lifting Plan Simulator Specification**: [`docs/architecture/INTEGIN_LIFTING_PLAN_SIMULATOR_2D_3D_4D_SPECIFICATION.md`](./docs/architecture/INTEGIN_LIFTING_PLAN_SIMULATOR_2D_3D_4D_SPECIFICATION.md) — Parametric dynamic blocks, ground bearing pressure, tandem kinematics, and 4D trajectory sealing.
11. **Lifting Plan Simulator Governance Debate**: [`docs/architecture/INTEGIN_LIFTING_PLAN_SIMULATOR_GOVERNANCE_DEBATE_2026-09-07.md`](./docs/architecture/INTEGIN_LIFTING_PLAN_SIMULATOR_GOVERNANCE_DEBATE_2026-09-07.md) — Multi-Lens Governance Review establishing scope-creep boundaries, dimensional tablet isolation (2D vs. 3D), geotechnical liability disclaimers, and strict sprint fencing.
12. **Multi-Agent Immunity & Governance Harness Debate**: [`docs/architecture/INTEGIN_MULTI_AGENT_IMMUNITY_HARNESS_DEBATE_2026-09-07.md`](./docs/architecture/INTEGIN_MULTI_AGENT_IMMUNITY_HARNESS_DEBATE_2026-09-07.md) — Multi-Lens Governance Review codifying the **40 Grand Hazards of Industrial AI Engineering**, deterministic CEL math vs. stochastic LLM drift, the "Rule of Three" anti-bloat protocol, 2-file root SSOT, non-blocking AI boundaries, and raw stdout test evidence verification.
13. **AI Token, Quota & Cost Optimization Guide**: [`docs/architecture/AI_TOKEN_AND_COST_OPTIMIZATION_GUIDE.md`](./docs/architecture/AI_TOKEN_AND_COST_OPTIMIZATION_GUIDE.md) — Definitive handbook for slashing agentic coding spend by 80%–95% via prompt caching, tiered model routing, surgical line-slices, and micro-sprint session resets.
14. **Token Cost Optimization Governance Debate**: [`docs/architecture/INTEGIN_AI_TOKEN_COST_OPTIMIZATION_GOVERNANCE_DEBATE_2026-09-07.md`](./docs/architecture/INTEGIN_AI_TOKEN_COST_OPTIMIZATION_GOVERNANCE_DEBATE_2026-09-07.md) — Multi-Lens Governance Review establishing disk-backed memory invariants, frontier tier-fencing for life-safety Go logic, and pointer-based output transparency.

---

## 9. 📊 Verified Progress Log & Provenance Trail

### 2026-09-07 Verified Milestones

#### Deliverable 2.1: Computational Nanocell (`pkg/rulesengine`) — IMPLEMENTED
*   Added `cel.dev/cel-go v0.32.0` to `go.mod` (canonical module path; `github.com/google/cel-go` retired).
*   Delivered `schema.go`, `versioning.go`, `evaluator.go`, `asme_b30_5.go`, `iso_4309.go`, `rigging.go`, `geotech.go` + 16 unit tests + 3 benchmarks.
*   Verification: `go test -count=1 ./...` **PASS (0 failures)**, `go vet ./...` **CLEAN**. Benchmarks: B30.5 gate **271ns/op 0 allocs**, ISO4309 gate **227ns/op 0 allocs**, Rigging **46ns/op 0 allocs** — all under the `<50µs` SLA with `0 allocs/op`.
*   `go test -race` BLOCKED on host (no `gcc`, cgo disabled by design — Hazard 40).

#### Live Matrix & Authentication Repair
*   Restored `postJSONAuth` as a standalone authenticated HTTP client function in [`cmd/integin-live-matrix/main.go`](file:///c:/MY_PROJECT/integin-pilot-source/cmd/integin-live-matrix/main.go).
*   Added 11 unit tests in [`cmd/integin-live-matrix/main_test.go`](file:///c:/MY_PROJECT/integin-pilot-source/cmd/integin-live-matrix/main_test.go) guarding `postJSON`, `postJSONAuth`, and `localConfig`.
*   Verification: `go test -v ./cmd/integin-live-matrix` $\longrightarrow$ **PASS (11/11, 0.071s)**, `go vet` clean.
*   Provisioned Keycloak `integin-pilot` realm, client `integin-live-matrix`, and verified JWT token acquisition.
*   Seeded PostgreSQL `identity_subject` and `identity_membership` mapping for matrix service account.
*   Live Matrix Execution: `seed` $\longrightarrow$ **PASS**; `exercise` $\longrightarrow$ **100% PASS** (`/sync`: `APPLIED`, `DUPLICATE`, `HELD`, `CONFLICT`, `SECURITY_FAILURE`; `/evidence`: `APPLIED`, `DUPLICATE`, `CONFLICT`, `SECURITY_FAILURE`).
*   Chaos & Security Proof: FoundationDB chaos simulation (120/120 mutations reconciled, **PASS**), PgCat connection pooling (450 txns, zero GUC leak, **PASS**), Adversarial RLS penetration (5/5 attacks blocked, **PASS**).

#### Documentation SSOT Consolidation & Multi-Agent Protocol
*   Merged 10 root markdown files down to exactly 2 canonical files:
    *   [`c:\MY_PROJECT\WORKSPACE.md`](./WORKSPACE.md) — Master permanent reference.
    *   [`c:\MY_PROJECT\TRACKER.md`](./TRACKER.md) — Master active sprint tracker.
*   Preserved pre-merge backup snapshots in `archive/planning-history-august-2026/pre_merge_backup/`.

#### Comprehensive 12-Item Workspace Refactoring & Git Commits
*   **Commit `3c5dadd`**: Committed live matrix fixes, 11 unit tests, and redirect tracking.
*   **Commit `686515e`**: Relocated loose `.exe` binaries to `bin/`, purged `.pytest_cache/`, and updated `.gitignore`.
*   **Commit `6190336`**: Sanitized candidate migration (dropped phantom table indexes on `webhook_deliveries` and `shortlinks`, and dropped nonexistent `status` column on `certificate_snapshot`), consolidated 7 verified indexes into `migrations/0070_add_certificate_performance_indexes.sql`, removed `db/` directory, aligned runbook ports (`19000`, `18080`, `15432`), and standardized `quiet-signal` on `pnpm-lock.yaml`.
*   **Branch Hygiene**: Safely deleted fully merged stale feature branch `feature/bigtech-chaos-hardening` (was `8a61073`). Single authoritative branch is `master`.

#### Ponytail Adoption for Relay Implementers (2026-09-08)
*   Installed `@dietrichgebert/ponytail@4.9.0` globally (machine-level, no repo dependency).
*   Adapted reuse ladder as `integin-pilot-source/AGENTS.md` §2b — mandatory for all relay implementers (`opencode/big-pickle`, `opencode/mimo-v2.5-free`): YAGNI → reuse → stdlib → smallest gated diff; guards/tests never cut. Direct response to the 89-file `gofmt -w` scope creep incident.
*   Whole-OpenCode adoption: `"plugin": ["@dietrichgebert/ponytail"]` added to global `~/.config/opencode/opencode.jsonc` (JSON-validated, `opencode --version` 1.18.29) — every OpenCode session now injects the ruleset; relay briefs additionally cite the ladder per-task.
*   **2026-09-08 wshobson triage (owner-decided)**: INSTALLED `go-concurrency-patterns` + `sql-optimization-patterns` into `.agents/skills/` (installer `skills-lock.json` removed to preserve 2-file root invariant). REJECTED `postgresql-table-design` — schema and multi-tenant RLS design are strictly anchored in `WORKSPACE.md` + `agent-immunity-harness.md`; an external generic skill invites schema drift. `block-no-verify-hook` HELD for OpenCode clarification (see below).
*   **OpenCode verdict on `block-no-verify-hook`** (read-only relay `ses_f8220dd07ffe3PgN9Ipf5TaNv2`, big-pickle, $0): (1) No bundled hook — skill instructs installing a Claude-Code `PreToolUse` hook (matcher `Bash`) scanning `$TOOL_INPUT` for `--no-verify`. (2) Runtime is a POSIX `sh`+`grep -qE` one-liner; engine is Claude Code's hook runner. (3) Fully inert under headless `opencode run` (never reads `.claude/settings.json`) and bypassed by real CI/GUI git — fail-nothing, zero protection; regex risks false-positive blocks. Recommendation: gate `--no-verify` via real git hooks, not this skill. NOTE: read-only run left cosmetic traces on `integin-pilot-source/AGENTS.md` (BOM strip + trailing newlines; content intact) — `--read-only` is advisory, not a sandbox; verify diff after every relay.
*   **Clean Working Tree**: `master` branch is up to date with `origin/master`, `0` uncommitted changes.

---

## 10. 💡 Architectural Findings & Discoveries

### 1. Keycloak Management vs. Application Ports
*   **Port 19090**: Dedicated solely to Keycloak internal management & metrics (`/health`, `/metrics`). Navigating here in a browser shows an empty or basic health page by design.
*   **Port 18180**: The actual HTTP application and admin console port:
    *   Admin UI: `http://127.0.0.1:18180/admin/`
    *   Token endpoint: `http://127.0.0.1:18180/realms/integin-pilot/protocol/openid-connect/token`

### 2. PostgreSQL RLS & `integin_runtime` Permissions
*   When executing against `integin_dev`, the `integin_runtime` user requires explicit table and sequence grants on `sync_device_state`, `sync_receipt`, `sync_held_transaction`, and `authority_package`.
*   All queries must set session GUCs using `SELECT set_config('integin.tenant_id', $1, true), set_config('integin.organization_id', $2, true)` to pass table RLS policies.
*   The identity resolution function `integin_resolve_identity_membership(issuer, subject)` is marked `SECURITY DEFINER` so the runtime role can authenticate tokens without broad direct grants on `identity_subject` or `identity_membership`.

### 3. Big Tech Nanocell & Google CEL Suitability
*   Traditional Wasm runtimes (Wasmer, Wasmtime) introduce excessive binary size (20MB+) and startup overhead for rugged mobile tablets.
*   **Google CEL (`cel.dev/cel-go`)** is non-Turing complete, has zero-allocation memory pools, evaluates in $< 15\mu\text{s}$, and provides compile-time type-safety for ASME B30.5 / ISO 4309 formulas with zero server recompile.

### 4. PostgreSQL HOT (Heap-Only Tuple) Optimization
*   Updating rows in append-heavy tables (`sync_receipt`) normally causes B-tree index splits and flash wear.
*   Configuring `WITH (fillfactor = 85)` reserves 15% free space in each 8KB disk page, allowing in-place HOT updates that completely bypass B-tree re-indexing.

### 5. Migration Index Validation & Phantom Schema Hazard
*   A candidate migration in `db/migration/` had attempted to create indexes on `webhook_deliveries` and `shortlinks`, plus a `WHERE status ...` filter on `certificate_snapshot`.
*   Direct schema introspection against PostgreSQL 16 in `integin_dev` revealed that neither table existed, and `certificate_snapshot` lacked a `status` column.
*   Running an isolated dry-run in a transactional block (`BEGIN ... ROLLBACK`) blocked a potential production migration failure, resulting in sanitized `migrations/0070_add_certificate_performance_indexes.sql` containing only 7 verified indexes.

---

## 11. 🎯 Immediate Execution Command: Sprint 3 Active Frontier

With Sprint 2 (D2.1–D2.5) 100% complete and Sprint 3 **D3.1–D3.5 complete** (D3.1 closure `cc742ef`, D3.2 closure + D3.3 ledger landed `dd091d8`, D3.4 passport landed `c9f6441`, D3.5 minimal wizard + real-browser CDP smoke landed `4edb1c4`/`dd9ddb5`), the active frontier is:

1. **Deliverable 3.6: TUS Chunked Resumable Media Streamer (`internal/storage/tus_handler.go`)**:
   ```powershell
   # Target: new internal/storage/ package, chunked 2MB upload over VSAT
   ```

---

## 12. 📱 Live-Device Session Notes (2026-09-09, Xiaomi Mi 9, `59af14c0`)

* SDK installed to `C:\Android` (cmdline-tools + platform-tools + API 35/36 + JDK 21 Temurin at `C:\Android\jdk21-home` — JDK 26 cannot build AGP projects; NDK/cmake removed after use). Licenses accepted via ConPTY (sdkmanager ignores piped stdin on Windows; memorized hashes are stale).
* Docker Desktop died (disk pressure) taking postgres/keycloak/rustfs down; restarted daemon + `docker start` in dependency order. Pilot server restarted reusing `integin-server-pilot.exe` with `INTEGIN_LOCAL_PROVISIONING_ENABLED=true`, org `org-phone-01`, user `inspector-phone` (canonical launcher untouched).
* field_app fixes landed: `Idempotency-Key` header on sync POSTs (`11ac22e` — server 400s keyless posts); APK built with `INTEGIN_SYNC_ENDPOINT` + `INTEGIN_LOCAL_PROVISIONING_ENDPOINT` defines + `adb reverse tcp:18080`.
* Phone provisioned for real: `field-1643bfc6…` / inspector-phone with server-issued authorities (verified in `device_registry` + `authority_package` with tenant GUCs set session-wide — note: `set_config(...,true)` is transaction-local, one `psql -c` per statement loses it).
* Debugging is self-serve: logcat (`adb logcat -d -s flutter:E`), curl-from-phone, phone screenshots via `screencap`+pull, server request log at `operations/pilot/runtime/server.stderr.log`.
* Gotcha: stale demo-session outbox entries fail forever (403 unregistered authority) and latch `blocked`; fix is `pm clear` for a clean provisioned slate.

### IAM review follow-up — Zitadel read-only spike (2026-09-09, v4.17.3, PASS)

* Free to self-host confirmed in practice: Apache 2.0 image `ghcr.io/zitadel/zitadel:latest` (**234MB** vs Keycloak's 766MB), runs against any Postgres, no license/account.
* Spike (fully removed after): throwaway pg16 + `init`/`setup` (needs 32-byte `ZITADEL_MASTERKEY`, `--tlsMode disabled` for localhost) → `/.well-known/openid-configuration` live: standard authorize/token/userinfo/jwks, `client_credentials` + `device_code` grants, `private_key_jwt`, EdDSA/ES256/RS256. Covers everything the live-matrix + app flows need.
* Idle footprint: **~90MB RAM** (Keycloak typically 5–10× that).
* Verdict: viable Keycloak replacement when F3 triggers (appliance freeze or resource incident). No migration authorized; image kept locally for re-eval, all spike containers/network/DB removed.

