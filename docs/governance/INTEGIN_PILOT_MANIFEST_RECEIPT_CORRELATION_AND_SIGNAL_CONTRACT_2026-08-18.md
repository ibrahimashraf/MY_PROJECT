# INTEGIN Pilot Manifest Receipt Correlation and Signal Contract

**Decision date:** 2026-08-18  
**Status:** Proposed hard-gate contract pending bounded independent review.  
**Scope:** Source-derived, public-redacted correlation and the eight public manifest receipt cases.  
**Non-authorization:** This document does not authorize candidate emitter composition, fixture inspection, candidate execution, package enforcement, acceptance changes, OIDC enablement, or OpenBao wiring.

## 1. Purpose

The public receipt adapter already validates eight source-redacted case receipts, but it cannot establish that a receipt belongs to the wrapper invocation that produced it. The current wrapper creates a fresh lower-case 32-hex `candidate_run_id` but does not yet make that identifier available to the candidate or Field-side evidence source. The source receipt writer can validate that identifier and atomically publish a single receipt per case, but it does not decide how a candidate run is correlated or which source path owns each case.

This contract defines the **minimum safe correlation boundary**. It preserves the protected fixture as opaque and forbids correlation based on fixture stdout/stderr, fixture payloads, private values, identity fields, proofs, manifest contents, or inferred database data.

## 2. Current source facts

| Source boundary | Existing safe fact | Limitation that the contract resolves |
| --- | --- | --- |
| Public wrapper | Generates a fresh 32-hex `candidate_run_id`, verifies acceptance at `/healthz` and `/readyz`, enforces exclusive `127.0.0.1:18080` ownership, suppresses fixture output, and always stops the candidate. | The generated run ID is not propagated to candidate/Field receipt producers. |
| `manifestreceipts.Writer` | Validates run ID and directory, allow-lists public cases/outcomes, applies bounded state arithmetic, hashes manifest IDs, writes atomically, and rejects duplicate filenames. | Its v1 shape requires an HTTP status and cannot honestly represent a Field binding event with no HTTP transport. |
| HTTP handler observation | Emits redacted outcome, reason, HTTP status, manifest ID, and replay acceptance; it excludes proof, signature, payload, key, identity, and credentials. | It currently has no candidate-run context, receipt sink, or explicit state-count scope. |
| Field binding observation | Emits only `verified_cached` after successful verify/bind/cache; it contains no manifest, identity, key, proof, or endpoint fields. | It has no nonsecret run-context bridge and no HTTP response status. |

## 3. Core correlation decision

The wrapper remains the only run-ID generator. Before it launches the candidate, it must create a fresh, empty, ACL-hardened public receipt directory scoped to that run and set the following **nonsecret, process-scoped** values for both the candidate and the invoked fixture:

| Environment name | Meaning | Validation/handling |
| --- | --- | --- |
| `INTEGIN_PILOT_MANIFEST_CANDIDATE_RUN_ID` | Wrapper-generated 32-hex run ID. | Must match `^[a-f0-9]{32}$`; no caller may supply it. |
| `INTEGIN_PILOT_MANIFEST_RECEIPTS_DIR` | Fresh wrapper-created receipt directory for that run. | Must be a regular, non-symlink directory with restrictive ACLs; never reuse a directory. |
| `INTEGIN_PILOT_MANIFEST_RECEIPT_CONTRACT_VERSION` | Public schema/bridge version. | Must be explicitly recognized; unknown versions fail closed. |

The wrapper restores or clears the process-scoped values after the invocation, stops the candidate in `finally`, verifies acceptance again, and supplies the same wrapper status record to the read-only adapter. Neither the public run ID nor receipt directory is a secret; they are correlation handles only. Their safety comes from wrapper generation, exclusive process lifecycle, source-derived output, and ACL/atomic publication controls—not from secrecy.

## 4. Receipt contract v2 decision

The existing v1 receipt schema is insufficient for a trustworthy Field receipt because `field_binding` has no HTTP response. It is also ambiguous when `0/0` counts are used on an invalid-proof path with no verified tenant scope. Therefore, the next reviewed source change must introduce **public receipt contract v2** rather than falsifying an HTTP status or presenting placeholder counts as an aggregate state measurement.

| v2 element | Rule |
| --- | --- |
| `contract_version` | Fixed to `2`; v1 remains historical adapter evidence only. |
| `event_source` | Required public enum: `manifest_http` or `field_binding`. |
| `transport` | Required object. For `manifest_http`, it carries the observed HTTP status. For `field_binding`, it declares `kind=field_binding` and carries no invented HTTP status. |
| `replay_state` | Required object with `scope=verified_tenant` plus `before/after/delta`, or `scope=not_applicable` with no numeric count. |
| `candidate_run_id`, `case`, `expected_outcome`, `observed_outcome`, `status`, `generated_at` | Retained, public, schema-bounded fields. |
| `correlation_id` | Optional opaque source-generated reference only; it must not be a proof request ID, manifest body, user/device identifier, key, token, or auth session identifier. |
| `manifest_id_digest` | Optional SHA-256 digest only; never a raw identifier. |

The public adapter must be upgraded in the same reviewed change to reject v1/v2 mixing within a candidate run, reject unknown bridge diagnostic artifacts, and require the expected event-source/transport/state-scope combination for each case.

## 5. Eight-case source-signal matrix

Only the following terminal source observations may create receipts. All other handler observations, including malformed requests, unconfigured service, assignment absence, or issuance failures, are diagnostic only and **must not** create a public case receipt.

