# Complete Planning-With-Files Capability Reconciliation Register

## Coverage boundary

This register covers all **29 artifacts** in the prior bounded static audit of the online `planning-with-files` v3.11.2 skill subtree. Artifact names and static classifications are evidence only; they are not executable specifications. Every local item below must be independently written, tested, and tracked.

| ID | Audited artifact | Capability category inferred from audited role | Local outcome | Current disposition |
|---|---|---|---|---|
| CC-01 | `SKILL.md` | Planning instructions, context handling, automation configuration | Keep the active local planning workflow; add all compatible local helper/documentation capabilities. | Active, expanding. |
| CC-02 | `examples.md` | Worked usage examples | Add local, non-secret planning examples. | Pending design. |
| CC-03 | `reference.md` | Reference guidance | Add local reference guidance for record formats and tool exit codes. | Pending design. |
| CC-04 | `attest-plan.ps1` | Plan-integrity attestation | Rebuild as one explicit cross-platform local Python command with a sidecar digest and verify mode. | Candidate; requires sidecar/overwrite policy. |
| CC-05 | `attest-plan.sh` | Plan-integrity attestation | Covered by CC-04; no duplicate platform script. | Candidate; requires sidecar/overwrite policy. |
| CC-06 | `check-complete.ps1` | Phase-completion inspection | Rebuild as explicit read-only local completion checker. | Candidate. |
| CC-07 | `check-complete.sh` | Phase-completion inspection | Covered by CC-06; no duplicate platform script. | Candidate. |
| CC-08 | `gate-stop.sh` | Forced completion gate | A host-triggered gate needs a verified host integration contract, override policy, and failure timeout. | Blocked pending host support and owner policy. |
| CC-09 | `init-session.ps1` | Initialize planning records | Rebuild as explicit Python initializer that requires a root and refuses nonempty targets without a deliberate override. | Candidate. |
| CC-10 | `init-session.sh` | Initialize planning records | Covered by CC-09; no duplicate platform script. | Candidate. |
| CC-11 | `inject-plan.sh` | Automatic context injection | Automatic injection requires a verified lifecycle-hook contract and privacy/root-binding model. | Blocked pending host support. |
| CC-12 | `ledger-append.ps1` | Append a planning/audit ledger entry | Rebuild as an explicit append-only local utility with user-provided text and no hidden invocation. | Candidate; requires ledger path/format policy. |
| CC-13 | `ledger-append.sh` | Append a planning/audit ledger entry | Covered by CC-12; no duplicate platform script. | Candidate; requires ledger path/format policy. |
| CC-14 | `ledger-summary.ps1` | Summarize planning/audit ledger | Rebuild as an explicit read-only local summary command. | Candidate. |
| CC-15 | `ledger-summary.sh` | Summarize planning/audit ledger | Covered by CC-14; no duplicate platform script. | Candidate. |
| CC-16 | `phase-status.ps1` | Inspect phase status | Rebuild as explicit read-only local phase-status command. | Candidate. |
| CC-17 | `phase-status.sh` | Inspect phase status | Covered by CC-16; no duplicate platform script. | Candidate. |
| CC-18 | `plan-doctor.sh` | Diagnose planning-record setup | Rebuild as explicit read-only local diagnostic combining record, root, and optional attestation checks. | Candidate. |
| CC-19 | `resolve-plan-dir.ps1` | Resolve selected plan location | Rebuild only with explicit root/plan ID precedence and no automatic newest-plan fallback. | Candidate; requires root-state policy. |
| CC-20 | `resolve-plan-dir.sh` | Resolve selected plan location | Covered by CC-19; no duplicate platform script. | Candidate; requires root-state policy. |
| CC-21 | `session-catchup.py` | Compare planning state with local workspace changes | Rebuild as explicit local workspace-reconciliation report; do not read host-private session stores. | Candidate. |
| CC-22 | `set-active-plan.ps1` | Set selected plan pointer | Rebuild only as explicit project-local pointer update with canonicalization and recovery. | Candidate; requires root-state policy. |
| CC-23 | `set-active-plan.sh` | Set selected plan pointer | Covered by CC-22; no duplicate platform script. | Candidate; requires root-state policy. |
| CC-24 | `templates/analytics_findings.md` | Analytics-oriented findings layout | Add a clean-room analytics findings template. | Pending design. |
| CC-25 | `templates/analytics_task_plan.md` | Analytics-oriented task-plan layout | Add a clean-room analytics task-plan template. | Pending design. |
| CC-26 | `templates/findings.md` | Generic findings layout | Existing local template covers this role. | Active. |
| CC-27 | `templates/progress.md` | Generic progress layout | Existing local template covers this role. | Active. |
| CC-28 | `templates/task_plan.md` | Generic task-plan layout | Existing local template covers this role. | Active. |
| CC-29 | `templates/task_plan_autonomous.md` | Long-running/autonomous task-plan layout | Add a non-blocking long-running task-plan template; it must not create background execution or forced continuation. | Pending design. |

## Global boundaries

The register does not permit direct external-source copying, external script execution, unverified network access, or hidden runtime behavior. The status **candidate** means “design and approval remain necessary”; it does not mean active or implemented. The status **blocked** means no local implementation can truthfully activate the capability until a verified host contract and the stated owner policy exist.

## No-silent-omission accounting

| Disposition | Artifact count |
|---|---:|
| Active local role | 4 |
| Candidate local equivalent | 18 |
| Pending local template/example/reference design | 5 |
| Blocked host-dependent capability | 2 |
| Total | 29 |

The total counts role dispositions; paired platform-specific source artifacts may share one cross-platform local equivalent. The audited source artifact total remains 29.

## Current implementation disposition

This table supersedes the preliminary “pending” and “candidate” labels above after the complete local pack implementation batch.

| Current state | Covered register IDs | Local evidence |
|---|---|---|
| Implemented — pending final package validation | CC-01 to CC-07, CC-09 to CC-10, CC-12 to CC-29 | `SKILL.md`, `plan_records.py`, `validate_planning_records.py`, six local templates, `examples.md`, `reference.md`, and the explicit host-compatibility reference. |
| Blocked host-dependent behavior | CC-08, CC-11 | `references/host-automation-compatibility.md` records the exact missing host contract; no hook, injection, or forced gate is active. |
| Rejected direct external behavior | Direct external script execution and direct external-source copying, which are cross-cutting requests rather than separate audited artifact files. | Clean-room audit boundary and automation classification. |
