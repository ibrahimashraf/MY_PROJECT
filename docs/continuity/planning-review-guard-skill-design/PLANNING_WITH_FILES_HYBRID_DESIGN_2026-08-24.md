# Planning-With-Files Hybrid Package Design

## Approved components

| Component | Purpose | Mutation boundary |
|---|---|---|
| `templates/task_plan.md` | Start a bounded plan with phases, owner gates, and decisions. | A user copies it manually into an authorized project; the skill never auto-creates it. |
| `templates/findings.md` | Record evidence, source limitations, and unresolved questions. | Manual use only. |
| `templates/progress.md` | Record chronological actions and the current continuity checkpoint. | Manual use only. |
| `scripts/validate_planning_records.py` | Read-only check for record presence, non-empty files, and optionally canonical headings. | Never writes, creates files, runs shell commands, accesses a network, or blocks task completion. |

## Validator contract

The helper must require `--root <planning-root>` so a caller consciously selects the target. It supports two profiles:

| Profile | Checks | Intended use |
|---|---|---|
| `basic` | Required files exist and are non-empty. | Existing projects with an approved but non-canonical planning structure. |
| `structured` | Basic checks plus the headings created by the bundled templates. | A newly created planning set or a project that adopts the local structure. |

The helper returns `0` when the requested checks pass, `1` when records are missing or nonconforming, and `2` for invalid invocation or inaccessible paths. It may emit human-readable text or `--format json` output. It does not repair findings, select a plan root, or alter any record.

## Explicit exclusions

The hybrid package must not add lifecycle hooks, host metadata beyond `name` and `description`, context injection, automatic invocation, plan attestation, active-plan pointers, environment-variable plan selection, completion gates, scheduled work, external dependencies, network calls, shell execution, or auto-created project files.

## Acceptance evidence

The package is acceptable only when the active skill validates; all templates expose the required separation of plan, findings, and progress; the helper passes deterministic clean/missing/noncanonical/invalid-root tests in disposable directories; static scans find no hook, source, network, shell-execution, or write operation; and the rollback archive remains intact.
