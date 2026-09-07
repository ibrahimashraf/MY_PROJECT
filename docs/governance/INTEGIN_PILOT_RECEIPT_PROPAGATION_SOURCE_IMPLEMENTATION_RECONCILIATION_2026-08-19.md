# INTEGIN Pilot Receipt Propagation — Source Implementation Reconciliation

**Recorded:** 2026-08-19  
**Scope:** Source-only hardening of the isolated wrapper and launcher.  
**Decision:** Accepted for source-control recording, subject to the residual risks and no-runtime boundary below.

## Accepted implementation controls

The wrapper now creates a canonical lowercase-hex run identifier only after protected acceptance checks return `200/200` and the candidate listener is absent. It creates the fresh receipt run and its restricted `candidate-control` subdirectory, passes the exact receipt context to the launcher as an in-memory PowerShell parameter, and omits receipt-context values from public status output.

The launcher validates that parameter object before it loads candidate configuration or creates any candidate artifact. It clears the candidate child environment, rebuilds it from an allow-list, adds receipt context only to that child specification, and keeps OIDC and package enforcement disabled. It creates PID/log artifacts solely beneath the validated per-run control directory, using exclusive creation and restrictive ACLs. Both scripts reject unsafe path components and broad-write ACLs, fail closed when required NetTCPIP/ACL capabilities are unavailable, and retain the fixture child’s minimal receipt-free environment.

| Evidence item | Result | Scope limit |
| --- | --- | --- |
| `test-receipt-propagation-static.ps1` | Passed | Parses and inspects source only; it does not start any process. |
| Security-boundary review | `ship`, high confidence | Source-only assessment; no host or runtime proof. |
| Reliability-and-validation review | `ship`, high confidence | Source-only assessment; no host or runtime proof. |

## Repository boundary

The final non-executing structural validation passed again. `integin-pilot-source` is clean, but `operations\` is not inside an existing Git worktree. The changed wrapper, launcher, static test, and governance records therefore cannot be committed without creating or selecting a separate operational repository, which this work did not do. This is a source-control boundary, not a runtime or implementation failure.

## Residual risks and explicit exclusions

Windows filesystem checks and ACL assignment cannot be fully atomic in the current PowerShell implementation. The scripts reduce and document the resulting local TOCTOU window, but do not claim to eliminate it. Host-level privileged observers can inspect child environment memory, including the short-lived signing key required by the current server contract. NetTCPIP ownership evidence is an operational prerequisite; unsupported or insufficiently privileged environments fail closed.

No wrapper, launcher, candidate, fixture, HTTP endpoint, Field transport, migration, package-enforcement, OIDC, OpenBao, or acceptance runtime action was executed. No private configuration, fixture body, signing key, sync secret, or database credential was read, printed, uploaded, or committed during this validation.

## Remaining authority gate

This accepted source-only work is **not** runtime authorization. A later candidate or fixture exercise requires a separate explicit authorization after an acceptance `200/200` preflight, listener-absence confirmation, runtime evidence plan, and cleanup/rollback plan. The next implementation activity must not add Field transport until the existing source-derived eight-case receipt coverage gate is separately satisfied.
