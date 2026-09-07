# INTEGIN Pilot Manifest-Delivery Candidate Execution Evidence — 2026-08-18

**Status:** Bounded candidate attempt completed and safely stopped. The candidate proved isolated route composition and malformed-request rejection. It did **not** complete the valid retrieval/proof matrix, so manifest delivery is not yet promoted from a candidate gate to a proven delivery capability.

## Scope and result

This evidence records the explicit isolated-pilot candidate authorized on 2026-08-18. The candidate was constrained to loopback `127.0.0.1:18080`; the protected acceptance control remained at `127.0.0.1:8080`. The candidate was never used to enable package enforcement, alter acceptance, enable OIDC, wire OpenBao, or process customer data. The source and operational boundaries were those defined in the candidate runbook. [1]

> **Decision:** The candidate is a successful **route-isolation and rejection-path smoke test**, but an incomplete **device-authenticated manifest-delivery proof**. It must not be represented as full manifest-delivery acceptance.

| Control | Observed result | Outcome |
|---|---|---|
| Acceptance preflight | Health and readiness returned `200/200`. | Passed. |
| Candidate target | No existing listener at `127.0.0.1:18080`; candidate started on that loopback address only. | Passed. |
| Source identity | `integin-pilot-source` remained at `a2af81e`; only the intentional untracked `cmd/pilot-manifest-fixture/` state was present. | Passed. |
| Recovery asset | The documented pre-manifest pilot backup was present; only metadata was checked. | Passed. |
| Pilot dependencies | `integin-pilot-postgres` and `integin-pilot-rustfs` were running. | Passed. |
| Candidate route | `POST /work-package-manifest` on the candidate returned `401` for an empty JSON object rather than `404` or a transport failure. | Passed: the route was mounted and rejected the unauthenticated malformed request. |
| Package enforcement | No enablement action or enforcement-capable persistent runtime was introduced. | Preserved disabled. |
| Candidate shutdown | The process was stopped; no listener remained at `127.0.0.1:18080`. | Passed. |
| Acceptance recovery | Health and readiness returned `200/200` after shutdown. | Passed. |

## Approved matrix results

The following matrix distinguishes evidence actually obtained from cases that remain intentionally unproven.

| Case | Expected result | Observed result | Evidence status |
|---|---|---|---|
| Candidate listener isolation | Candidate binds only to loopback `127.0.0.1:18080`. | Listener was present after launch and absent after controlled stop. | Passed. |
| Route composition | Manifest route is mounted only in the candidate process. | Empty disposable request returned `401`, not `404`. | Passed. |
| Missing/invalid request rejection | Request without a usable device proof is rejected. | Empty JSON returned `401`. | Passed. |
| Manifest issuance | One signed disposable manifest is issued. | Not run. | Unproven. |
| Field verify/bind/cache | One issued manifest verifies and caches on the test Field device. | Not run. | Unproven. |
| Valid proof | One valid disposable proof is accepted exactly once. | Not run. | Unproven. |
| Replay rejection | The identical proof is rejected on second submission. | Not run. | Unproven. |
| Invalid signature / package hash | Invalid variants are rejected deterministically. | Not run. | Unproven. |
| Expired / wrong-authority / unknown-key proof | Each invalid proof is rejected with its expected reason code. | Not run. | Unproven. |

## Why the proof matrix stopped

The untracked disposable fixture was intentionally treated as protected material: its contents, generated payloads, and any key material were not read, printed, or logged. A no-output help invocation and a no-output default invocation each returned process exit code `2`. Rather than guess undocumented arguments, expose fixture data, or weaken a security boundary, the candidate was stopped.

This is not a runtime defect in the mounted route. It is an **operator-contract gap**: the fixture lacks a documented, non-secret invocation contract that lets an approved operator create the disposable valid and invalid proof cases while keeping private material private.

| Required corrective artifact | Purpose | Boundary to preserve |
|---|---|---|
| Public fixture invocation contract | Documents argument names, supported case names, exit statuses, and non-secret output fields. | Must not include fixture payloads, private keys, database URLs, tokens, or credentials. |
| Non-secret case runner or receipt ledger | Executes named valid/replay/rejection scenarios and emits opaque IDs, reason codes, and table-count deltas. | Must never print a manifest body, proof body, signature, or private material. |
| Repeatable Field binding observation | Records fetch, signature verification, package-hash binding, and local cache result for one disposable work package. | Must not make an inspection approval or alter package enforcement. |
| Candidate evidence template | Captures expected versus observed results and source/binary identifiers. | Must retain the acceptance `200/200` and no-listener checks. |

## Continued safety state

The candidate process is stopped. Acceptance remains healthy at `127.0.0.1:8080`; no listener exists at `127.0.0.1:18080`. Package enforcement remains disabled; OIDC remains disabled; OpenBao remains sealed and unwired. No database restore was attempted, so the destructive restoration gate was not entered.

The logical next step is **source-only completion of the public fixture/operator contract**, followed by a fresh isolated-pilot candidate run using that contract. The later risk/JSA/hazard foundation remains downstream of a completed manifest-delivery gate.

## References

[1]: INTEGIN_PILOT_MANIFEST_DELIVERY_CANDIDATE_RUNBOOK_2026-08-18.md "INTEGIN Pilot Manifest-Delivery Candidate Runbook — 2026-08-18"
