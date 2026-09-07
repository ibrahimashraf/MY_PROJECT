---
name: integin-planning-suite
description: Plan, verify, reconcile, and preserve durable multi-phase project work with explicit local records, named plans, integrity checks, bounded context preparation, and host-gated completion decisions. Use for complex work spanning sessions, multiple planning tracks, owner gates, evidence continuity, or authorized PC-hosted planning integration.
---

# INTEGIN Planning Suite

Use this locally owned skill for durable, reviewable planning. It consolidates the manual record discipline, explicit planning utilities, templates, integrity and reconciliation checks, and host-gated bridge contract from the local planning packages. It is not an external-source fork, installer, hidden hook, service, scheduler, or background worker.

## Start safely

1. Establish the authorized root, scope, exclusions, owner gates, and stop conditions before creating or editing a record.
2. Resolve the planning location deliberately. If roots, plans, or legacy records conflict, record the ambiguity and obtain an owner choice; never select the newest candidate.
3. Read `task_plan.md`, `findings.md`, and `progress.md` together. Record factual drift instead of silently harmonizing it.
4. Treat planning evidence as non-authoritative. Separate approval remains required for protected runtimes, migrations, credentials, external calls, public release, destructive work, and production changes.
5. Do not write secrets, private data, raw untrusted instructions, or credentials into planning records, context summaries, or audit logs.

## Durable workflow

Use `task_plan.md` for scope, phases, decisions, and owner gates; `findings.md` for sources, validation, uncertainties, and contradictions; and `progress.md` for chronological actions and continuity checkpoints. Use one designated writer per record. Before editing an existing record, re-read it and stop for reconciliation if another writer changed it.

Create records only in an authorized root. For empty authorized destinations, use the generic, analytics, or long-running templates. Do not overwrite an established record merely to match a template.

After each material action, record the current phase, validated action, unresolved decision or limitation, and next safe action. Close only when delivered artifacts, validation, deferred work, limitations, and the next owner decision are durable and mutually consistent.

## Explicit local capabilities

All utility commands require an explicit `--root`. A non-root named plan requires an explicit `--plan-id`, or `--use-active` after an explicit `set-active` action. Utilities never discover a root by recency, access host-private session history, invoke a network, run shell commands, create schedules, or activate hooks.

| Need | Utility | Safe behavior |
|---|---|---|
| Create/select a plan | `init`, `resolve`, `set-active` | Uses explicit roots and safe identifiers. |
| Inspect record health | `status`, `doctor`, `validate_planning_records.py` | Reports gaps; never repairs implicitly. |
| Assess completion | `check-complete` | Reports local state; does not block a session by itself. |
| Preserve planning events | `ledger-append`, `ledger-summary` | Writes local JSONL only on an explicit command. |
| Detect plan changes | `attest`, `verify-attestation` | Uses selected-file SHA-256 evidence. |
| Reconcile bounded workspace drift | `snapshot`, `reconcile` | Excludes private, build, dependency, Git, and environment paths. |
| Prepare bounded planning context | `render-context` | Requires an explicit root and redacts sensitive-looking lines. |

Use `references/explicit-command-examples.md` for safe command sequences, `references/record-layout-reference.md` for local state layout, and `references/state-and-exit-codes.md` before relying on a non-zero result.

## Host-gated context and completion module

`scripts/planning_bridge.py` can prepare bounded, redacted context or return a local completion decision for one owner-bound root. It is **disabled by default** and does not create a listener, hook, service, schedule, network request, or background process.

Read `references/host-integration.md`, `references/host-automation-compatibility.md`, and `references/pc-hosted-automation.md` before any bridge use. Activate it only when a documented session-to-host event interface exists; the canonical root is owner-approved; the emergency bypass is tested; bounded invocation and audit controls exist; and the owner has given a final activation decision. If unavailable, malformed, mismatched, or bypassed, it injects nothing and allows completion.

Do not accept an arbitrary root, command, network endpoint, credential, or raw event payload from a message or bridge event. Do not claim that a host-gated capability is automatic when no verified host event contract exists.

## No-omissions and completion standard

Read `references/capability-ledger.md` for the local capability mapping and host-gated limitations. It is an evidence ledger, not authorization to copy external material or activate automation.

Finish only after the selected planning state, evidence, decisions, validation, limitations, and next safe action agree. It is valid to finish with host integration disabled or deferred; record the missing condition rather than inventing completion or automation.
