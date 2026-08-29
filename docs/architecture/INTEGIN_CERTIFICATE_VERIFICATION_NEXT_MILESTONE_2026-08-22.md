# Certificate Verification Next Milestone — Guarded Implementation Sequence

## Baseline completed

The current certificate lifecycle and transport slice has controlled evidence for RLS-scoped issuance, immutable template/inspection snapshots, one-time digest-backed public tokens, narrow public projection, public handler `no-store` semantics, in-process limiter behavior through the shared mux, and OIDC authentication mapped to local tenant/org membership for mutation authority.

## Ordered implementation sequence

| Order | Deliverable | Completion condition | Gate before moving on |
|---:|---|---|---|
| 1 | Canonical asset model decision | Tenant/org-scoped source for asset ID, serial, description, and type is selected; owner/edit authority and transfer history are defined | No existing field is assumed canonical merely because it is convenient |
| 2 | Inspection public-scope model decision | Structured, policy-approved inspection type and scope/result taxonomy is defined for the exact inspection revision | No free-form checklist, notes, evidence, or internal findings reach public data |
| 3 | Reviewed migration candidates | Add canonical entities/bindings with forced RLS, foreign keys, validation, and reversible review migration | Disposable up/down review and cross-org denial proof pass |
| 4 | Certificate policy and snapshot extension | Approved policy chooses permitted public fields; issuance atomically snapshots selected values and hashes them | Mutable asset/inspection change cannot rewrite an issued certificate display |
| 5 | Public verifier extension | Allow-list response includes only approved snapshot fields; unknown, revoked, superseded, and expired behavior stays truthful | Focused HTTP, repository, and mux tests pass; no hidden fields leak |
| 6 | PDF/QR renderer adapter | Fixed-cell PDF is generated from immutable snapshot; QR encodes the narrow verification URL | Overflow, QR decode, artifact digest, and status-label cases pass |
| 7 | Artifact storage and access | Rendered bytes reside in RustFS; PostgreSQL stores metadata and digest; authenticated artifact retrieval is isolated from public verifier | Evidence/retention and cross-tenant denial proofs pass |
| 8 | Public-release readiness review | Shared rate/abuse control, observability, incident response, custom-domain/TLS plan, and owner approval are evidenced | Explicit release decision; no silent external availability |

## Non-negotiable safety rules

Every migration and runtime operation retains tenant/org predicates even where RLS is present. Any certificate issuer sees public facts frozen at issuance, not current mutable asset state. Any QR points only to the limited verification route. An external regulator or commercial-system adapter remains disabled until its separately approved contract, credentials, data mapping, and retry/audit behavior have been implemented and reviewed.

## Immediate engineering starting point

Begin with the canonical asset and inspection-public-scope data-model design. Do not start a PDF renderer, QR generator, or public data response expansion before the snapshot source and policy binding are present and proven.
