# INTEGIN Data-Backed Loopback Pilot Policy Candidate Design

**Status:** Design approved for source implementation only. No candidate has been built or started.

## Objective

Create a command distinct from `cmd\integin-server` that can load the existing disposable pilot PostgreSQL state, prepare the real work-package policy against `workpackagepg.Repository`, attach it through `RegisterPilotPreAcceptancePolicyFromEnvironment`, expose the existing loopback HTTP surface on `127.0.0.1:18080`, and collect only non-secret receipt/state-count evidence.

## Fixed Boundary

| Concern | Candidate rule |
|---|---|
| Normal startup | `cmd\integin-server` remains unchanged and has no registration reference. |
| Scope | Candidate accepts only `INTEGIN_PILOT_RUNTIME=pilot` and `INTEGIN_HTTP_ADDR=127.0.0.1:18080`. |
| Data source | Candidate requires the existing disposable pilot PostgreSQL repository and rejects a nil/non-durable database path. |
| Policy preparation | Candidate uses `workpackagepg.NewRepository(database)` with `NewPilotPreparedPreAcceptancePolicyCompositionFromEnvironment`. |
| Policy registration | Candidate uses `RegisterPilotPreAcceptancePolicyFromEnvironment`; all preparation, registration, and enforcement gates must be enabled only in its child process. |
| Observation | Candidate accepts only a privacy-minimized observer that permits fixed categories and never emits payload, signature, credential, key, fixture value, or customer content. |
| Evidence | Matrix runner records result category, receipt/state counts, assigned package identity/hash prefix where already non-secret, and acceptance health/readiness. |
| Rollback | Candidate exposes an explicit controlled shutdown path that calls `Disable()` before stopping the listener. Force-kill is not evidence of rollback. |

## Candidate Composition

The candidate will reuse the current durable database, device, authority, sync-state, evidence-store, OIDC-disabled, and manifest composition from the normal executable only by copying/refactoring shared configuration into a candidate-specific path. It will then add the following narrow sequence after processor construction and before mux construction:

1. Reject any scope other than isolated pilot and loopback port 18080.
2. Reject absence of the durable disposable pilot database.
3. Construct `workpackagepg.Repository` from the already opened database.
4. Build the prepared policy with the repository, fixed-category observer, `time.Now`, pilot runtime name, and loopback address.
5. Reject a nil prepared policy or registration result.
6. Register through the explicit registration seam; keep the returned registration handle inside the candidate lifecycle controller.
7. Start the loopback server only after registration succeeds.

## Matrix Runner and Rollback

The candidate must have a candidate-only lifecycle controller, not a normal-server route. The controller starts the listener, provides a local non-secret status suitable for the matrix runner, serializes matrix cases, then calls `Disable()` before listener shutdown. The runner must stop immediately on an unexpected accepted receipt/state mutation, an acceptance health/readiness result other than HTTP 200, an identity boundary change, a non-disposable target, or an observation outside its allowed categories.

The runner will record before/after counts for only disposable pilot records and will not print raw database values. If an exercise must restore state, it uses the existing pilot backup rather than deleting records ad hoc.

## Preconditions for Any Launch

Pilot PostgreSQL and RustFS must be running; the pilot backup must be present; acceptance health/readiness must be HTTP 200; pilot port 18080 must be closed; OIDC must remain disabled; OpenBao must remain sealed/unwired; and the source candidate must pass full Go tests and static analysis. The process-scoped gate variables are removed after every candidate invocation.

> This design does not authorize package enforcement in normal pilot or acceptance services. It limits a future disposable candidate to one loopback process and requires explicit rollback evidence.
