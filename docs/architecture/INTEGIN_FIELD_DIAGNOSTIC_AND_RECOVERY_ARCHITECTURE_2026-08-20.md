# INTEGIN Field Diagnostic and Recovery Architecture

**Status:** Proposed architecture; no app update, schema migration, or runtime-policy change is authorised by this document.  
**Date:** 2026-08-20  
**Scope:** Flutter/Dart Field application, metadata-driven forms, offline capture, queued synchronization, and remote diagnosis.

## Direct decision

> The Field app is an **offline-first state machine**. It stores capture state, approved metadata packages, and operations locally before attempting synchronization. It emits structured, redacted diagnostics and provides a safe export path. Routine form, binding, rule, option, and layout corrections are delivered as signed metadata packages; an app release is reserved for native capability, security, or engine-level defects.

This design gives the inspector continuity during network outages and gives the support team enough evidence to identify whether a failure originates in the server, network, account, package, device, renderer, or application code. The server remains authoritative for tenant scope, assignment, validation, approval, issuance, and certificate state.

## 1. Local storage architecture

The Field app should use an encrypted SQLite-backed local store. The specific Flutter database library is an implementation decision; the logical entities and invariants below are architectural requirements.

| Local entity | Required identity and control fields | Purpose and minimization rule |
| --- | --- | --- |
| `local_package` | `package_id`, `version`, `sha256`, `signature_status`, `installed_at`, `active` | Caches signed forms, rules, option sets, binding catalogs, and layouts. Keep only packages needed by the current assignment scope. |
| `local_work_order` | `work_order_id`, `tenant_id`, `server_revision`, `assignment_revision`, `sync_status` | Caches assigned work orders and their locations/assets. Never use a local tenant or assignment value as authority. |
| `local_capture` | `capture_id`, `work_order_id`, `asset_id`, `form_package_id`, `form_version`, `capture_status`, `updated_at` | Stores draft answers, applicability state, evidence references, signatures, and completion state needed to resume work. |
| `local_attachment` | `attachment_id`, `capture_id`, `content_hash`, `mime_type`, `size`, `upload_state`, `storage_reference` | Stores evidence metadata and local encrypted file references. Raw bytes are not included in diagnostic exports. |
| `local_operation` | `operation_id`, `aggregate_id`, `operation_type`, `created_at`, `attempt_count`, `state`, `idempotency_key`, `last_error_code` | Stores the ordered actions required to synchronize work. No operation may be silently discarded. |
| `local_log_event` | `event_id`, `timestamp`, `level`, `category`, `correlation_id`, `code`, `package_version`, `redaction_class` | Stores bounded structured diagnostics without answers, credentials, client names, or raw media. |

Every local record must be attributable to the current authenticated tenant and user context, while retaining only the minimum data needed for offline execution. Sensitive values must be encrypted at rest, and access to local data must follow the device’s authenticated session and lock state.

## 2. Operation queue and synchronization

The app must not treat a network request as the source of truth for capture. A user action first commits local state and an operation to the queue. Synchronization is a separate process that can be retried, paused, inspected, and resumed.

When the inspector selects **Submit**, the app commits a `complete_inspection` operation locally. It does not construct an untracked one-shot payload that disappears if the process crashes. The queue then drains operations in dependency order, for example: work-order acknowledgment, inspection start, answer changes, evidence attachment, inspection completion, and partial-work submission.

### Queue state machine

`pending → sending → acknowledged` is the normal path. A transient failure returns the operation to `pending` with bounded backoff. An authentication failure moves it to `blocked_auth`; a business conflict moves it to `conflict`; a malformed package or unsupported capability moves it to `blocked_package`; and an unrecoverable protocol error moves it to `quarantined` for support review.

Each operation must have a deterministic `idempotency_key`. A retry therefore cannot create a second inspection, duplicate an evidence attachment, or issue a second certificate. The server must return an authoritative revision or receipt. Local `acknowledged` means only that the server accepted that specific operation; it does not mean that a certificate is approved or externally registered.

If the app crashes, the network drops, or the server rejects a submission, the capture state and queue remain intact. The inspector can continue unrelated offline work while the blocked operation is displayed with a clear reason and next action.

## 3. Structured diagnostic logging

The app must use structured event records rather than unbounded console output. Every event should include a timestamp, severity, category, stable error code, correlation ID, app version, package ID/version where relevant, and a redaction class.

| Log category | Examples of safe fields |
| --- | --- |
| **Sync** | Reachability result, request class, attempt number, payload size, HTTP status, server error code, duration, and queue state. |
| **Package** | Download result, hash verification, signature result, schema version, capability check, activation result, and rollback decision. |
| **Renderer** | Form load duration, missing binding key, rule evaluation code, layout validation result, and memory warning. |
| **Device** | OS version, app build, permission state, free storage bucket, camera/GPS availability, clock-skew bucket, and background-task result. |
| **User action** | Non-sensitive action code such as `health_check_started`, `retry_sync_selected`, or `diagnostic_export_created`. |

Logs must be bounded by size and age, rotated automatically, and protected from modification by ordinary capture actions. They must never record passwords, access tokens, private inspection answers, client names, full serial numbers, signatures, raw certificate contents, or photo bytes.

## 4. Self-tests and safe repair actions

The app should expose a non-destructive **Health Check** that can be run by the inspector or support staff. It must never modify an inspection answer or delete queued work without a separately confirmed, authority-checked action.

