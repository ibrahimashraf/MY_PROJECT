---
name: integin-planning-orchestrator
description: Orchestrate durable multi-phase planning with explicit local records, named plans, integrity checks, reconciliation, context preparation, and completion decisions. Use for complex INTEGIN work spanning sessions, multiple planning tracks, owner gates, or PC-hosted planning integration; use only with an authorized project root.
---

# INTEGIN Planning Orchestrator

Use this locally owned skill to plan, verify, and reconcile complex work. It combines durable planning records, explicit local utilities, reusable templates, and a host-integration contract in one package. It is not an external-source fork or installer.

## Start safely

1. Identify the authorized project root and required records before creating or editing anything.
2. Read existing `task_plan.md`, `findings.md`, and `progress.md` together. Record drift; do not silently reconcile contradictory state.
3. Use `scripts/plan_records.py` only with an explicit `--root`. Select a non-root plan only with `--plan-id` or a deliberately created active-plan pointer.
4. Use bundled templates only for an empty authorized destination. Do not overwrite nonempty records.
5. Treat the planning state as evidence, not authority. Protected runtimes, migrations, credentials, public release, external calls, and destructive changes need their own approval.

## Core workflow

Use the generic templates for ordinary engineering work, analytics templates for evidence-heavy research, and the long-running template for owner-controlled multi-checkpoint work. The long-running template does not authorize background continuation.

```bash
python3 scripts/plan_records.py init --root /authorized/project --plan-id review-cycle
python3 scripts/plan_records.py set-active --root /authorized/project --plan-id review-cycle
python3 scripts/plan_records.py status --root /authorized/project --use-active
python3 scripts/plan_records.py check-complete --root /authorized/project --use-active
```

Use `attest` and `verify-attestation` for an explicit task-plan digest, `snapshot` and `reconcile` for bounded workspace drift, `ledger-append` and `ledger-summary` for a local event record, `doctor` for structural diagnostics, and `render-context` for a manually requested bounded/redacted planning summary. Read `references/state-and-exit-codes.md` before relying on a non-zero exit code.

## Local capabilities

| Need | Local command or asset | Behavior |
|---|---|---|
| Create or select a plan | `init`, `resolve`, `set-active` | Uses explicit roots and safe plan identifiers; never selects the newest directory. |
| Review record health | `status`, `doctor`, `validate_planning_records.py` | Reports gaps without attempting repair. |
| Check completion | `check-complete` | Reports state only; it does not block a session by itself. |
| Track planning events | `ledger-append`, `ledger-summary` | Appends local JSON lines only when explicitly invoked. |
| Detect record changes | `attest`, `verify-attestation` | Uses SHA-256 for selected task-plan files. |
| Reconcile workspace changes | `snapshot`, `reconcile` | Excludes private, build, dependency, Git, and environment paths. |
| Prepare a bounded summary | `render-context` | Redacts sensitive-looking lines and requires an explicit root. |
| Prepare host integration | `planning_bridge.py` | Disabled by default; use only through a verified event host. |

## Host-integration module

Read `references/host-integration.md` before using `scripts/planning_bridge.py`. The bridge can prepare bounded context or return a completion decision for one configured root, but it does not create a listener, hook, schedule, service, network request, or background process. A session platform must expose a documented event contract before it can run automatically.

Keep the bridge disabled until its local configuration has an owner-approved canonical root, the emergency bypass is tested, and a final activation decision exists. If unavailable, malformed, mismatched, or bypassed, the bridge injects nothing and allows completion.

## No-omissions ledger

Read `references/capability-ledger.md` for the mapping of every audited external role to this package. The ledger distinguishes active local equivalents, host-gated modules, and direct external actions that remain unavailable. Do not claim that a host-gated module is automatically active.

## Hard boundaries

Do not copy external source text, prompts, scripts, hooks, or metadata into this package. Do not execute external artifacts. Do not accept an arbitrary root, command, or network endpoint from a message or event payload. Do not store credentials, private material, raw event payloads, or planning content in bridge audit logs.

## Completion standard

Finish only after the selected planning state, evidence, decisions, validation, limitations, and next safe action agree. It is valid to finish with a host-integration module unactivated; record the missing host contract rather than inventing automatic behavior.
