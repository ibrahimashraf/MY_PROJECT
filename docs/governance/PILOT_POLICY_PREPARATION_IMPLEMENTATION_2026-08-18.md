# INTEGIN Pilot Policy Preparation Implementation

**Status:** Implemented and validated as source-only preparation.  
**Activation status:** **Disabled.** No policy is registered on a processor and no runtime configuration has been changed.

## Purpose

This implementation completes the next governed prerequisite identified by the Phase 8 enforcement evaluation: a narrow, fail-closed way to construct a pilot-only work-package pre-acceptance policy and emit privacy-minimized preparation/decision observations. It deliberately does not activate that policy.

## Implemented Components

| Component | Responsibility | Safety boundary |
|---|---|---|
| `workpackageenforcement.PreAcceptancePolicy` | Evaluates only verified `InspectionSubmitted` transactions after the authoritative processor has completed identity, authority, signature, and sequence checks. | It is an adapter object only; it is not self-registering and cannot invoke the processor. |
| Current-assignment comparison | Resolves tenant/organization-scoped server-held inspection/device assignment and requires matching package ID, package version, inspection ID, and authority epoch. | Payload package identifiers alone cannot satisfy the policy. |
| Existing payload validator | Validates the package hash, field sequence, and typed findings against the approved server package. | Validation occurs only after assignment matching and only for the limited inspection-submission operation. |
| Privacy-minimized observations | Emits only fixed lifecycle/decision states and fixed categories such as `policy_prepared`, `binding_valid`, and `assignment_mismatch`. | No payload, signature, key material, user identifier, inspection identifier, or raw error text is present. |
| `server.PilotPolicyPreparationConfig` | Requires isolated pilot runtime, `127.0.0.1:18080`, explicit preparation state, complete dependencies, and enforcement inactive. | Rejects acceptance scope, non-loopback scope, incomplete dependencies, and any active-enforcement request. |
| Environment factory | Defaults to no policy unless `INTEGIN_PILOT_PACKAGE_POLICY_PREPARATION=enabled`. | `INTEGIN_PILOT_PACKAGE_ENFORCEMENT=enabled` is rejected; the factory never calls `SetPreAcceptancePolicy`. |

## Validated Behavior

The focused policy tests prove that an exact current assignment and approved typed package payload can be evaluated by the adapter, that an assignment mismatch is rejected, and that non-inspection operations are outside the policy scope. The pilot preparation tests prove that disabled preparation returns no policy, unsafe acceptance/enforcement configuration is rejected, and a complete pilot preparation configuration returns an unregistered policy object.

The full Go regression suite (`go test ./... -count=1`) and `go vet ./...` passed. A final source check confirmed that `cmd/integin-server/main.go` contains no policy-registration or work-package-enforcement startup reference. The protected acceptance runtime was revalidated healthy and ready on `127.0.0.1:8080`; the pilot candidate remains stopped.

## Recovery Incident

During final validation, Docker Desktop had stopped and its existing acceptance PostgreSQL container was unavailable, causing acceptance health to fail. Docker Desktop and only the existing `integin-postgres` dependency container were restored; direct `pg_isready` confirmed PostgreSQL availability before the reviewed acceptance restore script returned health and readiness to HTTP 200. No database data, environment value, runtime policy, OIDC state, OpenBao state, or source configuration was altered during recovery.

## Explicit Non-Changes

The implementation does not mount a route, set a runtime environment variable, call `SetPreAcceptancePolicy`, inject the policy into a processor, change the pilot candidate, start the pilot listener, or alter acceptance. Package enforcement remains disabled. OIDC remains disabled, and OpenBao remains sealed/unwired.

## Remaining Activation Matrix

Before activation can be considered, a separate pilot-only phase must implement the intentionally missing runtime composition and complete a disposable evidence matrix. That matrix must prove allowed exact binding, missing binding rejection, package hash/version mismatch rejection, expired/revoked assignment rejection, malformed typed-payload rejection, invalid-signature policy non-invocation, duplicate-safe behavior, privacy-minimized observations, and configuration rollback without data deletion.

> This source-only preparation is **not** an authorization to enable package enforcement.
