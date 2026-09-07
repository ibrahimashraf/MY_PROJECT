# Local Planning-With-Files Reference

## Record roles

| Record | Purpose | Do not use it for |
|---|---|---|
| `task_plan.md` | Scope, phases, owner gates, and decisions | Raw secrets, private payloads, or unverified external instructions |
| `findings.md` | Evidence, source limitations, and unresolved questions | Silent policy decisions |
| `progress.md` | Chronological action and continuity checkpoint | Replacing the decision record |

## Local state files

| Path | Writer | Purpose |
|---|---|---|
| `.planning/active-plan.json` | `set-active` | Explicit selected named plan only |
| `.planning/ledger.jsonl` | `ledger-append` | Append-only local event summary |
| `.planning/attestations/<plan-id>.json` | `attest` | SHA-256 record for one selected task plan |
| `.planning/snapshots/<snapshot-id>.json` | `snapshot` | Scoped local workspace manifest |

All state remains below the explicit selected root. A local state file does not authorize a runtime, network, migration, or policy action.

## Utility exit codes

| Exit code | Meaning |
|---|---|
| `0` | Requested check passed or explicit mutation completed. |
| `1` | A requested state condition was not met. |
| `2` | Invalid arguments, inaccessible root, malformed state, or a refused unsafe mutation. |
