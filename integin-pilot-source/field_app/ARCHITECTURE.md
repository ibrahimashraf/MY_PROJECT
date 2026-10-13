# INTEGIN Flutter Field App Architecture

## Purpose

The field app is a native Flutter client for assigned inspection work. It is designed to remain useful with no network connection while preserving the server-authoritative boundary: the app can capture signed, attributable work, but it cannot issue certificates, approve inspections, change authorization, or publish public verification data.

## Module boundaries

| Module | Responsibility | Explicit non-responsibility |
|---|---|---|
| `domain` | Immutable value objects, enums, validation, canonical payloads, and sync outcomes. | No network calls, persistence, or UI state. |
| `authorization` | Local preflight checks for tenant, organization, capability, scope, environment, workflow, device trust, and authority expiry. | It cannot grant authority; the server remains authoritative. |
| `inspection` | Work-pack presentation, guided checklist capture, measurements, findings, evidence metadata, local completeness checks, and draft status. | It does not compute authoritative certificates or approvals. |
| `outbox` | Append-only durable mutations, stable transaction identifiers, sequence allocation, payload hashes, signatures, and retry state. | It never silently deletes failed or held work. |
| `sync` | Connectivity-aware upload, idempotent retry, explicit Applied/Duplicate/Held/Rejected/Conflict/Security Failure states, and reconciliation UI. | It does not resolve business conflicts by overwriting server state. |
| `storage` | Encrypted local database, secure key storage, evidence file references, and retention/cleanup rules. | It does not store unrestricted tenant data or public QR secrets. |
| `presentation` | Offline/online status, last sync, authority expiry, queued/failed work, inspection flow, and persistent security notices. | It does not expose primary decision mutation controls. |

## Durable data model

Every local record carries `tenantId` and `organizationId`; repositories require both values and reject mismatches before reading or writing. The minimum offline transaction envelope mirrors the Go sync engine: transaction ID, tenant ID, device ID, user ID, monotonically increasing sequence number, operation/entity, canonical payload, payload hash, captured time, authority package ID/epoch, and signature.

Inspection drafts retain the assigned work-pack snapshot, procedure/template version, asset identity, checklist responses, measurements, findings, evidence references, notes, local completeness result, and current local status. A submitted revision is immutable; a returned inspection begins a new revision rather than rewriting the prior snapshot.

## Security model

The app stores device keys and authority material in platform secure storage. The authority package is checked locally for identity binding, tenant/organization binding, trusted device state, epoch, scope, capability, procedure context, issue time, and expiry. A revoked, restricted, locked, retired, or stale-epoch device cannot create an offline mutation. No client-side advisory result can alter inspection state.

## Connectivity and sync state

The user-visible state machine is `online`, `offline`, `syncing`, `degraded`, or `blocked`. The app always shows last successful sync, authority expiry, queued count, held count, failed count, local storage health, and the action required for recovery. Outbox records remain inspectable after network or server failure.

## Persistence strategy

The storage interface is platform-neutral. The mobile implementation will use an encrypted SQLite-backed adapter, while Flutter Web will use a browser-compatible adapter behind the same repository interface. This keeps the domain and sync rules shared while allowing platform-specific persistence and evidence-file handling.

## Delivery sequence

The first implementation slice is intentionally narrow: load an assigned work pack, capture findings and evidence metadata offline, validate locally, append a signed outbox transaction, and render explicit sync outcomes. Review, approval, certificate issuance, public QR, and AI advisory display remain server-side or secondary-console responsibilities.
