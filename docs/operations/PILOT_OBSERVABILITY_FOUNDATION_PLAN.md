# INTEGIN Pilot Observability Foundation Plan

**Status:** Prepared for post-acceptance implementation; **not deployed**.
**Scope:** Isolated local pilot only. This document does not authorize a collector, dashboard, alerting process, port, container, runtime setting, OIDC change, OpenBao change, or production authority path.

## Objective

After the human-visible field-to-operator workflow is accepted, introduce the smallest self-hosted observability slice that helps operators detect pilot degradation, investigate authoritative sync/evidence failure, and rehearse recovery. Observability observes behavior only: it cannot approve work, authorize a device, alter a receipt, change tenant scope, or make advisory output binding.

## Existing baseline

| Boundary | Existing evidence | Observability use |
|---|---|---|
| Runtime availability | Loopback health and readiness endpoints exist. | Measure availability and dependency readiness. |
| HTTP behavior | Correlation-aware structured request logging exists. | Preserve correlation in access-controlled pilot logs only. |
| Signed sync | Deterministic applied, duplicate, held, rejected, conflict, and security outcomes exist. | Aggregate outcome counts and latency only. |
| Evidence | Evidence storage is separate and digest-bound. | Measure result class and latency only. |
| Advisory | Advisory is non-blocking with `blocking=false`. | Measure availability separately from primary workflow health. |
| Recovery | Pilot backup/restore controls are documented. | Record rehearsal pass/fail and timestamp only. |

## Minimal signal contract

| Signal family | Required pilot measurement | Prohibited telemetry data |
|---|---|---|
| Availability | Process up; health/readiness result; dependency state. | Credentials, URLs beyond fixed route names, database text. |
| HTTP | Count and duration by method, fixed route template, status class. | Request/response bodies, query values, authorization headers, user identifiers. |
| Sync | Outcome count and flush duration by result class. | Transaction, device, authority, receipt, payload, signature, or hash values. |
| Evidence | Result/retry class and storage latency. | Object key, digest, body, bucket credential. |
| Advisory | Result class/latency and non-blocking boundary assertion. | Prompts, model inputs, evidence references, primary identifiers. |
| Recovery | Rehearsal pass/fail and timestamp. | Backup paths, manifests, recovery material. |

## Pilot ownership and response

| Alert/view | Initial owner | Required response |
|---|---|---|
| Availability failure | Pilot operator | Confirm isolated pilot service/dependency state; do not touch acceptance. |
| Sync/evidence degradation | Field workflow and storage owners | Inspect aggregate class first; use local correlation only through controlled access. |
| Advisory degradation | Advisory owner | Mark advisory unavailable; prove primary workflow remains unchanged. |
| Recovery rehearsal failure | Recovery owner | Stop promotion discussion and repair rehearsal evidence before broadening scope. |

## Implementation gates

1. Complete the isolated human-visible walkthrough.
2. Select only signals that resolve an observed operator or recovery question.
3. Approve a separate pilot-only, reversible, self-hosted deployment design.
4. Prove prohibited data does not enter telemetry and rollback leaves pilot/acceptance behavior unchanged.

## Explicit exclusions

This plan does not enable OIDC, unseal or wire OpenBao, create production enrollment, change RLS, introduce automatic remediation, collect behavioral analytics, or authorize a managed service.

## References

- `CURRENT_STATE.md`
- `operations\pilot\PILOT_WORKFLOW_ACCEPTANCE_PLAN.md`
- `docs\OBSERVABILITY_FOUNDATION_CONTRACT_V1.md`
