# INTEGIN Public Manifest Operator Interface Evidence — 2026-08-18

**Status:** Public operator interface completed, independently reviewed, and validated in the isolated pilot. The wrapper can run the protected fixture without reading or retaining protected content. The complete valid/replay/rejection matrix remains **unproven** because no schema-valid public case receipts are produced by the currently exposed interface.

## Scope and retained boundary

This work addressed the operator-contract gap discovered during the earlier manifest candidate smoke check. It created a public wrapper, public operator contract, and strict case-receipt schema while preserving the rule that protected fixture contents, proof/manifest bodies, signing material, credentials, and environment values are never read, printed, logged, or committed. [1] [2]

> **Evidence distinction:** a fixture process exit code of `0` proves only that the protected fixture completed under the approved candidate context. It is not evidence that any named proof-matrix case passed.

| Control | Verified result |
|---|---|
| Fixture invocation | The wrapper passed the single approved opaque fixture handle without opening or recording its contents. Fixture build and execution output remained suppressed. |
| Candidate isolation | The wrapper required no pre-existing listener, captured the launcher PID record, verified its freshness, rejected reparse points and broad writer ACLs, and required that exact process to own only `127.0.0.1:18080`. |
| Runtime artifact protection | `operations\pilot\runtime` was hardened so the project owner, Administrators, and SYSTEM retain modification authority; broad authenticated-user modification was removed. |
| Protected acceptance | Before and after wrapper execution, health and readiness were `200/200`. |
| Candidate recovery | The candidate was stopped in `finally`; no listener remained at `127.0.0.1:18080`. |
| Fixture runnability | The public wrapper completed with `candidate_started: true`, `fixture_exit_code: 0`, output suppression enabled, and an opaque `candidate_run_id`. |
| Source and Field validation | Focused Go tests for `contracts` and `internal/packagemanifestapi` passed; focused Go vet passed; Field static analysis reported `No issues found!`. |

## Independent review and monitored reconciliation

A separate non-sensitive AI reviewer received only the public wrapper, public contracts, and receipt schema. It never received the fixture, any key, any credential, or runtime access. The first review identified loopback-isolation, output-suppression, PID-ownership, receipt-schema, and evidence-correlation gaps. Those findings were corrected and the reviewer was rerun until it returned **acceptable** with `public_contract_ready`.

| Review control added | Result |
|---|---|
| Exclusive listener verification | Candidate must own exactly one listener at loopback `127.0.0.1:18080`. |
| Trusted PID binding | The listener owner must match a fresh launcher PID record, whose ACL and reparse-point state are checked. |
| Acceptance evidence | The wrapper records measured before/after `200/200` values only after a successful fixture invocation and shutdown. |
| Opaque correlation | The wrapper emits `candidate_run_id`; future receipts for the same run must reuse it. |
| Receipt semantics | The schema binds each named case to the corresponding required outcome; passed cases require the matching observed outcome. |
| State delta rules | A passed valid proof requires delta `1`; passed replay and rejected cases require delta `0`; the verifier must also check `after - before = delta`. |
| Unrun-case clarity | `observed_outcome` and `http_status` are forbidden for `not_run` cases, preventing fabricated placeholders. |

## Public receipt-matrix state

The public interface did not expose an artifact that can be treated as a case receipt. Metadata-only observation found no new non-log receipt, case, result, or evidence artifact after successful fixture execution. Therefore, all eight named cases remain unproven; none may be inferred from exit code `0`.

| Matrix case | Required public evidence | Current state |
|---|---|---|
| Field binding | `verified_cached` receipt linked to wrapper `candidate_run_id`. | Not run / no public receipt. |
| Valid proof | `proof_valid` receipt and exact accepted-proof delta `1`. | Not run / no public receipt. |
| Replay | `replay_rejected` receipt and no additional accepted-proof delta. | Not run / no public receipt. |
| Invalid signature | `signature_invalid` receipt and delta `0`. | Not run / no public receipt. |
| Invalid package hash | `package_hash_invalid` receipt and delta `0`. | Not run / no public receipt. |
| Expired proof | `expired` receipt and delta `0`. | Not run / no public receipt. |
| Authority mismatch | `authority_mismatch` receipt and delta `0`. | Not run / no public receipt. |
| Unknown key | `key_unknown` receipt and delta `0`. | Not run / no public receipt. |

## Required next implementation gate

The next task is not another raw fixture run. It is a **public case-receipt adapter** that can select the eight named cases through a documented non-secret interface and emit exactly one schema-valid, non-secret receipt per case. It must reuse the wrapper `candidate_run_id`, validate count arithmetic, discard any raw fixture output, fail closed if it cannot produce a receipt, and leave the existing protected fixture opaque. Only then may the isolated-pilot proof matrix be rerun and considered for acceptance.

## References

[1]: INTEGIN_PUBLIC_MANIFEST_FIXTURE_OPERATOR_CONTRACT_2026-08-18.md "INTEGIN Public Manifest Fixture Operator Contract — 2026-08-18"
[2]: INTEGIN_MANIFEST_CANDIDATE_CASE_RECEIPT_CONTRACT_2026-08-18.md "INTEGIN Manifest Candidate Case-Receipt Contract — 2026-08-18"
