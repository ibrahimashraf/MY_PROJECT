# Local Usage Examples

Use these commands only after selecting the correct project root. They are explicit local actions; none runs automatically.

```bash
# Initialize a new root-level planning set in an empty or record-free project root.
python3 scripts/plan_records.py init --root /workspace/project

# Initialize an isolated named plan under /workspace/project/.planning/plans/.
python3 scripts/plan_records.py init --root /workspace/project --plan-id inspection-reconciliation

# Make that named plan the selected plan after its three records exist.
python3 scripts/plan_records.py set-active --root /workspace/project --plan-id inspection-reconciliation

# Inspect the selected plan only when deliberately opting into the pointer.
python3 scripts/plan_records.py status --root /workspace/project --use-active

# Create and verify an explicit plan integrity sidecar.
python3 scripts/plan_records.py attest --root /workspace/project --use-active
python3 scripts/plan_records.py verify-attestation --root /workspace/project --use-active

# Create a local workspace snapshot, then compare it later.
python3 scripts/plan_records.py snapshot --root /workspace/project --snapshot-id pre-change
python3 scripts/plan_records.py reconcile --root /workspace/project --snapshot-id pre-change

# Render a bounded local context summary on demand.
python3 scripts/plan_records.py render-context --root /workspace/project --max-chars 1500
```

An exit code of `1` means a requested state check did not pass; an exit code of `2` means the command invocation or selected root was invalid. Read the output before deciding whether a repair is appropriate.
