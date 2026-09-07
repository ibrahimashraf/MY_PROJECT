# INTEGIN Controlled Planning Launcher — Root Binding Record

## Owner approval

The owner explicitly approved this binding on 2026-08-25:

> `I approve binding the controlled launcher to C:\MY_PROJECT.`

## Bound configuration

| Component | Configuration path | State |
|---|---|---|
| Existing local planning bridge | `C:\MY_PROJECT\.planning\automation.json` | Bound to `C:\MY_PROJECT`; `enabled: false`. |
| Controlled planning task launcher | `C:\MY_PROJECT\tools\planning-task-launcher\launcher-config.json` | Bound to `C:\MY_PROJECT`; `enabled: false`; API task-project allow-list is empty. |

The launcher configuration validates the expected root and bridge script. An explicit bridge `prepare_context` request returned `automation is disabled` with no injected context. No API call, task creation, credential setup, process, listener, scheduled task, runtime operation, private-material access, or protected INTEGIN change occurred.

## Outstanding activation conditions

The owner must configure the dedicated local environment variable `MANUS_PLANNING_LAUNCHER_API_KEY` with a new Manus API key and confirm only that it is configured. The task-project allow-list must remain empty until a project identifier is retrieved through the official API and the owner gives one final activation confirmation. The new launcher remains a controller for future API-created tasks only; it does not intercept ordinary interactive-task context construction or native completion.
