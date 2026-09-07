# INTEGIN Multi-Environment Portability Baseline

**Status:** Corrective architecture baseline. It changes no runtime, data volume, secret, deployment, or authority path.

## Direct answer

The platform **should have had an explicit multi-environment operations baseline from the beginning**. The core application was largely designed portably: Go modules build across operating systems, PostgreSQL and RustFS run in Linux containers, Flutter business code is portable, and Python/React are portable application technologies. However, the operational packaging was optimized around the available Windows pilot too early. That created a project that is **portable in core code but incomplete in repeatable deployment artifacts**.

This was not a reason to discard the existing work. It is a correctable architecture gap around deployment automation and verification, not a fundamental product rewrite.

## What was portable from the beginning

| Area | Portable choice already made | Evidence |
| --- | --- | --- |
| Core backend | Go modular monolith with platform-neutral networking, PostgreSQL driver, and HTTP boundaries | The acceptance server and pilot candidate cross-built for Linux `amd64`. |
| Durable services | PostgreSQL and RustFS as OCI container images | Existing Windows Docker Desktop services use the same Linux images that a Linux Docker Engine uses. |
| Field business logic | Flutter/Dart application layers | Field code is not tied to Windows APIs. |
| Advisory services | Python FastAPI and browser-based React/TypeScript UI | Both are runtime-portable application stacks. |
| Data model | PostgreSQL-backed tenant-scoped records and object storage | Not dependent on Windows file-system semantics for authority state. |

## What was too Windows-pilot specific

| Gap | Current effect | Corrective action |
| --- | --- | --- |
| Operations automation | The local controls are primarily PowerShell; the current inventory found 38 operation `.ps1` files. | Introduce POSIX shell equivalents and common declarative deployment files. |
| Service lifecycle | Acceptance is started as a direct Windows executable. | Provide Linux `systemd` unit templates and a documented native-binary release layout. |
| Container orchestration | Docker services were launched through local operational procedure rather than source-controlled Compose files. | Add source-controlled Compose profiles for acceptance and isolated pilot data services. |
| Cross-platform verification | Linux builds were not a routine release gate from the first day. | Add a build matrix for Windows and Linux Go binaries, Flutter analysis, Python tests, and configuration linting. |
| Field target runners | The current Flutter project has generated `web` and `windows` runners only. | Generate and validate Linux/Android/iOS runners before distributing those packages. |
| Environment contract | Private configuration exists correctly outside source, but environment lifecycle docs are Windows-centered. | Define one secret-free environment schema and per-platform secure-file instructions. |

## Corrective architecture baseline

The project should now treat **application logic**, **deployment packaging**, and **environment operations** as separate products with one shared contract.

| Baseline artifact | Purpose | Must be environment-neutral |
| --- | --- | --- |
| Go build matrix | Produces verified Windows and Linux binaries from the same commit. | Yes: target OS/architecture are build parameters, never hard-coded paths. |
| Docker Compose profiles | Starts PostgreSQL and RustFS for acceptance or isolated pilot with distinct named volumes. | Yes: use variable-driven ports and volume names, no local personal paths. |
| Service units | Starts the Linux Go binary through `systemd`; Windows retains a documented host-process wrapper. | Yes: share health, readiness, logging, and environment contracts. |
| Environment schema | Documents required variable names, permitted runtime values, and secure file location rules. | Yes: values remain private and are never committed. |
| Verification matrix | Runs Go tests/vet, target builds, Flutter analysis, Python tests, health checks, tenant-boundary checks, backup/restore rehearsal, and rollback proof. | Yes: same acceptance criteria, platform-specific commands only where unavoidable. |
| Release manifest | Records component versions, migration state, hash, target architecture, and rollback artifact. | Yes: no secret values or host-specific paths. |

## A better deployment model going forward

The correct long-term pattern is **build once, deploy by environment profile**.

1. The same reviewed source produces a Windows binary and Linux binary.
2. PostgreSQL and RustFS are defined through portable Compose profiles with separately named acceptance and pilot volumes.
3. Windows and Linux each have thin lifecycle adapters—PowerShell/Task Scheduler or service wrapper on Windows; `systemd` and shell scripts on Linux—but they call the same health, readiness, backup, and rollback logic.
4. Flutter platform runners are generated deliberately, then the same Field business code is built for target devices.
5. A release is not accepted until the required target build and non-secret verification matrix completes.

## What must not happen

This correction must **not** become an uncontrolled migration. Do not copy Windows secrets into another host casually, reuse acceptance database volumes for a Linux pilot, enable enforcement to test portability, or replace the Windows control before a Linux pilot proves restore and rollback. The current Windows acceptance remains the protected reference until a separately governed Linux pilot has completed its evidence gates.

## Next implementation gate

The next safe engineering task is to create the **source-controlled multi-environment deployment kit** in an isolated review scope: Compose profiles, Linux `systemd` templates, POSIX operational scripts, target build commands, and a secret-free environment schema. It is source-only work. Deploying that kit to a Linux host remains a later, separately governed pilot step.
