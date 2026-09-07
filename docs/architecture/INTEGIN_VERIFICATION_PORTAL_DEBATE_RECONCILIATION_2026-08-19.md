# INTEGIN Verification Portal — Debate Reconciliation

**Decision date:** 2026-08-19  
**Status:** Accepted as a staged, future post-manifest design.  
**Scope:** Read-only verification of selected INTEGIN-issued inspection/report and training-competence outputs.  
**Non-interference:** No source, database, runtime, QR, certificate, customer data, external connection, or active workflow was changed.

## Decision

Adopt a separate **read-only verification portal**. A QR code or manual verification reference printed on a selected future document resolves to a minimal, current, server-authoritative status view. The portal exists to verify a document—not to search records, download private evidence, issue a certificate, approve a record, alter data, renew an outcome, or act as a regulator portal.

All three independent lenses—trust/integrity, privacy/assurance, and adversarial fraud/field usability—returned `adopt_with_stages` at high confidence.

## What a verifier may see

| Field | Purpose |
| --- | --- |
| Verification reference | Lets the verifier match the printed reference to the portal result. |
| Document class and controlled scope/type | Identifies the kind of document without exposing full inspection content. |
| Current status | Communicates `valid/current`, `expired`, `revoked`, `superseded`, `not found`, or `pending external acceptance`. |
| Issue/expiry and verification time | Shows the controlled date status and when the portal checked it. |
| Version/integrity indicator | Helps identify a superseded or altered printed document. |
| Asset identity match | Shows the certificate-bound asset ID and the approved serial/unique-identifier display so the verifier can compare it with the physical item and printed document. |
| Full equipment description | Shows the issued document's full equipment type, manufacturer/model, capacity/rating, and other approved identifying description fields. |
| Completed inspection/test scope | Shows exactly what was completed and the controlled outcome, including test methods such as `load test performed`, `load test not performed`, `load test not applicable`, or `routine visual inspection only`. |
| Internal/external authority label | Makes clear whether the document is internal/company-issued or has a verified external reference. |

External accepted/registered status is displayed only after an authorized integration returns a verifiable official reference/status. Internal pending status is never represented as external approval.

## Asset-identity substitution resistance

For physical-equipment certificates, the public page must display the certificate-bound **asset ID** and a controlled asset serial/unique identifier. The printed certificate must show the same values in a protected certificate identity block. A verifier compares the physical label/serial, the printed certificate, and the portal result; a certificate relabelled from `ID-01` to `ID-02` will fail that comparison because the portal still resolves to the source asset bound when the document was issued.

The default display rule should be the full asset ID plus the full serial number where the serial is already printed on the equipment and certificate. If a tenant has a legitimate confidentiality concern, the portal may show the asset ID plus a serial suffix and require the verifier to enter the physical serial for a match/no-match response. That privacy-preserving mode must never weaken the certificate-to-asset comparison. Equipment identity is treated differently from training identity: no person name/photo disclosure is implied by this design.

The portal must also expose the **full issued equipment description** and an exact, normalized **completed inspection/test scope**. A generic title such as “equipment inspection certificate” is insufficient. The scope block must distinguish, for example, `routine inspection completed — no load test performed`, `load test completed at [approved test load]`, `load test not applicable to this equipment/class`, and `load test required but not completed`. It should show only controlled completed facts, not let a viewer infer that a test occurred merely because a certificate exists.

Asset identity and completed test scope must be bound into the immutable issued-document payload and its integrity reference at issuance. The user must not be able to edit only visible PDF text—such as changing `ID-01` to `ID-02` or adding “load test”—and retain a valid portal identity/scope result. A later asset rename, location change, metadata correction, or reinspection creates traceable record history; it does not silently rewrite the issued certificate's bound identity or completed-test scope. A reissued/superseded document receives its own reference/version and the portal states the current status.

## What the portal must never disclose

The public portal must not expose full documents/PDFs, inspection answers, photos/evidence, asset identifier/location history, client account data, commercial prices/invoices, personal contact or national-identification data, sensitive training history, audit logs, internal notes, credentials, secrets, or non-public regulator correspondence.

Training certificates need a separate tenant privacy decision. The default is no public photo or full name. A future privacy-preserving confirm-match flow or holder-consented reveal may be considered only after policy, consent, and retention rules are approved.

## Mandatory design safeguards

1. A separate verification domain/API that is read-only and has no issuance/admin endpoints.
2. High-entropy non-sequential references and QR codes containing only the canonical HTTPS verification URL/reference, never PII or document content.
3. Tenant-isolated, minimal verification views backed by server-authoritative version/revocation state.
4. Generic lookup behavior, strict rate limits, progressive anti-enumeration controls, abuse monitoring, and no public search/batch/prefix lookup.
5. No-store/no-index response controls and cache/revocation behavior that prioritizes current online status.
6. Clear status language for valid, expired, revoked, superseded, pending external acceptance, external acceptance, external rejection, and unknown/not found.
7. Explicit internal/company-issued versus externally verified labeling; never a self-asserted regulator claim.
8. An immutable issued-document identity binding that includes certificate type, full asset ID, full serial/unique identifier, full equipment description, template/version, exact completed inspection/test scope, issue state, and integrity reference; public identity/scope display must remain comparable to the printed certificate and physical asset.
9. Privacy-minimized verification access logs, key rotation/compromise runbooks, phishing/takedown process, accessible low-bandwidth multilingual UI, and independent security/privacy review before broad rollout.

## Rejected shortcuts

Sequential IDs; QR codes containing tenant IDs, asset IDs, names, or evidence; public document download; name/site/customer search; public batch verification; admin/issuance functions on the verification domain; unverified external status; automatic renewal; long-lived offline proof without revocation/freshness controls; visible document text that can be altered without a portal asset-identity or inspection-scope mismatch; and training photo/full-name disclosure by default are all rejected.

## Staged future sequence

| Stage | Outcome |
| --- | --- |
| 0 | Threat model, domain/tenant isolation, status vocabulary, privacy policy, reference/key strategy. |
| 1 | Read-only MVP: non-guessable reference + QR, full asset ID/serial/description, certificate-bound completed inspection/test-scope block, core statuses, no-store/no-index headers, rate limits, audit-minimized access logging. |
| 2 | Supersession/revocation integrity and training-certificate privacy controls. |
| 3 | Abuse hardening, phishing guidance, accessible multilingual low-bandwidth UX, and operational monitoring. |
| 4 | Optional short-lived offline verification only with signed status, explicit expiry, refresh/revocation policy, and clear online-precedence wording. |
| 5 | Authorized regulator adapters with official onboarding and verified external status mapping. |
| 6 | Independent privacy/security review, enumeration/cross-tenant red team, field gatekeeper usability validation, and controlled rollout. |

## Open decisions

The implementation design must decide reference length/entropy and manual-entry format; signing/HMAC/public-key approach; full serial display rules by equipment class; the canonical full equipment-description field set; normalized vocabulary/structure for completed test methods, test loads, test standards, not-performed/not-applicable states, and scope limitations; issuer-managed versus client-managed asset IDs; single versus tenant-branded canonical domain; offline verification need/TTL; revocation reason disclosure; training identity consent policy; verification-log retention; rate-limit/CAPTCHA thresholds; language/accessibility; and official adapter authentication/failure semantics.

## Residual risks

No portal can fully prevent phishing, users accepting an unscanned printout, short periods of stale offline status, distributed enumeration attempts, leaked printed references, key compromise, regulator outages, or tenant-isolation implementation bugs. The accepted controls reduce these risks and make their status visible; they do not claim to eliminate them.
