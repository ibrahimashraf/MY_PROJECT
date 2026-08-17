# INTEGIN — Dual-Service Implementation

## Goal
Complete the optional INTEGIN dual-service delivery: validate and checkpoint the Quiet Signal advisor console, implement and test the Python FastAPI advisory service, and preserve the advisory-only boundary across every integration point.

## Phase Status

| Phase | Status | Scope |
|---|---|---|
| 1. Shared Kernel | complete | Immutable tenant-scoped events, shared types, response definitions, notifications, feature flags, calibration records, tests, formatting, and vet validation. |
| 2. Core Engines | complete | Inspection, template, certificate, data completeness, cross-engine acceptance tests, formatting, and vet validation. |
| 3. Supporting Engines | complete | Authorization, device trust, offline sync, cross-engine acceptance tests, formatting, and vet validation. |
| 4. Infrastructure | complete | Event store, event bus, aggregate replay repository, MinIO storage, SMTP provider, migrations, documentation, tests, and vet validation. |
| 5. Platform Engines | complete | Notifications, audit, environment/release, planning, reporting, integrations, Standards Vault, feature flags, QR verification, calibration, Feature Console, acceptance tests, formatting, and vet validation. |
| 6. AI Advisory Layer | complete | Advisory contracts, AI-free zones, seven monitoring lenses, reasoning traces, regulation analysis, conflict resolution, client requests, canary metadata, acceptance tests, formatting, and vet validation. |
| 7. Go AI-Service Integration | complete | Versioned HTTP contract, guarded zones, timeouts, health checks, tenant validation, response normalization, configuration, and mock validation. |
| 8. Secondary Advisor UI | in_progress | Quiet Signal React/TypeScript console with evidence-linked insights, reasoning traces, health/canary metadata, and advisory safeguards. |
| 9. Python Advisory Service | complete | FastAPI `/healthz` and `/v1/advisory`, allowed-zone enforcement, deterministic monitoring-lens summaries, response metadata, and `blocking=false`. |
| 10. Cross-Service Handoff | in_progress | Final Go validation, persistent records, UI checkpoint, and delivery of the working console and service instructions. |

## Current Phase

Phase 8 — Secondary Advisor UI validation and checkpoint preparation.

## Acceptance Criteria

1. The advisor UI renders at desktop and mobile sizes, exposes evidence-linked signals and reasoning traces, and clearly communicates that insights cannot change primary INTEGIN workflows.
2. The UI passes TypeScript validation and production build validation, with only documented non-blocking asset/chunk warnings.
3. The Python service accepts only `MONITORING` and `REGULATION` zones, rejects all AI-free zones, and returns the v1 contract with confidence, rationale, evidence references, provider metadata, limitations, and `blocking=false`.
4. The Python service remains deterministic and does not mutate INTEGIN state or receive secrets, private prompts, or primary decision controls.
5. Go tests and vet remain passing after the dual-service additions.
6. A single pre-delivery UI checkpoint is created before presenting the console to the user.

## Deployment-Integration Acceptance Criteria

1. The AI service client communicates through an explicit versioned request/response contract and never receives AI-free-zone requests.
2. Provider timeouts, non-2xx responses, malformed responses, and unavailable services become explicit advisory errors; primary workflow state remains unchanged.
3. Provider responses are normalized through the existing advisory contract, including `blocking=false`, confidence, rationale, evidence references, model metadata, and limitations.
4. Secondary advisor view models expose only approved advisory fields and never expose secrets, hidden prompts, private credentials, or primary decision mutation hooks.
5. Health checks and canary metadata express baseline, candidate, health, traffic percentage, and rollback readiness.
6. Integration tests are opt-in or use local mocks; default `go test ./...` remains dependency-free.
7. Deployment configuration is documented without embedding credentials in source control.

## Non-Negotiables

