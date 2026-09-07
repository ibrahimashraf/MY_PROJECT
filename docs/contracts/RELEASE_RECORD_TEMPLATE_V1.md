# INTEGIN Release Record Template v1

> **Status:** Governance template only. Completing a record is evidence for an accountable decision; it does not constitute approval, publish an artifact, apply a migration, start a rollout, or modify an environment.

## Purpose

A release record makes a promotion or rollback decision reproducible. It connects an immutable artifact to reviewed migrations, dependency locks, validation evidence, recovery readiness, accountable approval, measurable rollback conditions, and post-release reconciliation. The record is metadata and evidence references only; it must never contain secret values, private keys, connection strings, access tokens, raw backup contents, or evidence payloads.

| Record area | Required v1 content |
|---|---|
| Release identity | Release ID, target environment, source revision, artifact SHA-256, and non-secret artifact reference. |
| Migration/dependencies | Each reviewed migration and the dependency-lock identifier, SHA-256, and evidence reference. |
| Verification | Evidence references for tests, race detection, static analysis, and security review. |
| Recovery | Pre-release backup and restore-rehearsal evidence references. |
| Approval | Accountable approver ID, UTC timestamp, decision, and approval evidence reference. |
| Rollback | At least one measurable trigger, approved target, owner, and evidence reference. |
| Reconciliation | Post-release reconciliation status and evidence reference. |

## Completion procedure

Create a record conforming to `contracts/release_record_v1.schema.json` only after evidence already exists. Use `DRAFT` while information is being assembled. An accountable human may change the status to `APPROVED` or `STOPPED`; `RELEASED` and `ROLLED_BACK` are factual outcomes, not requested actions. Every digest must be lowercase SHA-256. Every evidence reference must resolve to an approved, access-controlled non-secret location.

> A missing evidence reference, missing restore rehearsal, absent rollback trigger, or unapproved decision is a stop condition—not a reason to infer approval or fill the record with placeholders.

## Explicit boundaries

The template does not authorize production deployment. Production release, migration, secret handling, OpenBao wiring, acceptance OIDC enablement, production device enrollment, retention enforcement, or legal-hold/redaction/export actions remain subject to their dedicated approval gates.

## Artifacts

| Artifact | Purpose |
|---|---|
| `contracts/release_record_v1.schema.json` | Portable JSON Schema 2020-12 for a completed record. |
| `contracts/release_record_v1_test.go` | Guard that preserves required release, verification, recovery, approval, rollback, reconciliation, and no-secret fields. |
