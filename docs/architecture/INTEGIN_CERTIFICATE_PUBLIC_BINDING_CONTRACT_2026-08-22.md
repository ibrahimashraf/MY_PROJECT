# Certificate Public Binding Contract — Draft for Controlled Implementation

## Decision

>The verifier must render only **issuance-time immutable facts** that came from tenant/org-scoped canonical sources and were admitted by the approved certificate policy. It must not render mutable operational data, free-form inspector text, or inferred labels.

This is a design contract only. No schema, transport, or public response change is authorized by this document alone.

## Canonical entities and issuance boundary

| Entity | Required ownership boundary | Proposed purpose | Prohibited use |
|---|---|---|---|
| `asset` | Tenant ID and organization ID; forced RLS | Authoritative asset ID, serial number, description, and type | Do not use client-submitted values as a verifier source |
| `inspection_record` | Existing tenant/org scope | Links the inspection revision to the canonical asset and selected inspection type | Do not treat arbitrary checklist prose as a public test scope |
| `inspection_public_scope` | Tenant/org scoped; bound to inspection revision | Approved structured test/scope outcome derived from finalized inspection data | No secret evidence, internal notes, findings, or free-form results |
| `certificate_policy` | Existing tenant/org approved policy | Selects which public binding keys are permitted for a certificate profile | Cannot expand data exposure after issuance |
| `certificate_snapshot` | Existing tenant/org issuance snapshot | Stores the immutable selected public-binding snapshot and integrity digest | Never replace it with current asset values during verification |

## Proposed binding keys

The approved catalog may be extended only after canonical columns, migration review, and issuance snapshot logic exist. The candidate keys are deliberately narrow:

| Binding key | Canonical source at issuance | Inclusion condition | Public field |
|---|---|---|---|
| `asset.id` | `asset.id` | Always, when certificate profile permits public verification | `asset_id` |
| `asset.serial_number` | `asset.serial_number` | Non-empty and policy permits it | `asset_serial_number` |
| `asset.description` | `asset.description` | Non-empty and policy permits it | `asset_description` |
| `asset.type` | `asset.asset_type` | Policy permits it | `asset_type` |
| `inspection.type` | Persisted canonical inspection type | Policy permits it | `inspection_type` |
| `inspection.public_scope` | Approved `inspection_public_scope` for the exact inspection revision | Policy permits it and content validates against structured taxonomy | `test_scope` |

The raw public token, token digest, tenant ID, organization ID, personnel identity, internal audit events, evidence references, internal findings, reviewer reasoning, and non-approved checklist content remain prohibited from the public projection.

## Atomic issuance requirement

Issuance must lock the certificate record, approved policy/version, canonical asset, exact inspection revision, and approved public-scope record in the same tenant/org-scoped transaction. It must serialize only the policy-allowed binding values into the certificate snapshot, calculate a digest over the immutable snapshot, and issue the certificate only if all required values are present and valid.

Public verification must load the snapshot-backed projection from `certificate_record` plus immutable certificate snapshot data. It must not join directly to mutable live `asset` or inspection tables for issued certificates.

## Lifecycle and correctness rules

| Event | Required behavior |
|---|---|
| Asset details change after issuance | Existing certificate public display remains unchanged; a new inspection/certificate is required if the changed fact affects truthfulness |
| Inspection revision changes | Existing certificate remains tied to its revision; replacement certificate follows normal supersession rules |
| Public-scope record is incomplete | Issuance fails for a profile requiring scope display; it does not silently omit a required fraud-resistance field |
| Policy later reduces public disclosure | Newly issued certificates follow the new policy; historic certificate snapshot disclosure changes only through an explicit, audited policy-and-reissue decision |
| Certificate revoked, superseded, or expired | Public status remains truthful; identity fields may remain available only if the approved policy calls for them |

## Required proofs before enabling these fields

Implementation must provide reviewed up/down migration candidates, forced-RLS verification, cross-org denial cases, issuance snapshot immutability proof, public response allow-list proof, missing-required-scope rejection, mutable-source non-rewrite proof, and zero-fixture-residue evidence. The shared deployment-grade rate/abuse control remains a separate prerequisite for external release.
