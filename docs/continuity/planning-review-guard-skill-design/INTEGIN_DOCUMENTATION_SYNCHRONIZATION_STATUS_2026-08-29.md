# INTEGIN Documentation and Synchronization Status

## Overall status

The records are **documented but not fully synchronized**. The core post-manifest work-order, field-assignment, form, bridge, launcher, and activation-test records are present in the Windows workspace. Several later session decisions remain in the sandbox or local Manus skill collection only, and the root handoff documents have not been updated to reference them. The situation is correctable documentation drift, not a runtime escalation.

No protected runtime, database, route, migration, package-enforcement, OIDC, OpenBao, private-material, deployment, or tracker state was changed during this verification.

## Reconciliation matrix

| Area | Current evidence | Status | Gap | Safe correction |
|---|---|---|---|---|
| Post-manifest work-order foundation | `docs\architecture\INTEGIN_POST_MANIFEST_WORK_ORDER_FOUNDATION_SPEC_2026-08-22.md`, Stage 0 records, task plan, findings, and progress | Synchronized at the core-design level | Root handoff timestamps predate later session work and do not link to the new external-capability assessments | Add a dated handoff addendum after owner approval; do not rewrite historical records. |
| Field assignment and scoped asset sync | `INTEGIN_FIELD_ASSIGNMENT_SCOPED_ASSET_SYNC_2026-08-20.md` and related contracts | Synchronized for the existing design | The refined dependency sequence is not linked from the authoritative work-order records | Add a cross-reference to the approved field-package/form design only after review. |
| Metadata-driven forms | Existing form designer and metadata-driven delivery records | Synchronized for the earlier form direction | The refined contract adds explicit evidence binding, compatibility, cache invalidation, and conditional validation emphasis | Reconcile into a versioned “Work-Order Field Package and Form Contract” design record. |
| PC planning bridge and launcher | Four operations records are present: bridge deployment, root binding, offline safety evidence, and activation test | Documented and synchronized in operations | The activation test remains partial because both documented read-only task-status endpoints returned HTTP 404 | Preserve disabled state and add the API-status limitation to the next handoff update. |
| Controlled launcher configuration | Root-bound `.planning\automation.json` and launcher configuration exist with `enabled: false`; project allow-list was verified | Synchronized operationally | The root handoff does not yet summarize that the launcher is disabled and non-native to ordinary chat lifecycle | Add a concise status cross-reference; do not enable again until status monitoring is resolved. |
| Consolidated planning skill | `integin-planning-suite` exists in the local Manus skill collection; predecessor packages remain unchanged | Documented in the Manus local skill collection | It is not a Windows project artifact and is not listed in the INTEGIN root handoff; that separation is intentional | Record it as a tooling dependency or leave it as an external local skill; do not copy it into product source. |
| Mobtag public capability assessment | Sandbox assessment `MOBTAG_PUBLIC_CAPABILITY_ASSESSMENT_2026-08-25.md` | Documented only in sandbox | Not linked from Windows architecture records | If adopted as design inspiration, create an INTEGIN-owned design note with no external code or claims. |
| Refined capability sequence | Sandbox `INTEGIN_REFINED_CAPABILITY_SEQUENCE_2026-08-25.md` | Documented only in sandbox | Not synchronized into the Windows roadmap/handoff | Owner-review the sequence, then add a new dated architecture decision record rather than silently editing old records. |
| QR/NFC/RFID decision | Session discussion and refined sequence | Documented in session/sandbox only | No authoritative Windows architecture record currently captures QR + NFC first and UHF RFID later | Add it to the field-package/tag-identity design after requirements alignment. |
| Streamlit assessment | Session response and deployment review | Documented in session only | No project record says Streamlit is optional internal prototype tooling and not core product stack | Add only if the team wants to retain Streamlit as an explicit tooling option. |
| Railway LMT.PRO review | Sandbox `LMT_PRO_RAILWAY_PUBLIC_DEPLOYMENT_ASSESSMENT_2026-08-27.md` | Documented only in sandbox | No Windows handoff link; the deployment is an external product reference, not an INTEGIN component | Keep as a research note; do not integrate or treat its calculations as authoritative. |
| Core technology direction | Existing architecture/findings records identify Go, PostgreSQL, Flutter, and object storage direction; OIDC/OpenBao remain disabled/sealed | Mostly synchronized | The recent answer summarized the stack but did not create a new authoritative handoff record | Reconcile only through the normal architecture decision process. |

## What is synchronized now

The following are present and mutually consistent at the level checked: the approved post-manifest work-order foundation; scoped field assignment and offline package direction; metadata-driven form direction; the disabled PC bridge and launcher deployment records; root binding and test evidence; and the current protected-runtime exclusions.

## What is documented but not synchronized into the root handoff

The Mobtag capability assessment, the Railway/LMT.PRO public deployment assessment, the refined capability sequence, and the QR/NFC/RFID refinement are in sandbox/session records. The consolidated planning skill is in the Manus local skill collection, not the INTEGIN repository. These are not missing, but they are not yet authoritative INTEGIN architecture records.

The controlled launcher activation test is documented in the Windows operations area, but its post-stop completion-review path is unresolved. Therefore “automatic context preflight” is proven only for launcher-created tasks, while “automatic completion gate” remains unverified and must not be described as active.

## Correction sequence

1. Keep all current historical records unchanged and retain their dates.
2. Add one owner-approved dated handoff addendum summarizing the launcher’s disabled state, partial activation result, and HTTP 404 status-monitoring limitation.
3. If the capability direction is accepted, create one new INTEGIN-owned architecture record for the Work-Order Field Package and Form Contract, including the refined dependency order.
4. Add QR + NFC tag identity to that design only after entitlement, tenant/RLS, offline, revocation, and audit decisions are explicit.
5. Record UHF RFID, maps, MTC traceability, and Streamlit as deferred options with separate gates rather than active commitments.
6. Do not copy external Mobtag or LMT.PRO code, data, prompts, calculations, or product claims into INTEGIN.

## Conclusion

**Status: correctable documentation drift.** The main product foundation is documented and internally coherent. The later assessments and design refinements are not yet synchronized into the authoritative Windows handoff, and the launcher’s completion-review capability is not fully verified. No implementation or runtime action is authorized by this status check.