| Self-test | Checks | Safe repair or next action |
| --- | --- | --- |
| Storage integrity | SQLite readability, free space, migration marker, and attachment references | Rebuild indexes or mark an orphan for review; never delete capture data automatically. |
| Queue integrity | Missing references, duplicate idempotency keys, and stale `sending` leases | Reset a stale lease to `pending`; quarantine only the affected operation. |
| Package integrity | Hash, signature, schema version, binding catalog, and renderer capability requirements | Re-download or roll back to the last known-good package. |
| Work-order consistency | Tenant, assignment, server revision, and local revision | Request a fresh authoritative work-order snapshot. |
| Device readiness | Permission state, camera/GPS availability, clock-skew bucket, and storage threshold | Explain the required device action; never bypass evidence or authority rules. |
| Sync probe | Reachability, TLS, authentication, and a non-mutating server health endpoint | Resume the queue only after the probe succeeds. |
| Renderer smoke test | Load a sandbox form package and exercise representative field types | Mark the package as incompatible and preserve the inspector’s capture data. |

Each health check produces a `health_check_id` and structured result codes. A health check is diagnostic evidence only. It cannot approve an inspection, alter an answer, bypass a required evidence rule, or change certificate authority state.

## 5. Fault classification and escalation boundary

The app classifies faults so the person responding knows whether a metadata change, server repair, device action, or application release is appropriate.

| Fault level | Example | Resolution path without an app release |
| --- | --- | --- |
| **Network/server availability** | Timeout, DNS failure, or HTTP `503`. | Pause affected queue operations, continue offline, and retry with bounded backoff. |
| **Authentication/session** | Expired token or HTTP `401`. | Block synchronization, request re-authentication, and resume using the same idempotent operations. |
| **Tenant/assignment** | Work order no longer belongs to the inspector or revision is stale. | Refresh authoritative assignment state and show a manager resolution path. |
| **Business conflict** | Asset already submitted, duplicate serial, or partial-work overlap. | Keep the operation in `conflict`; allow authorized office resolution without deleting the local record. |
| **Package/metadata** | Missing binding, invalid rule, unsupported field type, or bad layout. | Reject activation, retain the previous package, and publish a corrected signed package. |
| **Device** | Camera permission denied, storage full, GPS unavailable, or clock skew. | Guide the inspector through the device correction and record the resulting health check. |
| **Renderer/application code** | Native crash, data corruption, or a field type the renderer cannot safely interpret. | Preserve evidence, quarantine the affected operation/package, collect diagnostics, and assess a controlled app release. |

An app release is required only when the defect is in native application code or a dependency, a new hardware or renderer capability is required, or a critical security issue must be patched. A routine form, binding, rule, option, certificate cell, or layout correction must not require an app-store release.

## 6. Safe diagnostic export and remote support

The app provides a **Diagnostic Export** action for persistent faults. The export is designed to help support reproduce the state without exposing inspection content.

The export contains the app version, operating system version, device capability buckets, package IDs and versions, package verification outcomes, queue metadata, operation states, stable error codes, health-check results, and redacted timing information. It may include opaque correlation IDs and server receipt IDs when those are safe to disclose.

The export must exclude private inspection answers, client names, full asset serial numbers, signatures, certificate contents, access tokens, passwords, and raw photos. If support needs business data to resolve a conflict, that data must be requested through an authorized server-side support workflow rather than silently included in a device bundle.

The bundle should be encrypted, integrity-protected, given a short retention period, and associated with a support case or consent record. The inspector should see a summary of what will be exported before creating it. The app should provide a cancellation path and should not upload the bundle automatically without an explicit, authenticated action.

A support case should reference:

| Support artifact | Reason |
| --- | --- |
| `diagnostic_export_id` | Tracks the redacted bundle without exposing its contents in ordinary UI. |
| `health_check_id` | Identifies the self-test run used for diagnosis. |
| `correlation_id` | Connects local events to server-side request logs. |
| `package_id` and version | Identifies the exact metadata behavior in use. |
| Queue operation IDs and states | Shows what was pending, acknowledged, conflicted, or quarantined. |

The support team may respond by correcting server state, reassigning a work order, resolving a conflict, revoking a bad package, publishing a corrected package, or issuing a narrowly scoped recovery instruction. It must not rewrite an immutable issued certificate or silently erase a local queue record.

## 7. Recommended implementation sequence

1. Define the local entity interfaces and invariants in the Field architecture package.
2. Implement encrypted local storage for packages, work orders, captures, attachments, operations, and logs.
3. Implement the idempotent operation queue and synchronization leases.
4. Implement structured logging, redaction, bounded rotation, and correlation IDs.
5. Implement health checks and the non-destructive repair actions.
6. Implement diagnostic export, consent/retention handling, and support-case correlation.
7. Add server-side conflict-resolution endpoints and operational dashboards.
8. Conduct controlled offline, crash-recovery, package-revocation, duplicate-retry, and tenant-isolation tests before enabling production use.

## Final architectural outcome

INTEGIN’s Field app should be recoverable through **state, metadata, and evidence**, not through guesswork. An inspector’s work remains locally durable; queued operations remain inspectable; retries are idempotent; package changes can be delivered without routine app updates; and support receives a privacy-preserving technical explanation of failures. Server authority, tenant isolation, immutable issuance, and required evidence rules remain non-bypassable throughout diagnosis and recovery.

**No implementation or runtime exercise is claimed by this document.** It is the approved design baseline for the later Field storage, queue, diagnostics, and support implementation stages.
