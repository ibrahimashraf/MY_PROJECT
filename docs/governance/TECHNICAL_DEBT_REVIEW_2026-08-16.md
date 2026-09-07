# INTEGIN Technical-Debt Review — 2026-08-16

## Purpose

This review prioritizes remaining INTEGIN work from the verified controlled-pilot state. It distinguishes repaired issues, current pilot limits, and production-gated capabilities. It does not authorize production rollout, OIDC enablement, OpenBao integration, or destructive cleanup.

## Priority summary

| Priority | Item | Current state | Recommended next action |
|---|---|---|---|
| Resolved | Live-matrix target ambiguity | The command now accepts only explicit pilot origin `http://127.0.0.1:18080`; regression tests reject missing, acceptance, alternate-port, HTTPS, and path targets. | Retain tests and manual pilot procedure. |
| Resolved | Pilot authority load diagnosis | A non-secret startup authority-count diagnostic exists and protected seed/restart/exercise passed. | Re-run only when authority loading changes. |
| P0 acceptance | Human-visible Flutter field-to-operator walkthrough | Automated evidence is complete; native Windows launch awaits C++ toolchain completion. | Complete offline queue, authoritative receipt, duplicate replay, operator review, advisory-only, cleanup evidence. |
| P1 reliability | Pilot matrix runner child-process behavior | Start-Process orchestration can misclassify a successful child exit. | Retain manual explicit-target procedure; redesign only with deterministic stage evidence. |
| P1 workstation | Flutter batch wrapper | `flutter.bat` exits 1 without output; direct bundled-Dart path is verified. | Defer SDK/workstation repair; use the direct command for the milestone. |
| P1 observability | Pilot telemetry foundation | Design is prepared, no component deployed. | Prioritize from walkthrough evidence; use self-hosted, pilot-only, reversible collection. |
| P2 documentation | Documentation ownership | Decision map now identifies current, snapshot, archive-only, and review-required records. | Maintain root continuity records and docs index. |
| P2 production | Device enrollment, production OIDC, OpenBao secret delivery | Intentionally not enabled. | Address only under separate approved production-authority plans. |
| P3 product | Pilot evidence-driven improvements | Not prioritized until real field/operator evidence exists. | Turn walkthrough observations into bounded UX/reliability work. |

## Current safety invariants

- Acceptance remains the protected control on `127.0.0.1:8080`; pilot experiments target only `127.0.0.1:18080`.
- OIDC is disabled at rest; `/identity/session` must return HTTP 404 unless a separate identity gate changes it.
- OpenBao remains sealed and unwired.
- Advisory behavior is always non-blocking and cannot mutate the primary workflow.
- Protected fixtures and private environment values are never displayed in reports or copied into source.
- The Windows pilot bootstrap is debug-only and is not a production enrollment mechanism.

## Evidence-based order of work

1. Complete the visible pilot walkthrough after the Windows toolchain is ready.
2. Recheck pilot/acceptance health and disabled OIDC state; clean temporary fixture material.
3. Use the walkthrough to choose a minimal pilot observability implementation and product-improvement backlog.
4. Consider production enrollment, OpenBao, and OIDC only through separate approval gates.

## References

- `CURRENT_STATE.md`
- `docs\continuity\VERIFICATION_COVERAGE_MATRIX_2026-08-16.md`
- `docs\PILOT_OBSERVABILITY_FOUNDATION_PLAN.md`
- `docs\MARKDOWN_DOCUMENT_DECISION_MAP_2026-08-16.md`
