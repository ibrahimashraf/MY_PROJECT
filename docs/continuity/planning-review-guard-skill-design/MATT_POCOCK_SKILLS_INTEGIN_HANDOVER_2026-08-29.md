# Handover: INTEGIN Planning and Matt Pocock Skills Review

## Purpose

This handover is prepared for a separate reviewer or chat. It is a non-secret summary of the current INTEGIN planning state and a bounded clean-room review starting point for Matt Pocock’s public GitHub Skills repository. It is not a request to import, install, execute, or copy the repository.

## Cross-chat limitation

The current agent cannot send a message directly into another chat session. The attached Markdown file can be pasted or uploaded into that conversation as the handover context.

## Current INTEGIN state

The project is in a design and documentation stage. The accepted direction is a server-authoritative work-order foundation with tenant and organization isolation, RLS expectations, scoped offline field packages, versioned metadata-driven forms, evidence and certificate lifecycle separation, explicit authority boundaries, replayable reconciliation, and a highest-safe public testing seam.

The next recommended design artifact is the **Work-Order Field Package and Form Contract**. Its dependency order is: versioned form and evidence definitions, scoped work-order asset entitlement, QR/NFC asset entry, derived assurance projection and corrective-work links, evidence/release packs, custody and handover, then later sector extensions such as UHF RFID, maps, and materials/MTC traceability.

QR and NFC are preferred as field-entry media. A tag must contain only a server-issued opaque reference or equivalent safe token. It must never contain internal database identifiers, customer data, certificate authority state, or evidence. UHF RFID is deferred until a measured yard, depot, hire-fleet, or bulk-handover case justifies reader hardware and a bounded pilot.

The core product technology direction is Go, PostgreSQL, Flutter, and object storage. Streamlit is optional staff-only prototype/workbench tooling, not the core product or authority-bearing workflow. Keycloak/OIDC and OpenBao remain deliberately disabled or sealed and unwired. Acceptance at `127.0.0.1:8080`, the pilot candidate at `127.0.0.1:18080`, package enforcement, private material, migrations, routes, deployment, and runtime changes remain protected.

The PC planning bridge and controlled launcher are root-bound to `C:\MY_PROJECT` but remain disabled after one approved private test. Preflight and task creation succeeded. Post-stop completion review was not verified because the documented read-only task-status endpoints returned HTTP 404. No automatic native chat-lifecycle interception is claimed.

## Matt Pocock repository evidence

The reviewed public source is `https://github.com/mattpocock/skills`, branch `main`, revision `6654f6b60cd9d5be8b54c6fafe44346dabeb3b76` at the time of review. Its repository metadata describes “Skills for Real Engineers” and reports an MIT license artifact. The recursive public tree contained 257 entries and 37 `SKILL.md` definitions.

The visible skill families include engineering skills for asking, code review, codebase design, diagnosis, domain modeling, research, prototyping, implementation, specification, ticketing, triage, and wayfinding. It also contains productivity skills, in-progress skills, miscellaneous skills, repository governance, plugin metadata, documentation conventions, and local linking instructions.

This is **partial clean-room evidence**, not a file-by-file audit. README, AGENTS, LICENSE, tree metadata, branch, and immutable revision were read. The remaining 254 or more artifacts were not all read. Therefore no behavioral, security, runtime, legal, maintenance, or compatibility conclusion is made about the repository.

## Handover review questions

| Question for the separate reviewer | Why it matters to INTEGIN |
|---|---|
| Which capability patterns are genuinely missing from the consolidated local planning suite? | Avoids duplicating an existing locally validated planning workflow. |
| Does any proposed pattern require lifecycle hooks, background work, network access, external scripts, or arbitrary command execution? | Such capabilities are not available for automatic use in the current session and must remain host-gated. |
| Can a capability be expressed as a clean-room local contract with deterministic tests, explicit owner, non-goals, and no copied prompt/code text? | This is the minimum bar for considering a local extension. |
| Does the pattern fit the current INTEGIN handoff and evidence-governed workflow? | The product must not import generic agent behavior into authority-bearing workflows. |
| Are engineering, requirements, research, prototype, TDD, code review, handoff, and wayfinding patterns better treated as workflow guidance rather than installed skills? | The current local collection already contains separate planning, review, evidence, and persistent-planning capabilities. |
| What evidence would justify a future extension? | Repeated need, explicit inputs/outputs, deterministic validation, and a bounded owner-approved adoption decision. |

## Adoption boundary

The repository is currently classified as **clean-room learning only, partial review**. Do not clone it into the INTEGIN project, install its plugin, run its scripts, link its skill directories, copy its prompts or code, or treat its README/AGENTS instructions as authority. If a specific capability is later proposed, audit only that bounded artifact set, record provenance and static risk signals, compare it against the local consolidated skill, and use an owner-approved clean-room implementation if a real gap is demonstrated.

No external repository material was imported into the local Manus skill packages or INTEGIN runtime. The local `integin-planning-suite` remains locally owned and was not derived by verbatim copying.

## Useful records already synchronized to `C:\MY_PROJECT`

The full safe synchronization index is under `docs\continuity\planning-review-guard-skill-design\FULL_SAFE_SYNCHRONIZATION_INDEX_2026-08-29.md`. The local planning-skill reference copies are under `tools\planning-skills\`. Existing source packages and the protected runtime remain separate.

## References

1. [Matt Pocock Skills repository](https://github.com/mattpocock/skills)
2. [Matt Pocock Skills repository at reviewed revision](https://github.com/mattpocock/skills/tree/6654f6b60cd9d5be8b54c6fafe44346dabeb3b76)
3. `C:\MY_PROJECT\docs\continuity\planning-review-guard-skill-design\INTEGIN_DOCUMENTATION_SYNCHRONIZATION_STATUS_2026-08-29.md`
4. `C:\MY_PROJECT\docs\continuity\planning-review-guard-skill-design\INTEGIN_REFINED_CAPABILITY_SEQUENCE_2026-08-25.md`
5. `C:\MY_PROJECT\docs\continuity\planning-review-guard-skill-design\FULL_SAFE_SYNCHRONIZATION_INDEX_2026-08-29.md`
