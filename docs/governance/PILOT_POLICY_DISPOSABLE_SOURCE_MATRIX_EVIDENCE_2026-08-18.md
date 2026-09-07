# INTEGIN Disposable Pilot Policy Source-Matrix Evidence

**Status:** Source/process-level evidence complete. This record does **not** enable a data-backed pilot service, package enforcement, acceptance enforcement, OIDC, or OpenBao.

## Scope and Safety Boundary

The exercise used the new non-networked `cmd/integin-pilot-policy-probe` command. Its only operational inputs were temporary process-scoped pilot runtime and policy-gate variables. It creates an in-memory authoritative processor, invokes the actual registration seam, proves registration activity, calls idempotent rollback twice, and exits. It neither opens a listener, loads a private file, connects to PostgreSQL/RustFS, nor starts the normal server.

Acceptance was checked before and after probe execution. Pilot port `127.0.0.1:18080` remained closed throughout. The normal `cmd\integin-server` entrypoint has no reference to `RegisterPilotPreAcceptancePolicy`.

## Acceptance Readiness Recovery

Before the matrix, acceptance `/healthz` was HTTP 200 but `/readyz` was HTTP 503. Read-only checks showed RustFS loopback health was HTTP 200 while PostgreSQL `127.0.0.1:5432` was unreachable. With explicit authorization, the retained `integin-postgres` container was started. The reviewed acceptance restore script correctly refused to duplicate the already running server. Acceptance `/healthz` and `/readyz` then both returned HTTP 200.

## Discovered and Corrected Gap

The initial expired-assignment regression proved a real source-only gap: `PreAcceptancePolicy` accepted a returned assignment whose `ExpiresAt` was already in the past. Package enforcement was not active in a runtime, so no live submission was affected.

Commit `80cb90b` (`Reject expired work package assignments`) resolves the gap. The policy now samples time once, passes that time to the current-assignment resolver, and rejects a zero or expired returned assignment with the fixed `assignment_unavailable` category. The existing positive fixture now supplies an explicit future expiry. Repository-wide Go tests and `go vet ./...` passed after the correction.

## Evidence Matrix

| ID | Required condition | Evidence collected | Result |
|---|---|---|---|
| A1 | Exact current package assignment binds successfully | `TestPreAcceptancePolicyRequiresCurrentAssignmentMatch` | Passed |
| A2 | Missing assignment rejects without acceptance | `TestPreAcceptancePolicyRejectsMissingAssignment`, commit `7a6b87e` | Passed |
| A3 | Package/epoch mismatch rejects without acceptance | Mismatch branch of `TestPreAcceptancePolicyRequiresCurrentAssignmentMatch` | Passed |
| A4 | Expired assignment rejects with recovery category | `TestPreAcceptancePolicyRejectsExpiredAssignment`, commit `80cb90b` | Passed after correction |
| A5 | Modified package hash or invalid typed response rejects | `TestValidatorRejectsModifiedPackageHashAndInvalidTypedResponse` | Passed |
| A6 | Signature failure precedes policy/acceptance | `TestPreAcceptancePolicyRunsAfterVerificationAndBeforeAcceptance` and `TestSyncRejectsTenantAndSignatureMismatches` | Passed |
| A7 | Duplicate request has no second mutation | `TestSyncAppliesAndDeduplicatesTransactions` | Passed |
| A8 | Explicit rollback removes only active policy | `integin-pilot-policy-probe` with both candidate-only gates enabled printed `PILOT_POLICY_REGISTRATION_PROBE=passed` | Passed |
| A9 | Fresh process with gates absent has no registration | Fresh probe exited as expected with `registration_absent`; no port was opened | Passed |

## Commits and Validation

| Commit | Purpose |
|---|---|
| `4a43a6b` | Adds the non-networked isolated pilot registration probe. |
| `80cb90b` | Rejects zero/expired assignments and adds expiry regression coverage. |
| `7a6b87e` | Adds explicit missing-assignment rejection coverage. |

The final `go test ./... -count=1` and `go vet ./...` runs passed. The race detector remains unavailable in this Windows Go environment because cgo is disabled; no toolchain setting was changed.

## Post-Exercise Isolation

After the probes exited, acceptance health/readiness were HTTP 200, acceptance `/identity/session` was HTTP 404, pilot port 18080 was closed, and normal server startup still had no registration reference. The temporary candidate-only process variables were removed at process exit. No pilot database record, receipt, assignment, package, runtime configuration, OIDC state, or OpenBao state was changed.

## Remaining Gate

This evidence proves the **source/process candidate** behavior, including the fixed expiration path. It is not a substitute for a future data-backed loopback pilot candidate that uses the prepared policy and real disposable assignment repository. Before package enforcement can be enabled in any listening pilot runtime, that separate candidate must collect the same non-secret receipt/state-count evidence against disposable pilot data and demonstrate the same rollback result. Acceptance enforcement remains out of scope.
