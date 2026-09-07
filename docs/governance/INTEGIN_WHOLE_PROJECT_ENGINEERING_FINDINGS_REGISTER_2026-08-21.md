# INTEGIN Whole-Project Engineering Findings Register

**Date:** 2026-08-21
**Scope:** Full-workspace read-only engineering inspection of `C:\MY PROJECT` (excluding `private\`, which was never accessed per boundary rules).
**Method:** Source-level review of the Go backend (`integin-pilot-source`, 234 Go files), Flutter field app (`field_app\lib`), PostgreSQL migrations (applied + candidates), PowerShell runtime wrappers (`operations\pilot\runtime`), AI advisory service (`ai_service`), deploy kit, contracts/vectors, root continuation records, and runtime artifact forensics under `operations\acceptance`.
**Verification performed:** `go vet ./...` — PASS (exit 0) on the working tree including uncommitted changes. No tests were executed; no files were modified; no runtime was started or stopped.

> This record is non-secret. It contains no private values, credentials, database URLs, keys, or protected fixture material.

---

## ⚠️ VERIFICATION ADDENDUM — 2026-08-21 (same day, post-export owner challenge)

The owner challenged whether anything was missed or skipped. A personal re-verification pass over the primary sources **refuted four findings** below that had been accepted from an unverified secondary analysis. This is exactly the residual-risk scenario documented in `ENGINEERING_CONTINUATION_GUIDE.md` §7.1. Corrections:

| ID | Status | Verified truth (source read directly) |
| --- | --- | --- |
| F-01 | **REFUTED** | `internal\domain\sync\sync.go` dedup switch explicitly checks `receipt.Outcome`: `Applied`→Duplicate, `Held`→falls through the sequence gate ("so a later replay can be applied exactly once"), default→Conflict. The HELD→DUPLICATE wedge does **not** exist in the current working tree. |
| F-02 | **REFUTED** (processor level) | The HELD path calls only `persistHeld`, which builds the receipt and held envelope and issues a single repository call (`SaveHeld`). There is no two-call receipt-then-held sequence. Residual question (downgraded): atomicity *inside* `syncstate.SaveHeld` was not separately re-verified. |
| F-06 | **REFUTED** | The algorithm switch has an explicit `default:` returning SECURITY_FAILURE "transaction signature algorithm is not supported". Only exact `Ed25519` and the legacy HMAC constant are accepted. |
| F-07 | **REFUTED** | Durable/in-memory dedup lookups execute **after** device registration, tenant binding, algorithm validation, authority-package validation, and signature verification. No unauthenticated existence oracle. |

**Personally confirmed as stated:** F-03 (escaping bug verbatim at `workorderpg/postgres.go` `stringArray.Value()`), F-04 (FORCE ROW LEVEL SECURITY grep: present only in `0003` + all candidate files; absent in `0002`, `0005_work_package_persistence`, `0007`, `0004`), F-11 (processor-wide mutex spans DB calls), F-13 (assignment transfer UPDATE without state precondition or RowsAffected check), F-15 (no `.timeout(` in sync/evidence transports; only package-manifest transport has one), F-19 (`ListHeld` referenced only by tests/interface/impl; `Queued` declared but never produced).

**Inventory corrections:** Go totals are **254 files / 132 test / 122 src** (register body said 234/121/113); migration inventory extends to **0011/0012**: `0011_evidence_metadata_encryption_export.candidate.sql` adds immutable encryption-provenance columns; `0012_certificate_template_binding_registry.candidate.sql` (created 2026-08-21 evening) lands the fixed-layout certificate-designer foundation — page geometry in points, cell kinds TEXT/CHECKBOX/DATE/STATIC_TEXT/REPEATING_REGION, fit policies, pairwise CHECK constraints for conditional rules, dual FORCE RLS. Additional small verified notes: the persistent outbox store also deduplicates by transactionId (dedupe is not memory-only); `SecureKeyValueStore` performs no error handling; evidence crypto uses no AAD and no explicit key-length pre-check; the Field live acceptance test is env-gated and skipped by default.

**Lesson recorded per guide §7:** secondary analyses must be treated as leads until spot-checked; SEV1/SEV2 claims in particular require eyes on source before entering any register.

---

## Severity definitions

| Level | Meaning |
| --- | --- |
| SEV1 | Correctness / data-integrity defect reachable in normal operation |
| SEV2 | Security-hardening gap (defense-in-depth or fail-closed discipline) |
| SEV3 | Robustness, performance, or API-semantics weakness |
| SEV4 | Hygiene, process, or repository-cleanliness item |

---

## SEV1 — Correctness / Data Integrity

### F-01 — HELD→DUPLICATE wedge in sync processor
- **Location:** `integin-pilot-source\internal\domain\sync\sync.go:148-154`
- **Finding:** Duplicate detection compares only `receipt.PayloadHash` and ignores `receipt.Outcome`. A transaction that was once persisted as `HELD` (never applied) will forever return `DUPLICATE` ("already applied") on retry.
- **Impact:** Silent data loss: any submission that ever entered HELD can never be applied afterward.
- **Recommendation:** Include outcome in the dedup predicate; only `APPLIED` receipts satisfy duplicate detection. Add a regression test: submit with sequence gap → HELD → fill gap → retry original → must become APPLIED (or CONFLICT by explicit policy), never DUPLICATE.

### F-02 — Non-atomic receipt + held-envelope persistence
- **Location:** `sync.go:245-252` (two separate repository calls); `internal\syncstate\postgres.go`
- **Finding:** The processor saves a HELD receipt and then the held envelope in two separate transactions. A crash between them leaves an orphan HELD receipt.
- **Impact:** Amplifies F-01 into a permanent wedge; also leaves inconsistent durable state.
- **Recommendation:** Persist receipt + held envelope atomically in one transaction, or add compensating cleanup.

### F-03 — Malformed PostgreSQL array literal escaping
- **Location:** `integin-pilot-source\internal\workorderpg\postgres.go:583-589`
- **Finding:** `stringArray.Value()` escapes `"` as `\\\"` (backslash-backslash-quote) instead of the PG-correct `\"`, and never escapes backslashes.
- **Impact:** Values containing `\` or `"` produce malformed/misparsed array literals used by the `?|` duplicate-inclusion check and `= ANY(...)` lookups — wrong results in Work-Order partial-submission safeguards.
- **Recommendation:** Replace with `pq.Array` or correct PG array-literal escaping; add round-trip unit tests including quote/backslash payloads.

### F-04 — Applied migrations lack FORCE ROW LEVEL SECURITY
- **Location:** `migrations\0002_device_trust_sync.sql`, `0005_work_package_persistence.sql`, `0007_manifest_proof_replay.sql`
- **Finding:** RLS is ENABLED but not FORCEd on applied tables (all candidate migrations 0005wo/0008/0009/0010 correctly use FORCE). Table-owner sessions bypass tenant isolation on the applied tables.
- **Impact:** Tenant isolation depends solely on connecting-role discipline (`RLS_RUNTIME_ROLE.md` mitigates operationally, but the schema boundary is weaker than designed).
- **Recommendation:** Draft a reviewed additive migration enabling FORCE RLS on the applied tables after verifying the runtime role is non-owner; keep parity with the candidate migrations.

### F-05 — Field sequence allocation vs crash window
- **Location:** `field_app\lib\application\field_app_controller.dart:148` (sequence increment) and `restoreOutbox` (:61-83)
- **Finding:** `_sequence` advances before outbox append; recovery rebuilds sequence state only from persisted outbox entries. A crash between mutation construction/signing and persistent append could desync the local sequence, producing a permanent gap (which interacts with F-01 server-side).
- **Impact:** Low-probability but permanent HELD wedge on that device.
- **Recommendation:** Derive/commit sequence atomically with the outbox entry write; verify ordering guarantees between `_outbox.add` and `outboxStore.append`.

---

## SEV2 — Security-Hardening Gaps

### F-06 — Unknown signature algorithms silently routed to HMAC
- **Location:** `sync.go:196`
- **Finding:** Any `SignatureAlgorithm != "Ed25519"` (including garbage values such as `"FOO"` or `"hmac-sha256"`) falls through to the legacy HMAC verification branch instead of being rejected.
- **Impact:** Algorithm-confusion surface; the shared HMAC secret becomes a verification oracle for arbitrary algorithm labels.
- **Recommendation:** Accept only exact `Ed25519` or exact legacy constant; reject everything else as SECURITY_FAILURE.

### F-07 — Existence oracle before authentication
- **Location:** `sync.go:148-167`
- **Finding:** Durable and in-memory dedup lookups execute before device-registration and signature checks.
- **Impact:** Unauthenticated callers can probe transaction-ID/hash pairs (existence/conflict oracle) without presenting valid credentials.
- **Recommendation:** Reorder so identity/device/signature verification precedes any dedup lookup, or make dedup responses indistinguishable until authenticated.

### F-08 — Unescaped pipe-delimited canonical strings
- **Locations:** `sync.go:271`; `domain\workpackage\manifest_canonical.go:39-74`; `domain\sync\device_proof.go` canonical proof; Dart mirrors (`transaction_signer.dart:22-40`, `package_manifest_client.dart:75-108`)
- **Finding:** Canonical signing strings join fields with literal `|` (and `,`/`=` inside assignment context pairs) with no escaping or length-prefixing.
- **Impact:** Theoretical signature-collision vector if any component field ever contains delimiter characters. Currently mitigated because most fields are server-controlled identifiers, but the invariant is not enforced anywhere.
- **Recommendation:** Either enforce a "no delimiter in field" validation at every constructor, or migrate canonicalization to length-prefixed/JSON-canonical encoding in a future contract version (vectors would need regeneration).

### F-09 — Manifest signed before replay consumption; error masking
- **Location:** `internal\packagemanifestapi\http.go:113-128`
- **Finding:** Issuance signs the manifest before `ReplayStore.Consume`; replayed requests waste an Ed25519 sign + manifest build (result discarded on 409). Generic Consume errors map to 401 `expired`, masking DB outages as auth failures.
- **Impact:** Minor DoS amplification and misleading telemetry; security-safe (manifest never emitted on conflict).
- **Recommendation:** Consume replay key before signing; map infrastructure errors to 5xx distinct from auth reasons.

### F-10 — SECURITY DEFINER resolver privilege boundary
- **Location:** Migration `0009_identity_work_order_actor.candidate.sql` (resolver function)
- **Finding:** `integin_resolve_identity_membership` necessarily bypasses RLS via owner rights; exposure is bounded by EXECUTE grants (currently only `integin_pilot_runtime`) and `SET search_path` hardening is present.
- **Impact:** Accepted design; recorded here as a standing invariant to preserve.
- **Recommendation:** Keep grant list minimal; add an adversarial test asserting other roles cannot EXECUTE.

---

## SEV3 — Robustness / Performance / Semantics

| ID | Location | Finding | Recommendation |
| --- | --- | --- | --- |
| F-11 | `sync.go:138-139` | Global mutex held across all DB I/O serializes every submission process-wide; DB latency multiplies lock hold time. | Narrow the critical section or shard by device/tenant. |
| F-12 | `application_service.go`; `workorderpg/postgres.go:451,568-575` | Idempotency request hash = `sha256(json.Marshal(command))` — struct-field reorder silently invalidates all historical idempotency keys across deploys. `findReceipt`'s hash parameter is dead code. | Use explicit stable canonical payload hashing; remove dead param. |
| F-13 | `workorderpg/postgres.go:320` | `ReassignScope` marks previous assignment `transferred` with no state precondition or RowsAffected check — can transition already-completed/revoked assignments. | Add `WHERE state='active'` + RowsAffected assertion. |
| F-14 | `domain/workorder/workorder.go` + migration 0005wo | Commercial and certificate state machines are dead vocabulary (no transition guards); all four state columns stored as unconstrained TEXT in DB. | Implement guards or reduce vocabulary; add CHECK constraints when states stabilize. |
| F-15 | `http_sync_transport.dart`; `evidence_upload.dart` | No HTTP timeout configured on either transport; evidence retry performs 3 attempts with zero backoff delay. | Add explicit timeouts and bounded backoff. |
| F-16 | `workorderhttp/http.go:78-87` | `sql.ErrNoRows` mapped to 403 conflates missing aggregate with denial; `ErrStaleRevision` flattened to generic 400 — clients cannot distinguish conflict from malformed input. | Introduce distinct status codes per domain error class. |
| F-17 | `syncstate/postgres.go:206-207`; epoch comparisons in enforcement/issuer | `uint64` scanned from BIGINT and int64↔uint64 epoch conversions alias for values ≥ 2⁶³. | Constrain epochs at DB level (CHECK within BIGINT range) and scan through int64. |
| F-18 | `workorderpg/postgres.go:533-540` | Assignment-scope inserts loop row-by-row (N round trips per command). | Multi-values INSERT or unnest. |
| F-19 | `sync.go:28`; `syncstate/postgres.go:291` | `QUEUED` outcome declared but never produced; `ListHeld` has zero production callers — there is no replay/recovery path for held transactions at all. | Design the governed held-transaction drain path (server-initiated, authorized) or document held as terminal-by-policy. |

---

## SEV4 — Hygiene / Process

| ID | Finding | Recommendation |
| --- | --- | --- |
| F-20 | Two junk files at repo root: `53ceeed0…​ pilot-current-schema.dump` and `e3b0c442…​ pilot-current-schema.dump` — both are 106-byte stray path strings pointing at `.sha256` checksum files from the 2026-08-21 Stage A drill (the second filename is the SHA-256 of the empty string). Not actual dumps. | Delete both; locate the writer that emitted paths instead of digests. |
| F-21 | ~279 lines uncommitted across 11 modified files plus entire untracked Work-Order slice (`internal\domain\workorder`, `workorderpg`, `workorderhttp`, `workorderauth`, `workorder_composition*.go`, migration candidates 0005wo/0008/0009, repo-local docs). Coherent, vet-clean slice. | Commit per repository reviewed-commit policy (separate commits per subsystem recommended). |
| F-22 | LF→CRLF conversion pending on touched files (git warnings). | Normalize via `.gitattributes` policy commit. |
| F-23 | Duplicate migration numeric prefix: two distinct `0005_*` files (applied work-package + candidate work-order foundation). | Rename candidate series (e.g., 0011+) or adopt explicit ordering manifest before tooling sorts wrongly. |
| F-24 | Orphaned/legacy operational artifacts: v1 receipt adapter has no caller; legacy pair `start-pilot-policy-candidate.ps1` + `run-pilot-policy-matrix.ps1` lacks the hardened boundary controls (no digest pinning, ACL hardening, PID binding, acceptance preflight). | Mark deprecated or port to hardened launcher pattern. |
| F-25 | `workpackage.ValidateSubmission` iterates maps for error selection — nondeterministic error choice when multiple violations exist. | Iterate ordered field IDs for deterministic diagnostics. |

---

## Positive assurance findings (verified sound)

- **Cross-language crypto parity:** Go/Dart canonicalization byte-identical; pinned deterministic Ed25519 vectors (`contracts\vectors\v1`) verified in both languages; keyId = hex(sha256(pubkey)); documented Go-64-byte vs Dart-32-byte seed interop handling.
- **Fail-closed composition:** every optional route/policy nil-unless-multi-gated (manifest route, policy registration, local provisioning, OIDC); OIDC disabled, package enforcement disabled, OpenBao sealed/unwired in persistent runtimes.
- **Tenant isolation pattern:** transaction-local GUCs (`set_config('integin.tenant_id', …, true)`) consistently applied through `scopedTx`/`beginTenant`/`begin()` across all repositories.
- **Durable monotonicity:** guarded counter bump (`WHERE last_accepted_sequence + 1 = $1`, exactly-1-row required) implements correct CAS semantics.
- **Receipt writers:** three-layer duplicate protection (in-memory set + Lstat probe + atomic temp-file/hard-link publish), 8 KiB bounds, digest-only identifiers, closed vocabularies, v2 structural validation with per-case HTTP-status tables.
- **Runtime wrapper boundary:** SHA-256 source pinning, per-component trusted-path walks (UNC/reparse/unsafe-ACL rejection), SID-based broad-write detection with InheritOnly exclusion, cleared child environments, PID↔listener binding, mandatory restore pass and shutdown, acceptance 200/200 preflight/postcheck.
- **Test density:** 121 test files vs 113 source files; `go vet ./...` clean on working tree.
- **AI boundary:** enforced at three layers (Python service zone allowlist/denylist, Go client pre-flight rejection, Flutter blocking=true rejection).

---

## Suggested remediation order

1. F-01 + F-02 (HELD wedge + atomicity) — correctness core
2. F-03 (array escaping) — active safeguard corruption risk
3. F-04 (FORCE RLS candidate migration)
4. F-06 + F-07 (algorithm pinning + authz ordering)
5. F-21 (commit the Stage 0 slice per reviewed-commit policy)
6. F-20, F-22, F-23 (repository hygiene)
7. Remaining SEV2/SEV3 items as scheduled design work

---

## Verification commands used

```
go vet ./...            # exit 0 (working tree)
git log --oneline       # 72 commits, single master branch
git diff --stat         # 11 modified files, +279/-8
git status --short      # untracked Work-Order packages + migration candidates
```

*End of register.*
