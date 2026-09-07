# Host Integration Contract

The local bridge accepts one JSON object on standard input and emits one JSON object on standard output. It recognizes `prepare_context` and `check_completion` only.

The future event host must bind the bridge to one owner-approved canonical root. It must not supply a filesystem path or executable command in the event payload. The host must provide an explicit owner bypass, an audit route, bounded invocation time, and a non-trapping failure policy.

For `prepare_context`, a failed, disabled, mismatched, unknown, or bypassed request returns no context. For `check_completion`, those same conditions return `allow: true`; only a successful local inspection of an enabled, bound root may return `allow: false`.

The bridge has no listener, scheduler, network access, hook registration, service installation, or background behavior. A local `.planning/automation-disabled` file is the emergency bypass.
