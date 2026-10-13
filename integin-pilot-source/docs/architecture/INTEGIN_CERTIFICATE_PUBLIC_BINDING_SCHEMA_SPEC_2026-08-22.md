# Canonical Public-Binding Schema Specification — 2026-08-22

## Compatibility decision

The existing `inspection_record.asset_id` remains the stable business identifier. This slice must not retrofit a mandatory foreign key into already-persisted inspection rows. Instead, the new tenant/org-scoped `asset_registry` resolves that existing identifier at certificate issuance; a certificate policy can require or omit each asset fact. The issuance snapshot remains the historical truth even when the registry changes later.

## Proposed relations

| Relation | Primary / uniqueness rule | Mandatory fields | Purpose |
|---|---|---|---|
| `asset_registry` | Primary key `id`; unique `(tenant_id, organization_id, asset_id)` | tenant/org, asset ID, asset type, serial number, description, lifecycle state, timestamps | Canonical registry for verification-safe equipment identity |
| `inspection_public_scope` | Primary key `id`; unique `(tenant_id, organization_id, inspection_id, inspection_revision)` | tenant/org, inspection/revision, inspection type, scope taxonomy version, result state, timestamps | One approved public scope summary for an exact inspection revision |
| `inspection_public_scope_item` | Primary key `id`; unique `(tenant_id, organization_id, scope_id, scope_code)` | tenant/org, parent scope, code, label, outcome, display order | Structured public test items; never free-form internal findings |
| `certificate_policy_public_binding` | Primary key `id`; unique `(tenant_id, organization_id, policy_id, binding_key)` | tenant/org, policy ID, approved key, required flag, display order | Policy admission control for public and rendering bindings |
| `certificate_snapshot.public_binding_snapshot` | Existing one-row-per-certificate snapshot row | JSON object of only approved issuance-time values | Immutable verifier/render source, sealed with the existing snapshot digest calculation |

## Validation rules

`asset_registry.serial_number` and `asset_registry.description` must be trimmed, non-empty values when an approved policy requires their corresponding binding. `asset_type` is a controlled canonical code rather than a presentation label. An asset may be retired but never deleted while referenced by an issued certificate.

`inspection_public_scope` is linked to the exact `inspection_id` and `inspection_revision`; it records a controlled `inspection_type`, a taxonomy version, and a public result state. Individual `inspection_public_scope_item` rows carry a controlled scope code, display label, outcome, and ordering. Evidence paths, notes, measured readings, reviewer comments, and raw inspection answers are excluded from this entity.

`certificate_policy_public_binding.binding_key` is restricted to `asset.id`, `asset.serial_number`, `asset.description`, `asset.type`, `inspection.type`, and `inspection.public_scope`. No policy row can admit a key outside that allow-list. Required keys cause issuance failure if their locked canonical values are unavailable.

## Issuance transaction shape

1. Set tenant and organization transaction scope.
2. Lock the signed certificate, its approved policy/version, the exact inspection record/revision, and the matching canonical asset registry row.
3. Lock the exact inspection public-scope summary and ordered items if policy admits `inspection.public_scope`.
4. Validate policy bindings and materialize only admitted values into `certificate_snapshot.public_binding_snapshot`.
5. Include template snapshot, fixed cell snapshot, and public-binding snapshot in the certificate snapshot digest.
6. Persist the certificate number/token digest and snapshot atomically.

## Snapshot structure

```json
{
  "asset.id": "asset-a",
  "asset.serial_number": "SN-EXAMPLE-001",
  "asset.description": "Example equipment description",
  "asset.type": "lifting_equipment",
  "inspection.type": "periodic_inspection",
  "inspection.public_scope": {
    "taxonomy_version": 1,
    "result_state": "PASS",
    "items": [
      {"code": "visual", "label": "Visual examination", "outcome": "PASS"}
    ]
  }
}
```

The example is a shape specification, not seeded or production data. The verifier reads only keys selected by the snapshot and current status rules; it does not construct new facts from mutable tables.

## Deliberate exclusions

No public binding will include owner/client identity, location, employee identity, evidence data, internal notes, raw checklist results, cost, audit metadata, token material, or tenant/org IDs. External recognition metadata is also excluded until a dedicated adapter is approved.
