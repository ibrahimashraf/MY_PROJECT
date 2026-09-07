# INTEGIN Pilot Fixture Retention and Automatic Revocation Policy

**Decision date:** 2026-08-18  
**Decision:** Retain the source-owned pilot fixture **only with automatic hard-gate revocation**.  
**Scope:** The local pilot manifest input-generator fixture and its public wrapper. This policy does not authorize a live proof matrix, enforcement, OIDC, OpenBao, or production use.

## Decision rationale

Three completed independent positions—from security/process isolation, industrial assurance evidence, and software-maintenance governance—converged on conditional retention. The local fixture has useful provenance value now that it is tracked and bounded. Immediate deletion would remove that value without reducing a demonstrated risk. Unconditional retention would be unsafe because future source or wrapper drift could expand the fixture beyond its approved local generation role.

> The fixture is retained as a **local generation-provenance component**, not as a source of candidate truth. Any failure of its approved isolation, redaction, lifecycle, source identity, or claim boundary revokes fixture use for that wrapper invocation.

## Automatic hard-gate revocation

Automatic revocation means the wrapper fails closed: it starts no candidate where failure occurs before launch, otherwise shuts down the candidate in `finally`, emits no accepted matrix result, and leaves package enforcement, OIDC, OpenBao, Field workflow, and acceptance authority unchanged. It does **not** delete source history or silently remove evidence needed for review.

| Trigger | Current enforcement | Required result |
| --- | --- | --- |
| Fixture entrypoint or signal-writer source drift | Exact SHA-256 verification of `cmd\pilot-manifest-fixture\main.go` and `internal\fixturesignals\signal.go` before candidate launch. | Fixture is not invoked and candidate must not start. |
| Missing, partial, unsupported, or unsafe fixture signal context | Source writer rejects the context. | Fixture exits nonzero; wrapper rejects the hard gate and performs mandatory shutdown. |
| Signal absence, duplicate, unexpected file, forbidden field, invalid version/step/ID/time, unsafe path, oversized artifact, or unsafe ACL | Wrapper exact inventory/schema/path/ACL validation. | No `fixture_generated` acceptance; mandatory cleanup and candidate shutdown. |
| Fixture process nonzero exit | Wrapper checks exit code. | Reject the hard gate and suppress output. |
| Candidate non-loopback/mismatched PID/binding, failed shutdown, or pre-existing listener | Wrapper listener/process checks. | Reject and stop the candidate where present. |
| Acceptance health/readiness not `200/200` before or after wrapper work | Wrapper health/readiness checks. | Reject; no fixture result is accepted. |
| Fixture signal directory or per-run artifact remains | Wrapper cleanup plus final absence verification. | Treat as failed cleanup; fixture is unavailable until corrected and revalidated. |
| Attempt to use fixture result as a live case verdict or matrix completion | Policy and validator claim boundary. | Quarantine the fixture from proof claims; require governance review before further use. |

The source-drift gate was tested twice using temporary public wrapper copies with a deliberately mismatched digest: once for the fixture entrypoint and once for the signal writer. Both tests rejected before candidate launch and preserved acceptance `200/200`.

## Controlled retention requirements

The fixture remains retained only while all of the following remain true: it remains a local input generator; it has no HTTP, database, network, Field-client, receipt-authority, proof-authority, or status-authority role; it receives only the child-specific signal directory and fixed contract version; its stdout/stderr remain blackholed; it emits exactly one closed redacted `fixture_generated` signal; and the wrapper validates, cleans, and records only a public boolean.

Any source modification requires an explicit review that updates the reviewed source digests, re-runs focused/full validation, re-runs source-drift rejection tests, and updates this policy/evidence. A wrapper change requires an equally explicit review because the wrapper is the independent lifecycle and validation boundary.

## Explicitly prohibited claims

No user, developer, report, dashboard, or automated gate may claim that the fixture validates an HTTP case, Field binding, replay state, enforcement, OIDC, OpenBao, tenant isolation, production readiness, certificate eligibility, final receipt provenance, or live matrix completion. `fixture_generated` proves only that the reviewed fixture completed its local generation path within the current bounded wrapper flow.

## Manual quarantine and recovery

Manual quarantine is required when a security concern, source/wrapper review gap, repeated cleanup/ACL/lifecycle flakiness, output leakage, local filesystem weakness, scope-expansion proposal, or claim misuse is discovered. The immediate response is to stop invoking the wrapper, preserve the tracked source and non-secret failure evidence, keep acceptance independent and healthy, and require a review with source/wrapper tests before re-enablement. No private fixture material is attached to public evidence during this process.

## Current verified position

The fixture is retained under automatic hard-gate revocation. The latest normal smoke check reported `fixture_source_verified=true`, `fixture_generated=true`, loopback candidate startup/shutdown, output suppression, and acceptance `200/200` before/after. The live eight-case matrix remains unproven and not ready.
