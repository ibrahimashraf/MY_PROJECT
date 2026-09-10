# Delegation Discipline (local fix for stalled background dispatches)

## Problem
Dispatching `relay.mjs` with `run_in_background: true` parks the task: completion
arrives nowhere, and the orchestrator only discovers it when prompted. This stalled
the workflow repeatedly (upstream: lean-ctx #1722 — no completion notifications for
background `ctx_shell` jobs; polling is the only mechanism).

## Rule: foreground dispatch, always
`relay.mjs` already blocks until `result.json` lands. Run it in the **foreground**
with a generous timeout (up to 2h for implementation runs). Never background it.
Completion then arrives in-band — the stall class disappears without any polling.

## Mandatory loop per dispatch (no exceptions, no prompts needed)
1. Write brief to **temp only** (`$env:TEMP/delegate-*.txt`), never into any repo tree.
2. Brief touch-paths **relative to `--cd`**; `--cd` = the repo the touches live in.
3. Dispatch foreground; wait for terminal `result.json`.
4. Review diff for scope creep — revert out-of-brief touches (delta-fix only if the
   extra change is correct AND the human approves the scope expansion).
5. Re-run the project's gates yourself (never trust the self-report).
6. Update TRACKER.md in the same turn as any landing.
7. Commit with hooks enabled — never `--no-verify`.
8. Report unprompted: what changed, files touched, gate outputs, deviations.
9. Delete the temp brief after dispatch.
