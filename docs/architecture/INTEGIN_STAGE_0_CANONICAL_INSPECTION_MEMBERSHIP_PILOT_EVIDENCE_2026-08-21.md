# INTEGIN Canonical Inspection Membership Pilot Evidence

**Date:** 2026-08-21

## Applied change

The reviewed `0008_inspection_persistence.candidate.sql` was applied to the local controlled pilot PostgreSQL 18 database only. The exact applied SHA-256 was `12484533879bb0fd374f6cc620119b56af2852bf0a965eb690f46b930ceb3ff3`. A verified post-cleanup pre-apply backup was retained at `docs/evidence/pilot-backups/INTEGIN-pilot-pre-0008-20260821-094050.dump` with SHA-256 `780684CB112025E26CA0A6F5F094ACE64CA05B5538221D99F0C08C32E306C3B0`.

## Pilot preflight and preservation

The read-only preflight found three residual `it-flow-*` integration namespaces. They were identified as controlled test fixtures, backed up, and removed in dependency order before migration. No non-test data was identified in the inspected Work-Order tables.

## Proven on the pilot

| Claim | Evidence |
|---|---|
| Candidate schema apply | Transactional apply completed after file hash match. |
| Schema state | `inspection_record` and `work_order_submission_item` exist; required containment constraints exist; both tables report RLS enabled and forced. |
| Canonical partial submission | Controlled test creates canonical inspection records, rejects an unknown inspection, writes two normalized submission items, updates both inspection records to `SUBMITTED`, rejects re-submission, and preserves certificate/commercial state. |
| Cross-organization Work-Order mutation denial | Existing controlled integration assertion passes. |
| Non-owner pilot write RLS denial | An ephemeral least-privilege role using Organization B was rejected when inserting an Organization A inspection record. The role and any probe row were verified absent afterward. |
| Test cleanup | Focused test completed with zero `it-flow-*` work orders, inspection records, submission items, and operation rows afterward. |
| Regression | `go test ./...` passed after the final cleanup fix. |

## Defect discovered and corrected

An initial canonical-reader assertion failure left fixtures because cleanup occurred only at the successful end of the test. Converting cleanup to `t.Cleanup` first exposed an ordering issue: the database close defer ran before cleanup. The test now registers database close with `t.Cleanup` before the fixture cleanup registration, ensuring LIFO execution cleans fixtures before closing the connection. The corrected focused run proved zero residual rows.

## Explicit limits

This does not prove production composition-root wiring: current source mapping has not identified a production constructor for the Work-Order PostgreSQL repository. Non-owner pilot write RLS denial is proven. Non-owner pilot read isolation remains unproven because the cleaned pilot currently contains no non-test canonical inspection record to query without introducing a dedicated read fixture. Certificate issuance, commercial release, provisional reconciliation authority, and Flutter runtime provisioning remain separate gates.
