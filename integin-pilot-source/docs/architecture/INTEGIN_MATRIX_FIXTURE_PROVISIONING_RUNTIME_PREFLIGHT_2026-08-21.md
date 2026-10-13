# INTEGIN Matrix Fixture Provisioning Runtime Preflight

## Result

The Field tooling is **not fundamentally unavailable**. The installed Flutter SDK includes a working Dart runtime and Flutter tools snapshot. The ordinary `flutter.bat --version` launcher exits with code `1` and no diagnostic output in the connected workstation process, but the tools snapshot runs successfully when invoked through Dart.

The first snapshot test attempt failed because the sidecar process did not expose the standard `PROGRAMFILES(X86)` environment variable. Supplying `C:\Program Files (x86)` for the process only, without changing the user or system environment, allowed the live provisioning acceptance test to run. It exited `0` with its single test **skipped**, as designed.

| Requirement | Status | Evidence |
|---|---|---|
| Flutter tools execution | Available through bundled Dart and `flutter_tools.snapshot` | Snapshot reported Flutter `3.47.0`; Field test runner started successfully. |
| Persistent workstation/toolchain mutation | Not performed | No SDK, PATH, registry, user environment, or project configuration change was made. |
| Live provisioning, sync, and evidence endpoints | Absent | All three `INTEGIN_LIVE_*` definitions were absent in the connected process. |
| Matrix Fixture Provisioning runtime proof | Not executed | The test skipped before any provisioning, sync, evidence, or database action. |

## Required next action

Provide an isolated, non-production set of `INTEGIN_LIVE_PROVISIONING_ENDPOINT`, `INTEGIN_LIVE_SYNC_ENDPOINT`, and `INTEGIN_LIVE_EVIDENCE_ENDPOINT` values through the approved controlled runtime harness. The commands must retain the process-only `PROGRAMFILES(X86)` workaround or repair the launcher through a separately reviewed workstation change. Before any execution, repeat acceptance preflight, confirm endpoint isolation, inventory fixtures, and define cleanup receipts.

> This preflight provides no Matrix Fixture Provisioning completion claim. It proves only that the Field test can be started through the snapshot workaround and that it safely skips when its required endpoints are absent.