| Public case | Source terminal observation | Expected public outcome | Transport | Replay state | Receipt rule |
| --- | --- | --- | --- | --- | --- |
| `field_binding` | Field observer emits `verified_cached` after existing verify/bind/cache succeeds. | `verified_cached` | `field_binding`, no HTTP status. | `not_applicable`. | Field bridge emits once only after success; observer failure cannot change Field result. |
| `valid_proof` | Handler emits `manifest_issued` / `valid_proof` after successful replay consumption. | `proof_valid` | HTTP `200`. | `verified_tenant`; exact `+1` replay-count delta. | Capture count only after verified tenant context exists and immediately before/after the successful consume boundary. |
| `replay` | Handler emits `replay_rejected` / `replay` after verified proof and replay-store rejection. | `replay_rejected` | HTTP `409`. | `not_applicable`. | No tenant aggregate is published; the observed rejection itself is the public signal. |
| `signature_invalid` | Handler emits `proof_rejected` / `signature_invalid`. | `signature_invalid` | HTTP `401`. | `not_applicable`. | Never infer tenant scope or aggregate state. |
| `package_hash_invalid` | Handler emits `proof_rejected` / `package_hash_invalid`. | `package_hash_invalid` | HTTP `403`. | `not_applicable`. | Never expose package content or calculated hash. |
| `expired` | Handler emits `proof_rejected` / `expired`. | `expired` | HTTP `401` when verification fails before issuance, or the actual terminal HTTP status if source code has a separately tested expiry path. | `not_applicable`. | The bridge must map the observed status; it may not normalize an unknown path into a false status. |
| `authority_mismatch` | Handler emits `proof_rejected` / `authority_mismatch`. | `authority_mismatch` | Observed HTTP status (`401` or `403` only where the handler emitted it). | `not_applicable`. | The bridge maps the source reason, not an identity or authority payload. |
| `key_unknown` | Handler emits `proof_rejected` / `key_unknown`. | `key_unknown` | HTTP `401`. | `not_applicable`. | No identity, registered-device data, or public key may enter the receipt. |

This deliberately exposes an aggregate replay count only for `valid_proof`, which is the sole case in which the current source has a verified tenant context and a required `+1` mutation. All other cases use `not_applicable` rather than misleading zero-count placeholders.

## 6. Source-owned bridge design

The implementation must add a small source-owned bridge package, conceptually `internal/manifestreceiptbridge`, with these responsibilities:

1. Load and validate the nonsecret run context only in the isolated pilot runtime. It returns no bridge when all three values are absent; partial, malformed, non-pilot, or unsafe contexts fail closed at candidate startup.
2. Own the deterministic mapping from allowed redacted handler observations to the seven HTTP public cases. No fixture data, HTTP request body, proof, key, identity, or stdout/stderr is a bridge input.
3. Receive a source-produced replay-state snapshot only for the verified `valid_proof` successful consume path. It must not query counts after an unverified failure merely to populate a public receipt.
4. Receive the source-redacted Field `verified_cached` observation through a separately provisioned Field bridge. The Field bridge may use the same public run context, but it cannot become an authority or mandatory workflow dependency.
5. Use the receipt writer for atomic output and write a strict, source-redacted bridge-failure artifact when a recognized receipt cannot be published. The public adapter must fail a run if that artifact exists. It must not rely on stdout/stderr, which remains suppressed.

The bridge must never be injected into `cmd/integin-server/main.go`. The existing `server.PilotManifestHandlerFromEnvironment(...)` composition seam is the only reviewed server-side location for receipt bridge injection, and it must continue to validate isolated pilot runtime, loopback use, and disabled package enforcement.

## 7. Failure, duplication, and completeness rules

| Situation | Required effect |
| --- | --- |
| Missing/invalid run ID, contract version, or receipt directory | Candidate bridge fails closed before mounting the receipt sink; no receipt is emitted. The wrapper/run is a failed hard-gate attempt. |
| Source observation is not one of the eight mapped terminal signals | No case receipt. It may create a bounded diagnostic only if the diagnostic contains no protected/private material. |
| Same mapped case occurs twice in one run | First atomic receipt remains; the duplicate is written as a strict public bridge failure. The adapter rejects the run. |
| Output directory is unsafe, missing, symlinked, or fails ACL checks | No receipt is emitted; strict bridge failure causes adapter rejection. |
| Field observer/bridge is absent or fails | Field workflow remains authoritative and successful cache stays valid; the hard-gate run lacks `field_binding` evidence and therefore fails completeness. |
| Candidate stops before all cases are emitted | Wrapper mandatory shutdown remains; adapter rejects incomplete set. |
| Acceptance changes or pilot listener remains | Existing wrapper invariant fails; no matrix claim is permitted. |

## 8. Required evidence before any emitter composition

Before changing the candidate or invoking the fixture, the reviewed implementation must provide source-only tests for run-context validation, v2 schema validation, state-scope behavior, case mapping, duplicate/failure publication, unsafe-directory rejection, source redaction, and Field observer non-interference. The public adapter must have synthetic positive and negative v2 tests. Go tests, Go vet, and accepted Flutter static analysis must pass.

Only after these source-only prerequisites are reviewed may the candidate bridge be composed into the isolated pilot. Composition is still not a proof-matrix execution authorization: it is followed by the existing wrapper, mandatory shutdown, adapter validation, and separate evidence review.

## 9. Explicit prohibitions

The implementation must not inspect or alter `cmd\pilot-manifest-fixture`, read fixture content, print fixture output, read private credentials, modify protected acceptance, enable package enforcement, enable OIDC, unseal/wire OpenBao, add a route to the default runtime, modify `cmd\integin-server\main.go`, use client-provided run IDs, derive public receipts from database queries alone, or convert public run-correlation data into a user/device tracking system.
