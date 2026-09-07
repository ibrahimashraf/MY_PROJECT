# INTEGIN Candidate Receipt-Bridge Composition Plan

**Decision date:** 2026-08-19  
**Status:** Source-only design pending bounded review.  
**Scope:** The future isolated-candidate bridge composition only.  

> This plan is not authorization to edit the bridge, start a candidate, invoke or inspect the protected fixture, mount a route, apply a migration, send Field traffic, enable package enforcement, enable OIDC, or wire OpenBao.

## 1. Decision and correction

The source-only lifecycle prerequisite is complete, but no bridge currently turns a redacted manifest-handler observation into a v2 receipt. The proposed next source change is a **candidate-only bridge injection** at the already reviewed `server.PilotManifestHandlerFromEnvironment(...)` and `NewPilotManifestHandler(...)` seams. It must remain uncalled by `cmd\integin-server\main.go` and default Field bootstrap code.

The earlier correlation contract’s wording that the same receipt values reach “the candidate and the invoked fixture” is superseded for the protected fixture: the protected fixture receives **no** candidate run ID, receipt directory, contract version, finalization capability, or receipt-writing ability. The wrapper may supply those nonsecret values only to the candidate child process and, in a later separately reviewed step, to a distinct Field pilot process. Fixture provenance remains on its unrelated, bounded fixture-signal path.

## 2. Required composition flow

| Step | Source owner | Required behavior | Forbidden behavior |
| --- | --- | --- | --- |
| 1. Wrapper preparation | Public wrapper | Generate the run ID; create a fresh ACL-restricted receipt directory; pass explicit candidate-only correlation values to the candidate launcher; restore/clear wrapper-scoped values after use. | Reuse a directory, accept caller run IDs, pass receipt values to the fixture, print values, or modify acceptance. |
| 2. Candidate startup validation | `PilotManifestHandlerFromEnvironment` | After existing pilot, loopback, retrieval-enabled, and enforcement-disabled checks, load all-or-absent v2 context. Partial, malformed, unsupported, or unsafe context fails candidate startup. | Read context in normal startup, make the context optional when partially supplied, or mount a default route. |
| 3. Handler bridge injection | `NewPilotManifestHandler` | Construct a mapper only from validated context plus source-owned dependencies and assign it to `Handler.Observer`. | Inject into `cmd\integin-server\main.go`, use request bodies, proof values, keys, identities, credentials, fixture output, or caller-provided correlation. |
| 4. HTTP observation mapping | New source-owned bridge mapper | Emit only the seven closed HTTP cases specified below. A failed/duplicate publication creates the fixed public bridge-failure artifact and never changes the handler’s determined HTTP result. | Emit diagnostics as cases, invent a status, publish raw errors, or query aggregate replay state after unverified failure. |
| 5. Terminal finalization | Wrapper after candidate/Field activity has stopped | Call `FinalizeV2` once, then invoke the read-only adapter. | Let the candidate self-finalize, finalize before child stop, finalize while a Field writer may still publish, or infer a successful matrix from finalization alone. |

## 3. Closed HTTP mapping and required source gaps

| Source observation | Receipt case | Required extra source fact | Current gap to close before composition |
| --- | --- | --- | --- |
| `manifest_issued` / `valid_proof` / HTTP 200 | `valid_proof` | Verified-tenant replay count immediately before and after successful consume; raw manifest ID only inside the mapper for hashing. | `Observation` has no count snapshot and currently carries no manifest ID. Add a source-owned verified-scope counter interface; never invoke it on rejected paths. |
| `replay_rejected` / `replay` / HTTP 409 | `replay` | None; replay state is `not_applicable`. | Mapper must reject any near-match rather than normalize it. |
| `proof_rejected` / `signature_invalid` / HTTP 401 | `signature_invalid` | None. | Replace string-derived failure categorization with a closed typed/explicit source mapping before relying on it for public evidence. |
| `proof_rejected` / `package_hash_invalid` / HTTP 403 | `package_hash_invalid` | None. | Mapper must accept only the observed status. |
| `proof_rejected` / `expired` | `expired` | Actual observed terminal status. | Current issuer-expiry path uses HTTP 403 while v2 writer/adapter currently permit only 401. Resolve the contract/adapter/source-status mismatch before composition; do not silently remap. |
| `proof_rejected` / `authority_mismatch` / HTTP 401 or 403 | `authority_mismatch` | None. | Mapper must preserve source status and accept only the two allowed statuses. |
| `proof_rejected` / `key_unknown` / HTTP 401 | `key_unknown` | None. | Mapper must reject non-401 variants. |

Malformed requests, unconfigured service, missing assignment, internal issuance failure, `proof_invalid`, and every observation not exactly listed above remain diagnostic only. They must produce neither a public case receipt nor an inferred failure case.

## 4. Field boundary

The committed `PilotManifestBindingReceiptBridge` remains an abstraction only. It observes one post-cache `verified_cached` event and suppresses its own publication failures, but has no concrete publisher or default bootstrap injection. This plan deliberately does **not** choose a Field-to-receipt transport because a process that merely possesses a nonsecret run ID is not sufficient provenance for a trustworthy Field case.

Candidate composition for seven HTTP cases may be source-designed, but a full matrix run remains incomplete until a separately reviewed Field pilot publisher proves provenance, ACL/transport controls, one-event semantics, and non-interference. No design may solve that gap by passing receipt context to the fixture or by converting the Field observer into an authority dependency.

## 5. Required source-only tests before a composition change

| Claim | Required proof |
| --- | --- |
| Default/runtime isolation | All-absent context returns no bridge; partial/invalid context fails only under the explicit pilot constructor; `cmd\integin-server\main.go` remains unchanged. |
| Context safety | Invalid run ID, unknown contract version, missing/unsafe directory, and symlink directory reject before observer attachment. |
| Closed mapper | Each exact allowed observation maps once; every near-match and all diagnostic observations write no case receipt. |
| Valid-proof arithmetic | The counter is called only after verified tenant context, immediately around successful consumption, and produces exactly `+1`; a counter failure creates public bridge-failure evidence without changing the HTTP response. |
| Source redaction | Mapper input/output tests prove no proof, signature, raw manifest, key, identity, credential, request ID, or raw error reaches artifacts. |
| Duplicate/publish failure | First receipt remains, one bridge-failure artifact appears, and the adapter rejects the finalized run. |
| Expiry mismatch | A source-only test fixes either the observed status contract or the explicit allowed-status rule before a mapper is composed. |
| Field non-interference | Existing bridge tests remain; no concrete Field publisher is added in this candidate-only change. |

## 6. Explicit stop conditions

Stop and require a new decision if a proposed solution needs fixture changes or context, a default runtime route, a database migration, acceptance alteration, a secret, client-supplied correlation, an unbounded diagnostic stream, a Field authority dependency, or an unreviewed expiry-status change. A passing source test suite is not candidate-launch authorization.
