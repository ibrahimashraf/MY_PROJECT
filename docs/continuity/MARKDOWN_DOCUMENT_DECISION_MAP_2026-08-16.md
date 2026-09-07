# INTEGIN Markdown Documentation Decision Map — 2026-08-16

## Scope and method

This is a read-only inventory and classification of non-private Markdown records in `C:\MY PROJECT`. Private material, generated dependencies/caches, build output, Keycloak distribution files, and archived binary artifacts were excluded. No document was deleted, moved, or rewritten by this review.

The initial broad scan found 139 Markdown files, but that number included Python virtual-environment and test-cache dependency documents. The reviewed documentation corpus therefore excludes `.verification-venv`, `.pytest_cache`, `node_modules`, `.dart_tool`, build output, `.git`, `private`, Keycloak distribution content, and `archive`.

## Decision vocabulary

| Decision | Meaning |
|---|---|
| **Keep current** | Active canonical record or code-adjacent document; update only when its governed capability changes. |
| **Keep and update** | Active record whose next scheduled milestone must be appended here. |
| **Reference / snapshot** | Retain read-only for recovery, parity, or history; do not bulk-sync it. |
| **Archive-only** | Historical recovery evidence; retain but do not update. |
| **Review required** | The document is missing from the actual Windows workspace or needs an explicit owner decision before it can be treated as canonical. |

## Workspace-level decisions

| Area | Reviewed Markdown records | Decision | Required action |
|---|---:|---|---|
| Workspace root | 8 | Keep the root continuity set canonical. | Update only `CURRENT_STATE.md`, `task_plan.md`, `findings.md`, `progress.md`, and `todo.md` after a material result, stop, or recovery. |
| `docs` | 27 | Keep as the structured design, policy, runbook, and contract library. | Update the documentation index and decision map when a new canonical document is added. |
| `operations` | 3 | Keep operational plans and local contract copies. | Update pilot workflow evidence only after the visible walkthrough. |
| `integin-pilot-source` | 17 source-owned records after excluding generated dependencies. | Keep code-adjacent documents with the active pilot source. | Update only with related source/runtime changes. |
| `integin-source` | 16 | Reference snapshot. | Exact Markdown hash parity with the pilot-source copy was confirmed for all 16 matching records; do not bulk-update. |
| `integin-pilot-admin` | 2 | Reference/administrative boundary. | Keep its local contract copy and boundary record. |
| `migration-backups` | 51 | Archive-only recovery evidence. | Do not update, delete, or relocate during normal work. |

## Root continuity records

| File | Decision | Update rule |
|---|---|---|
| `CURRENT_STATE.md` | **Keep and update** | Authoritative concise current-state ledger; update after the Windows walkthrough, a runtime recovery, or an explicit gate decision. |
| `task_plan.md` | **Keep and update** | Long-form phase history and current addenda; do not delete historical evidence. |
| `findings.md` | **Keep and update** | Append verified discoveries and limitations; do not treat it as the sole current-state source. |
| `progress.md` | **Keep and update** | Append completed, failed, stopped, and recovered actions with non-secret evidence. |
| `todo.md` | **Keep and update** | Maintain active unchecked work only; archive completed checklist sections only after a separate retention review. |
| `ENGINEERING_CONTINUATION_GUIDE.md` | **Keep current** | Stable engineer handoff; revise after material architecture or operating-boundary changes. |
| `AI_ENGINEER_RESUME_PROMPT.md` | **Keep current** | Resume prompt; refresh after current milestone changes. |
| `WORKSPACE_MAP.md` | **Keep and update** | Workspace navigation index; add a link to this decision map. |

## Structured documentation library (`docs`)

| Document set | Decision | Update rule |
|---|---|---|
| Architecture and local runtime: `ARCHITECTURE_EVOLUTION_ROADMAP.md`, `LOCAL_RUNTIME_STRENGTHENING_PLAN.md`, `LOCAL_PILOT_BASELINE_20260815.md`, `LIVE_INTEGRATION_RUNBOOK.md` | **Keep current** | Revise only after an approved architecture, runtime, or recovery change. |
| Signed protocol and release: `CANONICAL_SIGNED_VECTOR_SPEC_V1.md`, `EVIDENCE_EXPORT_MANIFEST_V1.md`, `SIGNED_AUDIT_CHECKPOINT_V1.md`, `RELEASE_RECORD_TEMPLATE_V1.md` | **Keep current** | Revise only through versioned contract or release work. |
| Controlled pilot decisions: `DECISION_01` through `DECISION_06` | **Keep current, approval-gated** | Retain as decision records; do not mark approved or implement production controls without accountable approval. |
| Identity and enrollment: `IDENTITY_AUTHORIZATION_DESIGN.md`, `OIDC_HANDOFF.md`, `PRODUCTION_DEVICE_ENROLLMENT_SPEC.md` | **Keep current, production-gated** | Update only after separate production identity/enrollment work. |
| Trust and governance: `OBSERVABILITY_FOUNDATION_CONTRACT_V1.md`, `OPENBAO_PILOT_REHEARSAL_PLAN.md`, `PRODUCTION_SECRET_INVENTORY_AND_OPENBAO_RECOVERY_DESIGN.md`, `RETENTION_LEGAL_HOLD_REDACTION_EXPORT_ACCESS_POLICY_V1.md`, `PLATFORM_GOVERNANCE_BACKLOG.md`, `PRODUCTION_TRUST_BASELINE_ROADMAP.md`, `PRODUCTION_READINESS_GAP_REVIEW.md` | **Keep current, staged** | Update only when the named workstream moves from specification to approved implementation. |
| `EXTERNAL_REVIEW_TRIAGE.md` | **Reference / snapshot** | Keep the review response and evidence, but do not treat it as a current plan. |
| `docs\README.md` | **Keep and update** | Make it the directory index linking this decision map and canonical documents. |

