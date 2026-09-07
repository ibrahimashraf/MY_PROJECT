# INTEGIN Pilot Receipt Context Propagation Plan

**Decision date:** 2026-08-19  
**Status:** Design-only; pending bounded review.  
**Scope:** A future one-way propagation of nonsecret receipt context from the approved wrapper to a future isolated candidate child.  

> This plan does not authorize editing or running the wrapper/launcher, starting a candidate, invoking the fixture, calling an HTTP route, sending Field traffic, applying a migration, or changing acceptance, enforcement, OIDC, or OpenBao.

## Context boundary

The wrapper already owns a fresh opaque candidate run ID and a trusted per-run fixture-signal directory. A future wrapper revision may create a separate trusted `manifest-receipts\<run-id>` directory and pass only the following three fields to the launcher for the candidate child.

| Child value | Rule |
| --- | --- |
| `INTEGIN_PILOT_MANIFEST_CANDIDATE_RUN_ID` | Existing wrapper lower-case 32-hex run ID; opaque and nonsecret. |
| `INTEGIN_PILOT_MANIFEST_RECEIPTS_DIR` | Newly created, empty, non-reparse-point directory beneath a wrapper-owned receipt root; inherits only trusted runtime ACLs and is revalidated immediately before launch. |
| `INTEGIN_PILOT_MANIFEST_RECEIPT_CONTRACT_VERSION` | Literal `2`. |

These values must never appear in fixture parameters, fixture files, fixture signals, logs, stdout, stderr, persistent process-wide environment, or user-facing output. The fixture's existing stripped child environment remains unchanged: its allow-list contains only `SystemRoot`, the fixture signal directory, and the fixture signal contract version.

## Proposed one-way lifecycle

| Step | Required behavior |
| --- | --- |
| Parent preparation | Verify acceptance `200/200` and absent candidate port; create and revalidate the fresh per-run receipt directory with no-reparse-point, restrictive-ACL, freshness, and exact-inventory discipline. |
| Child launch context | Wrapper passes the three values only as explicit launcher inputs. Launcher validates them and places them only in the candidate child environment immediately before process start. |
| Candidate boundary | The committed pilot constructor reads all three values only after existing explicit pilot gates. Missing/partial/invalid context fails closed; all absent values remain inert. |
| Fixture boundary | Fixture retains its stripped environment and receives no run ID, receipt directory, contract version, candidate endpoint, or receipt capability. |
| Parent cleanup | Wrapper stops the trusted candidate first, validates the public receipt directory as an exact artifact inventory for a later read-only adapter, and removes temporary resources under existing trusted cleanup rules. |

## Required implementation-time controls

The future implementation must use an explicit launcher parameter object rather than ambient inherited context. It must reject duplicate, non-empty, reparse-point, or broadly writable receipt directories before candidate start. It must not print values, generated paths, candidate stdout/stderr, private configuration, or process environment; launcher output remains suppressed.

The launcher must reject receipt context unless all three values are present and valid, then build a dedicated candidate child environment from an explicit allow-list and remove transient parent-side values after start. It must not inherit receipt context from its own process. Before launch it must create the candidate PID record and stdout/stderr sinks with restrictive ACLs, or refuse to launch; post-start ACL detection alone is insufficient. Candidate stdout/stderr must be restricted, ephemeral, and removed during trusted wrapper cleanup. Candidate identity remains subject to existing PID freshness, ACL, and listener-ownership checks. No receipt file is evidence until the child stops and a later adapter validates exact inventory and lifecycle finalization.

ACL validation must favor an allow-list of permitted principals over only name-matching broad groups, with explicit target-environment review for local and organizational identities. The future implementation must include non-production, non-Field operational tests for pre-start receipt-directory validation, explicit launcher parameterization, child-only receipt context, fixture stripped-environment preservation, candidate stop, and secure cleanup. Those tests remain separately gated and are not authorized by this plan.

## Mandatory stop conditions

| Condition | Required response |
| --- | --- |
| Acceptance is not `200/200` | Stop before receipt-directory creation or candidate launch. |
| Candidate listener already exists | Stop; never reuse or adopt it. |
| Receipt directory is missing, reparse-pointed, pre-existing, non-empty, or broadly writable | Stop before launch and leave no candidate process. |
| Context is partial, malformed, or version is not `2` | Launcher and candidate constructor fail closed. |
| Fixture would receive any candidate receipt value | Stop as a fixture-isolation violation. |
| Candidate exits, binds unexpectedly, or lacks port ownership | Stop and cleanup under trusted PID ownership. |
| Receipt inventory lacks terminal finalization or has bridge failure | Later adapter rejects; no matrix-success claim is possible. |

## Excluded work and next authority gate

This design does not create execution permission. It establishes no public HTTP case driver, Field publisher, terminal-finalizer integration, or complete evidence matrix. The next gate is independent operational-security review followed by explicit authorization for a **staged implementation**. It is not authorization to launch a candidate or invoke a fixture.
