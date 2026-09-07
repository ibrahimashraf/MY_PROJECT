# INTEGIN Pilot HTTP Receipt-Bridge Attachment Plan

**Decision date:** 2026-08-19  
**Status:** Source-only design pending bounded review.  
**Scope:** Attaching the already committed HTTP mapper only through the explicit pilot handler constructor.  

> This plan does not authorize a candidate launch, a wrapper or launcher change, fixture context, Field traffic, route mounting, database migration, acceptance action, enforcement, OIDC, or OpenBao operation.

## 1. Proposed attachment boundary

The only permitted source location is `server.PilotManifestHandlerFromEnvironment(...)`, after it has completed its existing retrieval-enabled, `runtime=pilot`, database, signing-key, loopback, and enforcement-disabled validation. `cmd\integin-server\main.go`, default server construction, `NewMux`, default Field bootstrap, and the protected fixture remain unchanged.

The constructor will read three candidate-only, nonsecret process values:

| Value | Rule |
| --- | --- |
| `INTEGIN_PILOT_MANIFEST_CANDIDATE_RUN_ID` | Lower-case opaque 32-hex value generated later by the public wrapper. |
| `INTEGIN_PILOT_MANIFEST_RECEIPTS_DIR` | Fresh wrapper-owned regular directory with restrictive ACLs. |
| `INTEGIN_PILOT_MANIFEST_RECEIPT_CONTRACT_VERSION` | Exactly `2`. |

It will call the existing `manifestreceiptbridge.Load(...)` once. All absent values preserve the current handler with no observer/counter attachment. Any partial, malformed, unsupported, missing, or unsafe context fails only the explicit pilot constructor before a handler is returned. The normal default server never reads these values.

## 2. Required source-level attachment

| Condition | Constructor result |
| --- | --- |
| Context absent | Return the existing uninstrumented pilot handler; `Observer` and `ReplayCounter` remain nil, and `Load` must not create, open for write, or enumerate a receipt directory. |
| Context valid | Attach `manifestreceiptbridge.NewHTTPObserver(writer)` and attach a `ProofReplayCounter` derived from the same concrete replay-store object already serving `Handler.ReplayStore`. |
| Counter unavailable or not implemented by that concrete store | Return a fail-closed constructor error; never attach a mapper that can silently emit a valid-proof failure artifact for every valid request. |
| Context invalid | Return a fail-closed constructor error; do not return handler/registry components for mounting. |
| Any attachment error | Return no handler/registry, expose no private value, and leave the default runtime untouched. |

The attachment must retain the one `*workpackagepg.ManifestProofReplayStore` value it constructs, assign that exact pointer to `Handler.ReplayStore`, and attach the same pointer as `Handler.ReplayCounter` only after an interface assertion. It must fail closed if the counter interface is unavailable. It must not construct a second store, query replay state outside verified handler scope, or inject a database dependency into the bridge package.

## 3. Source-only test matrix

| Claim | Required test |
| --- | --- |
| Default isolation | With retrieval disabled, no handler/registry/observer exists, regardless of receipt-related environment values. |
| Current candidate compatibility | With explicit pilot retrieval enabled but all receipt values absent, handler construction remains successful and `Observer`/`ReplayCounter` remain nil; no receipt directory or artifact is created. |
| Partial context | Each one- or two-value combination rejects before a handler is returned. |
| Invalid context | Invalid run ID, contract version, and unsafe directory reject before observer attachment. |
| Valid context | The returned handler contains an `HTTPObserver` and a counter from the same concrete replay-store instance. |
| No normal-startup drift | A source guard confirms `cmd\integin-server\main.go` has no receipt bridge import/reference. |
| No fixture/Field drift | Attachment tests exercise only source values and temporary directories; no fixture, Field publisher, launcher, wrapper, socket, database, or route is touched. |

## 4. Explicitly excluded work

This attachment does not alter the launcher or wrapper. Therefore no current process can provide the values required to attach it, and no candidate can become instrumented as a consequence of this code alone. Receipt write failures remain deliberately non-authoritative at the HTTP boundary; the existing redacted failure artifact and later read-only adapter own their evidence consequence. Finalization remains wrapper-owned and uncalled. The Field publisher stays abstract, so no completed public matrix is possible even after attachment.

The next gate after a reviewed source attachment is a separate **operational propagation plan** for wrapper-to-candidate-only context. That plan must preserve fixture isolation and still is not launch authorization.
