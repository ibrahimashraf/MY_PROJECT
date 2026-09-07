# Planning-With-Files Hybrid Revision Review Record

## Decision frame

The owner authorized a revision of the active `planning-with-files` skill to recover useful planning convenience without restoring incompatible external automation. The comparison point is the current one-file, hook-free active package. The revision may add independently written templates and explicitly invoked local validation, but must not add external source text, host hooks, automatic context injection, completion blocking, scheduler behavior, or hidden mutation.

The previously separate `integin-persistent-planning` package was retired at the owner’s direction. Its invoked principles are therefore treated as internal design requirements for the active skill, not as a second active skill.

## Evidence ledger

| ID | Source or test | What it supports | Limitation |
|---|---|---|---|
| E-01 | Owner’s current request | A hybrid revision under the active `planning-with-files` name is authorized. | Does not authorize external-source copying or hook activation. |
| E-02 | Active package validation and package inventory | The active package is hook-free, one-file only, and validator-compliant. | Does not prove the workflow is sufficient for every project. |
| E-03 | Preserved-package comparison | The original package included reusable assets and explicit helper behavior, but its host automation is incompatible with the current target. | Original source remains evidence only. |
| E-04 | Existing local planning workflow | The active workflow already defines durable records, root disambiguation, write coordination, and a continuity checkpoint. | It currently exposes record structures only inline. |

## Findings and dispositions

| ID | Lens | Finding | Evidence | Confidence | Disposition | Required next evidence or owner decision |
|---|---|---|---|---|---|---|
| F-01 | User workflow | Reusable, separately addressable templates would reduce repeated manual reconstruction for new planning records. | E-01, E-04 | High | Confirmed; add clean-room templates. | Template validation and no-overwrite guidance. |
| F-02 | Operability | A user-invoked record check can identify missing required planning records or headings without relying on host hooks. | E-01, E-03 | High | Confirmed; add a read-only local helper. | Deterministic positive and negative test evidence. |
| F-03 | Safety | A helper must not create, overwrite, invoke network services, run shell commands, or block task completion. | E-01 to E-03 | High | Confirmed; enforce as a non-goal. | Static scan and behavioral tests. |
| F-04 | Skill ownership | The retired successor must not be restored as a duplicate active trigger. | Owner decision and active skill inventory | High | Confirmed; transfer only its local workflow principles. | Final inventory verifies no successor package. |

## Recommendation and stop conditions

Create three clean-room Markdown templates and one explicit, read-only Python validator under the active `planning-with-files` package. Update the active skill to describe only manual template use and deliberate validator invocation.

Stop and re-review if a proposed addition needs hooks, automatic file injection, background execution, external dependencies, hidden writes, completion gates, or external source text.
