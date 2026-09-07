# INTEGIN Production Device Enrollment Specification

**Status:** Draft v0.1 — design-review required before implementation  
**Date:** 2026-08-15  
**Owner:** Security Operator and Engineering Change Owner  
**Applies to:** The authoritative Go service, PostgreSQL device-trust state, Flutter field clients, and the selected self-hosted OIDC provider.

## 1. Decision and safety boundary

This specification replaces the development-only `localprovision` bridge with a production design. The local bridge is deliberately constrained to loopback integration, must remain disabled in every shared or production deployment, and must never be extended into a production endpoint.

> **A field device may create a key pair and request enrollment, but only the server may trust a device, issue offline authority, or accept a signed offline mutation.**

The production design preserves four separate truths. The OIDC provider authenticates the human or service identity. The Go authority resolves membership, role, tenant, organization, policy, device state, and workflow permission. PostgreSQL is the durable record for that authorization decision and its audit trail. The Flutter client retains its private Ed25519 key solely in platform secure storage.

| Concern | Authoritative component | Explicitly not authoritative |
|---|---|---|
| Human authentication and MFA | Self-hosted OIDC provider | Flutter client, Go session cookies, device key |
| Tenant and organization membership | Go authority + PostgreSQL | OIDC group claim alone, client input |
| Device identity and trust lifecycle | Go authority + PostgreSQL | Device ID chosen by client, object storage |
| Offline authority package | Go authority signature | OIDC access token, Flutter local state |
| Inspection, certificate, and audit workflow | Existing Go domain/application services | AI service, field client, identity provider |

## 2. Scope and non-goals

The scope is enrollment of a human-operated field device capable of producing signed offline mutations. It covers key generation, proof of possession, tenant-administrator approval, authority issuance, revocation, recovery, auditability, and cross-tenant isolation.

It does **not** make the device an OIDC client with a client secret, permit a device to approve itself, permit private-key transport, replace the existing sync signature verification, or make the identity provider the system of record for INTEGIN tenant authorization. It does not allow an AI service to recommend, approve, or revoke a device.

## 3. Stable identifiers and minimum records

All values in this table are server-defined or server-validated. A caller may provide descriptive device metadata, but no caller-selected identifier can confer authority.

| Identifier / record | Created by | Purpose | Handling rule |
|---|---|---|---|
| `oidc_subject` | OIDC provider | Immutable human/service identity | Mapped to a INTEGIN identity subject; never accepted from the request body. |
| `tenant_id`, `organization_id` | Go authority | Tenant and organization boundary | Resolved from active membership; never trusted from a client claim alone. |
| `enrollment_request_id` | Go authority | Idempotent enrollment workflow record | Random, opaque, time-bounded, and tenant-scoped. |
| `device_id` | Go authority | Durable field-device identity | Created only after proof verification and approval. |
| `key_id` | Derived from public key | Stable non-secret key reference | Derived by a documented SHA-256-based function; uniqueness is enforced per tenant. |
| `authority_id`, `authority_epoch` | Go authority | Scoped offline authorization | Server-issued, expiring, and invalidated by revocation/epoch change. |
| `approval_id` | Go authority | Human approval evidence | Stores approver subject, time, reason, policy version, and correlation ID. |

The device must generate an Ed25519 key pair locally. The private key stays in the platform secure store and must be marked non-exportable wherever the operating system supports that policy. The client transmits only the public key, its derived key ID, and a proof signature. Backup, test fixtures, logs, audit records, analytics, and support bundles must never contain the private key.

## 4. Enrollment lifecycle

The server owns the state machine. A client may request a transition but cannot force it.

