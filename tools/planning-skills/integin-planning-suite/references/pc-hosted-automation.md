# PC-Hosted Planning Automation

The local bridge is disabled by default. It is not a background service, does not create a listener, and does not connect itself to a session lifecycle.

To prepare it for a bound local project, the owner manually copies `templates/automation.json` to `<project-root>/.planning/automation.json`, replaces the placeholder with the canonical local path, and leaves `enabled` set to `false` until a verified session-to-PC event interface is available.

When invoked manually by that future interface, the bridge accepts one JSON request on standard input:

```json
{"event":"prepare_context"}
```

or:

```json
{"event":"check_completion"}
```

The bridge always returns JSON. It fail-opens when unavailable and honors `<project-root>/.planning/automation-disabled` as an emergency bypass. It never accepts a root from the JSON request, uses a network, runs a shell command, or activates itself.
