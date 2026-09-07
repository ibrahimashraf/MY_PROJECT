# INTEGIN Pilot Manifest Lifecycle and Field Seam — Source Evidence

**Recorded:** 2026-08-19  
**Scope:** Source-only lifecycle artifacts, read-only adapter validation, and an advisory Field observation seam.  
**Status:** Implementation, focused validation, and bounded source review complete; source-only commit pending.  

> This record does **not** authorize candidate emitter composition, fixture invocation or inspection, candidate launch, migration application, route mounting, package enforcement, OIDC enablement, OpenBao wiring, or acceptance changes.

## Delivered source-only seams

| Area | Delivered behavior |
| --- | --- |
| Go lifecycle artifacts | `internal\manifestreceipts\lifecycle_v2.go` adds atomic public bridge-failure and terminal-finalization artifacts. The finalizer reports only `complete`, `incomplete`, or `failed` lifecycle state; it never claims receipt semantic validity. |
| Duplicate/publish visibility | `V2Writer.EmitWithFailure` records a fixed redacted failure code for a recognized case when receipt publication or duplicate detection fails. Raw errors are not serialized. |
| Public adapter | `run-public-manifest-case-receipt-adapter-v2.ps1` now requires one complete finalization artifact, rejects a public bridge-failure artifact, and continues to validate exact case inventory and receipt semantics. |
| Field pilot seam | `PilotManifestBindingReceiptBridge` accepts only a post-cache `verified_cached` observation, requires a 32-hex opaque run ID, and catches publication failure so the verified/cache-bound package remains authoritative. It is not constructed by the default Field bootstrap. |

## Validation evidence

| Verification | Result |
| --- | --- |
| `gofmt -d` on new Go lifecycle source/tests | Passed with no diff after one formatting correction. |
| Focused `go test .\internal\manifestreceipts -count=1` | Passed. |
| `go test ./... -count=1` | Passed. |
| `go vet ./...` | Passed. |
| Focused Dart analysis of bridge and delivery test | Passed: `No issues found!` using the independently validated direct Dart path. |
| Focused Flutter delivery test | Started through the established Windows toolchain but produced no output within 200 seconds; it was stopped. This is recorded as the existing Windows Flutter-runner limitation, not as a passing test. |
| Public v2 adapter parse | Passed. |
| Synthetic public-only adapter positive case | Passed with eight valid redacted receipts plus one complete finalization artifact. |
| Synthetic public-only bridge-failure negative case | Rejected. |
| Synthetic public-only missing-finalization negative case | Rejected as incomplete. |
| Protected acceptance after guarded recovery | `/healthz=200`, `/readyz=200`, one acceptance process, and no listener on `127.0.0.1:18080`. |

## Preserved boundaries

The protected fixture receives no correlation context and was not opened or invoked. No default server main, default Field bootstrap, route composition, migration, persistent enforcement, OIDC, OpenBao, private value, database data, or object storage state was changed by these seams.

The lifecycle artifacts prove only that a future source bridge can report its own public completion/failure state to the adapter. They do not prove any live manifest case. Candidate composition remains blocked until the source-only commit is recorded, the known Flutter test-runner limitation is explicitly retained, and the full candidate/Field composition plan is separately authorized.
