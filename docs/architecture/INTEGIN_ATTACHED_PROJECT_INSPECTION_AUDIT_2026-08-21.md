# INTEGIN Attached Project-Inspection Conversation Audit

## Scope and evidence discipline

The attached `project-inspection.json` is a historical conversation record from a separate project-inspection session. It contains 18 user-authored text messages and a prior agent’s source-review conclusions. It is useful as an engineering review input, but it is not current source of truth, runtime evidence, or authorization. The current source and controlled evidence records govern whenever they differ.

## Extracted user intent

The user asked the prior agent to inspect the project deeply, provide engineering-quality findings, export those findings, and improve the workspace’s AI collaboration protocol against omission and context-loss risk. The user also supplied a generic Go/PostgreSQL scaling blueprint for comparison. The attachment does not add a new product workflow, certificate, disclosure, commercial, or external-authority instruction.

## Comparison with current project direction

| Attached-conversation subject | Current assessment | Roadmap consequence |
|---|---|---|
| Keep Go and PostgreSQL; add specialized infrastructure only when justified. | Aligned. The current pilot is Go + PostgreSQL + RustFS/S3 adapter, with bounded operational HTTP behavior and no justified rewrite. | Do not introduce Redis, brokers, search, warehouses, or a framework rewrite without a measured use case. |
| Layered boundaries, context propagation, connection discipline, structured logs, health checks, tests, and CI. | Mostly aligned. Domain/repository/HTTP boundaries, request correlation/logging, health/readiness, and local test/vet gates are present. | Hosted CI, explicit performance/load evidence, and deployment/production configuration remain future operational work rather than certificate prerequisites. |
| Historical sync defects: held receipt replay, non-atomic held persistence, unknown signature-algorithm fall-through, and unauthenticated receipt lookup. | **Current-source recheck confirms these four code paths remain present** in `internal/domain/sync/sync.go` lines 148–167, 175–198, and 240–252. They are not resolved by the newer Work-Order/evidence-retention gates. | Create and prioritize a separate offline-sync corrective slice before relying on held-transaction recovery for production Field continuity. No current Stage A evidence proof should be read as fixing it. |
| Historical Work-Order array-escaping concern. | The current helper at `internal/workorderpg/postgres.go` lines 583–588 still manually constructs PostgreSQL array literals and only replaces quotation marks, so the precise edge-case behavior remains unproven. | Add a focused round-trip/escaping test or replace the custom helper before a broad high-volume special-character import path. |
| Existing whole-product workflow ideas: bulk work order, clone controls, conditional evidence, reassignment, verifier identity, office commercial separation, ZATCA/external readiness. | Aligned and already represented in the 2026-08-19 operating-model documents and the 2026-08-21 higher-level review. | No duplicate product stream is created. Retain the current certificate-first roadmap and separate evidence-export/commercial/external-adapter boundaries. |
| “Binding Key Registry before certificate designer” recommendation. | Current source search found no `binding_key`/`BindingKey` registry implementation. The requirement matches the owner’s prior requirement for data-bound certificate cells and server-driven template updates. | Add a **certificate-template binding registry** as the first sub-foundation inside the future certificate authority lifecycle; do not begin free-form designer rendering without stable, versioned binding keys and applicability rules. |
| AI collaboration residual-risk protocol. | The attachment shows this was written into the root engineering continuation guide. The current work also uses persistent task, findings, progress, and TODO records. | Maintain the protocol; do not claim it eliminates model error. New authority work must retain mechanical tests, explicit evidence, and owner decision points. |

## Important reconciliation

The attachment predates the completed Stage 0 Work-Order, Stage A recovery, evidence-retention, runtime-operations, and authenticated evidence-registration work. Therefore, its statement that `0010` was merely rehearsed is historical; the current controlled records show that `0010` and additive `0011` were subsequently reviewed, separately backed up, applied to the pilot, and proved through runtime evidence. Conversely, its older defect observations must not be treated as fixed merely because unrelated later milestones passed.

## Proposed priority adjustment

The certificate roadmap remains correct, but its first bounded slice should be: **certificate-template binding registry and eligibility contract**, followed by certificate approval/issue/revocation/supersession and then the public QR projection. Separately, the confirmed sync replay/held-state issues should be planned as an offline-durability corrective gate before production Field rollout. These are distinct workstreams: neither should be hidden inside certificate work.

> No source code, migration, runtime, certificate, export, commercial record, or external adapter was changed by this audit.
