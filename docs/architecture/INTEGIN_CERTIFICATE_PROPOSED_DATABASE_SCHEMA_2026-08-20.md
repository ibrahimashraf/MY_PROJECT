# INTEGIN Proposed Certificate Database Schema

**Status:** Proposed architecture only; no migration, database change, or data backfill is authorised by this document.  
**Date:** 2026-08-20  
**Scope:** Data needed to design, issue, verify, supersede, retain, and render fixed-layout inspection certificates and reports.

## Direct answer

**Yes, the important information is stored—but not all in one large certificate table and not one column for every visual cell.**

INTEGIN should use four storage forms:

| Storage form | What belongs there | Example |
| --- | --- | --- |
| Normalised PostgreSQL tables | Authoritative business and lifecycle records | Asset serial number, inspection outcome, certificate status, approval event. |
| Versioned JSON/JSONB documents | Flexible approved template layout and signed/dynamic form values | Fixed cell positions, template component configuration, a typed template-field value. |
| Immutable issuance snapshot | Exact resolved values used to issue one certificate | The client, asset, result, selected evidence, and signatories as they were at issuance. |
| RustFS/file-object references | Large files, never uncontrolled blobs in ordinary database fields | Final PDF, photo, signature image, stamp, attachment. |

> A certificate cell is normally stored as a **template-zone configuration**. The actual value belongs to its appropriate business record, then is copied into an immutable issuance snapshot only when the certificate is issued.

## Universal columns

Every tenant-owned operational table should include the following common columns. These are intentionally not repeated in every table below.

| Column | Type / rule | Purpose |
| --- | --- | --- |
| `id` | UUID / ULID primary key | Stable non-sequential record identity. |
| `tenant_id` | UUID, required | Mandatory tenant isolation key. |
| `organization_id` | UUID, required | Organisation scope under the tenant. |
| `created_at`, `created_by` | Timestamp / user ID | Creation audit. |
| `updated_at`, `updated_by` | Timestamp / user ID | Last mutable update audit. |
| `row_version` | Integer or revision token | Optimistic concurrency and offline-sync conflict control. |
| `archived_at` | Nullable timestamp | Archive/soft-retire marker where the lifecycle permits it. |

All data access must be tenant- and organization-scoped in the application and in PostgreSQL row-level security. No template binding may bypass this scope.

## A. Core source records

These tables store the real operational data that certificate cells bind to.

### Client, site, work order, and asset

| Table | Key proposed columns beyond universal columns | Why it exists |
| --- | --- | --- |
| `clients` | `client_code`, `legal_name`, `trading_name`, `certificate_address_json`, `contact_summary_json`, `status` | Certificate recipient/owner/employer details. |
| `sites` | `client_id`, `site_code`, `name`, `address_json`, `location_notes`, `status` | Reusable examination/work location. |
| `work_orders` | `order_number`, `client_id`, `site_id`, `client_reference`, `scope_summary`, `status`, `opened_at`, `due_at`, `closed_at` | Operational/commercial request that groups inspections and partial handover. |
| `assets` | `asset_id_display`, `asset_type_id`, `serial_number`, `manufacturer`, `model`, `manufactured_on`, `certificate_display_description`, `master_description`, `status`, `parent_asset_id` | Canonical item record. `master_description` can be complete; `certificate_display_description` is the approved bounded display value. |
| `asset_components` | `parent_asset_id`, `component_type_id`, `identifier`, `serial_number`, `manufacturer`, `model`, `quantity`, `status` | Fitted/nested assembly parts, such as shackles, links, or termination components. |
| `asset_measurements` | `asset_id` or `asset_component_id`, `measurement_key`, `numeric_value`, `unit_code`, `precision_scale`, `effective_from`, `effective_to`, `source` | Dimensions, weight, capacity, rope diameter, length, angle, and other structured technical values. |

`measurement_key` is a controlled value such as `safe_working_load`, `proof_load`, `rope_diameter`, `rope_length`, `tare_weight`, or `gross_mass`. Values must not be stored only as human-formatted free text.

### Inspection, forms, results, and findings

