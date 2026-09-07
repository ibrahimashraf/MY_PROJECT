# Complete Local Planning-With-Files Capability Pack Design

## Local package contents

| Local artifact | Audited roles covered | Design boundary |
|---|---|---|
| `SKILL.md` | Planning instructions, examples/reference navigation, explicit-command guidance | No lifecycle metadata, auto-run behavior, or source copying. |
| `scripts/plan_records.py` | Initialize, resolve, set active, status, completion report, doctor, ledger append/summary, attestation, verification, workspace snapshot/reconcile, explicit context rendering | Standard-library Python only; every mutation requires an explicit subcommand and an explicit `--root`. |
| `scripts/validate_planning_records.py` | Existing basic/structured record validation | Keep as a focused read-only checker. |
| `templates/*.md` | Generic, analytics, and long-running plan/finding/progress starters | Manual copying only; no automatic project creation. |
| `examples.md` | Common local command sequences | Examples only; no auto-execution. |
| `reference.md` | File layout, state-file schemas, exit codes, recovery and security behavior | Local semantics only. |
| `references/host-automation-compatibility.md` | Injection, lifecycle, and completion-gate compatibility contract | Documentation only; does not enable unsupported host automation. |

## Explicit subcommand contract

| Subcommand | Mutation | Audited role coverage | Guardrails |
|---|---:|---|---|
| `init` | Yes | Session initialization | Refuse a nonempty target unless `--force` is explicitly supplied; never infer root. |
| `resolve` | No | Plan-directory resolution | Requires `--root`; accepts explicit `--plan-id`; never selects the newest plan. |
| `set-active` | Yes | Active-plan selection | Writes a project-local pointer atomically after path containment checks. |
| `status` | No | Phase status | Parses only the local supported status markers. |
| `check-complete` | No | Completion inspection | Reports state and exit code; never blocks an agent or user action. |
| `doctor` | No | Planning setup diagnosis | Checks root, files, pointer, and optional attestation without repair. |
| `ledger-append` | Yes | Ledger append | Requires explicit `--message`; append-only JSON Lines ledger within the root. |
| `ledger-summary` | No | Ledger summary | Reads only the local ledger format. |
| `attest` | Yes | Plan integrity record | Writes a local sidecar SHA-256 record only after explicit invocation. |
| `verify-attestation` | No | Plan integrity verification | Compares only the local sidecar and selected plan. |
| `snapshot` | Yes | Session/workspace continuity | Writes a local file-manifest sidecar, excluding sensitive/large/common build directories. |
| `reconcile` | No | Session catchup replacement | Compares current scoped manifest against an explicitly selected snapshot; never reads host-private histories. |
| `render-context` | No | Context-injection equivalent | Prints a bounded, redaction-aware local summary only when explicitly invoked. |

## State layout and precedence

```text
<root>/
├── task_plan.md
├── findings.md
├── progress.md
└── .planning/
    ├── active-plan.json
    ├── ledger.jsonl
    ├── attestations/<plan-id>.json
    └── snapshots/<snapshot-id>.json
```

The explicit `--root` value always wins. An explicit `--plan-id` identifies `<root>/.planning/plans/<plan-id>/`. The active pointer may be used only when `--use-active` is explicitly supplied. An environment variable may be supported later only as a lower-priority, opt-in override after a separate policy decision; it is not part of this implementation batch.

## Host-dependent items

| Requested item | Local equivalent | Activation status |
|---|---|---|
| Automatic context injection | `render-context` produces a bounded summary on explicit command. | Available manually; automatic activation remains blocked. |
| Lifecycle hooks | Document the exact contract a host would need to provide. | Not enabled. |
| Forced stop/completion gates | `check-complete` returns a report and exit code. | Available manually; forced gating remains blocked. |
| Background work and scheduling | No local background process or schedule is created by this package. | Requires a separate deployment/schedule decision. |
| Network access | No network client is included. | Requires named endpoint and data-flow approval. |
| External script/source execution | No direct implementation. | Rejected. |

## Validation plan

Test each read-only subcommand against a disposable root. Test each writer against a disposable root with path-containment, refusal-on-nonempty, malformed-pointer, invalid-plan-ID, ledger append, attestation mismatch, snapshot/reconcile, and content-rendering negative cases. Confirm no package script contains network imports, process execution, shell invocation, host hooks, or external-source identifiers.
