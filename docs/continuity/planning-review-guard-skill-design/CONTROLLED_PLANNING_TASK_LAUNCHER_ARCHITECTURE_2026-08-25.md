# Controlled Planning Task Launcher Architecture

## Decision

The owner selected a controlled task launcher. The launcher is a separate, disabled PC-local component for **future API-created planning tasks**. It must not represent itself as a hook into the lifecycle of ordinary interactive tasks.

## Boundary

| Concern | Launcher responsibility | Explicit limitation |
|---|---|---|
| Context preparation | Invoke the existing bridge with `prepare_context` for one locally configured canonical root before an API task is created. Include only the bridge’s bounded, redacted context in the new task input. | It cannot inject context before the platform creates an ordinary interactive task. |
| Task start | Create a new private Manus API task only after preflight succeeds. Attach an immutable task/run receipt to the local audit record. | It must not create a task if the bridge is disabled, bypassed, unavailable, root-mismatched, or non-injecting. |
| Completion review | Monitor only launcher-created task IDs and invoke `check_completion` after a verified stop state. Record `approved`, `warn`, `bypassed`, or `unavailable`. | It cannot retract, hide, or block the platform’s own final message or completion control. A non-allowing result is a local workflow-review warning. |
| Root selection | Read a fixed canonical root only from owner-local launcher configuration. | The task prompt, webhook, command line, and API response may never select a root or a command. |
| Failure policy | Do not create a task when context preflight fails. For post-stop review, record an unavailable/warn result and never trap task completion. | No remote listener, arbitrary command execution, or background action is permitted in the disabled initial implementation. |

## Fixed control points

The launcher will use a configuration file adjacent to itself, not the bound project, containing a disabled flag, canonical project root, bridge-script path, approved API endpoint base, maximum prompt/context sizes, a short network timeout, and an allowlist of project IDs. It will not contain an API credential. The credential will be supplied only through a user-managed environment variable after a separate request and approval.

The launcher will never inspect the root until an explicit owner binding is approved. It will keep `C:\MY_PROJECT` unbound during implementation and testing. A disposable test root is required for test execution.

## Activation prerequisites

1. The owner approves one canonical root in a separate, explicit message.
2. The owner creates a dedicated Manus API key with the minimum available task-creation scope and stores it locally as an environment variable.
3. The launcher configuration is reviewed with `enabled: false`, then activated only after offline, bypass, root-mismatch, completion-review, and recovery tests pass.
4. The PC remains available only for the launcher-created task workflow; no service, scheduled task, listener, or browser-data access is introduced without a separately reviewed change.

## References

1. [Planning Bridge Interface](../skills/integin-planning-orchestrator/scripts/planning_bridge.py)
2. [Planning Lifecycle Interface Research](PLANNING_LIFECYCLE_INTERFACE_RESEARCH_2026-08-25.md)
3. [PC-Hosted Planning Automation Security Contract](PC_HOSTED_PLANNING_AUTOMATION_SECURITY_CONTRACT_2026-08-24.md)
