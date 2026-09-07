# INTEGIN Verification Coverage Matrix — 2026-08-16

## Purpose

This is the current controlled-pilot evidence index. **Passed** means a documented verification was observed. **Pending** means intentional unfinished evidence remains. **Manual-only** means a validated human-controlled procedure exists because the local automation path is not reliable. It does not claim production readiness.

## Current verification status

| Verification area | Status | Evidence / limitation | Next trigger |
|---|---|---|---|
| Go tests and vet | **Passed** | Focused live-matrix target-guard test, complete `go test ./...`, and `go vet ./...` passed after the strict pilot-origin guard change. | Re-run after Go source changes. |
| Go race detection | **Passed** | Offline cached Docker `go test -race ./...` passed against the pilot source. | Re-run after concurrency-sensitive changes. |
| Flutter tests | **Passed** | Full field-app suite passed: 31 passed, 1 intentional skip. | Re-run after Flutter storage/sync/UI changes. |
| Flutter native Windows build | **Pending** | Windows runner exists; MSVC and CMake are registering, Windows SDK installation remains in progress. | Run `flutter doctor`, then launch pilot-only field app. |
| Advisory service | **Passed** | Isolated advisory verification passed and primary workflow remains `blocking=false`. | Re-run after advisory service changes. |
| Secondary advisor UI | **Passed** | Type and production-build verification passed with non-blocking packaging observations. | Re-run after UI changes. |
| Acceptance/pilot health | **Passed** | Read-only checks returned 200 for health/readiness on acceptance 8080 and pilot 18080. | Recheck before/after pilot workflow. |
| OIDC disabled-at-rest | **Passed** | `/identity/session` returned 404 on both runtimes. | Recheck before/after walkthrough or identity work. |
| OpenBao boundary | **Passed at current scope** | Rehearsal complete; OpenBao remains sealed and unwired. | Separate approved integration plan only. |
| Tenant RLS and durable pilot state | **Passed** | Pilot tenant-context, identity-negative, durable authority, and sync controls passed. | Re-run after migrations/identity changes. |
| Evidence storage and recovery | **Passed for pilot integration** | Signed S3, evidence, archive/manifest restore, and isolated PostgreSQL restore drills passed. RustFS RC is not production-qualified. | Re-run after storage/recovery changes. |
| Explicit pilot target guard | **Passed** | Live matrix accepts only `http://127.0.0.1:18080`; omitted, acceptance, alternate-port, HTTPS, and path targets fail before HTTP activity. | Re-run after matrix target changes. |
| Signed live sync/evidence matrix | **Passed** | Protected pilot seed, pilot restart, and exercise passed against explicit pilot 18080; acceptance stayed isolated. | Re-run after authority-loading changes. |
| Pilot runner orchestration | **Manual-only** | Child-process exit-code propagation can misclassify success; manual explicit-target procedure is the validated interim control. | Redesign only with deterministic proof. |
| Field-to-operator acceptance | **Pending** | Human-visible offline queue, receipt, duplicate, operator review, and advisory checks remain. | Complete after Windows toolchain verification. |

## Boundaries for the pending walkthrough

The field app must use the Windows-debug-only pilot bootstrap, an explicit `http://127.0.0.1:18080/sync` endpoint, and the retained isolated pilot fixture. It must not target acceptance, enable OIDC, unseal/wire OpenBao, expose fixture material, or make advisory output binding.

## Next evidence refresh

After the visible walkthrough, update this matrix with queue durability, authoritative receipt, replay-safe duplicate, operator review, advisory-only, cleanup, health/readiness, and disabled-OIDC evidence. Use those observations to prioritize the first pilot observability implementation slice.

## References

- `CURRENT_STATE.md`
- `operations\pilot\PILOT_WORKFLOW_ACCEPTANCE_PLAN.md`
- `docs\PILOT_OBSERVABILITY_FOUNDATION_PLAN.md`
