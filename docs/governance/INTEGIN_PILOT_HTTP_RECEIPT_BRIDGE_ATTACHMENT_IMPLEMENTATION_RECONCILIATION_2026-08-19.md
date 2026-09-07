# INTEGIN Pilot HTTP Receipt-Bridge Attachment Implementation Reconciliation

**Recorded:** 2026-08-19  
**Template:** Delegate and Debate — complex source-only implementation review.  
**Decision:** Commit the reviewed pilot-only attachment source; preserve the separate operational-propagation gate.

## Independent review reconciliation

| Lens | Result | Owner treatment |
| --- | --- | --- |
| Runtime isolation and fail-closed attachment | Approved without blockers. It requested early missing-dependency rejection, bounded decoded-key handling, initialized durable-store tests, and an unready-store rejection. | All accepted. The constructor now rejects missing processor/authority dependencies after its existing pilot guard; the decoded source buffer is cleared after construction; the attachment requires `ManifestProofReplayStore.Ready()`; and tests use a repository-backed adapter plus an explicit unready-store rejection. |
| Non-authority behavior and test coverage | Approved without blockers. | Accepted. It confirmed all-absent/partial/unsafe context coverage, default-disabled isolation, exact counter identity, and absence of launch, route, Field, or fixture behavior. |

The issuer independently copies the signing key at construction (`append(ed25519.PrivateKey(nil), privateKey...)`), so clearing the transient decoded buffer cannot invalidate the issuer’s key.

## Validation evidence

| Verification | Result |
| --- | --- |
| Focused server and replay-store tests | Passed. |
| Go formatting check on changed source/tests | Passed. |
| `go test ./... -count=1` | Passed. |
| `go vet ./...` | Passed. |
| Default main source guard | No receipt-bridge import/reference found in `cmd\integin-server\main.go`. |

## Preserved boundaries

The attachment code reads candidate-only receipt context only after the existing explicit pilot retrieval and runtime gates pass. With all receipt values absent, it returns the existing uninstrumented pilot handler and does not inspect or create a receipt artifact. Partial, invalid, unsafe, non-durable, and unready state fails closed before observer attachment.

No wrapper or launcher passes those values today. No candidate is launched, no fixture receives context, no Field transport exists, no route is mounted, and no production/default startup path changes. The next gate is a separate operational propagation plan that may describe—but must not execute—how the isolated wrapper later supplies the candidate-only values. It is not authorization to launch.
