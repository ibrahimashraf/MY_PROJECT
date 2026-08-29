# INTEGIN Inspection Membership Authority Contract

**Status:** Precision prerequisite; no persistence migration or runtime wiring authorized by this document.

## 1. Decision

Work-Order partial submission must remain fail-closed until the server can resolve every submitted inspection through a canonical inspection read model. The current codebase does not yet provide that model.

## 2. Evidence inventory

| Source | What it proves | What it does not prove |
|---|---|---|
| `internal/domain/inspection/inspection.go` | An in-memory inspection aggregate has tenant, organization, root asset, assigned inspector, and lifecycle status. | No database repository, no work-order link, no scope-item link, and no production composition root. |
| `migrations/0005_work_package_persistence.sql` | A tenant-scoped work-package assignment links an inspection ID to a device, package, and expiry. | No work-order link, scope-item link, inspector identity, or inspection lifecycle state. |
| `migrations/0006_work_package_assignment_context.sql` | A tenant-scoped inspection context stores root asset, procedure, schedule, and field asset IDs. | No work-order link, assignment link, inspector, lifecycle state, or finalization state. |
| `migrations/0005_work_order_foundation.candidate.sql` | A Work-Order has scope items, inspector assignments, submission segments, revisions, and RLS. | No canonical inspection-record relation; segment inspection IDs are JSON references only. |

## 3. Exact prerequisite

Before a real validator is implemented, the server needs one canonical, RLS-scoped inspection record/read model that can resolve, for every inspection ID: tenant ID, organization ID, work-order ID, scope-item ID, root asset ID, assigned inspector ID, lifecycle status, revision, and finalization/submission state.

## 4. Proposed boundary

The Inspection domain owns inspection lifecycle state. The Work-Order repository consumes a read-only server interface; it must not query a device cache or infer membership from an ID string.

```go
type InspectionMembershipReader interface {
    ResolveForPartialSubmission(ctx context.Context, actor workorder.ActorContext, workOrderID, assignmentID string, inspectionIDs []string) ([]InspectionMembership, error)
}
```

For every accepted inspection, the resolver must prove all of the following in one tenant-scoped server read transaction:

1. The inspection belongs to the actor tenant and organization.
2. The inspection belongs to the requested work order.
3. The inspection scope item is included in the requested active assignment.
4. The inspection root asset agrees with the authoritative scope item asset.
5. The inspection is assigned to the assignment inspector or an explicitly authorized server policy permits the actor.
6. The inspection lifecycle status is eligible for partial submission. The initial proposed eligible states are `completed` and `pending_review`; this policy requires domain-owner approval before implementation.
7. The inspection is not finalized, voided, already included in an accepted segment, or otherwise blocked by authoritative policy.

## 5. Denial criteria

A partial submission must be rejected when any requested inspection is unknown, belongs to another tenant or organization, belongs to another order, maps to an out-of-scope asset, is assigned to another inspector, has an ineligible state, is finalized, or has already been submitted. A batch is atomic: one invalid inspection rejects the entire segment.

## 6. Required persistence shape

The canonical inspection record must be stored separately from `work_order_submission_segment` and have server-controlled foreign keys or equivalent verified references to work order, scope item, asset, and assignment. It must be RLS-protected with the same transaction-local tenant and organization settings. The Work-Order migration must not fabricate this relation by parsing JSON.

## 7. Evidence required before wiring

| Claim | Required proof |
|---|---|
| Membership resolution | Controlled database test with valid multi-inspection batch |
| Tenant and organization isolation | Negative cross-organization read and submit attempts |
| Scope and asset consistency | Out-of-scope and asset-mismatch denial tests |
| Inspector authority | Different-inspector denial test |
| Lifecycle policy | Each allowed and disallowed status tested |
| Atomicity | Mixed valid/invalid batch creates no segment and no revision increment |
| Cleanup | Test fixture inventory and verified dependency-order cleanup |

## 8. Current conclusion

The existing `InspectionMembershipValidator` is correctly fail-closed but cannot yet be given a production canonical implementation. The next safe action is a separate inspection-persistence design and migration review. No runtime authority path should be claimed complete until that prerequisite is implemented and tested.
