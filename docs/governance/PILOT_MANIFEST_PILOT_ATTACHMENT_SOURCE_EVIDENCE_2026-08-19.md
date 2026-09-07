# INTEGIN Pilot Manifest Pilot-Only Attachment — Source Evidence

**Recorded:** 2026-08-19  
**Source revision:** `af79c5a` — `Attach receipt bridge only in pilot constructor`  
**Scope:** Source-only optional attachment inside the explicit pilot constructor.  

> This evidence record does not authorize operational context propagation, candidate launch, wrapper/launcher change, fixture context, Field traffic, route mounting, migration, acceptance action, enforcement, OIDC, or OpenBao work.

## Committed behavior

| Behavior | Evidence |
| --- | --- |
| Explicit pilot-only attachment | Attachment occurs only after the existing retrieval-enabled and `runtime=pilot` checks in `PilotManifestHandlerFromEnvironment`. |
| All-empty inert state | All candidate receipt values absent leaves observer/counter nil and creates no receipt artifact. |
| Fail-closed invalid state | Partial, unsupported, unsafe, non-durable, and unready states return an error before observer/counter attachment. |
| Exact replay object identity | The same initialized durable replay-store pointer serves both `ReplayStore` and `ReplayCounter`. |
| Default isolation | Disabled pilot retrieval returns no handler/registry despite receipt-related values. |
| Key buffer handling | The transient decoded buffer is cleared after the issuer constructor independently copies its signing key. |

## Validation evidence

| Verification | Result |
| --- | --- |
| Focused server and replay-store tests | Passed. |
| Go formatting check | Passed. |
| `go test ./... -count=1` | Passed. |
| `go vet ./...` | Passed. |
| Bounded independent reviews | Two usable reviews: runtime isolation/fail-closed attachment and test/non-authority behavior. Both found no blocker after the accepted hardening. |
| Source state after commit | Clean source tree at `af79c5a`. |
| Protected acceptance after commit | `/healthz=200`, `/readyz=200`; no listener on `127.0.0.1:18080`. |

## Retained gate

No current wrapper or launcher provides the three candidate-only attachment values. Therefore the committed source remains inert in every current process. The next permitted action is a **design-only operational propagation plan** for how a future isolated wrapper would pass those values to a future child candidate while retaining fixture isolation. It is not execution authorization.
