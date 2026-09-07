# INTEGIN Pilot Processor-Enforcement Evaluation

**Status:** Complete evaluation; **not approved for activation**.  
**Scope:** Isolated pilot only. This evaluation does not enable a policy, register a validator, restart a runtime, alter acceptance, or change OIDC/OpenBao state.

## Decision

Phase 7 established that a Field-style device can retrieve a signed inspection work-package manifest, verify its Ed25519 signature, preserve an immutable inspection/hash identity, and encounter durable request-proof replay rejection. That delivery and binding evidence is necessary for enforcement, but it does not by itself make processor-level enforcement safe to activate.

The current codebase has the correct authoritative insertion point: `sync.PreAcceptancePolicy` executes after transaction signature, device, authority, and sequence checks, and before an accepted receipt or accepted-state mutation. The policy is `nil` by default, and `SetPreAcceptancePolicy(nil)` disables evaluation. However, no current startup composition builds a work-package policy and injects it into the processor. The HTTP handler's optional `PackageValidator` is also `nil` by default and is not suitable as the final enforcement path because it runs before the authoritative processor verification sequence.

> **Decision:** Keep package enforcement disabled. A separate pilot-only implementation and activation phase is required before any policy is injected into the authoritative processor.

## Evidence Reviewed

| Control or behavior | Evaluation result | Phase 8 implication |
|---|---|---|
| Signed manifest retrieval and public-key verification | Demonstrated in isolated pilot during Phase 7 | Field has a trustworthy package identity before any enforcement decision. |
| Immutable inspection/hash binding | Demonstrated as `inspection_id + NUL + package_hash` | A policy may require a specific approved package binding without silently substituting a newer package. |
| Proof replay protection | Immediate duplicate request ID returned HTTP 409 | Retrieval replay is protected; this does not replace transaction-level duplicate handling. |
| Authoritative policy position | Verified in `sync.Processor` after verification and before acceptance persistence | This is the only acceptable final enforcement seam. |
| Default behavior | Policy remains unset; handler validator remains nil | Current acceptance and pilot workflows remain unchanged. |
| Rejection behavior | Focused tests prove a rejecting policy yields `REJECTED` without accepting the transaction; a valid transaction can be accepted after policy removal | A rollback semantic exists at processor level, but it is not yet exposed through a pilot-only configuration path. |
| Invalid-signature ordering | Focused tests prove a tampered transaction does not invoke the policy | Enforcement cannot become an alternate identity/authority decision path. |
| Focused validation | `go test` passed for domain sync, work-package enforcement, and sync API packages | Current source behavior is covered, but live policy composition is not implemented. |

## Current Gaps That Block Activation

The following gaps are material. They are not documentation gaps and must not be bypassed with an environment-variable shortcut.

| Blocking gap | Why it matters | Required resolution before activation |
|---|---|---|
| No processor policy adapter | The validator is not composed into `PreAcceptancePolicy` and has not been shown to consume authoritative transaction scope safely. | Implement a dedicated adapter that derives tenant, organization, device, inspection, and package binding from the already-verified transaction and server-held assignment state. |
| No fail-closed pilot activation configuration | There is no distinct processor-enforcement configuration that proves the process is the isolated pilot and refuses incomplete inputs. | Add a pilot-only configuration validator requiring pilot runtime identity, loopback pilot address, explicit feature state, and all policy dependencies. Default must remain disabled. |
| No dedicated enforcement observation model | Manifest-retrieval observations do not explain why a primary transaction would be rejected or whether rollback succeeded. | Add privacy-minimized policy decision observations: evaluated, allowed, rejected category, configuration state, and rollback confirmation. Never log payloads, signatures, or private material. |
| No live enforcement matrix | Source tests do not yet prove a mounted pilot policy against durable state, receipt behavior, and a real Field-bound inspection. | Run a disposable pilot matrix covering valid, missing, expired, mismatched, malformed, duplicate, disabled-policy, and rollback cases. |
| No operator recovery contract | A rejected primary transaction requires an operator-visible explanation and a safe retry path. | Define Field and operator handling for policy rejection, package refresh, assignment expiry, and explicit rollback; do not silently mutate drafts or package bindings. |

