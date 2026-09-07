# INTEGIN Closed HTTP Receipt-Bridge Review Reconciliation

**Recorded:** 2026-08-19  
**Template:** Delegate and Debate — complex source review, high evidence burden.  
**Decision owner:** INTEGIN engineering owner.  
**Decision:** Commit the unmounted source-only prerequisites; do not compose a candidate bridge or launch a runtime.

## Review outcome

| Lens | Result | Owner reconciliation |
| --- | --- | --- |
| Security and redaction | Approved with limitations and a requested explicit failure-artifact redaction assertion. It also highlighted that counter snapshots are separate operations rather than a single atomic replay/count transaction. | Added the focused assertion: a raw manifest identifier supplied to an invalid valid-proof observation cannot appear in its public bridge-failure artifact. The counter limitation is accepted because any non-`+1` observed delta becomes a failed public receipt and blocks the adapter; it cannot create a false successful case. |
| Assurance and failure semantics | Approved after reviewing the lifecycle failure writer, receipt contract, counter, mapper, and tests together. | Accepted. It confirmed exact mapping, typed reasons, both observed expiry status paths, missing-snapshot failure handling, duplicate/failure artifacts, and redaction boundaries. |
| Correctness and verification | Did not complete within the bounded response window. | Not counted as approval. Deterministic focused tests, repository-wide Go tests, and Go vet substitute only for the missing perspective; this is not represented as a third independent approval. |

## Accepted behavior

The device-proof verifier now exposes only a closed typed reason to the unmounted handler. The handler retains its existing HTTP response behavior while optionally capturing before/after replay counts only after verified tenant scope is available and only around a successful consume. If a counter is absent or unusable, the handler remains authoritative; the mapper receives an incomplete valid-proof observation, whose receipt attempt records a redacted bridge failure and causes adapter rejection later.

The new mapper emits only seven exact HTTP cases and ignores diagnostics or near-matches. It does not receive proofs, signatures, identities, keys, credentials, request IDs, or raw errors. The issuer-expiry HTTP 403 path is now a deliberately accepted v2 observed status; proof-verification expiry remains HTTP 401. No status is silently normalized.

## Residual limitations and prohibitions

The counter snapshot is intentionally a source-owned observation seam, not a cross-store atomic transaction. A concurrent change can produce a delta other than `+1`; that becomes failed evidence rather than a passing receipt. No candidate constructor attaches this mapper, no default runtime uses it, no Field publisher is added, and no live matrix claim is permitted. Candidate composition, fixture invocation/context, Field traffic, migrations, runtime changes, package enforcement, OIDC, and OpenBao remain closed.
