# Installed Skills: Audit and Revision Register

## Scope and evidence

This is a read-only audit of the **54 installed skill packages** under `/home/ubuntu/skills`. The review used a static inventory, frontmatter and reference checks, a lexical overlap heuristic, and the current package validator. It did not execute bundled skill scripts, contact upstream repositories, install updates, or alter skill behavior.

The audit can confirm local structure and package-validator results. It cannot prove that an installed skill is current upstream, semantically correct for every host, safe to execute, or compatible with an unrecorded external source. No per-skill upstream remote, local Git checkout, manifest, or provenance file is available, so an actual version-update check is currently **not reproducible**.

| Evidence item | Confirmed result | Limitation |
|---|---|---|
| Installed package inventory | 54 `SKILL.md` packages were present. | Does not establish ownership or upstream source. |
| Required metadata scan | Every package has a frontmatter block, `name`, and `description`; no duplicate names or missing internal skill-path references were detected. | Does not assess whether descriptions are precise enough for every trigger. |
| Standard structural validator | 53 packages pass. `planning-with-files` fails because its `hooks` and `user-invocable` keys are not accepted by the validator’s current schema. | The failure does not by itself prove that either the skill or its host-specific hook model is wrong. |
| Placeholder scan | Four packages contain the token `TODO`, but all observed uses are template or example comments, not unresolved authoring placeholders. | An automated search cannot establish semantic completeness. |
| Length and lexical overlap scan | Five onboarding/reference packages exceed 500 lines; 24 lexical-overlap pairs were identified. | Size and word overlap are maintenance signals, not proof of duplication or a defect. |

## Revision register

| ID | Area | Verified finding | Classification | Proposed handling | Owner or source decision needed |
|---|---|---|---|---|---|
| SR-01 | `planning-with-files` frontmatter | The installed validator rejects `hooks` and `user-invocable`, although the skill declares them. | **Confirmed compatibility mismatch** | Do not remove these keys blindly. First determine whether this is a host-managed/Claude-specific skill or whether the validator schema is outdated for the intended target. | **Yes** — removing hooks could silently disable its core behavior. |
| SR-02 | Upstream updates | No package has local upstream provenance or a repository checkout. | **Confirmed evidence gap** | Add a non-secret provenance record only after owner identifies the supported source and allowed update policy for each externally maintained skill. | **Yes** — no safe upstream update route exists today. |
| SR-03 | Long reference skills | `excel-generator`, `webdev-readme-fullstack`, `webdev-readme-mobile`, `webdev-readme-mobile-backend`, and `webdev-readme-static` exceed 500 lines. | **Maintenance risk; not a defect** | Consider progressive disclosure only when real context-window pressure or a concrete retrieval failure is observed. Do not rewrite broad reference manuals merely for length. | No for deferral; **yes** before a structural rewrite. |
| SR-04 | `TODO` matches | The observed matches describe templates/examples, not unfinished operating instructions. | **False positive; no change** | Retain current wording. Improve the audit rule rather than altering the skills. | No. |
| SR-05 | Apparent overlaps | The lexical heuristic flags related workflows, particularly WebDev integration skills and adjacent INTEGIN quality workflows. | **Candidate only; no duplicate proven** | Preserve separate skills. Their distinct triggers and evidence boundaries are intentional. | No. |
| SR-06 | INTEGIN review family | `integin-code-review`, `integin-review-governance`, and `integin-evidence-guard` are distinct: fixed-point review, owner-facing governance/debate, and claim verification respectively. | **Confirmed separation** | No merge or rewrite needed. Keep the distinctions explicit in future revisions. | No. |
| SR-07 | Newly created internal skills | `integin-review-governance` and `integin-evidence-guard` pass the current validator and their direct clean-room scans found no external repository, forge-adapter, or framework-specific references. | **Verified current state** | Keep unchanged until real usage reveals a gap. | No. |
| SR-08 | Broad content correctness | Static checks do not prove technical accuracy of platform-specific commands, APIs, dependency versions, or external-service instructions across all 54 skills. | **Unproven** | Run targeted primary-source currency checks only for skills that will be used in an upcoming task or for owner-identified high-risk skills. | **Yes** for a full external currency program. |

## Recommended controlled sequence

Begin by resolving **SR-01** and **SR-02**, because they determine what “update” can safely mean. Then retain the installed packages unchanged, correct the audit tool’s false-positive logic for template `TODO` markers, and schedule targeted currency checks only when a specific skill is about to be invoked. This avoids mass edits to platform or host-specific skills without a source of truth.

> No installed skill has been edited by this audit. No external update has been installed, and no bundled script, hook, or runtime automation has been executed.

## Approval gate

Before any skill changes, the owner must choose one of the following revision scopes:

| Scope | Permitted action | Safety effect |
|---|---|---|
| A. Audit tooling only | Improve the local static-audit script and reports; do not edit skills. | Lowest risk; does not change skill behavior. |
| B. Internal skills only | Revise the INTEGIN-owned skills and the two new review/guard skills using actual usage evidence. | Bounded; avoids host/platform-managed skills. |
| C. Full installed collection | Revise every package after identifying ownership, provenance, host compatibility, and upstream version evidence. | Requires a separate source and compatibility plan; do not begin without explicit approval. |
