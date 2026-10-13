# INTEGIN Canonical Inspection Persistence Candidate

**Status:** Candidate design only. No migration execution, runtime wiring, data backfill, or authority claim is authorized by this document.

## Purpose

This candidate supplies the missing server-owned inspection read model required by the Work-Order `InspectionMembershipValidator`. It separates authoritative inspection identity and lifecycle from Field caches and normalizes accepted submission membership.

## Authority ownership

| Concern | Authoritative owner | Work-Order role |
|---|---|---|
| Inspection identity, tenant, organization, asset, inspector, lifecycle, revision | Inspection persistence service | Read-only validation consumer |
| Work-order scope and inspector assignment | Work-Order service | Owns order/assignment lifecycle |
| Submission segment membership | Work-Order service | Writes immutable accepted segment items only after validator acceptance |
| Certificate issuance and commercial release | Separate services | Explicitly out of scope |

## Candidate tables

| Table | Purpose | Core integrity |
|---|---|---|
| `inspection_record` | Canonical server inspection projection. | Tenant and organization ownership; work-order and scope link; asset, inspector, lifecycle, revision, finalization state. |
| `work_order_submission_item` | Normalized membership of an accepted partial-submission segment. | One accepted segment per inspection; composite foreign keys bind the segment, inspection, work order, assignment, and scope item to the same authoritative containment chain. |

## Required server validation

Database relationships establish tenant, organization, work-order, scope, and assignment containment. The `InspectionMembershipReader` must additionally verify asset equality, inspector equality, eligible lifecycle status, and non-finalized status in the same tenant-scoped transaction. These checks remain in service code because they depend on state policy rather than static foreign keys.

## Initial lifecycle policy for review

| Inspection state | Eligible for partial submission? | Reason |
|---|---|---|
| `SCHEDULED`, `ASSIGNED`, `IN_PROGRESS` | No | Field work is incomplete. |
| `COMPLETED`, `PENDING_REVIEW` | Proposed yes | Inspector can submit a work summary without issuing a certificate. |
| `APPROVED`, `CLOSED` | Proposed no | Requires an explicit later rule if resubmission is wanted. |
| `REJECTED` | No | Must be corrected or resolved first. `VOIDED` and `SUPERSEDED` are not current `InspectionStatus` values and require a separate lifecycle evolution decision. |

## Required validation sequence

1. Start the authenticated tenant- and organization-scoped transaction.
2. Lock the Work-Order revision and assignment.
3. Resolve every requested inspection record through the canonical reader.
4. Reject the entire batch if any inspection is missing, cross-scoped, assigned to another inspector, out of assignment scope, asset-mismatched, ineligible, finalized, or already present in a submission item.
5. Create the immutable segment and normalized segment items.
6. Update the order revision once and store the idempotent receipt.
7. Commit atomically.

## Explicit review gates before SQL execution

1. Confirm inspection lifecycle vocabulary with the Inspection domain owner.
2. Confirm whether inspector transfer preserves, reassigns, or blocks pre-sync completed inspection records.
3. Review composite foreign keys and normalized submission-item constraints with existing migration conventions.
4. Review RLS policies under the non-owner runtime role and force RLS on new tables.
5. Define a non-destructive backfill or declare candidate-only pilot semantics.
6. Prepare rollback and restore drill with fixture inventory.
7. Add controlled integration tests for every acceptance and denial criterion in the authority contract.

## Current claim boundary

This design does not prove canonical membership validation. It defines the minimum data boundary required to implement and prove it safely. A fresh disposable PostgreSQL 18 exercise proved SQL apply, valid containment insertion, same-organization cross-order foreign-key denial, forced-RLS metadata, non-owner cross-organization read denial, and candidate rollback. It did not apply to the pilot database, test non-owner write denial, wire a production reader, or prove pilot-data compatibility.
