# INTEGIN Manifest Candidate Case-Receipt Contract — 2026-08-18

**Status:** Source-only, non-secret evidence contract. This contract defines the minimum public evidence required before the isolated-pilot manifest proof matrix can be declared complete. It does not expose or authorize access to protected fixture content.

## Purpose

Each candidate case must emit exactly one schema-valid receipt using `contracts/manifest_candidate_case_receipt_v1.schema.json`. A receipt confirms only a named case, expected/observed public outcome, HTTP result, opaque correlation material, and optional aggregate state count. Every receipt for one wrapper invocation must reuse that wrapper's `candidate_run_id`. `observed_outcome` and `http_status` are required for executed `passed` and `failed` cases, and prohibited for `not_run` cases. It must never contain a fixture path, tenant/device/user identifier, manifest or proof payload, signature, public key, private key, token, database configuration, or customer information.

| Case | Expected public outcome | Required state-count interpretation |
|---|---|---|
| `field_binding` | `verified_cached` | No server workflow-state delta required. |
| `valid_proof` | `proof_valid` | Exactly one durable replay-protection acceptance delta. |
| `replay` | `replay_rejected` | No additional accepted-proof delta. |
| `signature_invalid` | `signature_invalid` | No accepted-proof delta. |
| `package_hash_invalid` | `package_hash_invalid` | No accepted-proof delta. |
| `expired` | `expired` | No accepted-proof delta. |
| `authority_mismatch` | `authority_mismatch` | No accepted-proof delta. |
| `key_unknown` | `key_unknown` | No accepted-proof delta. |

## Acceptance rule

The matrix passes only when all eight named cases yield `status: "passed"`, the observed outcome exactly equals the expected outcome, the valid-proof state delta is exactly one, every rejected case has no additional accepted-proof delta, acceptance stays `200/200` before and after the candidate, and the candidate listener is absent after shutdown. The schema binds every case to its required expected outcome, enforces the observed outcome for passed cases, and enforces the valid-proof and rejected-case state-delta requirements. The receipt verifier must additionally reject any `state_counts` record for which `after - before` does not equal `delta`; this arithmetic rule is intentionally enforced by the verifier rather than represented as an unsafe placeholder in JSON Schema.

> A fixture exit code alone is never a case receipt. It only confirms that the protected fixture ran under the approved candidate context.

## Adapter responsibility

The future public case adapter must select a fixture case using a documented non-secret invocation, validate the produced receipt against the schema, and discard any raw fixture output after extracting only schema-allowed values. If it cannot produce a schema-valid receipt without exposing protected material, it must fail closed and mark the case as `not_run`.
