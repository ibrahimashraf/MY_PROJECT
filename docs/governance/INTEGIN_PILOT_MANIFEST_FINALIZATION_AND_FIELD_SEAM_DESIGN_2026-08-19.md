# INTEGIN Pilot Manifest Finalization and Field Receipt-Seam Design

**Decision date:** 2026-08-19  
**Status:** Source-only implementation design; not authorization for candidate composition or execution.  
**Scope:** Public-redacted lifecycle completion, bridge-failure visibility, and a non-interfering Field binding receipt seam.

> This design does **not** authorize a candidate launch, manifest route mount, migration application, package enforcement, OIDC enablement, OpenBao wiring, acceptance change, fixture inspection, or transmission of run context to the protected fixture.

## 1. Resolved boundary

The current v2 receipt writer and context validator correctly model individual public receipt shapes, but they do not prove that a candidate run reached a trustworthy terminal state. A receipt publish error or duplicate case can otherwise be returned only to a local source caller and disappear when the candidate stops. The missing work is therefore a **lifecycle evidence seam**, not a new authority path.

The protected fixture is excluded from this seam. The wrapper may give nonsecret correlation values only to the isolated candidate and, when explicitly started for a later Field exercise, to a separately provisioned Field pilot process. The fixture receives neither a run ID nor a receipt directory and cannot emit, alter, or certify a public receipt.

## 2. Required public artifacts

All artifacts reside only in the wrapper-created, ACL-restricted, single-run receipt directory. They contain no proof, key, identity, user, device, tenant, organization, payload, manifest body, database value, private path, or raw error text.

| Artifact | Owner | Required fields | Adapter effect |
| --- | --- | --- | --- |
| `receipt-v2-<case>-<run>.json` | Source receipt bridge or Field bridge | Existing v2 receipt contract. | Counts toward the exact eight-case inventory. |
| `bridge-failure-v2-<run>.json` | Source bridge | Contract version, run ID, fixed artifact type, fixed failure code, optional public case, UTC timestamp. | Its presence makes the matrix fail, even if all receipt files exist. |
| `bridge-finalization-v2-<run>.json` | Wrapper-invoked source finalizer after candidate and Field activity end | Contract version, run ID, fixed artifact type, terminal status, emitted-case count, bridge-failure presence, UTC timestamp. | Required exactly once; `complete` is necessary but never sufficient for a matrix pass. |

The finalizer accepts only the known eight receipt names plus its own finalization artifact and the optional fixed failure artifact. Any other file, duplicate case file, missing expected case, unsafe directory, or failure artifact produces a non-complete finalization state. It never parses a receipt body to create an authority claim; the public adapter remains responsible for full schema and semantic validation.

## 3. Source-only lifecycle protocol

The following ordered protocol is required before any candidate emitter is composed.

1. The wrapper creates a fresh empty receipt directory and a fresh 32-hex run ID. It starts the isolated candidate with child-only `INTEGIN_PILOT_MANIFEST_CANDIDATE_RUN_ID`, `INTEGIN_PILOT_MANIFEST_RECEIPTS_DIR`, and `INTEGIN_PILOT_MANIFEST_RECEIPT_CONTRACT_VERSION=2` values. It does not mutate a parent environment.
2. Candidate startup loads only the explicit pilot context. All-empty context preserves the ordinary no-bridge path. Partial, malformed, unsafe, unsupported, or non-pilot context fails candidate startup before a receipt sink is available.
3. A mapped source observation asks the bridge to publish its public receipt. If publication or duplicate detection fails, the bridge writes one fixed, redacted failure artifact on a best-effort basis. It returns an error only to the evidence bridge caller; it must not alter an already determined manifest HTTP or Field workflow result.
4. The separately provisioned Field bridge receives only a post-cache `verified_cached` observation. It uses the same explicit nonsecret context and can emit only `field_binding`; observer or write failure is caught so that verified cache persistence remains authoritative.
5. The wrapper stops the candidate in its existing mandatory shutdown path and ensures any explicitly started Field pilot process has ended. Only then does it invoke the source finalizer with the run ID and receipt directory.
6. The finalizer atomically writes exactly one terminal finalization artifact. It writes `complete` only when all eight expected receipt filenames exist, no failure artifact exists, the directory remains trusted, and no unexpected artifact is present. It writes `incomplete` or `failed` for every other condition without exposing implementation error text.
7. The read-only public adapter requires a terminal finalization artifact, rejects a failure artifact, rejects any non-complete finalization state, and still independently validates the eight receipt schemas, case mapping, redaction limits, replay arithmetic, wrapper invariants, and post-shutdown listener absence.

## 4. Fixed failure vocabulary

The source bridge and finalizer use only the following public reason codes. They must not serialize a Go, Dart, PowerShell, database, HTTP-client, or filesystem error string.

| Code | Meaning |
| --- | --- |
| `duplicate_case` | A recognized public case was observed more than once in one run. |
| `receipt_publish_failed` | A recognized public receipt could not be atomically published. |
| `unsafe_receipt_directory` | The receipt directory failed trusted-directory validation. |
| `unexpected_public_artifact` | The finalizer found an unrecognized file in the public receipt directory. |
| `missing_expected_case` | The finalizer found fewer than the required eight case receipts. |
| `bridge_failure_present` | The finalizer found the single fixed bridge-failure artifact. |
| `finalization_publish_failed` | The finalizer could not atomically publish its terminal public artifact. |

## 5. Independent Field pilot seam

The Field seam is a small optional observer implementation, constructed only by an explicit isolated-pilot bootstrap. It accepts the already-redacted `PackageManifestBindingObservation`, creates only the `field_binding` v2 observation, and catches all bridge errors. It cannot fetch a manifest, change cache state, add an HTTP endpoint, accept caller-supplied correlation values, inspect Field storage, or modify primary workflow state.

The observer is intentionally not created by the default Field application bootstrap. A future pilot launcher must pass the opaque run context as an explicit child-process configuration only after the lifecycle source tests and independent review accept the implementation.

## 6. Required source-only proof before composition

| Claim | Required proof |
| --- | --- |
| Duplicate or publish failure is publicly visible | Focused Go tests assert one fixed failure artifact, no raw error text, and no duplicate final receipt. |
| Finalization is reliable and fail-closed | Focused Go tests cover complete, missing-case, failure-present, unexpected-artifact, unsafe-directory, and duplicate-finalization paths. |
| Field bridge is non-interfering | Focused Dart tests prove it emits only after verified cache binding and that observer failures leave the returned cached package unchanged. |
| Public adapter rejects incomplete lifecycle evidence | Synthetic adapter tests cover missing/non-complete finalization and present failure artifact. |
| Existing scope remains protected | `go test ./... -count=1`, `go vet ./...`, accepted Flutter static analysis, and an explicit no-default-composition check pass. |

## 7. Explicit non-goals

This design does not implement an emitter, start a candidate, call the protected fixture, provide a Field transport configuration, apply migrations, mount a route, query live aggregate state, enable package enforcement, or establish live proof-matrix evidence. Those remain later gates after source-only implementation and independent review.
