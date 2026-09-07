# PC-Hosted Planning Bridge Protocol

## Local configuration

The bridge reads only `<bound-root>/.planning/automation.json`. The file is local, contains no secret, and is disabled by default.

```json
{
  "enabled": false,
  "project_root": "<canonical local path>",
  "max_context_chars": 2000,
  "completion_timeout_seconds": 3,
  "fail_open_completion": true
}
```

The bridge rejects relative paths, mismatched canonical roots, a missing config, a disabled config, a malformed config, or any request that supplies a different root. It never reads an event-provided filesystem path.

## Stdio request and response

The local bridge accepts one JSON object on standard input and writes one JSON response to standard output. It accepts only these event values:

| Event | Response | Local behavior |
|---|---|---|
| `prepare_context` | `inject`, `reason`, `context` | Reads bounded/redacted planning records only when enabled and not bypassed. |
| `check_completion` | `allow`, `reason`, `status` | Returns `allow: false` only when enabled, no bypass exists, and the planning completion check fails. |
| Other value | `allow: true`, `inject: false`, `reason` | Fail safe without interpreting unknown events. |

The request may carry `bypass: true` only when a future verified host contract designates it as an owner control. Until then, the local `automation-disabled` file is the primary bypass.

## Fail-open completion rule

The bridge returns `allow: true` when disabled, unavailable, misconfigured, timed out, or unable to inspect the plan. It may return `allow: false` only after a successful local inspection of the configured bound root. This prevents a defect from trapping the user or agent indefinitely.

## Audit log

Each request appends one redaction-safe JSON line to `<bound-root>/.planning/automation-events.jsonl` containing timestamp, event, result, bypass state, and non-sensitive reason. The log contains no planning record contents, arbitrary paths, tokens, or event payload copies.

## Test plan

Use a disposable root to verify disabled, enabled-pass, enabled-incomplete, bypass-file, mismatched-root, malformed-config, unknown-event, and audit-log cases. No test activates a host lifecycle event or starts a persistent service.
