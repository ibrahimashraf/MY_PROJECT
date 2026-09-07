# INTEGIN Controlled Planning Task Launcher

This local helper is a **disabled-by-default controller for future API-created tasks**. It is not a Manus session hook, service, scheduled task, listener, background process, or replacement for native task completion. It first invokes the fixed-root planning bridge, creates a new private API task only after explicit confirmation, and can record a local completion-review decision after the task stops.

## Safety contract

The launcher reads a single explicit configuration file. The configuration fixes the project root, bridge path, approved Manus API base URL, credential environment-variable name, task project allow-list, and size/time bounds. It contains no API key. A prompt file must be inside the bound root and outside `private` and `.planning` paths. The bridge owns redaction, bypass, and project-root validation.

The launcher must remain disabled and unbound until the owner separately approves a canonical root and supplies a dedicated API key through the configured environment variable. A failed preflight prevents task creation. A failed or non-allowing post-stop completion review only records `warn` or `unavailable`; it never blocks the platform’s own task-completion behavior.

## Disabled setup and commands

Copy `templates\launcher-config.json` to a chosen local configuration path, replace only after review, and retain `"enabled": false`.

```powershell
python .\scripts\planning_task_launcher.py validate-config --config .\launcher-config.json
python .\scripts\planning_task_launcher.py preflight --config .\launcher-config.json
python .\scripts\planning_task_launcher.py create --config .\launcher-config.json --project-id "<approved-project-id>" --prompt-file "<bound-root>\planning-request.md" --confirm-create
python .\scripts\planning_task_launcher.py monitor --config .\launcher-config.json --receipt .\receipts\<task-id>.json --watch
```

The final two commands are intentionally unavailable while the configuration remains disabled. `--confirm-create` is required on every task-creation call. The monitor is a foreground process only; it creates no scheduler or listener.

## Audit and rollback

The launcher writes bounded event metadata to `launcher-events.jsonl` and non-secret task receipts under `receipts\`. No prompt, planning context, credential, private data, or response body is written to those files. Delete this new launcher directory to remove the component; keep the existing planning bridge and its archives unchanged.
