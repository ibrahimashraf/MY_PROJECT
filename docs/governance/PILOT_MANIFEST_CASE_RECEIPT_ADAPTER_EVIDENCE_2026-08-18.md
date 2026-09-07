# INTEGIN Public Manifest Case-Receipt Adapter Evidence — 2026-08-18

**Status:** The public, read-only case-receipt validator adapter is implemented and validated. The full isolated-pilot proof matrix remains **unproven** until the protected fixture gains a source-redacted receipt emitter. No protected fixture content was read during this work.

## Delivered public components

The adapter is installed at `operations\pilot\runtime\run-public-manifest-case-receipt-adapter.ps1`. It validates only pre-redacted, non-secret receipt artifacts and a non-secret wrapper record. It neither starts a candidate nor opens a fixture, reads fixture stdout/stderr, accesses credentials, or alters acceptance or package enforcement. [1]

| Component | Delivered control |
|---|---|
| Source contract | `contracts\manifest_candidate_case_receipt_v1.schema.json`, committed as `a603555` (`Add public manifest case receipt schema`). |
| Runtime schema | ACL-hardened deployment copy under `operations\pilot\runtime\manifest_candidate_case_receipt_v1.schema.json`. |
| Adapter | Read-only validation of exactly eight named receipt files, wrapper correlation, schema identity, safe paths and ACLs, file-size bounds, outcome mapping, state-count arithmetic, and candidate-shutdown state. |
| Positive synthetic test | Eight fully public synthetic receipts passed the adapter contract test with acceptance `200/200` and no listener at `127.0.0.1:18080`. |
| Negative synthetic test | A receipt containing one deliberate unexpected public field was rejected; the adapter returned a fail-closed incomplete result while acceptance remained `200/200` and the candidate remained stopped. |

## Multi-AI bounded review

Three independent reviewers assessed only public materials: an architecture review, a security review, and a Field/evidence-flow review. A fourth bounded source-review pass examined the adapter source, public contract, and public schema. I reconciled the findings and applied the resulting validation controls.

| Reconciled control | Adapter behavior |
|---|---|
| Artifact path safety | Rejects reparse points and broad `Everyone`, `BUILTIN\Users`, or authenticated-user write access. |
| File-set safety | Requires exactly eight case files with names tied to the current opaque `candidate_run_id`; rejects extras and omissions. |
| Schema boundary | Rejects unexpected top-level and nested `state_counts` fields; validates outcome enum, HTTP-status range, digest pattern, correlation-ID bounds, and runtime schema identity. |
| Semantic integrity | Requires filename/case/run-ID/expected-outcome agreement, passed observed-outcome equality, `after - before = delta`, valid-proof delta `1`, rejected-case delta `0`, and no side-effect count change for Field binding. |
| Fail-closed summary | Any error yields an incomplete matrix result and fills missing cases as `not_run`; it never fabricates observed outcomes or receipt data. |

## Explicitly unproven boundary

The adapter validates source-redacted receipt files; it cannot create authoritative receipts from the current public candidate interface. The existing wrapper correctly suppresses fixture output, and no receipt artifacts are exposed today. Therefore, the following remain unproven in the live isolated pilot: Field binding, valid proof, replay rejection, invalid signature/hash, expired proof, authority mismatch, and unknown-key handling.

> A fixture exit code, candidate start, or synthetic adapter test is **not** a proof-matrix result.

## Next required gate

The protected fixture, or a sidecar inside the fixture's existing trust boundary, must add a **source-redacted emitter**. For each named case, it must atomically write one schema-valid receipt to an ACL-hardened directory using the wrapper `candidate_run_id`, without fixture stdout/stderr, raw payloads, keys, tokens, credentials, tenant identifiers, device identifiers, or customer data. The public adapter can then validate the complete receipt set and determine whether the isolated-pilot proof matrix passes.

## References

[1]: INTEGIN_PUBLIC_MANIFEST_CASE_RECEIPT_ADAPTER_CONTRACT_2026-08-18.md "INTEGIN Public Manifest Case-Receipt Adapter Contract — 2026-08-18"
