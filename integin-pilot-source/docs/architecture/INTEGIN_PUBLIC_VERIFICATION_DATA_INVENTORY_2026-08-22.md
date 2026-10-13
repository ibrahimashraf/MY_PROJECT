# Public Verification Data Inventory — 2026-08-22

## Purpose

This inventory establishes the source-of-truth boundary for the next certificate-verification milestone. It does not add a database column, binding key, HTTP field, or document-rendering behavior.

| Desired verifier datum | Current canonical source | Current use | Safe public-binding conclusion |
|---|---|---|---|
| Certificate number, status, issue/expiry timestamps | `certificate_record` | Current digest-backed public projection | Already public within the narrow verifier contract |
| Asset ID | `inspection_record.asset_id` | Current digest-backed public projection and certificate snapshot | Already public within the narrow verifier contract |
| Asset type | `work_order_scope_item.asset_type` | Tenant-scoped work-order scope metadata | A canonical relational source exists, but it is not yet captured in the certificate snapshot or approved for public projection |
| Inspection type | `inspection.Inspection.inspectionType` domain aggregate | Canonical domain concept | Not persisted as a first-class field in `inspection_record` or included in the certificate snapshot; do not expose yet |
| Asset serial number | None located | Not in current work-order scope, inspection persistence, or certificate snapshot | Must be introduced through a canonical asset source and approved binding; never substitute the asset ID |
| Asset description | None located | Not in current work-order scope, inspection persistence, or certificate snapshot | Must be introduced through a canonical asset source and approved binding |
| Test scope / result summary | Template checklist prompts and findings, but no approved canonical public summary | Domain supports metadata-driven items and findings | Requires an explicit, policy-approved derived snapshot; do not assemble arbitrary checklist content for public display |

## Evidence basis

`work_order_scope_item` persists the tenant-scoped asset ID and asset type but no serial or description. `inspection_record` persists the asset ID, inspector, lifecycle, revision, and finalization fields but no inspection type or test scope. Certificate issuance snapshots the approved template and a fixed inspection-state cell snapshot; it does not yet capture asset type, serial, description, inspection type, or a derived test-scope statement.

The current certificate-template binding catalog likewise permits only the approved inspection identity/lifecycle keys. Its validation path rejects unapproved keys, which is the correct current protection against accidental public or PDF binding of unavailable equipment facts.

## Design consequence

The next design must add a tenant/org-scoped canonical asset record (or formally bind to an already-approved authoritative asset source), capture the selected public facts immutably at issuance, and expose only a policy-approved subset. The public verifier must never query mutable client-facing asset data at verification time in a way that can rewrite the truth of an already-issued certificate.

Until that design and its controlled migration/proofs are complete, the verifier remains limited to the currently truthful five-field projection.
