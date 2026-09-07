# INTEGIN Observability Foundation Contract v1

> **Status:** Governance-only vocabulary. No collector, SDK, exporter, pipeline, dashboard, alert, endpoint, secret, or runtime telemetry emission is introduced by this artifact.

## Purpose and authority boundary

Observability is diagnostic evidence. It helps operators determine whether a dependency, request class, sync flow, or service state needs investigation. It must never become a workflow decision-maker, authorization system, tenant directory, device registry, certificate authority, or evidence store. Go/PostgreSQL remains authoritative for all primary state.

## Low-cardinality data contract

| Signal | Allowlisted values | Prohibited values |
|---|---|---|
| Metrics | Service, environment, route template, method, status, operation, outcome, dependency, error class. | Tenant/org/user/device/resource/evidence/object identifiers, payload hashes, request IDs, and exception messages. |
| Traces | OpenTelemetry service/environment fields, route template, HTTP method/status, operation, outcome, error type. | Request/response bodies, SQL, connection strings, user identity, tenant/device IDs, evidence/payload/object content, tokens, and exception messages. |
| Logs | UTC time, severity, stable event name, service/environment, operation/outcome/error class, generated trace ID; selected bounded operational fields. | Headers, cookies, tokens, JWTs, secrets/keys, connection strings, payloads, raw evidence or object contents, tenant/org/user/device IDs, and raw exception messages. |

The contract requires route **templates**, not raw URL paths, and error **classes**, not raw error strings. This preserves operational aggregation without creating high-cardinality or privacy-sensitive telemetry.

## Baseline signals and alerts

The contract reserves seven metric names covering HTTP requests/latency, sync outcomes, evidence operations, authority validation, dependency health, and telemetry drops. It also defines the minimum alert classes below. Numerical thresholds, windows, paging routes, suppression, and escalation ownership must be set by the accountable environment owner after baselining; they are intentionally not inferred here.

| Alert class | Detects |
|---|---|
| Service health/readiness unavailable | Health or readiness cannot establish a usable service state. |
| Elevated server-error ratio | A material rise in classified server-side request failures. |
| Sync rejection/retry spike | A material rise in sync rejection or retry behavior. |
| Evidence persistence failure | Failure to persist or retrieve required evidence metadata/object operations. |
| Authority validation failure spike | Unexpected rise in authority-package validation failures. |
| Dependency health degraded | Database, object storage, or approved identity dependency health degradation. |
| Telemetry drop/pipeline silence | Instrumented telemetry is lost or unexpectedly absent. |

## Implementation prerequisites

Implementation requires a separately approved collector/exporter topology, secret-safe transport and workload policy, data retention/access decision, sampling/cost rationale, negative prohibited-data tests, and a pilot-only verification campaign. No acceptance OIDC or production configuration work is implied.

## Artifacts

| Artifact | Purpose |
|---|---|
| `contracts/observability_foundation_v1.json` | Machine-readable signal vocabulary, prohibited values, alert classes, and prerequisites. |
| `contracts/observability_foundation_v1_test.go` | Guard for high-risk allowlist/prohibited-data/alert invariants. |
