# INTEGIN Quality-Recovery Audit — 2026-08-16

**Scope:** Current workspace, protected runtime boundaries, recent Flutter tooling changes, and continuation records.
**Method:** Read-only verification except for the already documented reversible Flutter wrapper shim and prior Visual Studio component attempts. No private value was read or displayed.

## Overall Assessment

The project is **not evidenced as damaged**. Acceptance and pilot runtimes are healthy and separated. The current field-app acceptance remains blocked by one verified Windows native dependency gap, not by backend, database, identity, or Flutter core-toolchain failure. The audit did identify process and traceability weaknesses that must be corrected before treating the workflow as accepted.

| Area | Audit status | Evidence |
|---|---|---|
| Acceptance and pilot isolation | Verified | Health and readiness returned HTTP 200 on ports 8080 and 18080. |
| OIDC at rest | Verified | Identity-session route returned HTTP 404 on both runtimes. |
| OpenBao state | Verified | Pilot OpenBao reported initialized and sealed through its local HTTP listener. |
| Pilot fixture handling | Verified presence only | Fixture pointer and referenced protected file exist; contents were not read. |
| Flutter core Windows toolchain | Verified | Flutter Doctor recognized Visual Studio and a disposable Windows Flutter probe built. |
| Flutter wrapper repair | Verified and reversible | Current shim returns Flutter 3.47.0; original wrapper backup hash matches. |
| Flutter test suite | Verified | 31 passed and 1 intentional skip through the repaired wrapper. |
| Go test suite | Verified | `go test ./...` completed successfully. |
| INTEGIN Windows field-app build | Blocked | Native secure-storage plugin reaches C1083 while its ATL header dependency is absent. |

## Verified Recent Flutter Changes

The current Flutter SDK remains version 3.47.0 at `C:\flutter`. Before the shim, its tracked wrapper files matched the corresponding official source. The active wrapper is now an 11-line compatibility shim that directly invokes the existing bundled Dart executable and Flutter tool snapshot, bypassing the failed shared bootstrap path. The original 74-line wrapper is retained with a SHA-256 record under `C:\MY PROJECT\tools\flutter-wrapper-shim-backup-20260816`. The backup hash was rechecked during this audit.

This is an intentional local deviation, not a standard Flutter installation state. It is currently justified by successful wrapper, Doctor, test-suite, and fresh-probe evidence, but it must remain reversible and be compared against the matching official repository revision after pilot acceptance.

## Verified Field-App Native Blocker

The field app includes `flutter_secure_storage_windows`. Its C++ source includes `<atlstr.h>`. The Windows build reached that plugin and emitted C1083. The required `atlstr.h` file is absent from the installed Build Tools tree. Microsoft documents C++ ATL as an optional component and directs users to select the matching ATL component in Visual Studio Installer when ATL libraries are required.[1] [2]

**Confidence:** High that the missing ATL installation is the immediate native dependency. The retained verbose log did not yield a clean, single-line include-name capture, so the report does not claim a text-exact compiler message beyond the observed C1083 and verified absent ATL header.

## Findings Requiring Correction or Discipline

| Priority | Finding | Required control |
|---|---|---|
| P0 | The previous optional-component guidance omitted C++ ATL even though the actual field plugin requires it. | Inspect active Flutter Windows plugin sources before advising any future workstation component selection. |
| P1 | The active pilot source folder has no Git metadata. | Establish or restore a controlled source baseline before any broad refactor; do not claim a complete source-diff history from this folder. |
| P1 | The current Flutter wrapper is locally modified. | Retain its backup until the field acceptance and official-repository comparison are complete. |
| P1 | The current field app cannot build until ATL is installed. | Use Visual Studio Installer to select `C++ ATL for x64/x86`, then verify `atlstr.h` before rebuilding. |
| P2 | Full non-private secret-value scan was not completed because broad recursive scans exceeded the connected-session budget. | Use a bounded, offline file-scanning procedure before a release or source-control import; do not infer global absence of secret leakage from this audit. |
| P2 | Remote desktop command-session disconnects interrupted diagnostics. | Treat workstation connector stability as an operational limitation and validate important commands through persisted evidence. |

## Corrective Continuation Plan

1. Install **C++ ATL for x64/x86** visibly in Visual Studio Installer and verify `atlstr.h` exists.
2. Rebuild the field app without changing Flutter, INTEGIN source, pilot fixture, acceptance, OIDC, or OpenBao.
3. Launch only the guarded Windows-debug pilot mode against `http://127.0.0.1:18080/sync`.
4. Perform the offline queue, authoritative receipt, duplicate-safe replay, operator review, and advisory-only walkthrough.
5. Compare the active Flutter SDK with official revision `4cf24164269a5ebf0c16a028a00727d0e77bbb05` and document only meaningful differences, including the wrapper shim.
6. Complete a bounded non-private secret scan before source-control or production-readiness work.

## References

[1]: https://learn.microsoft.com/en-us/visualstudio/msbuild/errors/msb8041?view=visualstudio "MSB8041: MFC/ATL Libraries are required for this project"
[2]: https://learn.microsoft.com/en-us/visualstudio/install/workload-component-id-vs-build-tools?view=visualstudio "Visual Studio Build Tools workload and component IDs"
