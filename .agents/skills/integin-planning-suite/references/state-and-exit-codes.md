# State and Exit Codes

The `.planning` directory may contain named plans, an `active-plan.json` pointer, ledger entries, attestation records, and workspace snapshots. State is project-local and must remain under the explicit root.

| Exit code | Meaning |
|---:|---|
| `0` | Requested local check succeeded or found no difference. |
| `1` | The check ran and found an incomplete plan, changed workspace, malformed ledger entry, or failed attestation. |
| `2` | The command could not safely run because the root, plan identifier, state, or input was invalid. |

Treat an exit status as evidence for that command only. It does not grant authority to repair records, choose a root, change a runtime, or activate host automation.