AI remains advisory only. The integration client cannot set verdicts, approve inspections, issue certificates, validate calibration, authorize requests, accept sync transactions, or expand public QR projections. Service failure must degrade advisory panels gracefully without blocking or mutating primary workflows.

## Design Decisions

| Decision | Rationale |
|---|---|
| Use a versioned HTTP contract with an injectable transport. | It supports the Python service while preserving deterministic Go tests and allowing future provider changes. |
| Normalize all provider output through `internal/advisory`. | The existing contract already enforces confidence, rationale, evidence, metadata, and `blocking=false`. |
| Keep secondary advisor data separate from primary domain aggregates. | A deployment failure or stale insight must never alter authoritative business state. |
| Require explicit environment configuration for live integration. | Credentials and service endpoints must remain outside source code and default tests. |

## Phase 1–6 Closure

Phases 1–6 are complete with no unresolved implementation blockers. The mounted project contains the full deterministic INTEGIN core, platform engines, advisory contracts, monitoring, regulation/conflict/client-request flows, canary metadata, and passing `gofmt`, `go test ./...`, and `go vet ./...` validation.

## Errors Encountered

| Error | Attempt | Resolution |
|---|---:|---|
| Unix commands were initially used against the Windows project path. | 1 | Switched to PowerShell-compatible inspection and mounted-file operations. |
| Desktop terminal sidecar disconnected during verification. | 1 | Validated the mounted source directly from the sandbox and recorded reconnect requirement. |
| Inspection lifecycle test expected the submitted event at the wrong index. | 1 | Corrected the assertion; all Phase 2 tests passed afterward. |
| Initial SMTP test stub used an incorrect callback signature. | 1 | Replaced it with a clean `smtp.Auth`-compatible sender test; the email package then passed. |
| Phase 6 conflict merger declared an unused `baseOK` variable. | 1 | Removed the unused variable before rerunning the suite. |
| Initial conflict test fixture did not contain genuine concurrent conflicts. | 1 | Updated serial and status to diverge on both sides; the suite then passed. |
| A targeted edit used pre-gofmt test text. | 1 | Read the formatted file and applied a line-accurate edit. |
| A final-status edit targeted a duplicate error row. | 1 | Re-read the plan, removed duplicate rows, and applied final status edits separately. |

Deployment integration is complete. The optional Python AI service and secondary advisor UI can now be integrated through the documented contract without changing the advisory-only primary decision boundary.

## Dual-Service Validation Notes

The UI screenshot review described the design as strong and aligned with Quiet Signal. `pnpm check` and `pnpm build` pass. The CSS import ordering warning was fixed. Vite still reports that the generated `/manus-storage` hero asset remains runtime-resolved and that the JavaScript bundle is above the default 500 kB advisory threshold; neither blocks the build.

The Python service tests pass in the sandbox: 4 passed. The connected Windows Python environment does not currently have `pytest`, so local Windows test execution requires installing `ai_service/requirements.txt` first. Go validation was started on Windows but did not produce output within three 60-second waits; it must be confirmed or reported as an environment-timeout limitation before final delivery.

## Errors Encountered — Dual-Service Session

| Error | Attempt | Resolution |
|---|---:|---|
| Windows Python environment lacked `pytest`. | 1 | Validated the service in the sandbox after installing pytest; documented the Windows dependency setup. |
| CSS font import appeared after stylesheet rules. | 1 | Moved the font import before Tailwind imports; the warning is gone. |
| Windows Go validation produced no output after three 60-second waits. | 1 | Inspected the terminal and retained the process for diagnosis; final status remains pending. |
| Mounted-file read used an invalid negative range. | 1 | Re-read the file with a valid range and logged the issue. |
| An incorrect root `README.md` path was queried. | 1 | Used the existing `ai_service/README.md` and planning files instead. |

## Next Step

Confirm the Windows Go validation status without repeating the timed-out wait, then create the single pre-delivery UI checkpoint and deliver the UI/service handoff.