| Table | Key proposed columns beyond universal columns | Why it exists |
| --- | --- | --- |
| `inspection_templates` | `template_code`, `display_name`, `asset_type_id`, `status` | Business identity of an inspection form family. |
| `inspection_template_versions` | `inspection_template_id`, `version_number`, `schema_json`, `package_hash`, `status`, `approved_at`, `approved_by` | Approved dynamic Field form definition; immutable after publication. |
| `inspections` | `work_order_id`, `asset_id`, `inspection_template_version_id`, `inspection_number`, `assigned_to`, `started_at`, `completed_at`, `status`, `outcome`, `safe_to_operate`, `location_id` | One authoritative inspection event. |
| `inspection_field_values` | `inspection_id`, `inspection_template_version_id`, `question_instance_id`, `parent_instance_id`, `order_index`, `field_key`, `value_json`, `value_type_code`, `capture_state`, `applicability_code`, `source_code`, `captured_at`, `captured_by`, `ruleset_hash` | Typed dynamic form values governed by the approved inspection template version. Instance/group identity supports repeaters and tables; it is not a raw unvalidated key/value store. |
| `inspection_assessments` | `inspection_id`, `inspection_template_version_id`, `question_instance_id`, `assessment_key`, `answer_code`, `capture_state`, `applicability_code`, `answered_at`, `answered_by`, `ruleset_hash`, `basis_note` | Controlled yes/no/not-applicable questions, including conditional statutory/examination-basis answers. One answer code prevents contradictory checkboxes. |
| `inspection_findings` | `inspection_id`, `finding_code`, `severity`, `description`, `immediate_danger`, `corrective_action`, `status`, `resolved_at` | Defects, non-conformances, repair needs, and action tracking. |
| `inspection_recommendations` | `inspection_id`, `recommendation_type`, `description`, `due_at`, `status` | Separate recommendations/next actions from observations. |

The final outcome and safety decision are normal columns because they drive workflow, filtering, certificate eligibility, dashboards, and reminders. They should not exist only inside JSON.

### Tests, measurements, NDT, and controlled evidence

| Table | Key proposed columns beyond universal columns | Why it exists |
| --- | --- | --- |
| `test_runs` | `inspection_id`, `test_type`, `performed_at`, `performed_by`, `standard_reference`, `outcome`, `summary` | General load, dimensional, function, or other test occurrence. |
| `test_measurements` | `test_run_id`, `measurement_key`, `numeric_value`, `unit_code`, `precision_scale`, `acceptance_limit_json`, `result_code` | Structured measured values and result. |
| `ndt_inspections` | `inspection_id`, `method_code`, `procedure_reference`, `specification_reference`, `acceptance_criteria`, `inspection_area_code`, `condition_code`, `conclusion`, `result_code` | NDT-specific controlled data. |
| `ndt_consumables` | `ndt_inspection_id`, `role_code`, `product_name`, `batch_number`, `expiry_on`, `manufacturer` | Consumable traceability such as penetrant/developer or magnetic media. |
| `inspection_instruments` | `inspection_id` or `test_run_id`, `instrument_id`, `used_for`, `calibration_status_at_use`, `calibration_certificate_reference`, `calibration_valid_to` | Links actual inspection/test use to calibration evidence. |
| `file_objects` | `storage_key`, `content_hash`, `mime_type`, `byte_size`, `original_filename`, `uploaded_at`, `retention_class`, `legal_hold_flag`, `malware_scan_status`, `tenant_id` | Controlled content-addressed reference to a RustFS object; does not store the large file bytes in normal rows. |
| `evidence_items` | `inspection_id`, `file_object_id`, `evidence_type`, `collected_at`, `collected_by`, `original_hash`, `review_status`, `redaction_status`, `caption`, `source_context` | Photo, signature, scan, attachment, or other evidence provenance. |
| `evidence_transformations` | `evidence_item_id`, `operation`, `input_hash`, `output_file_object_id`, `performed_at`, `performed_by`, `reason_code` | Crop/redaction/format-conversion lineage. |

## B. People, competence, and authority records