### Documentation records requiring reconciliation

| Expected record | Actual Windows status | Decision | Required action |
|---|---|---|---|
| `PILOT_OBSERVABILITY_FOUNDATION_PLAN.md` | Not visible in actual Windows `docs`. | **Review required** | Recreate from the approved design-only scope before treating it as canonical. |
| `VERIFICATION_COVERAGE_MATRIX_2026-08-16.md` | Not visible in actual Windows `docs`. | **Review required** | Recreate from current verified evidence; do not claim it exists until copied locally. |
| `docs\governance\TECHNICAL_DEBT_REVIEW_2026-08-16.md` | Not visible at the actual workspace root. | **Review required** | Recreate or relocate only after historical reconciliation with `CURRENT_STATE.md`. |

## Active source and operations records

| Location / document set | Decision | Update rule |
|---|---|---|
| `integin-pilot-source\README.md`, `AI_DEPLOYMENT.md`, `LOCAL_INFRASTRUCTURE.md`, `LIVE_INTEGRATION_RUNBOOK.md`, `RLS_RUNTIME_ROLE.md` | **Keep current** | Code-adjacent operational source records; revise with implementation/runtime changes. |
| `integin-pilot-source\ARCHITECTURE_EVOLUTION_ROADMAP.md`, `IMPROVEMENT_PLAN.md`, `task_plan.md`, `findings.md`, `progress.md` | **Reference / source-local history** | Retain for source provenance; root continuity records are canonical for the consolidated workspace. |
| `integin-pilot-source\field_app\README.md`, `field_app\ARCHITECTURE.md` | **Keep current** | Update with Flutter architecture, storage, sync, or user-visible workflow changes. |
| `integin-pilot-source\internal\domain\README.md`, `internal\shared\README.md`, `internal\INFRASTRUCTURE.md`, `internal\SERVER.md` | **Keep current** | Update only with the governed package boundary. |
| `integin-pilot-source\ai_service\README.md` | **Keep current** | Update only with advisory service contract or local-development changes. |
| `integin-source` Markdown set | **Reference / snapshot** | All 16 equivalent Markdown files matched the active pilot-source copy by SHA-256. Retain as a stable snapshot; do not bulk-edit. |
| `operations\CONSOLIDATION_COPY_RECORD.md` | **Reference / snapshot** | Keep as a transfer/audit record. |
| `operations\pilot\PILOT_WORKFLOW_ACCEPTANCE_PLAN.md` | **Keep and update** | Update only after the isolated human-visible walkthrough. |
| Local vector copies under `operations\pilot\contracts` and `integin-pilot-admin\contracts` | **Reference copies** | Keep because they bind local contract consumers; the `docs` vector specification is the explanatory canonical record. |
| `integin-pilot-admin\CONSOLIDATION_BOUNDARY.md` | **Keep current** | Update only when the administrative/source consolidation boundary changes. |

## Archive and backup records

All 51 Markdown files under `migration-backups` are **archive-only**. They are recovery and transfer evidence, not active documentation. Keep them in place and do not update, move, de-duplicate, or delete them as part of ordinary documentation maintenance.

## Safe update order

1. Update the root continuity records after each material pilot result.
2. Update the specific source or operations document that governs a changed capability.
3. Refresh `docs\README.md`, `WORKSPACE_MAP.md`, and this decision map when a canonical document is added or relocated.
4. Preserve snapshot, administrative, and migration-backup records unchanged.
5. Treat the three Windows-missing records above as reconciliation work, not as proof that their content is already available locally.

## Reconciliation completion — 2026-08-16

The three previously absent local records are now present and structurally validated in the actual Windows workspace: `docs\PILOT_OBSERVABILITY_FOUNDATION_PLAN.md`, `docs\continuity\VERIFICATION_COVERAGE_MATRIX_2026-08-16.md`, and `docs\governance\TECHNICAL_DEBT_REVIEW_2026-08-16.md`. Their earlier **Review required** status is resolved. The observability plan remains design-only; the verification matrix retains the Windows walkthrough as pending; and the debt review retains production authority work as separately gated.