| State | Meaning | Permitted next states | Offline authority allowed? |
|---|---|---|---|
| `requested` | Authenticated user submitted public-key and device metadata. | `challenge_issued`, `rejected`, `expired` | No |
| `challenge_issued` | Server issued a short-lived, single-use proof challenge. | `proof_verified`, `expired`, `rejected` | No |
| `proof_verified` | Device has proven possession of the submitted private key. | `awaiting_approval`, `rejected`, `expired` | No |
| `awaiting_approval` | Identity, membership, key proof, and policy checks passed; human approval is outstanding. | `trusted`, `rejected`, `expired` | No |
| `trusted` | Device is permitted to receive policy-scoped authority. | `restricted`, `locked`, `revoked`, `retired` | Only with a current authority package |
| `restricted` | Device has a limited policy scope or requires remediation. | `trusted`, `locked`, `revoked`, `retired` | Only if a newly issued restricted authority explicitly permits the operation |
| `locked` | Temporary security or administration hold. | `trusted`, `revoked`, `retired` | No |
| `revoked` | Trust ended; all authority is invalid. | `retired` only | No |
| `retired` | Historical endpoint, retained for audit. | None | No |

No server process may convert a state directly from `requested` to `trusted`. Every transition records actor, reason code, policy version, captured time, correlation ID, and prior/new state in the tenant-scoped audit record.

## 5. Production protocol

### 5.1 Human sign-in and request creation

The field application authenticates a human through OIDC Authorization Code with PKCE. The Go authority validates the resulting access token and resolves the immutable issuer/subject pair against its own tenant membership projection. The application must ask the human to choose an authorized organization when more than one active membership exists; the selected organization is revalidated by the server.

`POST /v1/device-enrollment/requests` accepts only the following material:

| Field | Validation |
|---|---|
| `protocol_version` | Exact supported version; initially `v1`. |
| `public_key` | Canonical Ed25519 public-key encoding and length. |
| `key_id` | Must equal the server-derived value for `public_key`. |
| `device_attestation` | Optional at v1; retained only if a supported platform attestation verifier approves it. |
| `device_metadata` | Allowlisted OS/app/version/model fields, bounded in size, treated as untrusted descriptive data. |
| `idempotency_key` | Bound to the authenticated subject, organization, public-key ID, and request body hash. |

The endpoint does not accept `tenant_id`, `organization_id`, user ID, role, approval state, authority scope, authority duration, or device state as client-controlled fields. It returns a server-generated request ID and a single-use challenge after validating policy, duplicate-key rules, and request rate limits.

### 5.2 Proof of possession

`POST /v1/device-enrollment/requests/{request_id}/proof` requires the same OIDC subject and active membership used for request creation. The device signs a canonical versioned enrollment-proof envelope containing:

```text
INTEGIN-DEVICE-ENROLLMENT/v1
request_id
server_challenge
key_id
public_key_sha256
oidc_issuer
oidc_subject
organization_id
challenge_expires_at
```

The server generates the challenge with cryptographic randomness, stores only a hashed challenge value, marks it single-use, and expires it after five minutes. It verifies the signature against the submitted public key before changing the request to `proof_verified`. A request can never be replayed under another subject, tenant, organization, key, or challenge.

### 5.3 Human approval and trust creation

Approval is a separate authenticated server action: `POST /v1/device-enrollment/requests/{request_id}/approve`. The Go authority re-resolves the approver's membership and capability at the time of approval. It must verify the request belongs to the approver's tenant and organization, the proof remains current, the key is not revoked or already associated with an incompatible identity, and all applicable enrollment policy checks pass.

The default approver is a **Tenant Administrator**. A request seeking a high-risk capability, Security Operator role, elevated evidence export access, or an administrator's own device must use a second approver with the applicable role. Approval records must include an explicit reason and policy reference; a generic “approved” action is insufficient.

After approval, the server atomically creates the device record, transitions it to `trusted`, advances the device epoch, writes the approval/audit record, and registers the device with the active sync processor. The operation must use a PostgreSQL transaction and must not require a service restart.

### 5.4 Offline authority issuance

Trust does not automatically yield unbounded offline access. The server issues an authority package only after device trust and approval are current, with a scope derived from assignment, membership, competency, procedure/policy version, environment, and tenant policy.

