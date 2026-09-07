# INTEGIN PC Planning Bridge Deployment Contract

## Purpose

This contract governs a local, owner-controlled planning bridge placed outside product source and all protected runtime paths. It provides explicit local commands that can later serve a verified session lifecycle integration. It does not alter acceptance, the pilot candidate, package enforcement, OIDC, OpenBao, database state, routes, migrations, or private material.

## Deployment location

| Item | Approved local location | Handling |
|---|---|---|
| Bridge package | `C:\MY_PROJECT\tools\planning-bridge\` | Non-product local development helper. |
| Bound planning root | Not set during deployment. | Must be explicitly set by the owner in a local configuration file only after reviewing the canonical project path. |
| Local audit log | `<bound-root>\.planning\automation-events.jsonl` | Contains timestamp, event, result, bypass state, and bounded reason only. |
| Emergency bypass | `<bound-root>\.planning\automation-disabled` | Any presence disables injection and forces completion to allow. |
| Rollback archive | `C:\MY_PROJECT\archive\planning-bridge\` | Preserve a complete pre-change or deployed-package archive before later modifications. |

## Deployment state

The package must be copied in a **disabled-by-default** state. It must not create a service, scheduled task, port listener, background process, environment variable, session hook, or network request. The only enabled action after deployment is a manually invoked command operating on an explicit root.

## Binding rules

The bridge reads a configuration file only from `<bound-root>\.planning\automation.json`. The configuration must contain one canonical absolute `project_root`, `enabled: false` by default, a bounded context size, a short completion timeout, and fail-open completion behavior. The bridge must reject a mismatch between the selected root and its configured root.

The configuration must never reference `private\`, acceptance/pilot runtime folders, a database, a credential file, browser data, or an arbitrary session-supplied path. The event request contains only the event kind; it does not select a root or command.

## Local command protocol

The bridge accepts one JSON object on standard input and writes one JSON object to standard output. Its only supported events are `prepare_context` and `check_completion`. Unknown, invalid, disabled, unavailable, mismatched, or bypassed requests must fail safely: no context is injected and completion is allowed.

## Rollback and activation

Before any later modification or activation, archive the complete bridge package and record a SHA-256 checksum. A future lifecycle activation requires all of the following: an owner-selected bound root; a verified official session-to-PC event interface; a separate owner confirmation; a bypass-file proof; disabled, unavailable, and incomplete-plan tests; and a documented reverse procedure. The current deployment is not lifecycle activation.

## Current limitation

The connected PC is available at `C:\MY_PROJECT`, but no verified session lifecycle interface is exposed to invoke the bridge automatically before context preparation or completion. Until that interface exists, the bridge remains a local explicit command only.

## Verified disabled deployment

The local package was deployed to `C:\MY_PROJECT\tools\planning-bridge\` with no service, scheduled task, port listener, background process, project configuration, or lifecycle integration enabled. Its Windows compatibility handling accepts UTF-8 BOM text written by standard PowerShell commands in configuration, planning records, and standard-input JSON event requests.

| Evidence | Result |
|---|---|
| Explicit disposable-root test | Passed 8 cases: disabled context request, enabled local context response, incomplete completion response, emergency bypass, root mismatch fail-open, unknown-event fail-open, audit-log creation, and cleanup. |
| Python compilation | Passed for `scripts\planning_bridge.py`. |
| Deployed script SHA-256 | `202B71FC2275C91BE7AA2DF3435500EF129726FF03542FD7FABA49D9C3041314` |
| Pre-test disabled archive | `archive\planning-bridge\planning-bridge-disabled-baseline-20260825.zip`, SHA-256 `53D8436DBAB7413293E0CD41E4FD073BDA8CAFDFEF80899C5FC46C48DDD06218` |
| Tested disabled archive | `archive\planning-bridge\planning-bridge-tested-disabled-20260825.zip`, SHA-256 `A2A72B66917E1A4903344DD08A0DE0B04022302C7669AEC45B4AE3DE7CC2C374` |

The enabled test exercised only a disposable root and only invoked the bridge explicitly through standard input. It did not activate the bridge against `C:\MY_PROJECT`, modify any protected runtime, or create automatic session behavior.

## No-unintended-activation verification

The deployed bridge still matches its pre-activation checksum baseline. No root `automation.json`, emergency-bypass file, bridge process, or bridge scheduled task exists. A broad scheduled-task name scan matched one unrelated pre-existing Lenovo Service Bridge updater under `\Lenovo\Lenovo Service Bridge\`; its executable is not part of this planning bridge and it was not altered.
