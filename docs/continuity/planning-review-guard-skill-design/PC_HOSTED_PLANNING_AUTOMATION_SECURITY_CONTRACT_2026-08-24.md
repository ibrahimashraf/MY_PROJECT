# PC-Hosted Planning Automation Security Contract

## Owner intent

The owner requested opt-in automatic planning-context injection and an automatic completion gate at their own responsibility. This contract defines safeguards that remain mandatory even with owner consent.

## Non-negotiable safeguards

| Control | Injection behavior | Completion-gate behavior |
|---|---|---|
| Default state | Disabled. | Disabled. |
| Project binding | One canonical, owner-approved local project root only. | The same bound root only. |
| Read/write scope | Read only `task_plan.md`, `findings.md`, `progress.md`, and the local `.planning` state. | Read only the selected planning records and local `.planning` state. |
| Private material | Never read `private`, secrets, credentials, browser data, or unapproved files. | The same exclusion. |
| Output bound | Redact sensitive-looking lines and impose a fixed size limit. | Return only a small status record and failure reason. |
| Audit | Append a timestamped local event record with root identifier, event, result, and bypass state. | The same. |
| Timeout / failure | Fail closed: inject nothing and report an error. | Fail open after a short bounded timeout: do not trap completion; report an unavailable gate. |
| Emergency bypass | Owner-local disable file and an explicit one-run bypass flag. | The same bypass always overrides the gate. |
| Network | No listener, outbound request, credential, or remote command in the first local bridge. | The same. |

## Event contract required from the session platform

The session platform must provide an official, documented way to invoke a local command for these two events:

1. **Before agent context is prepared**, passing only a signed or trusted project identity—not an arbitrary filesystem path.
2. **Before a completion action is finalized**, accepting a bounded allow/warn result and respecting the local owner bypass.

The bridge must receive the fixed local root from a local owner configuration, not from event payload text. It must never run a supplied shell command, interpret a path from a chat message, or choose the latest directory automatically.

## Local owner configuration concept

```json
{
  "enabled": false,
  "project_root": "<canonical local path>",
  "max_context_chars": 2000,
  "completion_timeout_seconds": 3,
  "fail_open_completion": true
}
```

The configuration is local-only. It must be created by the owner at an explicit chosen project root, must reject relative/unresolvable paths, and must not contain secrets.

## Emergency bypasses

The bridge must honor either of the following before any injection or gate decision:

1. A local `.planning/automation-disabled` file in the bound project root.
2. A one-run explicit bypass supplied through the verified host event contract.

No chat message by itself is a bypass signal unless the host contract conveys it as a verified control value.

## Stop condition

Do not enable the automation until a connected local folder and a verified session-to-PC event interface are available. If either is absent, retain the explicit local `render-context` and `check-complete` commands only.

## Current integration inspection

| Check | Result | Consequence |
|---|---|---|
| Local folder bound to this task | No. | No local project root is available for a PC-hosted bridge. |
| Desktop/local-PC connector in current task configuration | No matching configuration entry. | No session-to-PC event path is available to inspect or activate. |
| Session hook contract | Not present in the installed skill validator or current task configuration. | Automatic injection and an automatic completion gate remain unavailable. |

The owner must first bind the intended project folder through the local desktop connection. After that, a separate verified event interface is still required; folder access alone does not create lifecycle hooks.
