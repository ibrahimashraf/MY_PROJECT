# INTEGIN Local Planning Bridge

This is a disabled-by-default local helper for an owner-bound planning root. It is not a service, scheduler, port listener, session hook, or background process.

The bridge is invoked explicitly through standard input. It accepts only `prepare_context` and `check_completion` events. Its selected project root must match `<root>\.planning\automation.json` exactly.

Before any manual test, copy `templates\automation.json` to `<bound-root>\.planning\automation.json`, replace the placeholder with the canonical local root, and keep `enabled` set to `false`.

```powershell
'{"event":"prepare_context"}' | python .\scripts\planning_bridge.py --root "C:\Bounded\Project"
'{"event":"check_completion"}' | python .\scripts\planning_bridge.py --root "C:\Bounded\Project"
```

The emergency bypass is `<bound-root>\.planning\automation-disabled`. When present, the bridge injects nothing and allows completion. The detailed non-secret deployment contract is `C:\MY_PROJECT\docs\operations\INTEGIN_PC_PLANNING_BRIDGE_DEPLOYMENT_CONTRACT_2026-08-25.md`.

Do not set `enabled` to `true` or connect this helper to a lifecycle event until a verified session-to-PC event interface, a final owner confirmation, bypass proof, and bounded tests are available.
