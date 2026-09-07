---
name: integin-review-governance
description: Run a bounded, evidence-governed engineering review or debate for a proposed change, design, implementation, incident finding, or release gate. Use when INTEGIN needs independent lenses, explicit owner decisions, and traceable final dispositions without delegating authority to an external model or automation.
---

# INTEGIN Review Governance

Use this skill to make an important engineering decision reviewable, evidence-linked, and closable. It is a governance workflow, not a code scanner or automatic approval mechanism.

## Apply this skill

Use this skill when a change is cross-cutting, security-sensitive, authority-sensitive, difficult to reverse, or likely to create disagreement about risk or sequencing. Typical triggers include a new design gate, a materially changed implementation plan, a migration proposal, a public-verifier or certificate-lifecycle change, an offline/replay boundary, a release readiness decision, and a significant defect whose root cause is uncertain.

Do not use this skill for a routine factual lookup, a small formatting change, or when a deterministic test failure has an obvious local repair. Do not use it to replace requirements alignment where owner intent is unknown.

## Preserve review boundaries

Treat the declared baseline, requirements, architecture records, test evidence, and explicitly authorized scope as the review inputs. Do not silently broaden the scope, infer missing policy, access secrets or private material, execute untrusted artifacts, create tickets, commit changes, publish, deploy, or alter runtime state unless a separate owner authorization explicitly permits it.

Never present model agreement as proof. Separate confirmed facts, supported inferences, assumptions, and unproven claims. A missing test or inaccessible environment is a limitation, not a passing result.

## Review workflow

1. **Frame the decision.** State the decision to be made, the comparison point, the owner-controlled policy questions, the protected boundaries, and the definition of a safe stop.
2. **Build the evidence ledger.** List each relevant source and record what it can prove. Mark unavailable or contradictory evidence explicitly.
3. **Choose independent lenses.** Select only the lenses that materially apply: requirement fit, tenant/organization authority, data lifecycle and audit, offline/replay/reconciliation, migration/rollback, performance/operability, public exposure, or user workflow. Do not manufacture disagreement merely to have a debate.
4. **Assess alternatives.** For each credible option, identify benefits, failure modes, dependencies, negative paths, and evidence needed before an approval claim would be valid.
5. **Record findings.** Give each finding a stable identifier and describe the affected decision, evidence, confidence, impact, recommended treatment, and whether owner confirmation is needed.
6. **Close every finding.** Resolve each finding as `confirmed`, `refuted`, `deferred`, `accepted risk`, `blocked`, or `owner decision required`. A deferred finding must name its trigger and required future evidence. Do not let a finding disappear because it is inconvenient.
7. **Issue a bounded result.** State the recommended next safe action, explicit stop conditions, and claims that remain unproven. Approval remains with the owner or designated human authority.

## Required review record

Use this minimum structure, adapting the technical detail to the decision:

```markdown
# [Decision] Review Record

## Decision frame
Decision, comparison point, authorized scope, protected boundaries, and owner policy questions.

## Evidence ledger
| ID | Source or test | What it supports | Limitation |
|---|---|---|---|

## Findings and dispositions
| ID | Lens | Finding | Evidence | Confidence | Disposition | Required next evidence or owner decision |
|---|---|---|---|---|---|---|

## Recommendation and stop conditions
State the next safe action, the actions not authorized, and what would require re-review.
```

## Quality rules

Require a review finding to identify the evidence it depends on or be clearly labelled as an assumption. Prefer the narrowest reversible action when a material policy issue, isolation condition, or lifecycle truth remains unresolved. Escalate instead of resolving silently when authority, tenant isolation, RLS behavior, public exposure, certificate issuance, or migration order is in dispute.

## Completion standard

The review is complete only when the decision frame is bounded, material evidence and limitations are recorded, every finding has a disposition, and the result identifies both the approved next action and the actions still prohibited.
