# INTEGIN Pilot Policy Composition Seam

**Status:** Source-only composition prepared and validated.  
**Enforcement status:** **Disabled.** The new seam cannot register a policy on a processor.

## Purpose

This implementation adds the narrow composition boundary required between the previously prepared work-package policy and a future, separately governed pilot runtime activation. Its purpose is to prove that policy construction can be limited to complete isolated-pilot dependencies without broadening startup behavior or accepting a processor dependency.

## Composition Contract

`server.NewPilotPreparedPreAcceptancePolicyCompositionFromEnvironment` accepts only a repository that implements both server-held approved-package lookup and server-held current-assignment lookup, a privacy-minimized observation sink, a clock, and the runtime identity/listen address. It does not accept `sync.Processor` and has no path to call `SetPreAcceptancePolicy`.

| Input condition | Result |
|---|---|
| Preparation environment absent or not `enabled` | Returns `nil` policy; normal behavior remains unchanged. |
| Enforcement environment requested | Returns an error; an active-enforcement request cannot be prepared through this seam. |
| Preparation enabled but repository missing | Returns an error; no partial policy is returned. |
| Runtime is not the isolated pilot or listener is not `127.0.0.1:18080` | Returns an error from the fail-closed preparation configuration. |
| Complete isolated-pilot preparation | Returns an **unregistered** policy object only. |

## Rollback Boundary

There is no runtime rollback action because no runtime is changed. The disabled environment path returns no policy, and the composition object itself cannot alter a processor. The future activation phase must retain this property: rollback must remove the explicit pilot composition/registration while preserving package history, assignments, receipts, drafts, and authority state.

## Validation

Focused server tests cover disabled default behavior, missing repository failure, enforcement-active rejection, and complete-pilot unregistered policy construction. Full `go test ./... -count=1` and `go vet ./...` completed successfully. A direct startup-source check confirms `cmd/integin-server/main.go` has no reference to this composition seam, `SetPreAcceptancePolicy`, or work-package-enforcement construction.

Acceptance health and readiness remained HTTP 200 during final validation. The separate pilot listener was not started. OIDC remains disabled, and OpenBao remains sealed/unwired.

## Remaining Disposable Activation Matrix

The next phase must not simply call the new seam. It must implement a separate, auditable pilot runtime registration path and then prove the following against disposable pilot data:

| Scenario | Required evidence |
|---|---|
| Exact signed package assignment | One accepted authoritative receipt with matching inspection, package ID/version/hash, and authority epoch. |
| Missing or mismatched assignment | Deterministic rejection, no accepted receipt, no accepted-state mutation. |
| Expired or revoked package/assignment | Deterministic rejection with recovery category, no data deletion. |
| Malformed typed payload | Rejection after authoritative verification, no accepted receipt. |
| Invalid transaction signature | Security failure before policy invocation. |
| Duplicate transaction | Existing duplicate-safe result; no second accepted mutation. |
| Explicit pilot rollback | Policy unregistered/disabled only in pilot; drafts, packages, assignments, receipts, and acceptance remain intact. |

> This seam is preparation evidence only. It does **not** authorize or activate package enforcement.
