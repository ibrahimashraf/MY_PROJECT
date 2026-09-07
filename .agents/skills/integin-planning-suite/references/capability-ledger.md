# Capability Ledger

This ledger accounts for every role in the audited 29-artifact planning-with-files source subtree without importing the source.

| Role group | Local module | State |
|---|---|---|
| Core workflow, examples, and reference guidance | `SKILL.md` and `state-and-exit-codes.md` | Active local documentation. |
| Attestation, verification, completion, initialization, ledger, phase status, doctor, root resolution, snapshots, reconciliation, and active plan selection | `scripts/plan_records.py` | Active explicit local utilities. |
| Structural record validation | `scripts/validate_planning_records.py` | Active explicit local utility. |
| Context preparation and completion-gate decision | `scripts/planning_bridge.py` | Implemented local module; automatic invocation requires a verified host event contract. |
| Generic, analytics, and long-running plan records | `templates/` | Active manual assets. |
| Host hooks and stop gate | `host-integration.md` plus bridge protocol | Host-gated; not automatically active. |

Direct external-source copying and direct external-script execution are unavailable. They are not missing modules: they are excluded actions.
