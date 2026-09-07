# INTEGIN Planning Orchestrator Module Map

## Planned package structure

```text
integin-planning-orchestrator/
├── SKILL.md
├── scripts/
│   ├── plan_records.py
│   ├── validate_planning_records.py
│   └── planning_bridge.py
├── templates/
│   ├── task_plan.md
│   ├── findings.md
│   ├── progress.md
│   ├── analytics_task_plan.md
│   ├── analytics_findings.md
│   ├── task_plan_long_running.md
│   └── automation.json
└── references/
    ├── capability-ledger.md
    ├── state-and-exit-codes.md
    └── host-integration.md
```

## No-omissions capability map

| Audited role | New independently written module | Activation status |
|---|---|---|
| Core planning instructions | `SKILL.md` | Active manual workflow. |
| Worked examples | `SKILL.md` command sections plus `state-and-exit-codes.md`. | Active manual documentation. |
| Reference guidance | `state-and-exit-codes.md`. | Active manual documentation. |
| Attest/verify plan (PowerShell and shell variants) | `plan_records.py attest` and `verify`. | Active explicit local command. |
| Completion check (PowerShell and shell variants) | `plan_records.py check-complete`. | Active explicit local command. |
| Stop gate | `host-integration.md` gate contract and `planning_bridge.py check_completion`. | Host-gated: no automatic trigger. |
| Session initialization (PowerShell and shell variants) | `plan_records.py init`. | Active explicit local command. |
| Context injection | `planning_bridge.py prepare_context` and `host-integration.md`. | Host-gated: no automatic trigger. |
| Ledger append/summary (PowerShell and shell variants) | `plan_records.py ledger-append` and `ledger-summary`. | Active explicit local command. |
| Phase status (PowerShell and shell variants) | `plan_records.py status`. | Active explicit local command. |
| Planning doctor | `plan_records.py doctor`. | Active explicit local command. |
| Plan directory resolution (PowerShell and shell variants) | Explicit `--root`, `--plan-id`, and `--use-active` contract in `plan_records.py`. | Active explicit local command. |
| Session catchup | `plan_records.py snapshot` and `reconcile`. | Active explicit local command. |
| Active plan setting (PowerShell and shell variants) | `plan_records.py set-active`. | Active explicit local command. |
| Generic templates | Three generic templates. | Active manual assets. |
| Analytics templates | Two analytics templates. | Active manual assets. |
| Autonomous template | One long-running template with owner checkpoints and no autonomous execution. | Active manual asset. |
| Automatic lifecycle hooks and user-invocation metadata | Host-integration module states the exact future contract. | Host-gated; no direct imported metadata. |

The new package also carries the current local bridge’s Windows BOM handling, canonical root binding, bounded redaction, audit log, bypass file, fail-open unavailable behavior, and no-network/no-listener/no-scheduler constraints.
