# LMT.PRO Railway Deployment — Public Read-Only Assessment

## Scope

This assessment covers only what was visibly rendered by the public deployment at `https://lmt-pro-production.up.railway.app/`. No credentials, form submissions, calculations, downloads, ratings, payments, source code, Railway controls, or deployment settings were accessed or changed.

The application identifies itself as **Lifting Tool / LMT.PRO Lifting Commandor**, a preliminary crane-selection and ground-load calculator. Its public text states that results are indicative and require qualified Appointed Person verification before a lift. That is a product claim, not independent verification of calculation correctness or safety.

## Visible capabilities

| Visible capability | What was observed | Confidence |
|---|---|---|
| Crane selection | Inputs include working radius, load weight, longest load dimension, maximum utilization, drive type, obstacle state, and lift/slings dimensions. The results show matching crane configurations, capacities, utilization, boom/jib, and hook-height values. | High that this is publicly rendered; calculation correctness not assessed. |
| Crane load-chart library | Public debug output lists 10 JSON library files, crane names, numbers of mobile/crawler cranes, and load-point counts. One listed mobile crane was flagged as lacking specs. | High public exposure; data quality/completeness unverified. |
| Structural temporary-works modules | Navigation exposes crane platforms, haul roads, protection slabs, scaffolding, falsework, and formwork. The protection-slab page presents a preliminary design aid, calculations, a design brief, risk assessment, inspection checklist, RAMS, and report path. | High public presence; engineering correctness and standards currency unverified. |
| Export/report path | Public UI advertises CSV export and describes report/PDF capabilities, with some features presented as paid stages. | Moderate; no export was generated or downloaded. |
| Public ratings | The interface visibly displays ratings/review counts on several pages. | High visibility; provenance and authenticity not assessed. |

## Relevant observations

| Observation | Why it matters | Recommended handling |
|---|---|---|
| Debug/library information is visible to every public visitor | Reveals runtime path conventions, asset file names, library inventory, load-point counts, and a missing-spec condition. This is not automatically a vulnerability, but it is unnecessary operational disclosure on a production calculator. | Move detailed library diagnostics behind staff authorization or disable them in production. Retain a concise safe error state for users. |
| A listed crane has “no specs” | Selection logic must make incomplete data impossible to treat as an eligible configuration. | Add an explicit server/engine eligibility invariant and a negative test proving missing/incomplete charts cannot produce a candidate or export. |
| Engineering disclaimers are displayed | Disclaimers are helpful but do not replace traceability, calculation versioning, verified input capture, competent-person review, or controlled release. | Link each exported result to formula/library version, source provenance, validation status, and reviewer/signatory record. |
| The application publicly claims no personal data is stored | This cannot be confirmed from the public UI. | Treat it as an unverified product statement until a data-flow, log, cookie, export, and hosting review is performed. |
| Calculator is a separate decision-support domain | Crane selection and ground-bearing calculation are safety-critical engineering aids, not ordinary inspection data entry. | Keep any future connection to INTEGIN as a separately governed reference/attachment workflow, not as direct automatic lift authorization. |

## Relevance to INTEGIN

The useful design lessons are **not** the calculator logic or presentation. INTEGIN could later support a work-order link to a qualified lifting-planning artifact, with the asset/work order, document version, engineer/AP review, evidence, restrictions, and final handover recorded under INTEGIN authority.

INTEGIN should not import, embed, call, copy, or rely on this public calculator for safety-critical decisions. A future integration would require a separate technical, legal, safety, data-governance, tenant-isolation, API, and validation assessment; a controlled test seam; explicit ownership; and review by an appropriately qualified lifting/temporary-works authority.

## Immediate conclusion

The site is a credible **product-pattern reference** for presenting a preliminary lifting decision-support workflow: constrained inputs, match results, assumptions, safety disclaimer, review-oriented documents, and tiered report outputs. It is not evidence that the underlying calculation library, formulas, reporting, security, or engineering controls are suitable for INTEGIN.

No INTEGIN runtime or source change is recommended from this inspection. The immediate current design sequence remains the approved work-order foundation, versioned form/evidence contract, and scoped field package; lifting-planning integration is a later, separately specified domain extension.

## Reference

1. [LMT.PRO Railway deployment](https://lmt-pro-production.up.railway.app/), publicly rendered 2026-08-27.