| Table | Key proposed columns beyond universal columns | Why it exists |
| --- | --- | --- |
| `users` | `display_name`, `email`, `status` | Person identity; existing auth identity can be linked rather than duplicated. |
| `user_qualifications` | `user_id`, `qualification_code`, `issuer`, `reference`, `valid_from`, `valid_to`, `file_object_id`, `status` | Inspector/NDT/reviewer competence and qualification validity. |
| `certificate_authorities` | `authority_code`, `issuing_body_id`, `jurisdiction_code`, `scope_json`, `status`, `valid_from`, `valid_to` | Controlled basis/scope for issuance, not merely an image of a signature. |
| `authority_delegations` | `certificate_authority_id`, `delegator_user_id`, `delegate_user_id`, `scope_json`, `valid_from`, `valid_to`, `status` | Explicit sign-on-behalf/delegation trail. |
| `signatory_assignments` | `certificate_issue_id`, `role_code`, `user_id`, `qualification_snapshot_json`, `signature_evidence_id`, `signed_at`, `decision_code` | Role-indexed certificate signatures. Supports inspector, issuer, authorizer, reviewer, client witness, or other approved role. |

The design should use a generic `role_code` rather than hard-coding only `inspector_name` and `authorizer_name` columns.

## C. Fixed-layout document templates

### What is a column and what is JSON

The high-level template lifecycle deserves normal columns. The flexible page geometry and component settings belong to a versioned declarative document, because templates vary by certificate family.

| Table | Key proposed columns | Stored as versioned JSON / child rows | Reason |
| --- | --- | --- | --- |
| `certificate_templates` | `template_code`, `display_name`, `document_family`, `status` | None required | Stable template identity. |
| `certificate_template_versions` | `certificate_template_id`, `version_number`, `page_profile_code`, `renderer_profile_code`, `status`, `approved_at`, `approved_by`, `effective_from`, `effective_to`, `template_hash` | `layout_json`, `static_content_json`, `binding_policy_json` | Draft/approved fixed layout; immutable after approval. |
| `template_zones` *(optional normalized companion)* | `template_version_id`, `zone_id`, `component_type`, `page_number`, `x_mm`, `y_mm`, `width_mm`, `height_mm`, `binding_key`, `display_order` | `fit_policy_json`, `visibility_rule_json`, `format_json`, `component_config_json` | Useful where querying/validation of zones benefits from columns. Alternatively derive from `layout_json`. |
| `binding_catalog_versions` | `catalog_version`, `status`, `published_at` | `bindings_json` | Server-owned allowed binding names, types, component compatibility, sensitivity, and template rules. |
| `render_profiles` | `profile_code`, `renderer_version`, `font_manifest_hash`, `locale_policy_json`, `page_unit_policy`, `status` | Optional `render_config_json` | Pins the rendering environment that controls fit and historical repeatability. |

No template record may contain raw SQL, a database connection string, an external unapproved asset URL, a secret, arbitrary JavaScript, or an unvalidated internal record ID.

### Mandatory data-model corrections from independent validation

The initial certificate schema is sound as a foundation, but the following structures are mandatory before it is implemented. They make the form designer deterministic, repeatable, offline-capable, and safe for multi-tenant use.

| Table / structure | Key columns or fields | Why it is required |
| --- | --- | --- |
| `binding_key_registry` | `binding_key`, `namespace`, `value_type_code`, `cardinality`, `unit_policy`, `nullable`, `sensitivity_class`, `permission_policy`, `introduced_in`, `deprecated_in`, `status` | Formal server-owned dictionary for values such as `core.asset.serial_number`. It prevents collisions, weak typing, and raw query access. |
| `value_type_registry` | `value_type_code`, `validation_schema_json`, `canonicalisation_policy`, `display_policy`, `status` | Defines text, integer, decimal, date, datetime-with-timezone, enum, multiselect, signature, file, geo, and measurement behavior. |
| `option_set_versions` | `option_set_code`, `version_number`, `options_json`, `default_locale`, `status` | Stores stable option codes separately from translated labels. |
| `form_ruleset_versions` | `inspection_template_version_id`, `ruleset_version`, `ruleset_json`, `ruleset_hash`, `status` | Versioned visibility, requiredness, calculation, and applicability DSL. The same ruleset runs offline and is revalidated by the server. |
| `form_repeaters` / template structure | `repeater_key`, `parent_key`, `min_instances`, `max_instances`, `order_policy` | Defines repeatable groups/tables so answers use stable `question_instance_id` values rather than only a static field key. |
| `sync_operations` | `operation_id`, `entity_type`, `entity_id`, `base_row_version`, `operation_kind`, `payload_hash`, `created_on_device_at`, `received_at`, `status`, `error_code` | Offline operation log with deterministic client IDs and idempotent server reconciliation. |
| `sync_conflicts` | `operation_id`, `entity_id`, `field_key`, `server_value_hash`, `client_value_hash`, `resolution_code`, `resolved_at` | Explicit conflict state instead of silent overwrites. |

