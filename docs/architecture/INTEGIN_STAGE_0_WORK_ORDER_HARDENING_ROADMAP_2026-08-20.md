# INTEGIN Work-Order Stage 0 Hardening Roadmap

**Review decision:** The prior roadmap is revised. The repository adapter already contains create, assignment, transition, partial submission, reassignment, provisional reconciliation, and certificate-validation methods. The immediate gap is not method presence; it is proof depth and a few authority-boundary risks.

## Independent review lenses

| Lens | Finding | Decision |
|---|---|---|
| Architecture | Transaction shape and revision predicates are present; migration remains additive and isolated. | Continue repository-level hardening before new domains. |
| Security and data isolation | Reads and writes carry tenant and organization predicates; integration proof must cover mutation denial, not only reads. | Add negative cross-organization mutation tests. |
| Assurance and testing | Existing proof covers create idempotency and read isolation only. | Add assignment, transition, stale revision, and partial-submission evidence. |
| Field/offline workflow | Idempotency and revision are the recovery boundary; partial submission must not imply order closure. | Test replay, stale retry, and partial segment semantics. |
| Operations | No runtime authority or external integration should be changed in this slice. | Keep changes local, reversible, and migration-neutral. |

## Revised logical work plan

1. Prove the existing assignment and transition transactions with valid and stale-revision cases.
2. Prove partial submission creates an immutable segment and advances the order exactly once without certificate or commercial release.
3. Prove cross-organization mutation denial for assignment, transition, and partial submission.
4. Re-run the full Go suite and controlled PostgreSQL integration suite.
5. Review and harden provisional reconciliation separately because accepting a device-supplied canonical ID is an authority-boundary concern requiring a canonical-record lookup/create policy, not a blind local patch.
6. Perform the migration rollback/restore drill only after repository evidence is complete and the migration path is explicitly authorized.

## Explicit non-goals for this slice

No Flutter runtime work, certificate issuance, external SASO/SAAC/ZATCA integration, production deployment, destructive migration, or autonomous AI behavior is included.

## Current proof status

| Claim | Status |
|---|---|
| Create idempotency and payload mismatch rejection | Proven by controlled integration test |
| Cross-organization read denial and receipt isolation | Proven by controlled integration test |
| Assignment, transition, partial submission transaction behavior | Proven by controlled integration test; assignment-state, duplicate-inclusion, and fail-closed membership-boundary safeguards fixed |
| Provisional reconciliation authority safety | Design concern identified; not claimed complete |
| Flutter matrix fixture runtime gate | Blocked by unavailable workstation Flutter toolchain |
