# INTEGIN Disposable Pilot Enforcement Activation and Rollback Matrix

**Status:** Prepared for a future controlled pilot exercise.  
**Current runtime state:** Policy registration and package enforcement remain disabled everywhere.

## Purpose and Scope

This matrix is the required evidence plan for a future, isolated-pilot-only enforcement exercise. It applies only to the disposable pilot database, disposable fixture device, separate candidate binary, and loopback listener at `127.0.0.1:18080`. It must not alter the acceptance listener at `127.0.0.1:8080`.

The source registration path is intentionally uncalled. This matrix does not authorize its use; it defines the evidence that must be collected if a separate controlled activation is approved.

## Preconditions

| Requirement | Evidence before the exercise |
|---|---|
| Recovery point | Existing pilot database backup is present and verified. |
| Isolation | Pilot candidate listener is separate from acceptance; acceptance health and readiness are HTTP 200 before and after each pilot action. |
| Feature state | Pilot preparation/registration/enforcement gates are absent or disabled before setup; no acceptance environment is edited. |
| Identity controls | OIDC remains disabled. OpenBao remains sealed and unwired. |
| Disposable data | Only the existing disposable tenant, organization, device, authority, package, assignment, and inspection fixture are used. |
| Observation handling | Evidence captures only status/result category, transaction ID or request correlation reference where already non-secret, and non-sensitive hash prefixes. No payload, signature, private key, sync secret, or user content is printed. |

## Activation Steps

The future activation must be executed in this exact sequence: verify the preconditions, start only the separate pilot candidate, confirm pilot health, build an in-memory policy from the approved repository/assignment resolver, set both explicit pilot gates only in the candidate process environment, invoke the registration seam only in the pilot startup composition, and confirm acceptance is unchanged. There must be no persistent acceptance configuration change and no data migration.

## Required Matrix

| ID | Scenario | Expected authoritative result | Required non-secret evidence |
|---|---|---|---|
| A1 | Exact signed inspection submission with current device assignment, matching package ID/version/hash, matching authority epoch, and valid typed findings | `ACCEPTED`; exactly one receipt/state mutation | Pilot status/result, receipt reference, policy category `binding_valid`, acceptance health/readiness 200 |
| A2 | Current signature and authority but no assignment for inspection/device | `REJECTED`; no receipt or accepted-state mutation | Result category `assignment_unavailable` or equivalent, before/after receipt/state count, pilot and acceptance health |
| A3 | Assignment exists but package ID, version, or authority epoch differs | `REJECTED`; no receipt/state mutation | Result category `assignment_mismatch`, before/after receipt/state count |
| A4 | Assignment/package expired or revoked | `REJECTED`; no receipt/state mutation | Result category for unavailable/invalid assignment, before/after receipt/state count |
| A5 | Matching assignment but malformed package payload or invalid typed field response | `REJECTED`; no receipt/state mutation | Result category `malformed_payload` or `package_validation_failed`, before/after receipt/state count |
| A6 | Invalid transaction signature | `SECURITY_FAILURE` before policy invocation | Security result plus absence of a policy-evaluated observation; no receipt/state mutation |
| A7 | Reuse of an already accepted transaction | Existing duplicate-safe result; no second receipt/state mutation | Duplicate result, receipt/state count unchanged |
| A8 | Registration rollback after a controlled rejection/acceptance | Pilot policy unregistered; future valid pilot submission follows the pre-policy path only if the rollback plan explicitly permits it | `policy_rolled_back` observation, no data deletion, pilot/acceptance health, registration inactive |
| A9 | Candidate restart after rollback | Policy is absent unless both gates and pilot-only composition are intentionally recreated | Startup evidence showing no registered policy; acceptance remains unaffected |

## Stop Conditions

The exercise stops immediately if acceptance health/readiness is not HTTP 200, any action targets port 8080, an environment scope is not unequivocally pilot-only, an unexpected receipt/state mutation occurs, a non-disposable record is selected, a secret would be logged, or any OIDC/OpenBao state changes. The response is to disable/unregister the policy in the pilot process, stop the pilot candidate, preserve non-secret evidence, and restore the pilot backup only if data integrity requires it.

## Rollback Procedure

The controlled rollback is configuration and registration removal, not data deletion. Call `Disable()` on the pilot registration, clear both candidate-only registration/enforcement gates, stop the pilot candidate, confirm port 18080 is closed, and verify acceptance health/readiness. Preserve the pilot database and evidence unless a matrix failure requires restoration from the verified pilot backup. Do not alter acceptance, OIDC, OpenBao, package enforcement defaults, or the user-facing INTEGIN workflow.

## Completion Criterion

This matrix is complete only when all scenarios have non-secret evidence, every rejected case proves no accepted receipt/state mutation, rollback is demonstrated, and acceptance remains healthy throughout. Only then can a separate human-owned decision consider whether enforcement should remain pilot-only, be revised, or be rejected.

> Completing this matrix is evidence gathering. It is not a production rollout and it does not authorize acceptance enforcement.
