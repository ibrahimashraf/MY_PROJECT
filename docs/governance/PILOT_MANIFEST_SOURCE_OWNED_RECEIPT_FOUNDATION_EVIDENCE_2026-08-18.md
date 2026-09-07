# INTEGIN Pilot Manifest Source-Owned Receipt Foundation Evidence

**Recorded:** 2026-08-18  
**Scope:** Source-only receipt-foundation and Field-binding evidence work.  
**Authority boundary:** This record does not authorize a candidate launch, a route mount in acceptance, package enforcement, OIDC enablement, OpenBao wiring, migration application, or any persistent-runtime change.

## Purpose and decision

This record closes the **source-foundation** portion of the isolated-pilot manifest evidence stream. The project now has a source-owned, redacted receipt-writing primitive; a closed handler observation seam; aggregate replay-count interfaces; and a Field binding signal that is emitted only after successful signature verification, cache binding, and persistence.

> The receipt writer is **not yet wired into the isolated-pilot candidate**. This is intentional. A live receipt is trustworthy only when its candidate-run correlation, case mapping, verified context, and aggregate-state arithmetic are all source-derived. The completed Field signal resolves one of those inputs, but it does not create a safe substitute for the remaining candidate-only correlation design.

## Completed tracked source changes

| Commit | Change | Safety contribution |
| --- | --- | --- |
| `384c3fb` | Added `internal/manifestreceipts`, closed `packagemanifestapi.ObservationSink`, handler-path observations, replay `Count()` seams, PostgreSQL tenant-scoped counting, and handler coverage. | Gives the source a bounded, atomic, redacted receipt primitive without exposing proofs, payloads, keys, identities, or tenant data. |
| `56f0422` | Added `PackageManifestBindingObservation` and an optional Field delivery observer. | Records `verified_cached` only after the existing verifier/binder has successfully persisted the approved package; observer failure cannot block or mutate the authoritative Field workflow. |

The handler observation contains only outcome, reason code, HTTP status, a permitted manifest digest field, and replay acceptance state. The receipt writer enforces outcome vocabulary, non-negative aggregate values, an 8 KiB bound, and atomic file creation. The PostgreSQL replay count is tenant-scoped and exposes no identity or proof material.

## Field signal contract

The Field delivery path now calls the optional observer strictly after `verifyBindAndCache(...)` resolves successfully. The callback payload contains only `occurredAt` and the fixed `verified_cached` outcome. It deliberately excludes manifest bodies, manifest identifiers, user/device identity, public/private keys, proof material, and source payload data.

An observer exception is caught locally. This preserves the authoritative result: a successfully verified and cached approved package remains usable even when non-blocking evidence observation is unavailable.

## Validation record

| Verification | Result | Notes |
| --- | --- | --- |
| `C:\flutter\bin\cache\dart-sdk\bin\dart.exe analyze .\field_app` | Passed: `No issues found!` | Accepted Field static-validation path; the known Flutter test-runner stall was not treated as a passing test result. |
| `go test ./... -count=1` | Passed | Includes `internal/manifestreceipts`, `internal/packagemanifest`, `internal/packagemanifestapi`, and `internal/workpackagepg`. |
| `go vet ./...` | Passed | No diagnostics. |
| Acceptance check | Passed | `GET http://127.0.0.1:8080/healthz` and `GET http://127.0.0.1:8080/readyz` both returned HTTP `200`. |
| Candidate isolation check | Passed | No listener existed on `127.0.0.1:18080`. |

The initial acceptance check used obsolete `/health` and `/ready` paths and correctly returned `404`. Read-only source inspection confirmed the supported contract is `/healthz` and `/readyz`; the corrected checks succeeded. This was a probe-path correction, not a runtime degradation or a runtime repair.

## Independent review decision

Bounded architecture, security, and Field-evidence reviews agreed that a source-owned receipt foundation is appropriate but rejected premature candidate composition. The deciding safety concern is that invalid-proof paths have no verified tenant scope, Field binding occurs outside the server handler, and state-count evidence must be tied to verified source context rather than reconstructed from fixture output.

## Remaining live-matrix gate

Before the receipt writer is composed into the isolated-pilot candidate, the source design must demonstrate trustworthy, source-redacted coverage for each public case: `field_binding`, `valid_proof`, `replay`, `signature_invalid`, `package_hash_invalid`, `expired`, `authority_mismatch`, and `key_unknown`.

The composition design must also bind every receipt to the wrapper-created opaque `candidate_run_id`, derive expected and observed outcomes from the source boundary, preserve aggregate state-count arithmetic, write only to the ACL-hardened public receipt directory, and retain mandatory candidate shutdown. It must not inspect the protected fixture, expose fixture output, alter acceptance, enable package enforcement, or use OIDC/OpenBao.
