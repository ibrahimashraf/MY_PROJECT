# INTEGIN Current Discussion — Idea Coverage Audit

**Audit date:** 2026-08-19  
**Scope:** Product/workflow ideas discussed during the current design conversation.  
**Boundary:** This audit records future product direction only. It changes no source, database, migration, runtime, certificate, external integration, customer data, package enforcement, OIDC, OpenBao, or manifest-gate behavior.

## Audit conclusion

Most user-directed ideas were already represented in the high-volume work-order model, final integrated operating-model reconciliation, verification-portal reconciliation, competitor assessment, and roadmap todo list. The audit found four improvements that needed explicit reconciliation: location-first expandable work-order presentation; fully editable cloning that may carry defect values but never proof; per-inspector closeout/combined work summary/commercial release; and bounded support, ticketing, feedback, and advisory-AI history. It also confirmed that the latest verification protection—full asset identity, full description, and exact completed-test scope—belongs in the integrated model as well as the verification-specific record.

## Coverage matrix

| Discussed idea | Coverage before audit | Audit outcome |
| --- | --- | --- |
| Client/internal request → authorized work order → inspector assignment | Captured | Retained as accepted future model. |
| Multiple orders, partial/selected submission, work across days | Captured | Retained as accepted future model. |
| High-volume native table, scan/select/import staging, exception queue | Captured | Retained as accepted future model. |
| Location-first expandable work-order sections and asset-row form navigation | Partial | Added explicitly as future operating experience. |
| Type-driven forms for equipment/person/site/document items | Partial | Added as a controlled type/template selection rule. |
| Clone across locations with all editable draft data | Partial and internally inconsistent | Repaired: cloned values, including findings/defects, may carry as visible editable draft values; evidence, signatures, receipts, approvals, and certificate state never carry as current proof. |
| Conditional photos and rejected inspection without photo where template permits | Captured | Retained as accepted template-governed policy. |
| Shared order: each inspector closes assigned scope, individual summary, combined office report, then commercial release | Partial | Added explicitly; commercial release remains separate from technical authority. |
| Completion/timesheet reports, certificates/reports, training competence | Captured | Retained and clarified. |
| Renewal dashboard, reminders, proposals, add selected items to existing/new order | Captured | Retained as future planning workflow. |
| Lawful SASO/SAAC/platform adapters and separate ZATCA commercial integration | Captured | Retained with no-bypass/no-false-claim boundary. |
| Public certificate verification portal | Captured | Retained as future staged read-only trust layer. |
| Full asset ID, serial, description, and exact completed inspection/test scope in verifier | Captured in verification record only | Added as an integrated-model requirement; distinguishes load test performed/not performed/not applicable/required but incomplete. |
| Customer support, formal tickets, feedback, manager quality view, advisory-only AI history | Not yet explicit in core operating model | Added as a future bounded support and advisory workspace stream. |
| Scope-change control and corrective-action/reinspection workflow | Proposed by assistant; not explicitly approved as a user requirement | Recorded as a candidate design decision requiring owner confirmation before implementation planning. |
| Public competitor/positioning assessment | Captured | Retained as evidence-led research with public-claim limitations. |

## Recorded versus implementation status

All entries above are **future design or tracked future work**. They must not be represented as released INTEGIN features. The active source-only manifest receipt gate remains the current implementation priority.
