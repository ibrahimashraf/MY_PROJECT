# Planning-With-Files Automation Authorization Matrix

## Decision options

| Option | What it provides | Trade-offs | Cost / operational impact | Required owner decision |
|---|---|---|---|---|
| A. Opt-in local multi-plan tools | Explicit commands to select, inspect, and validate a project-local plan root; no hidden execution. | Adds convenience but still requires deliberate command use. | No recurring execution or network traffic. | Approve a project-local state-file format, root precedence, and concurrent-writer policy. |
| B. Low-frequency planning review | A recurring prompt to review one planning root and report record health. | Suitable only for infrequent review; it is not a real-time monitor or background worker. | Ongoing task executions; needs a schedule, expiry, and delivery behavior. | Specify timing, expiry, exact review task, and failure delivery. |
| C. Persistent planning service | A continuously available planning API or monitor with optional network integration. | Highest complexity and exposure; needs a managed execution environment, observability, persistence, and an interface. | Requires a deployment and operating-cost decision. | Provide a concrete persistent use case, named users, latency, data retention, endpoint, and delivery requirements. |

## Per-capability gate

| Capability | Eligible option | Minimum design evidence | Explicit authorization still required | Current status |
|---|---|---|---|---|
| Manual plan-root selection | A | Canonical project-local path layout, root precedence, collision recovery. | Approve state-file format and write semantics. | Not approved. |
| Environment-variable root override | A | Exact variable name, precedence relative to explicit argument, canonicalization, stale-value behavior. | Approve override policy. | Not approved. |
| Active-plan pointer | A | Atomic pointer-write and recovery behavior; no automatic newest-plan selection. | Approve pointer location and multi-writer policy. | Not approved. |
| Automatic context injection | None currently | Official host hook/injection contract and privacy review. | Approval after host support is verified. | Blocked. |
| Lifecycle hooks | None currently | Official host schema, trigger list, allowed actions, failure policy, and opt-out. | Approval after host support is verified. | Blocked. |
| Completion gate | None currently | Official host mechanism plus user override, timeout, and non-availability policy. | Approval after host support is verified. | Blocked. |
| Scheduled planning review | B | Frequency, timezone, expiry, task detail, delivery, failure behavior. | Confirm the exact schedule activation. | Not approved. |
| Background worker | C | Persistent-host architecture, resources, logs, failure recovery, data retention, and ownership. | Approve deployment and operational model. | Not approved. |
| Network integration | C or a bounded local client | Named endpoint, allowed request/response data, authentication, allowlist, retry/timeout, logging/redaction. | Approve endpoint and data flow; provide credentials securely if needed. | Blocked. |
| External script execution | None | N/A; external artifacts are untrusted. | Cannot authorize direct execution through this package. | Rejected. |
| Direct external-source copying | None | N/A; violates clean-room boundary. | Cannot authorize direct copying through this package. | Rejected. |

## Recommended next decision

Choose **Option A**, **B**, or **C**. The selection determines the next design and testing scope. No automation is enabled by this matrix.
