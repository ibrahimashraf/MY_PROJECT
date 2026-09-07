# Planning Skill Reconciliation Change Ledger

## Baselines

| Artifact | Pre-reconciliation SHA-256 | Preservation rule |
|---|---|---|
| `integin-persistent-planning/SKILL.md` | `30fbe03d06c647b09dd536f3392444a1ac81f700f51da6a83ca7228d902a1f2c` | May change only through independently written clean-room revisions recorded below. |
| `planning-with-files/SKILL.md` | `e489570affc62409010bc22988150e9dffa20d91e0eaa1899b380e110ea6d6b4` | Must remain unchanged. |

## Source boundary

The external comparison point is the bounded clean-room audit of the canonical `planning-with-files` v3.11.2 skill subtree. The external material is evidence only. No source text, scripts, hooks, templates, or installation behavior may be copied, executed, or activated.

## Change dispositions

| ID | Verified upstream change category or observed capability | Internal clean-room disposition | Status | Evidence required |
|---|---|---|---|---|
| CR-01 | Newer external release exists; the canonical subtree differs locally only in its instruction document. | Track as provenance only. Do not duplicate external version numbering or source history in the local replacement. | Excluded | Rollback archive preserves the original package; the active package is now a local implementation. |
| CR-02 | The external package declares host-specific lifecycle hooks and user-invocable metadata. | Exclude. The internal successor remains manual and hook-free. | Excluded | Static scan confirms no hook metadata or activation directive. |
| CR-03 | The external package relies on host-specific context and stop-control automation. | Exclude. Add no automatic injection, completion blocking, background loop, or scheduler. | Excluded | Internal instructions state manual operation and no automation. |
| CR-04 | External planning behavior addresses ambiguity when locating durable planning state. | Add a clean-room manual rule: enumerate candidate planning roots, use an explicit owner-approved root, and stop on ambiguity rather than choosing a newest file. | Accepted and validated | Local validator passed; static check confirmed the rule is present. |
| CR-05 | External planning behavior addresses concurrent modifications to durable plan state. | Add a clean-room manual re-read-before-write and reconciliation rule for planned records. | Accepted and validated | Local validator passed; static check confirmed the rule is present. |
| CR-06 | External planning behavior emphasizes structured state continuity. | Add a clean-room checkpoint summary requirement that states current phase, last validated action, unresolved decision, and next safe action. | Accepted and validated | Local validator passed; static check confirmed the required summary is present. |
| CR-07 | Existing internal successor already distinguishes plan, evidence, and progress records. | Preserve and strengthen; do not replace the local record model. | Preserved | Diff review confirms the separation remains explicit. |
| CR-08 | Owner selected an existing-name replacement rather than a separate successor. | Replace the active `planning-with-files` body with the independently written local workflow; remove residual external resources from the active package. | Accepted and validated | Active package validates; rollback archive is readable; successor is retired from active use. |
| CR-09 | Active package resource treatment | Remove the residual external scripts, templates, references, and example artifacts; retain only the local `SKILL.md` as the active package. | Accepted and validated | Active package has one file only; rollback archive retains the full pre-replacement package. |
| CR-10 | Separate successor treatment | Retire `integin-persistent-planning` from active use to prevent duplicate triggers and ambiguous ownership. | Accepted and validated | Successor path is absent; rollback archive retains its pre-retirement state. |
| CR-11 | Owner-approved hybrid revision | Add independently written templates and one explicitly invoked, read-only local planning-record validator to the active package. | Accepted and validated | Templates and helper passed deterministic tests; static boundary checks found no hooks or hidden automation. |

## Tracking rules

Every future planning-skill change must add a row with the reason, clean-room origin, validation, and final disposition. A failed check must remain visible; do not delete or rewrite the prior row to conceal it.

> A disposition of “accepted” means an independently written internal behavior was validated against this host. It does not mean external source code or prompts were adopted.
