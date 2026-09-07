# INTEGIN Pilot Manifest Receipt v2 Source Prerequisites Evidence

**Recorded:** 2026-08-18  
**Scope:** Source-only public receipt v2 primitives and synthetic public-adapter validation.  
**Non-authorization:** No candidate was started, no fixture was inspected or modified, no receipt emitter was composed, and no enforcement, acceptance, OIDC, OpenBao, or private-material control changed.

## Decision

The earlier source receipt foundation is now extended by commit `f6aceef` with a reviewed **v2 prerequisite layer**. v2 distinguishes a real manifest HTTP event from a Field binding event without fabricating HTTP data, and it publishes replay arithmetic only where the source has a verified tenant context.

The work does **not** make the live proof matrix complete. Independent review found that it would be unsafe to pass public run context to the protected fixture, rely on parent-process environment mutation, or compose a candidate bridge before a reliable bridge-failure/finalization lifecycle exists. The corrected contract is `docs\governance\INTEGIN_PILOT_MANIFEST_RECEIPT_CORRELATION_AND_SIGNAL_CONTRACT_2026-08-18.md`.

## Delivered source-only prerequisites

| Item | Delivered behavior |
| --- | --- |
| `contracts\manifest_candidate_case_receipt_v2.schema.json` | Closed public receipt v2 contract with HTTP/Field transport distinction and replay-state variants. |
| `internal\manifestreceipts\receipt_v2.go` | Atomic, bounded, redacted v2 writer. It rejects invented Field HTTP status, wrong source status, invalid replay-state scope, malformed opaque correlation, unsafe directories, and duplicate cases. |
| `internal\manifestreceiptbridge\context.go` | Pure explicit run-context validation. All-empty context means no bridge; partial/invalid/unsupported context fails closed. It does not read environment values, start a candidate, mount a route, or alter workflow authority. |
| `operations\pilot\runtime\run-public-manifest-case-receipt-adapter-v2.ps1` | Separate read-only v2 public adapter; the historical v1 adapter remains unchanged. |
| Runtime v2 schema | Non-secret public v2 schema copied to the runtime directory and assigned the existing hardened public-artifact ACL. |

## Signal semantics

| Case group | Transport and replay rule |
| --- | --- |
| `field_binding` | `event_source=field_binding`; `transport.kind=field_binding`; replay state is `not_applicable`; no invented HTTP status. |
| `valid_proof` | `event_source=manifest_http`; HTTP `200`; verified-tenant replay state with exact `after - before = +1`. |
| `replay`, signature, hash, expiry, authority, key rejection | `event_source=manifest_http`; source-mapped closed HTTP status; replay state is `not_applicable` with no numeric placeholders. |

## Validation record

| Verification | Result |
| --- | --- |
| Focused Go tests for `manifestreceipts` and `manifestreceiptbridge` | Passed. |
| `go test ./... -count=1` | Passed. |
| `go vet ./...` | Passed. |
| `C:\flutter\bin\cache\dart-sdk\bin\dart.exe analyze .\field_app` | Passed: `No issues found!` |
| Public v2 adapter synthetic positive case | Passed eight redacted v2 receipts with exact inventory and expected mapping. |
| Public v2 adapter synthetic negative case | Passed by rejecting an unexpected public artifact. |
| Protected acceptance check | `/healthz=200`; `/readyz=200`. |
| Candidate isolation check | No listener on `127.0.0.1:18080`. |

The adapter synthetic wrapper record is deliberately synthetic validation input. It validates adapter shape and rejection behavior; it is not a live candidate or acceptance proof.

## Preserved boundary and next gate

The protected fixture receives no run ID, receipt directory, bridge context, or ability to influence public receipts. The default runtime remains without a receipt bridge and `cmd\integin-server\main.go` remains unmodified.

The next design gate is a pilot-only candidate lifecycle/finalization protocol that can make every recognized duplicate or publish failure visible to the public adapter despite candidate shutdown. A separate Field pilot composition seam is also required. Until both are independently reviewed and source-tested, the v2 writer and adapter remain source-only prerequisites rather than an isolated-pilot emitter.