## Required Pilot-Only Activation Criteria

Activation may be considered only when every criterion below has evidence. These are cumulative conditions; satisfying one criterion does not waive another.

| Category | Required criterion | Evidence required |
|---|---|---|
| Scope | The process proves it is the named pilot runtime, listens only on the pilot loopback address, and excludes acceptance. | Startup fail-closed tests and live listener check. |
| Composition | A processor-level work-package policy is built only when all pilot configuration, repository, and observation dependencies are valid. | Unit tests for disabled, incomplete, and enabled composition paths. |
| Authority | The policy receives only transactions already verified by the authoritative processor and resolves server-held assignment/package state under tenant and organization scope. | Ordering regression tests and PostgreSQL RLS integration evidence. |
| Field binding | The submitted inspection binding must match the signed-and-cached package identity for that inspection. | Controlled Field/pilot scenario with exact hash accepted and mismatch rejected. |
| Failure safety | Missing, expired, revoked, mismatched, malformed, duplicate, and unavailable package conditions return deterministic rejection categories without accepted receipt or workflow mutation. | Full pilot failure matrix and receipt/state assertions. |
| Observability | Policy decisions are visible through privacy-minimized pilot observations and distinguish configuration failure from package mismatch. | Captured pilot observations with no secret or payload leakage. |
| Rollback | Removing the explicit pilot gate or setting the processor policy to `nil` restores prior behavior without deleting drafts, receipts, assignments, or package history. | Controlled rollback test followed by health, receipt, and retry evidence. |
| Acceptance protection | Acceptance is never restarted, configured, or subjected to the policy during pilot work. | Before-and-after acceptance health evidence and configuration review. |

## Required Pilot Matrix

The future activation run must use disposable pilot records and record only status, category, transaction/inspection identifiers permitted by the pilot fixture, and hash prefixes. It must prove that an exact signed binding is accepted and that each unsafe condition is rejected without accepted-state mutation.

| Scenario | Expected policy result | Required non-mutation proof |
|---|---|---|
| Exact current assignment and package binding | Allowed | One authoritative accepted receipt with matching package identity. |
| Missing package binding | Rejected | No accepted receipt or sequence advance. |
| Package hash or version mismatch | Rejected | No accepted receipt; historic Field draft remains bound to its original hash. |
| Expired or revoked assignment/package | Rejected | No accepted receipt; operator/Field category identifies refresh or recovery requirement. |
| Malformed typed payload | Rejected | No accepted receipt or workflow mutation. |
| Tampered transaction signature | Security failure before policy | Policy observation count does not increase. |
| Duplicate transaction | Existing duplicate-safe result | Policy does not create a second accepted mutation. |
| Pilot policy rollback | Prior non-enforcement behavior restored only in pilot | Existing records remain intact; acceptance remains healthy and unchanged. |

## Rollback Boundary

The future implementation must treat rollback as a configuration and process-composition change, not as a database cleanup operation. The immediate rollback action must be to disable the dedicated pilot gate and ensure the processor has no pre-acceptance policy. It must not delete replay records, package history, assignment context, receipts, drafts, or authority records.

Rollback success requires that the pilot listener returns healthy, a previously valid transaction can be evaluated under the documented non-enforcement behavior, no protected acceptance process is restarted, and the pilot decision observations show the policy is disabled. Any uncertainty in that sequence is an abort condition, not a reason to broaden the change.

## Explicit Non-Changes

Package enforcement remains disabled. `SetPreAcceptancePolicy` was not called by a runtime startup path. The handler-level package validator was not enabled. The acceptance runtime, OIDC-disabled default, sealed/unwired OpenBao boundary, pilot backup, migration set, and authority paths were not changed by this evaluation.

## Next Governed Gate

The next implementation phase is **pilot-only processor-policy composition and observability**, followed by the required disposable pilot matrix. It is not an authorization to enable enforcement. Activation remains blocked until that implementation, its tests, its rollback procedure, and its pilot evidence satisfy every criterion in this document.
