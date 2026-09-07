# Full Installed-Skill Revision Proposal

## Decision requested

The requested Scope C review is complete through local structure, provenance, and eligible upstream-currency assessment. The evidence does **not** justify a mass content update. The recommended action is to preserve all installed skill package contents and close the current revision pass with documented deferrals.

| Disposition | Packages | Reason |
|---|---:|---|
| Preserve unchanged | 53 | The standard package validator passes. No verified local defect, authoritative update route, and compatible target-host contract jointly support a revision. |
| Preserve unchanged; host compatibility decision required | 1 (`planning-with-files`) | Upstream `v3.11.2` is newer than installed `3.10.0`, but its canonical subtree differs only in `SKILL.md`, and its frontmatter/hook model is incompatible with the local validator. A direct update is not safe. |
| No external update route available | 51 | No local upstream provenance or source-of-record metadata exists. A broad version check cannot be reproduced. |
| Locally created clean-room packages | 2 | `integin-review-governance` and `integin-evidence-guard` were created and validated in this session; no revision need was observed. |

The counts overlap as a useful distinction: the 53 structural passes include the two newly created packages, while the 51 packages without external update routes exclude the one externally identified package and the two session-created packages.

## Concrete findings and proposed treatment

| ID | Affected package or scope | Evidence-led finding | Proposed action | Requires confirmation |
|---|---|---|---|---|
| RP-01 | All 54 packages | The collection has a reproducible 54-package checksum baseline and 53 current-validator passes. | Preserve package content; retain audit artifacts for future comparison. | No package change. |
| RP-02 | `planning-with-files` | The installed `3.10.0` package has one newer upstream tag (`v3.11.2`), but 28 of 29 canonical artifacts match and the sole delta is the instruction document. Local validator rejection of host-specific `hooks` and `user-invocable` remains unresolved. | Do not update, strip metadata, copy a document delta, or enable hooks. Start a host-specific compatibility prototype only under a separate owner-approved scope. | **Yes.** |
| RP-03 | `planning-with-files` | Two local relative documentation links point outside the installed package and do not resolve in the current environment. | Leave unchanged in this pass. Any repair must be source-owned and host-aware because the package is externally sourced and hook-dependent. | **Yes.** |
| RP-04 | `content-gap-analysis`, `internet-skill-finder`, `listicle-blog-writer`, `seo-competitor-analysis-will` | Eight apparent missing local links are output placeholders or expected generated deliverables, not broken static references. | No change. Improve audit interpretation, not skills. | No. |
| RP-05 | Platform-associated packages | No official source-of-record or update channel is stored locally. | Preserve unchanged. Request or discover official provenance only when a specific package is needed for a future task. | **Yes** for any external update. |
| RP-06 | Unattributed general and existing INTEGIN packages | No external source ownership can be inferred safely from package text or file names. | Preserve unchanged. Use clean-room revisions only against a new owner-approved requirement and evidence record. | **Yes** for a semantic change. |
| RP-07 | `integin-review-governance`, `integin-evidence-guard` | Both packages pass validation and have no direct external-source content or known usage failure. | Preserve unchanged; revise only after real usage demonstrates a gap. | No. |

## Not approved by this proposal

This proposal does not authorize package-manager actions, repository cloning, source import, external script execution, hook activation, credential use, runtime changes, removal of frontmatter keys, or any claim that an external source is safe or compatible.

## Recommended closure

Approve **preserve-all closure** for this revision pass. The deliverable will include the checksum baseline, structural inventory, full ownership/provenance registry, clean-room planning-with-files audit, and an unresolved-update register. Re-open a specific package only when an official source and target-host contract are available, or when a concrete usage failure is observed.

> No installed skill has been changed during Scope C.
