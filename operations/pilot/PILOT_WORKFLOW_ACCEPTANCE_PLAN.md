# INTEGIN Complete Pilot Workflow Acceptance Plan

**Scope:** isolated pilot only (`127.0.0.1:18080`). Acceptance (`127.0.0.1:8080`) is a protected control and is never a target.

## Objective

Prove one evidence-linked field workflow end to end: field capture and local durable outbox; encrypted evidence creation/upload; signed transaction submission; authoritative receipt and duplicate/replay behavior; operator-visible review data; advisory-only insight with no ability to mutate primary workflow state.

## Existing verified evidence

| Step | Existing evidence | Current result |
|---|---|---|
| Signed sync/evidence transport | Explicit-target live pilot matrix | Passed manually against port 18080. |
| Field offline and sync state | Field controller, outbox, sync transport tests | Authored test coverage exists. |
| Evidence transport and crypto | Field evidence upload and crypto tests; Go evidence API/storage tests | Authored test coverage exists. |
| Advisory boundary | Go advisory and advisor-view tests | Advisory model is non-blocking. |
| Authority and receipt path | Go sync/sync-api/device-trust tests | Authoritative server path is covered. |

## Controlled acceptance scenario

1. Start from an isolated pilot-only identity/device fixture; do not use acceptance identities.
2. In the Flutter field app, create an inspection result while transport is unavailable; verify a durable queued transaction and evidence metadata.
3. Restore the pilot transport and flush the outbox; verify an applied authoritative receipt and evidence linkage.
4. Re-submit the same signed transaction; verify duplicate/replay-safe behavior without an additional workflow mutation.
5. View the resulting event/receipt/evidence metadata through the operator-facing server surface or reviewed read model.
6. Request advisory analysis and verify it is evidence-linked, marked non-blocking, and has no primary workflow mutation endpoint.
7. Capture only non-secret pass/fail evidence, clean pilot fixtures, and recheck acceptance/pilot health plus OIDC-disabled resting state.

## Execution rules

- Use the validated manually controlled explicit-target matrix procedure for signed server transport until the bounded child-process runner is corrected.
- Keep private values in process memory and never include them in acceptance notes, logs, or source control.
- Stop at any indication that the pilot tool targets port 8080, crosses tenant boundaries, exposes a secret, or mutates primary state from advisory logic.

## Remaining acceptance evidence

The automated tests and live sync/evidence matrix establish transport and boundary behavior. A human-visible Flutter field capture and operator review walkthrough is still required to close complete product acceptance.