| Authority field | Rule |
|---|---|
| Subject, tenant, organization, device, key ID | Bound server-side; all are mandatory. |
| Scope and allowed operations | Derived from server policy; never copied from a client request. |
| Authority epoch | Equals current device epoch and invalidates on epoch change. |
| Issue/expiry time | Server clock only. The initial production default is eight hours; a tenant policy may reduce it. A global server cap of 24 hours applies until the Safety Authority approves a revised policy. |
| Policy/procedure references | Immutable identifiers and versions, allowing later evidence review. |
| Signature | Produced by a server-managed signing key with versioned key ID and algorithm. |

The authority package may be returned to the authenticated field client over TLS after approval; it is not a private key or bearer credential. The client must verify the server signature before storing it and must show expiry, scope, and revocation/refresh status to the field user.

### 5.5 Revocation, recovery, and re-enrollment

Revocation is immediate for future server acceptance. `POST /v1/devices/{device_id}/revoke` requires the applicable tenant or security capability, a reason code, and an audit record. It increments the device epoch, invalidates current authority, stops future authority issue, and causes the sync processor to reject new offline mutations from the revoked authority.

Lost, replaced, or suspected-compromised devices must not reuse an old key. The recovery flow creates a new enrollment request with a new key and applies the same proof/approval process. The old device is locked or revoked first according to incident severity. Re-enrollment is not a way to silently clear an audit trail: linkage between replacement request and prior device is retained in the tenant-scoped audit record.

## 6. Persistence, RLS, and audit requirements

The production migrations must add append-only, tenant-scoped records for enrollment requests, enrollment challenges, proof-verification outcomes, approvals/rejections, key history, device state transitions, authority issue/revoke events, and recovery links. Challenges and rate-limit counters must have retention policies, but durable audit records must follow the approved retention/legal-hold policy.

Every repository operation must set the PostgreSQL tenant context inside the transaction before reading or writing. Global key-reuse detection requires a narrowly scoped security-owner query or a keyed irreversible public-key fingerprint index; it must not expose another tenant's device or identity data to a tenant administrator.

## 7. Abuse controls and security tests

| Threat | Required control | Required acceptance evidence |
|---|---|---|
| Private-key extraction | Public-key-only endpoints; secure storage; log/data-export scanning | Source, integration, and support-bundle tests show no private key leaves device storage. |
| Enrolling another user's device | OIDC subject binding, proof challenge, approver authorization | Subject/organization swap attempts are rejected. |
| Challenge replay | Single use, short expiry, hashed storage, request/key/subject binding | Replay and expiry tests are rejected deterministically. |
| Cross-tenant enrollment | Server membership resolution and RLS transaction context | Cross-tenant request/proof/approval/revoke tests fail closed. |
| Unauthorized authority expansion | Server-derived scope and duration only | Client-supplied scope/duration has no effect; max-duration test passes. |
| Administrator abuse | Separation of duties for privileged/self approval and audit | Self-approval and insufficient-role tests fail; approver record is retained. |
| Duplicate or recycled key | Tenant/device key history and policy checks | Same key/different subject and revoked-key reuse tests fail. |
| Offline device after revocation | Epoch binding and server sync checks | Authority issued before revocation is rejected after revocation. |
| OIDC outage | Existing valid authority remains bounded by expiry; no new trust/authority issue | Outage test proves no fail-open enrollment or authority refresh. |

## 8. Release gate

Production enrollment cannot be enabled until the following are complete: OIDC implementation and token-validation tests; migrations and RLS tests; canonical proof test vectors for Go and Flutter; approval/rejection/revoke/recovery tests; adversarial replay/cross-tenant/duplicate-key tests; operator runbooks; backup/restore verification for enrollment records; OpenAPI v1 publication; and an approved rollback plan.

The enablement flag is environment-specific and defaults to **disabled**. The legacy local-provisioning flag and production enrollment flag must be mutually exclusive in configuration validation.

## References

[1]: ./PLATFORM_GOVERNANCE_BACKLOG.md "Production device-enrollment design gate"

[2]: ./ARCHITECTURE_EVOLUTION_ROADMAP.md "Technology ownership, safety contracts, and staged evolution"

[3]: https://www.keycloak.org/docs/latest/server_admin/index.html "Keycloak Server Administration Guide"
