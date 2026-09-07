# INTEGIN Pilot Receipt Propagation — Staged Implementation Plan

**Decision date:** 2026-08-19  
**Status:** Design-only; no script changes authorized.  
**Scope:** A future staged source change to the isolated wrapper and launcher only.  

> This plan is not permission to modify or execute the wrapper/launcher, start a candidate, invoke the fixture, call an endpoint, send Field traffic, or change acceptance/runtime policy.

## Stage A: Secure receipt-run preparation in the wrapper

The wrapper will derive `manifest-receipts\<candidate-run-id>` only after acceptance health/readiness checks pass and after confirming port `18080` is absent. It will create a fresh receipt root and run directory using explicit allowed-principal ACLs, reject reparse points, reject pre-existing/non-empty directories, and validate that the final resolved run directory remains beneath the expected root. It will not log the run directory or receipt values.

The wrapper will construct an in-memory parameter object containing exactly the opaque run ID, receipt directory, and contract version `2`. It will pass that object to the launcher through an explicit PowerShell parameter, never through an inherited process environment. It will retain the object only until child creation returns.

## Stage B: Child-only environment in the launcher

The launcher will add a mandatory parameter object with three validated properties. It will reject absent/partial/extra/invalid values before loading candidate configuration or creating any candidate artifact. It will use `System.Diagnostics.ProcessStartInfo` with a cleared environment and an explicit allow-list of required existing candidate values plus the three receipt values. It will not use process-environment mutation for receipt context.

Before child creation, the launcher will resolve and validate every runtime input path it reads, including candidate executable, private environment/key paths, fixture handle, runtime roots, PID/log paths, and receipt paths. Each must remain beneath its expected root, avoid reparse points, and satisfy the target allowed-principal ACL policy. It will reject an existing PID/log artifact rather than overwrite it, create fresh artifacts using a restrictive explicit ACL, and revalidate them before child creation. The candidate child gets the receipt context; fixture child construction remains unchanged and excludes all receipt values.

The implementation must minimize the interval between final path checks and child creation and document the residual Windows TOCTOU window. `Assert-TrustedPathComponents`, `New-RestrictedDirectory`, and `New-ExclusiveRestrictedArtifact` reduce reparse and ACL risks, but cannot claim to eliminate every Windows filesystem race. A stronger OS-level bind/handle-passing mechanism is explicitly deferred. The current server contract also requires short-lived launcher-memory handling of the pilot signing key solely to construct the child environment; the launcher must never print or persist that value. The key and other child-environment secrets remain visible to host-level privileged observers and possible candidate descendants; that exposure is a documented residual of the present server contract, not a claim of secret isolation. The wrapper's existing public JSON contract version remains unchanged by this work.

## Stage C: Post-child cleanup and deferred evidence inspection

The wrapper will retain the existing trusted PID/listener shutdown path. After it verifies candidate stop, it will validate receipt-root/run-directory ACLs and exact inventory without reading receipt payloads. Candidate stdout/stderr and PID artifacts will be securely removed as part of wrapper cleanup. A later separately authorized adapter step may inspect only public receipt artifacts; this stage will not invoke it.

## Required non-executing tests and review evidence

| Test area | Required claim |
| --- | --- |
| Wrapper source structure | No wrapper call can create receipt context before acceptance `200/200` and absent candidate port checks. |
| Receipt path controls | Tests cover duplicate, non-empty, reparse-point, outside-root, and unsafe-ACL rejection before launcher invocation. |
| Launcher parameters | Tests cover absent, partial, malformed, and unsupported-version rejection before child creation. |
| Environment containment | Tests prove only the candidate child specification receives the three receipt fields; fixture child specification contains none. |
| Artifact controls | Tests prove PID/log paths are fresh, non-reparse-point, restricted, and cleaned after trusted stop. |
| Default isolation | Tests confirm neither normal startup nor the disabled pilot path references the staged receipt parameters. |

## Non-negotiable exclusions

No stage may alter `cmd\integin-server\main.go`, the Field app, fixture source, migrations, package enforcement, OIDC, OpenBao, acceptance configuration, or existing acceptance processes. The staged implementation must not execute any candidate/fixture process during development validation. A separate explicit authorization is required before any runtime exercise, even after code review and commit.
