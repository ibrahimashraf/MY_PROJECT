# INTEGIN Certificate and Export Sequence Decision Review

## Decision owner and boundary

The **INTEGIN owner** is the decision owner. This review does not issue a certificate, approve an evidence export, transmit a package, call an external authority, change data, or create a release-capable endpoint. It determines only the next bounded implementation order after the completed Work-Order, inspection, evidence-retention, and authenticated evidence-registration foundations.

## Options under review

| Option | General next authority boundary | Core decision controlled |
|---|---|---|
| A — certificate-first | Implement certificate review, approval, issuance, expiry, QR verification, revocation, and supersession. | Whether an inspection outcome becomes an official client-facing certificate. |
| B — export-first | Implement audited approval of a sealed evidence manifest for a defined recipient and purpose. | Whether a specific verified evidence package may leave INTEGIN. |

## Non-negotiable constraints

1. **A certificate must never be issued merely because files were uploaded or an export was approved.** It requires its own inspection, template, review, signatory, and policy conditions.
2. **An export must never release raw records solely because a certificate exists.** It requires recipient, purpose, scope, privacy/retention/redaction, and approval controls.
3. **External authority adapters may impose a submission-before-recognition sequence.** This is a per-integration exception and must not rewrite the general INTEGIN certificate lifecycle.
4. **Neither option may bypass server-derived identity, tenant/organization isolation, immutable evidence verification, or the owner’s later external-registration policy.**
5. **No external authority integration is assumed.** The review does not claim SASO, SAAC, ZATCA, client-platform, or other external compliance.

## Evidence threshold for recommendation

The recommendation must distinguish the normal operational flow from a future external-authority exception; state the resulting user/client workflow; identify which risk is controlled first; list what remains unproven; and name the next bounded implementation artifact without treating the design review as release authorization.

## Bounded review perspectives

| Lens | Certificate-first finding | Export-first finding | Implication |
|---|---|---|---|
| Inspector, client, and site workflow | The certificate is the ordinary deliverable after inspection. It supplies a clear client/site outcome and supports QR verification without disclosing internal evidence. | Export is an exception-driven action: a client, auditor, insurer, or platform asks for deeper support. It creates recipient/purpose/privacy choices the inspector should not make by default. | The normal operational path should reach a controlled certificate before any optional evidence release. |
| Assurance and disclosure control | Certificate issuance controls whether a reviewed inspection decision can be represented as official. Its main risks are improper approval, invalid template/policy state, expiry, and revocation. | Export approval controls whether sensitive supporting information can leave INTEGIN. Its main risks are over-disclosure, wrong recipient/purpose, changed manifest scope, and releasing evidence that is later superseded. | These controls are distinct. Certificate approval must not imply export approval, and export approval must not imply certificate issuance. |
| External integration and future authorities | A locally issued certificate can be internally valid and QR-verifiable even before any external integration exists. | Some future authority adapters may require a reviewed evidence package before they accept, recognize, or register a certificate. That is an adapter-specific submission sequence. | Model an export-approval capability after certificate issuance, but allow a future integration policy to require an approved export before external transmission or external recognition. |

## Reconciliation

The recommended **general INTEGIN build and workflow order is certificate-first**:

> Assigned Work Order → canonical inspection → verified evidence → internal review/approval → certificate issuance → QR/client verification → optional export approval → external transmission or recipient access.

The earlier apparent contradiction is resolved by separating the general lifecycle from an external adapter sequence. A future regulator/client platform may demand an evidence-package submission before it recognizes a certificate. In that specific adapter flow, INTEGIN prepares and approves the export before sending it externally. The local certificate workflow remains a separate authority boundary; the adapter does not rewrite its normal implementation order.

## Recommended next bounded action

Design and implement the **certificate issuance authority boundary** first: eligibility from canonical inspection state, template snapshot and layout binding, reviewer/signatory separation, certificate identity/status/expiry, QR-safe public projection, revocation, and supersession. Do not implement external transmission, export release, or external-registration bypass inside that boundary.

## Deferred export-approval boundary

After certificate issuance is evidenced, implement export approval as a separate release ledger over a sealed verified manifest. It must carry recipient, purpose, manifest checksum, included evidence scope, privacy/retention/redaction policy, requested-by and approved-by identities, validity/withdrawal state, and immutable audit history. No export endpoint or external connector is authorized by this review.

## Residual risk and owner confirmation

This recommendation is a design sequence, not a release authorization. It does not decide the owner’s final approver matrix, commercial/legal policy, external platform requirements, or when a manager may override external registration. Those policy values must be confirmed before implementing certificate issuance or export approval mutations.
