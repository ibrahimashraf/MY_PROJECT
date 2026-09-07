# INTEGIN Public Manifest Case-Receipt Adapter Contract — 2026-08-18

**Status:** Read-only public validator. This adapter validates source-redacted receipt artifacts after the candidate has stopped. It does not start a candidate, read a protected fixture, read fixture stdout/stderr, access keys or credentials, or change acceptance or package enforcement.

## Invocation

```powershell
& 'C:\MY PROJECT\operations\pilot\runtime\run-public-manifest-case-receipt-adapter.ps1' \
  -CandidateRunId '<opaque 32-character run ID>' \
  -PublicReceiptsDir '<ACL-hardened receipt directory>' \
  -WrapperRecordPath '<public wrapper JSON record>'
```

The adapter expects exactly eight public receipt files named `receipt-<case>-<candidate_run_id>.json`. The receipt directory must be a regular, ACL-hardened directory with no additional files. Each receipt and the wrapper record must be a regular file no larger than 8 KiB. The adapter reads its schema from the ACL-hardened runtime deployment copy at `operations\pilot\runtime\manifest_candidate_case_receipt_v1.schema.json`; the canonical tracked source copy remains `integin-pilot-source\contracts\manifest_candidate_case_receipt_v1.schema.json`.

| Validation layer | Required control |
|---|---|
| Wrapper correlation | `candidate_run_id`, candidate start, output suppression, acceptance `200/200` before/after, and absent candidate listener must all validate. |
| Schema boundary | No additional fields; public field presence rules must match `passed`, `failed`, or `not_run`. |
| Semantic binding | Filename, in-file case, candidate run ID, expected outcome, and passed observed outcome must agree. |
| State counts | `after - before = delta`; valid proof requires delta `1`; passed rejected cases require delta `0`; Field binding never changes the accepted-proof count. |
| Fail-closed result | Any missing, malformed, oversized, unsafe, duplicated, mismatched, or incomplete artifact yields `matrix_status: "incomplete"` and a non-zero exit. |

> The adapter treats the receipt files as **pre-redacted public evidence artifacts**. It must never be extended to parse raw fixture output or protected fixture content.

## Current integration boundary

The adapter is intentionally complete as a validator, but it cannot pass the matrix until a source-redacted fixture emitter writes the eight receipt files through a documented public interface. The emitter must create each file atomically, reuse the wrapper `candidate_run_id`, use only schema-permitted data, and never use stdout/stderr as an evidence channel.
