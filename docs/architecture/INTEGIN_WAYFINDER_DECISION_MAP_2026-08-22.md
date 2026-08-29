# INTEGIN Wayfinder Decision Map — 2026-08-22

> Decision map only: no implementation authority.

## Destination

Reach a reviewable engineering baseline: protected working-tree history, an explicit legacy evidence-ingress posture, and an owner-approved multilingual rendering direction that preserves INTEGIN authority boundaries.

## Standing Constraints and Owners

| Class | Owner |
|---|---|
| Tenant/org isolation, server-derived authority, lifecycle truth, AI advisory-only | Non-negotiable engineering constraint |
| Public, external, credential, migration, production runtime, document delivery | Owner approval required |
| Repository baseline, endpoint posture, renderer/font selection | INTEGIN owner |
| Evidence, options, and a bounded implementation proposal | Engineering team |

## Reconciliation of Supplied Analysis

The supplied material is advisory only. `git status --short` currently verifies a large uncommitted worktree, including certificate, migration, operational, and documentation slices.

| Claim or concern | Reconciled status | Map consequence |
|---|---|---|
| Uncommitted work is the greatest recoverability risk | Verified current-state concern | Establish an owner-approved reviewed baseline before further cross-cutting changes. |
| Legacy `/evidence` accepts caller-supplied tenant/org fields | Current-source concern; legacy handler exists separately from authenticated metadata registration | Owner must choose retire, gate, or explicitly retain a bounded device-flow contract. |
| Go renderer is disconnected and not Arabic-capable | Verified for the frozen Go prototype; a separate isolated WeasyPrint candidate has only partial synthetic evidence | Do not wire or replace it. Complete the owner-gated renderer decision first. |
| Process-local verifier limiter is insufficient for public release | Verified known limitation | Keep public exposure blocked pending deployment-grade shared controls. |
| Sync authentication ordering, held-to-applied atomicity, and certificate lifecycle basics | Supplied report is stale on several points; current durable records state those paths were corrected and regression-tested | Do not reopen without a reproduced failing case. |
| 0013/0014 migration gaps and candidate/applied status | Conflicting or insufficiently current evidence | Treat as a separate schema-audit decision before further migration work; do not infer an unapplied or unsafe state. |
| Go renderer byte determinism | Not material to the advanced candidate decision; the synthetic WeasyPrint smoke had a repeat-byte check | Reassess only if the frozen Go prototype is proposed again. |
| Broad work-order vocabulary and assignment-transfer concerns | Plausible historical candidates, but not confirmed against current source in this mapping pass | Keep outside the first frontier until a focused reproduction and owner priority decision. |

No claim in the supplied material authorizes a migration, runtime activation, public exposure, external operation, or commit.

## Decision Records

| ID | Precise question | Type | Owner | Evidence required |
|---|---|---|---|---|
| D-01 | What owner-approved preservation and review unit should protect the current uncommitted worktree before any new cross-cutting implementation? | Prerequisite task | Owner | Reviewed change inventory, focused test evidence where source changes exist, and an approved checkpoint/commit boundary. |
| D-02 | Should the legacy caller-scoped `/evidence` ingress be retired, placed behind derived device/OIDC authority, or retained under a documented bounded device-flow contract? | Policy and design | Owner | Current-route contract, threat model, compatibility analysis, and migration/deprecation proposal. |
| D-03 | Does the owner approve the advanced multilingual renderer direction, including a versioned licensed font pack and restricted declarative template model? | Architecture and product | Owner | Qualification corpus/evidence, representative redacted layouts, font-license review, output requirements, and final stack proposal. |
| D-04 | What is the sealed renderer-job and private artifact lifecycle after D-03? | Design and prototype | Engineering, then owner approval | Job schema, provenance, resource limits, deployment egress policy, storage metadata, audit, retrieval policy, and failure matrix. |
| D-05 | What current schema facts and constraints must be independently re-audited before any new certificate or work-order migration is proposed? | Research prerequisite | Engineering | Exact applied-versus-candidate inventory, RLS/FK/CHECK review, migration backups, and a no-change audit record. |
| D-06 | What deployment-grade rate/abuse and document-delivery controls are required before any verifier or PDF becomes public? | Architecture prerequisite | Engineering, then owner approval | Shared-control design, proxy model, abuse tests, no-store behavior, private object policy, and operational ownership. |
| D-07 | Which unresolved work-order state-vocabulary, assignment-transfer, and held-drain concerns deserve a focused reproduction after the above foundation decisions? | Not yet specified | Owner priority, then engineering | Reproducible current-source cases and a bounded impact assessment. |

## Dependency Graph and Initial Frontier

| Decision | Depends on | Blocks |
|---|---|---|
| D-01 Baseline preservation | None | New cross-cutting implementation and any commit/checkpoint action |
| D-02 Legacy evidence ingress posture | D-01 for implementation only | Any alteration or retirement of legacy evidence ingress |
| D-03 Renderer direction | D-01 for implementation only | D-04 and all document-artifact work |
| D-04 Renderer-job/artifact lifecycle | D-03 | Any renderer integration, storage metadata, or retrieval |
| D-05 Schema audit | D-01 for any change | New certificate/work-order migration proposals |
| D-06 Public controls | D-04 where PDF delivery is involved | Public PDF or broad public-verifier release |
| D-07 Work-order candidates | Owner prioritization | No current committed decision |

The initial frontier is **D-01**: preserve and review the current worktree. It is the only prerequisite that protects all later work from loss or ambiguous provenance. In parallel, the owner may make the policy choices in **D-02** and **D-03**, but neither becomes implementation work until D-01 is resolved.

## Out of Scope for This Map

This map does not start migrations, enable OIDC or OpenBao, change the legacy endpoint, choose a renderer, wire an artifact flow, commit changes, expose a verifier/PDF publicly, contact external authorities, or process customer content.

## Next Handoff

Resolve D-01 only: select and approve the reviewed preservation boundary. After it is complete, return to this map and claim D-02, D-03, or D-05 according to owner priority. Each is a separate bounded session and requires its stated evidence.
