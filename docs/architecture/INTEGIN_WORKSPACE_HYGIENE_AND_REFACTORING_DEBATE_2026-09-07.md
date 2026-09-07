# 🏛️ INTEGIN Governance Review & Architectural Debate: Workspace Refactoring & Technical Debt

**Document ID:** DEBATE-2026-09-07-WORKSPACE-HYGIENE  
**Governing Standard:** [`integin-review-governance`](file:///c:/MY_PROJECT/.agents/skills/integin-review-governance/SKILL.md)  
**Date:** 2026-09-07  
**Scope:** Evaluation of Workspace Hygiene Items 6 through 12 across 4 Independent Engineering Lenses.

---

## 1. Decision Frame

*   **Subject Under Debate:** Whether, how, and under what constraints to execute the 7 newly identified cleanup, ordering, and refactoring items (Items 6–12) across `operations/`, `docs/`, `quiet-signal/`, and `integin-pilot-source/`.
*   **Comparison Baseline:** The current repository state following the initial consolidation of root markdown files into `WORKSPACE.md` and `TRACKER.md`.
*   **Protected Boundaries:**
    1.  *Evidence Preservation (ISO 17020)*: Do not destroy raw test drill outputs, manifests, or audit records.
    2.  *Runtime Stability*: Do not execute unverified SQL migrations or alter running container state.
    3.  *Zero Hallucination Standard*: Ensure all documentation links resolve to existing files on disk.

---

## 2. The 4 Independent Review Lenses

1.  **🔍 Lens 1: Operational Hygiene & Agent Determinism (Pragmatist / Tooling)**
    *   *Focus:* Eliminating noise, stale scripts, dead markdown links, and duplicate lockfiles that induce agent hallucination and build drift.
2.  **📜 Lens 2: Audit Trail & Non-Repudiation (Compliance / ISO 17020 Auditor)**
    *   *Focus:* Ensuring that historical test runs, backups, and verification artifacts are preserved in immutable archive vaults rather than hard-deleted.
3.  **🗄️ Lens 3: Database & Migration Integrity (DBA / SRE)**
    *   *Focus:* Validating sequential migration numbers, schema ledger tracking (`schema_migrations`), and verifying that migration SQL does not reference phantom tables.
4.  **⚛️ Lens 4: Frontend & Toolchain Determinism (Web Architect)**
    *   *Focus:* Preventing dual-lockfile divergence between npm and pnpm in `quiet-signal/` and ensuring clean workspace isolation.

---

## 3. Evidence Ledger

| Evidence ID | Source / Test Artifact | What It Supports | Limitation / Caveat |
|---|---|---|---|
| **EV-01** | `operations/pilot/*.bak` | 21 backup script files created during August Keycloak and OpenBao debugging. | Files are non-executable backups; they clutter the active operational path. |
| **EV-02** | `operations/acceptance/` | 21 timestamped drill directories and two 11MB pre-compiled server binaries. | Drills contain historical stage evidence; binaries are outdated August builds. |
| **EV-03** | `docs/README.md` | Lines 4–6 link to deleted files (`../CURRENT_STATE.md`, `../WORKSPACE_MAP.md`, `../ENGINEERING_CONTINUATION_GUIDE.md`). | Confirmed dead links causing agent navigation errors. |
| **EV-04** | `quiet-signal/` | Both `package-lock.json` (Aug 31) and `pnpm-lock.yaml` (Sept 2) exist. | No `"packageManager"` constraint specified in `package.json`. |
| **EV-05** | `LOCAL_INFRASTRUCTURE.md` & `LIVE_INTEGRATION_RUNBOOK.md` | Text specifies port `9000` (S3) and `8080` (HTTP) with bucket `integin-evidence`. | In conflict with live verified standard: port `19000`, port `18080`, bucket `integin-pilot-evidence`. |
| **EV-06** | `integin-pilot-source/db/migration/20260903_add_certificate_indexes.up.sql` | Migration defines indexes on `webhook_deliveries` and `shortlinks`. | **CRITICAL:** `\dt` in `integin_dev` proves neither table exists. Migration will fail with SQLSTATE 42P01. |
| **EV-07** | `integin-pilot-source/apply_migrations.ps1` | Line 52 hardcodes discovery path to `.\migrations\*.sql`. | Ignores `db/migration/` entirely, creating two disconnected migration runners. |

---

## 4. Multi-Lens Debate & Findings

### Debate on Item 6: `operations/pilot/` Backup Clutter (21 `.bak` files)
*   *Pragmatist:* "Having 21 `.bak` files mixed with `.ps1` launchers is an operational hazard. An engineer or agent can easily execute an outdated `.bak` script."
*   *Auditor:* "These scripts record how OpenBao and Keycloak were debugged. They must not be permanently deleted without provenance."
*   *Disposition:* **CONFIRMED FOR ARCHIVAL**. Move all 21 `.bak` files and large logs to `archive/operations-pilot-baks/`.

### Debate on Item 7: `operations/acceptance/` 22MB Loose Binaries & 21 Drill Dirs
*   *Pragmatist:* "Two 11MB `.exe` binaries sitting in operations are unversioned and redundant. Binaries belong in `bin/` or git releases, not loose in scripts."
*   *Auditor:* "The 21 drill folders represent the August 21 acceptance baseline. Move them to `archive/acceptance-drills-aug2026/`, but do NOT delete them."
*   *Disposition:* **CONFIRMED FOR ARCHIVAL & BINARY PURGE**. Relocate drill folders to archive; delete the redundant 22MB loose executables.

### Debate on Item 8: `docs/README.md` Dead Links
*   *All Lenses Unanimous:* "Dead links in documentation violate the Single Source of Truth rule. Any new agent reading `docs/README.md` immediately attempts to read deleted files."
*   *Disposition:* **CONFIRMED FOR IMMEDIATE REPAIR**. Update `docs/README.md` to link to `WORKSPACE.md` and `TRACKER.md`.

### Debate on Item 9 & 10: `quiet-signal/` Lockfile Drift & Checkpoint
*   *Web Architect:* "`pnpm-lock.yaml` was modified on Sept 2, whereas `package-lock.json` was generated on Aug 31. Running `npm install` and `pnpm install` concurrently will cause dependency tree drift."
*   *Auditor:* "`CHECKPOINT_2026-08-29.md` is historical and belongs in `archive/`."
*   *Disposition:* **CONFIRMED**. Purge `.pytest_cache/`, archive the checkpoint file, and standardize on `pnpm-lock.yaml`.

### Debate on Item 11: Runbook Port & Bucket Drift
*   *SRE / Tooling:* "Having runbooks state port `9000` when Keycloak and RustFS run on `19000` causes immediate onboarding friction and failed health checks."
*   *Disposition:* **CONFIRMED FOR ALIGNMENT**. Update `LOCAL_INFRASTRUCTURE.md` and `LIVE_INTEGRATION_RUNBOOK.md` to match the canonical live environment variables in `WORKSPACE.md`.

### Debate on Item 12: Dual Migration Paths & Phantom Tables (`CRITICAL`)
*   *DBA / SRE:* "We uncovered a serious architectural hazard. `db/migration/20260903_add_certificate_indexes.up.sql` was placed in a non-standard directory (`db/migration/` instead of `migrations/`), completely bypassed by `apply_migrations.ps1`. More dangerously, **it references tables `webhook_deliveries` and `shortlinks` which do not exist in the database!**"
*   *Auditor:* "Renumbering this file to `0070_...` and applying it blindly would cause migration failures and break CI."
*   *Disposition:* **DEFECT IDENTIFIED / REFACTORING REQUIRED**.
    1. Do NOT apply `20260903_add_certificate_indexes.up.sql` as-is.
    2. Extract only the valid indexes for existing tables (`certificate_record`, `certificate_policy`, `inspection_record`, `certificate_snapshot`, `certificate_template_cell`, `asset_registry`, `work_order_assignment_scope`, `work_order_scope_item`).
    3. Remove the speculative indexes on non-existent tables (`webhook_deliveries`, `shortlinks`).
    4. Move the cleaned migration into `migrations/0070_add_certificate_performance_indexes.sql` so all schema changes remain strictly sequential.

---

## 5. Findings & Dispositions Summary

| Item | Focus | Risk / Rationale | Disposition | Action to Execute |
|:---:|---|---|:---:|---|
| **6** | `operations/pilot/*.bak` | Operational script confusion | **CONFIRMED** | Move 21 `.bak` files & logs to `archive/operations-pilot-baks/` |
| **7** | `operations/acceptance/` | 22MB loose binaries & 21 drill dirs | **CONFIRMED** | Move drills to `archive/acceptance-drills-aug2026/`; remove duplicate `.exe` |
| **8** | `docs/README.md` | Dead links inducing agent hallucinations | **CONFIRMED** | Update links to `WORKSPACE.md` and `TRACKER.md` |
| **9** | `quiet-signal/` Lockfiles | Dual lockfile drift & stray Python cache | **CONFIRMED** | Remove `.pytest_cache/` and stale `package-lock.json` |
| **10** | `quiet-signal/` Checkpoint | Clutter in frontend source root | **CONFIRMED** | Move `CHECKPOINT_2026-08-29.md` to `archive/` |
| **11** | Runbook Ports | Port `9000` / `8080` drift from live runtime | **CONFIRMED** | Align runbooks with `WORKSPACE.md` live parameters |
| **12** | Dual Migration & Phantom Tables | Broken migration referencing non-existent tables | **CRITICAL FIX** | Sanitize indexes, drop phantom tables, move to `migrations/0070_...` |

---

## 6. Recommended Next Actions & Boundaries

1.  **Authorized Immediate Execution:**
    *   Execute safe file relocations to `archive/` (Items 6, 7, 10).
    *   Repair dead documentation links in `docs/README.md` (Item 8).
    *   Clean `quiet-signal` lockfiles and caches (Item 9).
    *   Align runbook port numbers with live infrastructure (Item 11).
2.  **Gated Execution on Database Migration (Item 12):**
    *   Create clean, safe `migrations/0070_add_certificate_performance_indexes.sql` containing only verified table indexes.
    *   Verify with `go test ./migrations/...` before committing.
3.  **Prohibited Actions:**
    *   Do NOT delete raw audit test drills (always archive with timestamp).
    *   Do NOT execute raw SQL referencing uncreated tables (`webhook_deliveries`, `shortlinks`).