`capture_state` must distinguish at least `answered`, `unanswered`, `inapplicable`, `unknown`, and `system_derived`. `answer_code` is the substantive answer. This prevents a hidden conditional question from being confused with a question the inspector forgot to answer.

`question_instance_id` is mandatory for repeating sections. For example, four shackle components can each have the same `serial_number` field key but different stable instance IDs and order positions. This prevents rows from being mixed when a user edits offline or when a certificate table spans pages.

## D. Certificate issuance and lifecycle

| Table | Key proposed columns beyond universal columns | Why it exists |
| --- | --- | --- |
| `certificate_issues` | `certificate_number`, `certificate_template_version_id`, `work_order_id`, `client_id`, `status`, `issued_at`, `issued_by`, `next_due_at`, `verification_reference`, `supersedes_certificate_issue_id`, `voided_at`, `void_reason_code`, `lock_version` | Central certificate lifecycle record. |
| `certificate_issue_items` | `certificate_issue_id`, `sequence`, `asset_id`, `inspection_id`, `item_display_scope`, `status` | Keeps every rendered item tied to its asset and inspection when one certificate covers many items. |
| `certificate_lifecycle_events` | `certificate_issue_id`, `from_status`, `to_status`, `reason_code`, `occurred_at`, `actor_user_id`, `actor_role_code`, `comment` | Immutable transition/audit history. |
| `certificate_supersession_links` | `original_certificate_issue_id`, `replacement_certificate_issue_id`, `reason_code`, `effective_at`, `created_by` | Explicit replacement/withdrawal/continuity chain. |
| `certificate_issuance_snapshots` | `certificate_issue_id`, `snapshot_schema_version`, `snapshot_hash`, `snapshot_json`, `created_at`, `immutable_at` | Exact resolved values, selected evidence, signatories, locale, template version, and policy context used for issuance. |
| `certificate_renders` | `certificate_issue_id`, `certificate_issuance_snapshot_id`, `render_profile_id`, `rendered_at`, `page_count`, `render_hash`, `file_object_id`, `status`, `failure_code` | One authoritative final rendered document and its controlled renderer metadata. |
| `certificate_evidence_links` | `certificate_issue_id`, `evidence_item_id`, `purpose_code`, `display_order`, `included_in_annex`, `inclusion_rationale` | Selected evidence for the certificate; never an arbitrary file path in a template. |
| `certificate_verification_records` | `certificate_issue_id`, `public_reference`, `qr_payload_version`, `public_status`, `disclosure_policy_code`, `revoked_at`, `revocation_reason_code` | Public verification policy/state. |
| `certificate_verification_events` | `certificate_verification_record_id`, `verified_at`, `channel_code`, `outcome_code`, `viewer_context_hash` | Minimal verification audit; privacy policy determines what context is retained. |

## E. Later operational and external tables

These are valuable but should not delay the work-order, inspection, template, and certificate foundations.

| Table | Key proposed columns | When to add |
| --- | --- | --- |
| `certificate_notification_rules` | `trigger_code`, `recipient_group_code`, `schedule_rule_json`, `throttle_policy_json`, `active` | Renewal/reminder dashboard phase. |
| `certificate_notification_deliveries` | `certificate_issue_id`, `rule_id`, `recipient_reference`, `scheduled_at`, `sent_at`, `outcome_code`, `acknowledged_at` | Renewal/reminder dashboard phase. |
| `external_submission_schemes` | `scheme_code`, `jurisdiction_code`, `mapping_version`, `status` | Only when a real integration is approved. |
| `external_submission_records` | `certificate_issue_id`, `scheme_id`, `submission_status`, `export_file_object_id`, `external_reference`, `submitted_at`, `accepted_at`, `receipt_file_object_id`, `error_code` | Later regulatory/platform readiness. |
| `record_retention_policies` | `record_class`, `retain_for`, `archive_after`, `purge_rule`, `status` | Governance/retention phase. |
| `record_legal_holds` | `record_type`, `record_id`, `reason_code`, `placed_at`, `released_at`, `status` | Governance/retention phase. |
| `certificate_transmittals` | `work_order_id`, `recipient_group`, `handover_status`, `sent_at`, `acknowledged_at`, `cover_file_object_id` | Handover/reconciliation phase. |
| `certificate_deviations` | `certificate_issue_id`, `deviation_type`, `reason`, `approved_by`, `approved_at`, `valid_to`, `linked_finding_id` | Later, only if an owner-approved exception process is designed. |

