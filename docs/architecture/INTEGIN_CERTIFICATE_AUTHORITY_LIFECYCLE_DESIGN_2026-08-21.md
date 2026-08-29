# INTEGIN Certificate Authority Lifecycle Design

**Status:** Approved implementation design; no certificate mutation, document rendering, public endpoint, or schema application is created by this record.

## Decision

The next server-authoritative slice will establish a tenant- and organization-scoped certificate lifecycle above a frozen canonical inspection revision and an approved certificate-template version. It will retain the current pure `internal/domain/certificate` transition model as a starting point, but it must add durable authority, eligibility, snapshot, numbering, and public-projection boundaries before any certificate can be issued.

The approved owner policy is **both profiles**: independent review is the default; senior self-issue is a narrowly governed exception. A request body cannot choose either profile, the reviewer, signer, issuer, tenant, organization, template, number, validity, public identifier, or exception. These are derived by the server from membership, persisted policy, canonical inspection state, and approved template state.

## Eligibility gate

A certificate draft may be created only when the resolved actor is authorized within the canonical inspection tenant and organization, the inspection lifecycle is `APPROVED`, its finalization state is `FINALIZED`, and the exact inspection revision, asset reference, and approved template version can be read in the same RLS-scoped transaction. The initial slice must reject voided, unfinalized, rejected, non-approved, cross-organization, missing, or revision-stale inspections. It must reject an already-active certificate for the same inspection/revision unless the request is an explicit governed supersession.

The current registry proves a template and nine inspection fields only. It does not establish asset serial/description, test scope, photos, evidence sufficiency, signer display identity, or document rendering. Therefore the first lifecycle slice must not claim evidence completeness or expose unavailable values. These become eligibility requirements only after their server-authoritative sources and binding keys exist.

## Authority profiles

| Profile | Default path | Required durable evidence | Prohibitions |
| --- | --- | --- | --- |
| `INDEPENDENT_REVIEW` | Inspector prepares; distinct authorized reviewer approves; distinct authorized signatory signs; authorized issuer issues. | Actor IDs, role/capability checks, review/signature/issue timestamps, inspection revision, template snapshot digest. | Inspector cannot approve, sign, or issue their own inspection certificate. |
| `SENIOR_SELF_ISSUE` | A specifically authorized senior inspector may review/sign/issue their own inspection only under an organization/template policy. | Policy identifier, exception reason, server-resolved capability, actor ID, decision time, complete immutable audit trail. | No client profile selection, no silent exception, no bypass of inspection eligibility, template approval, snapshot, numbering, expiry, or revocation rules. |

The existing pure aggregate currently rejects the inspector at signing and issue transitions. The implementation must deliberately reconcile that behavior with the governed self-issue profile through an explicit policy-aware domain method and focused negative tests; it must not weaken the independent default through a generic boolean or a client-provided exception.

## Immutable issuance content

At successful issuance, one transaction must persist: the certificate identity and allocated number; tenant/organization; canonical inspection ID and revision; asset ID; approved template identity and the exact template/cell snapshot; catalog revision; resolved cell-value snapshot and digest; policy profile and any governed exception; lifecycle actors/timestamps; validity calculation inputs and resulting expiry; and a random public verification identifier stored only as a digest. The original template and canonical inspection remain source records, while the immutable certificate snapshot proves what the issuer relied upon at issuance.

## Numbering, status, validity, and correction

Certificate numbers are allocated by a tenant/organization-scoped server sequence inside the issuing transaction and protected by a unique constraint. The status flow is `DRAFT -> PENDING_REVIEW -> APPROVED -> SIGNED -> ISSUED`, with terminal or corrective outcomes `EXPIRED`, `REVOKED`, and `SUPERSEDED`. Validity is derived from an approved template/inspection-type policy, not a global fixed duration; an absent approved policy blocks issuance. Revocation preserves the record and reason. Supersession links a new certificate to the prior issued certificate atomically and leaves the prior certificate verifiably superseded rather than deleted.

## QR-safe public projection

The public verifier resolves only an unguessable raw token, compares its digest, and returns a narrow no-store projection. For an issued certificate it may show certificate number, status, issue/expiry timestamps, inspection type only when later canonical, and the current asset ID. It must not expose evidence bytes, evidence-object keys, internal actor IDs, internal tenant/organization identifiers, audit reasons, draft data, verifier-token hashes, or private source snapshots. Full serial number, full asset description, and completed-test scope are mandatory future verifier fields requested by the owner, but they remain unavailable until the authoritative asset/test sources and approved binding keys exist. A non-issued, revoked, superseded, expired, missing, or cross-scope record returns its truthful status without supplying a private reason or existence oracle beyond the public certificate token.

