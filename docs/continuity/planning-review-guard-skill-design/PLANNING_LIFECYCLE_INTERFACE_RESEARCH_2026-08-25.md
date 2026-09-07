# Planning Lifecycle Interface Research

## Scope

Assess only documented Manus capabilities relevant to invoking the disabled local planning bridge automatically before task context preparation and before task completion. This note does not authorize configuration changes, new services, a Windows scheduled task, API-key creation, or activation against `C:\MY_PROJECT`.

## Verified documentation findings

| Capability | Documented behavior | Fit for the requested two automatic events |
|---|---|---|
| My Computer | Runs terminal commands on the connected PC; every command requires explicit approval unless the owner selects the product's trusted-command option. | Provides PC command execution, but the evidence reviewed does not define pre-context or pre-completion callback events. |
| Custom MCP | Manus can call focused tools/resources/prompts served by a user-hosted MCP endpoint. The server must be reachable, authenticated, and respond promptly. | Can provide explicit planning tools, but does not itself register mandatory task lifecycle interception. |
| Manus API task webhooks | The API sends `task_created` and `task_stopped` events to an HTTPS callback for API-created tasks. | Provides post-creation and stopped/waiting notifications only; it is not a before-context or before-completion interception point for ordinary interactive sessions. |
| Manus API task creation | An API caller can create a new task with message, project ID, and configuration. | Enables a companion controller to run a planning preflight *before creating a new API-managed task*, but does not retrofit the current task's lifecycle. |

The interactive official reference labels task creation as `POST /v2/task.create` and identifies the `x-manus-api-key` authentication header. The reviewed rendered reference did not expose a concrete API hostname in the captured content; the launcher must therefore retain the official API base URL as a disabled configuration value rather than guessing it.

## Current configuration inspection

The refreshed current task configuration had no entries matching `desktop` or `lifecycle`. Its sole `hook` text match was a disabled Typeform connector description; it is unrelated and must not be repurposed.

## Design conclusion

No official interface found in this review can inject local context before an existing interactive task's context is built or block that task's native completion action. A custom PC or hosted service must not impersonate such a hook.

The only potentially supportable automatic pattern is a **companion task controller** for future API-created tasks: it holds a locally or securely stored API credential, invokes the fixed-root local bridge as a preflight, then creates a new task only after a valid result. The controller may react to `task_stopped` webhook events for audit/notification, but it cannot make the platform's own completion behavior conditional. This is architecturally distinct from a native session lifecycle hook and requires owner selection, a safe persistent host, API-key provisioning, and separate activation approval.

## Sources

1. [Introducing My Computer: When Manus Meets Your Desktop](https://manus.im/blog/manus-my-computer-desktop)
2. [Custom MCP Servers — Manus Documentation](https://manus.im/docs/integrations/custom-mcp)
3. [How Webhook Events Work — Manus API](https://open.manus.ai/docs/v2/webhooks-overview)
4. [Task Lifecycle — Manus API](https://open.manus.ai/docs/v2/task-lifecycle)
5. [task.create — Manus API](https://open.manus.ai/docs/v2/task.create)
6. [Authentication — Manus API](https://open.manus.ai/docs/v2/authentication)