## What is deliberately not stored as a normal certificate column

| Do not create/store as a general certificate column | Correct alternative |
| --- | --- |
| `serial_number_box_x`, `serial_number_box_y` on `certificate_issues` | Template zone bounds in the approved template version. |
| A separate `client_name_cell_1`, `client_name_cell_2` per template | One canonical `clients.legal_name` plus template bindings. |
| Raw SQL/query text from template designers | A server-owned binding key such as `asset.serial_number`. |
| Full image/PDF binary in ordinary rows | `file_objects` reference with object-store key and content hash. |
| Secrets, private signing keys, connection strings, or external API tokens | Approved private secret storage, never a certificate table. |
| Raw uncontrolled HTML/JavaScript in templates | Declarative component and zone configuration. |
| A display-only `yes_column` and `no_column` | One controlled `answer_code` such as `yes`, `no`, or `not_applicable`. |
| A text-only `safe_to_operate_label` as the source of truth | `inspections.safe_to_operate` as the authoritative decision; template renders it. |
| Page number as permanent certificate business data | `certificate_renders.page_count` and render-time `document.page_number`. |
| A single unstructured “all certificate data” text field | Normalized records plus controlled snapshots. |

## Snapshot and immutability rule

When a certificate is issued, INTEGIN should **not** keep reading live records to recreate the old document. It must preserve a controlled snapshot of the exact resolved values, template version, binding-catalog version, render profile, selected evidence IDs/content hashes, signatories, and source-record revisions used at issuance.

The issuance snapshot must additionally preserve the ruleset hash and evaluated rule outcomes, canonical values with units, locale and time zone, renderer/font profile, template version, selected evidence content hashes, and the final snapshot content hash. The server must reject issuance if a required template/binding/rule/version is not approved or if a snapshot cannot be made self-contained.

After issuance:

1. The source asset or client record may later change, but the old certificate stays unchanged.
2. The template may later have a newer approved version, but the old certificate remains tied to its original version.
3. A correction creates a new certificate issue and a supersession link; it does not overwrite the old final PDF or snapshot.
4. Verification reads the certificate lifecycle/verification state, so a voided or superseded certificate can be reported correctly.

## Recommended implementation sequence

The first implementation should not add every later table at once.

| Stage | First tables/columns to implement |
| --- | --- |
| Work-order and inspection foundation | `clients`, `sites`, `work_orders`, `assets`, `asset_components`, `asset_measurements`, `inspections`, `inspection_field_values`, `inspection_assessments`, `inspection_findings`, `test_runs`, `test_measurements`, `file_objects`, `evidence_items`. |
| Certificate-template foundation | `certificate_templates`, `certificate_template_versions`, `binding_catalog_versions`, `render_profiles`; start with `layout_json` and introduce `template_zones` only if query/validation requirements justify it. |
| Issuance and verification foundation | `certificate_issues`, `certificate_issue_items`, `certificate_lifecycle_events`, `certificate_issuance_snapshots`, `certificate_renders`, `signatory_assignments`, `certificate_verification_records`, `certificate_verification_events`. |
| Technical extensions | `ndt_inspections`, `ndt_consumables`, `inspection_instruments`, qualification/authority tables, and type-specific controlled keys. |
| Later operational work | Notification, external submission, retention, transmittal, and deviation tables only after their workflows are approved. |

## Final answer

The information **will be stored**, but in the correct place:

- Actual client, asset, inspection, test, finding, and signatory information is stored once in authoritative normalized tables.
- Fixed certificate boxes, their positions, fonts, maximum lines, and binding keys are stored in a versioned template definition.
- The exact values used in a final certificate are stored in an immutable issuance snapshot.
- Photos, signatures, PDFs, and stamps are stored in RustFS with database references and content hashes.
- Lifecycle, replacement, verification, audit, authority, and evidence provenance are stored as controlled operational records, mostly not displayed on the certificate.

This gives INTEGIN flexible certificate designs without data duplication, missing audit trails, or a different database schema for every certificate style.
