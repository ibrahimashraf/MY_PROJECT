# INTEGIN Pilot Policy Registration Path

**Status:** Source-only preparation complete. Registration is not wired into server startup and no pilot runtime activation has occurred.

## Registration Contract

Commit `0bbcc26` adds `RegisterPilotPreAcceptancePolicy` and `RegisterPilotPreAcceptancePolicyFromEnvironment` as explicit, uncalled registration seams. They exist solely for a future isolated pilot candidate and require all of the following before a policy can be attached to the authoritative processor:

| Requirement | Failure behavior |
|---|---|
| Runtime scope equals `pilot` | Registration returns an error and attaches nothing. |
| `INTEGIN_PILOT_PACKAGE_POLICY_REGISTRATION=enabled` | Registration remains disabled by default. |
| `INTEGIN_PILOT_PACKAGE_ENFORCEMENT=enabled` | Registration remains disabled by default. |
| A complete, prepared policy dependency is supplied | Registration returns an error and attaches nothing. |
| A pilot-only processor seam is supplied | Registration returns an error and attaches nothing. |

The contract deliberately requires both feature gates. A preparation gate alone cannot register enforcement, and an enforcement gate alone cannot register the policy. The registration constructor is absent from `cmd/integin-server` startup.

## Rollback Contract

Successful registration returns `PilotPolicyRegistration`. Its `Disable()` method is deterministic and idempotent: it removes only the attached pre-acceptance policy from the supplied pilot processor. It does not remove pilot data, delete receipts, alter assignments, change feature-gate files, affect OIDC/OpenBao, or touch the protected acceptance runtime.

The policy emits only fixed, privacy-minimized `policy_registered` and `policy_rolled_back` observation categories for this lifecycle. It does not log work-package content, signatures, private keys, sync secrets, or protected fixture values.

## Validation Record

The staged source changes passed repository-wide `go test ./... -count=1` and `go vet ./...` before commit. Focused registration tests cover unsafe-scope rejection, register/rollback idempotency, and disabled-default environment behavior. A post-commit race-detector attempt could not run because the Windows Go configuration has `CGO_ENABLED=0`; no environment/toolchain change was made to bypass that limitation.

The post-commit no-enablement check found zero startup references to `RegisterPilotPreAcceptancePolicy`, `SetPreAcceptancePolicy`, or the work-package policy package. The pilot listener on `127.0.0.1:18080` was absent. Acceptance `/healthz` and `/readyz` each returned HTTP 200.

## Explicit Non-Changes

No environment variable was enabled, no runtime was restarted, no registration function was called, no processor policy was attached, no database/data/volume was changed, no migration was applied, and no acceptance source or runtime was touched. OIDC remains disabled and OpenBao remains sealed and unwired.

## Next Gate

The next possible step is a separately governed disposable pilot exercise defined in `DISPOSABLE_PILOT_ENFORCEMENT_ACTIVATION_MATRIX_2026-08-18.md`. It requires scenario evidence for valid binding, all defined rejection paths, pre-policy signature failure, idempotent duplicate handling, explicit policy removal, candidate restart absence, and continuous acceptance health. Completion of the matrix is evidence gathering only; it does not authorize acceptance enforcement or production rollout.