## Initial persistence model

| Record | Purpose | Core constraints |
| --- | --- | --- |
| `certificate_policy` | Approved template/inspection-type policy, profile eligibility, validity rule, and public projection policy. | Forced RLS; immutable version after approval; no executable expression or arbitrary duration script. |
| `certificate_sequence` | Tenant/organization number allocation state. | Forced RLS; locked in issuing transaction; unique certificate number remains the final guard. |
| `certificate_record` | Authoritative lifecycle, canonical inspection revision, asset reference, template version, number, status, actor/timestamp fields, validity, public token digest, revocation/supersession links. | Forced RLS; foreign-key containment; unique active certificate per inspection/revision policy; append-only audit events. |
| `certificate_snapshot` | Exact approved template/cell and resolved value snapshot plus digest. | Immutable one-to-one issued-record snapshot; server-written only. |
| `certificate_audit_event` | Authority decisions and actor-derived policy evidence. | Append-only; forced RLS; no evidence bytes, raw public token, secrets, or unverified client claims. |

All certificate mutation endpoints will derive tenant, organization, actor ID, roles/capabilities, and allowed profile from validated OIDC plus local membership. They will decode unknown JSON fields as an error and will treat values in a body only as bounded identifiers/reasons requiring server validation.

## Acceptance matrix

| Claim | Positive proof | Required negative proof |
| --- | --- | --- |
| Eligibility is canonical and scope-safe | Eligible finalized/approved inspection creates only a draft. | Missing, cross-organization, unfinalized, voided, rejected, non-approved, or revision-stale inspection creates nothing. |
| Independent review is the default | Distinct authorized preparation/review/sign/issue flow succeeds. | Inspector self-approval, self-signature, and self-issuance fail without a policy exception. |
| Senior self-issue remains exceptional | An authorized policy-scoped senior self-issue creates a complete audit trail. | Missing policy, missing capability, inactive template, missing reason, or client-selected exception fails before mutation. |
| Snapshot and number are immutable | Issuance stores exact approved template/value digests and one allocated number. | Later template/inspection edits cannot alter the issued snapshot; duplicate number/active revision fails atomically. |
| Status corrections are truthful | Revocation, expiry, and supersession persist complete bounded reasons and links. | Illegal state transition, cross-scope correction, replacement collision, or deletion attempt fails. |
| Public verification is safe | Correct raw token returns only the allowed projection. | Guessable/missing/wrong/cross-scope/revoked token leaks no private facts, object references, or authority history. |

## Review outcome and implementation boundary

A complex/deep architecture-and-testing review was selected. Independent persistent reviewers are not available in this environment, so the owner performed two bounded passes: an architecture/authority pass and a verification/negative-case pass. The resulting design is accepted for the next source and migration-design slice because it preserves the existing certificate-first decision, creates no release authority yet, and names all conditions that must be proven before a lifecycle-complete claim. The next action is to inspect repository migration/test conventions, draft the lifecycle schema and rollback candidate, and review them on a disposable restored copy before any pilot application.

## Controlled implementation proof — 2026-08-21

The initial lifecycle persistence slice is implemented and partially proven. Migration `0013` passed an isolated restored-copy up/down review before pilot application, with five forced-RLS lifecycle tables. The controlled pilot proof passed the independent draft-to-issued sequence, immutable snapshot and number allocation, hashed public-token storage, duplicate-issuance denial, correct/wrong token behavior, revocation, public revoked status, dependency-ordered cleanup, and final full regression/vet. The proof does not establish an HTTP public portal, rendering, external authority recognition, senior self-issue at PostgreSQL level, expiry, or supersession; these remain separate acceptance boundaries.

## Complete lifecycle persistence proof — 2026-08-21

The repository-level lifecycle acceptance matrix is now closed for the current canonical scope. Controlled proof covered the independent default path, senior self-issue exception, approved/finalized inspection eligibility, approved template/policy eligibility, number allocation, snapshots, issue, revoke, supersede, expire, narrow digest-backed public projection, negative repeats/cross-scope behavior, cleanup, and final regression/vet. The accepted result remains strictly below HTTP/public portal, PDF rendering, QR image generation, background expiry scheduling, real asset serial/test-scope projection, and external-recognition authority.
