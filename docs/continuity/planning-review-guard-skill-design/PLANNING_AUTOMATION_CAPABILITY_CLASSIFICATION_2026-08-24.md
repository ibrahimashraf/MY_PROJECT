# Planning-With-Files Automation Capability Classification

## Decision frame

The owner requested inclusion of automatic context injection, lifecycle hooks, forced stop/completion gates, active-plan pointers, environment-variable root selection, background work, scheduling, network access, external script execution, and direct external-source copying.

The active package currently runs as a local skill with supported `name` and `description` metadata only. This classification does not activate any automation, create schedules, contact a network service, run external code, or alter project planning records.

## Capability matrix

| ID | Requested capability | Current host evidence | Risk / authority requirement | Safe alternative | Disposition |
|---|---|---|---|---|---|
| AC-01 | Automatic context injection | The local validator previously rejected host-specific hook metadata; no supported injection interface is evidenced. | Could inject stale, private, or wrong-project planning state. Requires an exact host contract and owner acceptance of the privacy model. | Manual re-read and explicit continuity checkpoints. | **Blocked pending verified host support.** |
| AC-02 | Lifecycle hooks | No compatible hook schema or execution contract is evidenced for this host. | Hooks can execute on user actions and silently alter behavior. Requires host support, trigger definition, failure policy, and no-secret guarantee. | Explicit user-invoked validator. | **Blocked pending verified host support.** |
| AC-03 | Forced stop/completion gates | No supported stop-gate interface is evidenced. | Can prevent user-directed completion and create an availability failure. Requires explicit owner policy for override, timeout, and failure mode. | Non-blocking completeness report. | **Blocked pending host support and policy decision.** |
| AC-04 | Active-plan pointers | Can be designed locally, but is not yet designed or validated. | Pointer ambiguity can select a wrong project or stale plan. Requires project-local location, atomic update/recovery behavior, and multi-writer rules. | Explicit `--root` argument and owner-approved root. | **Candidate; design approval required.** |
| AC-05 | Environment-variable root selection | Technically feasible only as an explicit command input convention. | Inherited or stale environment values can select the wrong root. Requires precedence, canonicalization, and mismatch handling. | Explicit required `--root` argument. | **Candidate; design approval required.** |
| AC-06 | Background work | Default sandbox cannot provide durable background execution. | Requires a persistent host, resource limits, observability, cleanup, and cost/ownership decision. | Manual execution or a project-hosted service. | **Blocked pending architecture choice.** |
| AC-07 | Scheduling | A schedule must have frequency, intended action, expiry, delivery location, and failure behavior. | Recurring work creates ongoing cost and execution scope; schedule creation is a separate activation action. | Manual reminder/check or a low-frequency recurring task. | **Blocked pending schedule specification and explicit activation confirmation.** |
| AC-08 | Network access | No endpoint, purpose, authentication, or data classification is specified. | Can expose planning data or create outbound side effects. Requires a named allowlist, authentication route, data handling, and explicit approval. | Offline local validation. | **Blocked pending endpoint and data-flow specification.** |
| AC-09 | External script execution | External artifacts remain untrusted even after static review. | Arbitrary execution can compromise the workspace or access private data. Requires a separate owned, reviewed, local implementation—not direct execution. | Rebuild a bounded local helper with deterministic tests. | **Rejected as direct execution; local reimplementation may be designed.** |
| AC-10 | Direct external-source copying | Clean-room boundary prohibits copying external code or prompt text. | Obscures provenance, compatibility, and security review. | Capability-level clean-room reimplementation with ledger. | **Rejected.** |

## Required decisions before implementation

The owner must decide separately whether the desired outcome is: (a) an **opt-in local multi-plan utility** with no persistence beyond project files; (b) a **low-frequency reminder** that triggers a planning review; or (c) a **persistent project service** with an interface, an approved execution environment, and a named delivery target.

For any networked or external action, name the target endpoint or script owner, describe the exact data allowed to leave the workspace, and provide any required credentials through the approved secret mechanism. For any scheduling request, state the frequency, expiration, expected action, and failure-delivery behavior.

> Direct source copying and direct external-script execution remain rejected. They are not deferred implementation items.
