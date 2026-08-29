# INTEGIN High-Level Operating Model Review

## Purpose and owner boundary

This owner review examines the whole INTEGIN operating model, not only the immediate choice between certificate issuance and evidence export. It is a design and roadmap review. It does not change live workflow authority, issue a certificate, expose evidence, transmit a record, create an invoice, or represent compliance with an external authority.

## Reconstructed operating model

INTEGIN is not one straight line. It is a controlled system of linked but distinct tracks. A Work Order gives an inspector legitimate scope; inspections create canonical facts; evidence supports facts; certification represents an official conclusion; exports govern disclosure; verification answers what outsiders may see; commercial handoff prices completed work; and external integrations are optional adapters governed by a separate policy.

| Track | Primary question | Current proven foundation | Not yet an approved release authority |
|---|---|---|---|
| Job and field execution | Who may inspect which client, location, asset, and scope? | Server-authoritative Work Orders, assignment scope, canonical inspections, partial submission, reassignment, RLS, and offline recovery foundations. | Work-order completion does not issue a certificate or invoice. |
| Evidence and assurance | What supports the inspection conclusion, and can bytes still be trusted? | Tenant-scoped metadata, immutable registration, object re-verification, sealed manifest projection, recovery drill, and authenticated metadata ingress. | Stored or sealed evidence does not itself approve a result. |
| Certification | May INTEGIN represent the result as an official certificate? | Existing certificate domain concepts only; no newly approved issuance workflow in this Stage A slice. | No certificate issue, signatory, expiry, revocation, or supersession mutation has been created by this review. |
| Client and verifier access | What can an owner, site, or QR verifier safely see? | QR-verification platform foundation and user requirement for identity-rich verification projection. | Internal notes or full evidence must not be revealed merely by QR verification. |
| Evidence disclosure and external exchange | May a defined evidence package leave INTEGIN, to whom, and why? | Verified sealed manifest projection. | No release ledger, recipient-purpose policy, download/share endpoint, or external transmission exists. |
| Commercial handoff | What completed scope may be billed, summarized, or handed to invoicing? | Work-order and partial completion foundations. | No invoice approval, ZATCA process, pricing, tax decision, or payment authority is created here. |
| External authority recognition | Does an external platform require a submission, review, or registration? | Adapter-ready principle only. | No SASO/SAAC/ZATCA/platform integration or compliance claim is established. |

## Review constraints

1. **Operational speed must not create a hidden authority shortcut.** Bulk entry, cloning, partial work, reassignment, offline onboarding, and senior-inspector exceptions need explicit status and audit behavior.
2. **Assurance, disclosure, commercial, and external-recognition decisions are separate.** No status in one track automatically grants authority in another.
3. **Templates and rules must drive the flow.** Equipment type, inspection class, client contract, risk level, evidence rule, external requirement, and certificate template may alter required gates without requiring an app release.
4. **The owner controls policy selection.** INTEGIN must make strict/default/exception policies visible rather than silently forcing a presumed regulatory or commercial path.
5. **Every exception must be explainable.** The system records why it was allowed, by which policy version, by whom, for how long, and whether later review is required.

## Review questions

The remaining phases reconcile: the default end-to-end lifecycle; bulk/high-volume and multi-inspector handling; evidence requirements and no-photo outcomes; certificate review/signature and self-issuance; export and external recognition; QR/client visibility; reminders and renewal demand; office confirmation/invoicing handoff; and the exception, revocation, and dispute model.

## Multi-lens assessment

### 1. Field operation and high-volume execution

The inspector’s primary working object should be the **Work Order workspace**, not a collection of unrelated certificates. The workspace contains location groups, scope items, and inspection records. A location can be collapsed or expanded; an asset row opens the form appropriate to the selected asset type; cloning creates a new editable record with a new identity and selected destination location; and partial submission can synchronize ready records without forcing incomplete work to be finalized. The Work Order may remain open across days and across inspectors.

High-volume execution needs a **record factory**, not a spreadsheet substitute: add asset, select type, open dynamic form, save, return to location group, clone, correct, remove unstarted draft, and submit independently ready records. Import can exist later as a controlled staging mechanism, but imported rows must enter the same validation, review, evidence, and approval states as manually entered rows. No bulk operation may generate a certificate merely because a row is imported or cloned.

| Decision | Default | Configurable exception | Prohibition |
|---|---|---|---|
| Work allocation | A Work Order carries client, locations, scope, assigned inspector(s), contract/job context, and required template/policy references. | Emergency offline provisional work may be created by an enabled inspector and requires later office reconciliation. | A Field user cannot silently create a final client, Work Order, or certificate authority while offline. |
| Inspector closeout | An inspector closes their own work session or assigned share, creating an immutable work summary/timesheet candidate. | Multiple inspectors close separately; Work Order closure waits for the required combined condition. | One inspector’s closeout must not falsely close another inspector’s remaining scope. |
| Bulk and clone | Clone copies permitted operational values into a new editable record with a new ID. | Destination location and every copied value may be changed before completion. | Clone must never copy certificate identity, approval, signature, evidence ownership, or a pass decision as an unreviewed final fact. |

### 2. Assurance, inspection facts, and evidence

Inspection facts, evidence, and conclusions require separate states. A form template and policy profile decide whether evidence is not applicable, optional, required on a condition, or required for a particular inspection type. A passing wire-rope or shackle inspection can therefore carry no photo; a rejected inspection may also have no photo when the applicable rule says photos are optional. The system records **why no photo is required**, rather than inventing an evidence requirement or treating its absence as a defect.

Evidence registered today is durable and verifiable, but signature/authority-package verification of device-origin claims remains a separate future gate. INTEGIN must distinguish: “object bytes were re-verified and linked to the inspection” from “an offline device signature or external authority statement was cryptographically validated.”

### 3. Certification and signatory control

Certificate issuance is an independent representation decision. It consumes selected canonical inspection revision(s), applicable template/version, asset identity, inspection/test result, required evidence condition, approved text/layout bindings, expiry calculation policy, and approval decision. It should normally issue one certificate per asset/inspection outcome, but a template policy may permit a controlled group or batch certificate when the business and applicable rules allow it.

The system should support three approval profiles rather than one hard-coded rule: **independent review required**, **senior-inspector self-issue allowed**, and **external review/recognition required**. The template/policy profile, client contract, risk class, and owner-selected rule determine which profile applies. Self-issue is not a shortcut: it needs a valid signatory capability, exception basis, audit trail, and optional later office review.

### 4. Client, owner, and QR-verifier experience

The public verifier should receive a deliberately minimal verification projection: certificate status, certificate number, asset ID, full serial number, full description, inspection type, applicable tests performed, issuer identity, issue/expiry dates, and revocation/supersession status. This is the correct protection against a client changing “01” to “02” on a PDF while the verifier cannot see the asset identity or test type.

The equipment owner/client portal can have a richer, permissioned view: asset fleet, certificates, expiry schedule, work order progress, requests, and approved reports. It still does not receive raw internal evidence by default. Evidence disclosure remains its own recipient/purpose approval decision.

### 5. Evidence disclosure, external exchange, and external recognition

An export package is a **release case** over a sealed evidence manifest. Its approval must state the recipient, purpose, permitted contents, policy profile, approver, expiry or withdrawal condition, and any external submission reference. A certificate does not automatically create or approve a release case.

External recognition is a third, distinct lane. A policy can declare that no external recognition is required, that an approved export must be transmitted before an external party recognizes a locally issued certificate, or that an external response must be received before INTEGIN may issue its certificate. The latter two are future adapter policies, not assumed Saudi or client requirements. No owner override should be hidden: the owner chooses whether a contract/policy follows the external process, and the system records the decision and its reason.

### 6. Office, commercial, and renewal operation

Field closeout produces an operational **Work Execution Summary**—dates, locations, quantities, inspection types, inspector effort, completion/exceptions, and optionally a certificate list. The office reviews that summary before commercial handoff. The invoice is a separate commercial record that may show a concise billed scope, date/days, quantities, and price; it may attach or reference the fuller work summary. It should not need to expose every raw inspection result unless the contract requires it.

Renewal/expiry monitoring becomes an opportunity and service-control track: the system identifies expiring or expired assets, allows office staff to filter/select them, propose a new or existing Work Order, and notify/queue client contact. A reminder is not an inspection request, Work Order, quote, invoice, or certificate; those follow their own approvals.

The future ZATCA handoff belongs after office commercial confirmation. ZATCA’s official description of structured electronic invoicing and system integration supports a dedicated invoicing adapter, not a direct coupling between inspection completion and tax invoice generation. [ZATCA](https://zatca.gov.sa/en/E-Invoicing/Introduction/Pages/What-is-e-invoicing.aspx)

### 7. Governance, resilience, and dispute handling

Every authoritative lifecycle needs reversible status changes: certificate revocation and supersession; export withdrawal; external submission correction/resubmission; invoice credit/cancellation under the commercial policy; and inspection amendment through a new revision rather than silent rewrite. The existing evidence, RLS, recovery, and correlation foundations support these future boundaries but do not replace their domain rules.

The owner should configure policy profiles by template/inspection class/client contract. Examples include required review, allowed self-issue, mandatory evidence condition, maximum certificate validity, whether public QR is permitted, whether export requires manager approval, whether external recognition is optional/required, and whether invoice creation waits for office confirmation. This makes the platform adaptable without hidden bypasses or routine app releases.

## Reconciled INTEGIN operating model

### Normal lifecycle

The normal lifecycle is deliberately certificate-first but not certificate-only. Each transition has a different authority owner and a different outcome.

| Step | Primary owner | Input | Controlled outcome | Does **not** automatically happen |
|---|---|---|---|---|
| 1. Plan work | Office/client-authorized requester | Client need, assets, location, job request, contract context | Work Order and scope are created, assigned, and made available to Field. | Inspection, invoice, or certificate creation. |
| 2. Perform work | Assigned inspector(s) | Assigned scope, template/policy profile, asset data | Canonical inspection records, drafts, findings, and optional/required evidence are captured. | Final approval, certificate issuance, or evidence disclosure. |
| 3. Synchronize and close field share | Each assigned inspector | Ready records and their own work session | Ready records synchronize; personal Work Execution Summary/timesheet candidate is produced. | Closing another inspector’s work or automatically closing the Work Order. |
| 4. Complete inspection package | Server policy plus assigned team | Canonical inspection revision, required template data, evidence condition | Record becomes technically complete or remains in correction/awaiting state. | Certificate issue or commercial approval. |
| 5. Review and issue certificate | Reviewer/signatory or valid senior self-issuer profile | Complete inspection package, approved template snapshot, applicable policy | Certificate is issued with identity, status, expiry, and immutable source references. | Release of raw evidence, external registration, or invoice issuance. |
| 6. Verify and serve client | Public verifier/client portal policy | Certificate identity or QR token | A safe status projection confirms the exact asset and inspection/test context. | Access to internal notes, raw evidence, or other client assets. |
| 7. Close office and commercial handoff | Office manager/commercial role | Consolidated Work Execution Summary, contract/pricing context, customer confirmation as required | Completed scope is accepted for quoting/invoicing workflow. | Tax invoice generation, payment collection, or external disclosure without commercial policy. |
| 8. Renew and retain | Office/customer-success policy | Certificate expiry and client asset portfolio | Reminder, service opportunity, or Work Order proposal is created. | Automatic Work Order, quote, invoice, or inspection. |

### Configurable exception policies

| Policy profile | What changes | Required record | Hard limit |
|---|---|---|---|
| Senior self-issue | A valid senior inspector may review/sign their own completed inspection. | Role/capability, policy basis, same-person indicator, and optional later office review. | It is prohibited where the applicable policy requires independent review. |
| Independent review required | An inspector cannot issue; a distinct reviewer/signatory must decide. | Reviewer identity, decision, reason, template/source revisions. | A manager cannot silently convert it to self-issue; a documented policy exception is required. |
| External-recognition required | A sealed package is approved and transmitted before an external party recognizes a certificate, or before local issue when the policy expressly says so. | Recipient, purpose, manifest checksum, adapter status, external response/reference. | External submission does not overwrite local certificate or inspection facts. |
| Evidence optional/no-photo | No evidence object is required for the result. | Template/policy condition explaining why absence is valid. | No user can turn off evidence where the selected policy marks it mandatory. |
| Emergency offline onboarding | Enabled inspector creates provisional client/Work Order context while disconnected. | Provisional identity, actor/device context, later office reconciliation result. | It cannot issue final certificates or represent a provisional client as office-approved until reconciled. |
| Multi-inspector work | Several inspectors execute portions of one Work Order. | Per-inspector assigned scope and individual closeout summary. | One inspector cannot mark the full Work Order closed while required work remains for another. |
| Reissue/correction | A certificate needs correction after issue. | Supersession/revocation reason and links old-to-new. | An issued certificate or evidence record is not silently edited in place. |

### Non-negotiable prohibitions

1. A pass/fail result, clone, imported row, uploaded object, evidence manifest, Work Order closeout, reminder, or invoice preparation does not on its own issue a certificate.
2. A certificate does not on its own approve evidence export, external authority transmission, a client portal disclosure, or invoice generation.
3. A QR verifier never receives the authority to alter a certificate and never receives evidence beyond the explicitly permitted verification projection.
4. An external platform response never silently rewrites canonical inspection results or certificate history; it is recorded as an external status linked to an immutable local version.
5. Owner overrides are visible policy decisions with actor, reason, scope, effective window, and audit evidence; they are not hidden backdoors.

### Owner policy choices still required

The system can support the following choices, but their initial values should be decided by the owner before authority mutations are built.

| Policy decision | Recommended initial default | Why it remains owner-controlled |
|---|---|---|
| Certificate approval model | Independent review by default; senior self-issue permitted for named policy profiles. | Risk class, contract, team capacity, and accreditation policy vary. |
| Certificate validity | Per inspection template/type, not globally hard-coded. | Validity can differ by equipment, service, contract, and jurisdiction. |
| Evidence requirement | Template/rule-driven: not applicable, optional, conditional, or mandatory. | The business needs no-photo pass and no-photo rejection cases without bypassing actual mandatory rules. |
| Public verification | QR projection shows certificate status, asset ID, serial, description, inspection type, tests, issuer, and dates. | The owner decides exact privacy fields, branding, and public exposure. |
| Export control | Manager/release-approver approval for every external evidence package. | Recipient, purpose, retention, redaction, and client agreement dictate disclosure. |
| External recognition | Disabled by default; enable per client/contract/template adapter policy. | No external body requirement or interface has been proven for INTEGIN yet. |
| Commercial handoff | Office confirms Work Execution Summary before invoice proposal. | Pricing, tax, credit, and client acceptance are commercial rather than inspection decisions. |
| Reminder behavior | Create an office action queue and optional client communication, never automatic billing or Work Order creation. | Outreach timing and consent are business-policy decisions. |

## Dependency-ordered roadmap

| Priority | Bounded capability | Why it comes here | Explicitly excluded from this slice |
|---:|---|---|---|
| 1 | **Certificate authority lifecycle** | It turns a complete verified inspection into the primary client-facing outcome. It needs eligibility, approval profile, template snapshot, certificate ID/status/expiry, revocation, and supersession. | Evidence export release, external connector, invoicing, payment. |
| 2 | **QR verification projection and certificate portal** | It delivers safe verifier value and blocks simple PDF/serial tampering by displaying server-held asset/inspection facts. | Raw evidence access and certificate amendment. |
| 3 | **Work Execution Summary and office closeout** | It connects individual and multi-inspector work across days to office acceptance and commercial handoff. | Pricing, tax invoice, payment. |
| 4 | **Expiry/reminder and renewal Work Order queue** | It turns issued-certificate validity into planned service demand without conflating a reminder with billing. | Automatic contract acceptance or job creation. |
| 5 | **Audited evidence-export approval ledger** | It governs controlled disclosure of a sealed manifest after certificate issuance, and prepares for external adapters. | Automatic external release or any particular regulator integration. |
| 6 | **Commercial/invoicing adapter boundary** | It allows office-confirmed work to create a compliant invoice payload through a dedicated future commercial system. | Tax applicability determination or ZATCA compliance claim. |
| 7 | **External authority adapters** | It supports client/authority-specific sequence rules after local certificate/export foundations are proven. | Universal SASO/SAAC recognition claim or unapproved owner bypass. |

## References and limitations

ISO’s current overview identifies ISO/IEC 17020:2026 as requirements for competence, impartiality, and consistent operation of inspection bodies; SAAC publicly describes its accreditation/assessment role for conformity-assessment bodies; and ZATCA publicly describes structured electronic invoicing and phased system integration. These sources inform the separation of inspection, certification, accreditation, and invoicing tracks. They do not establish accreditation, certification, tax applicability, legal compliance, or external-platform acceptance for INTEGIN. [ISO/IEC 17020:2026](https://www.iso.org/standard/17020) · [Saudi Accreditation Center](https://saac.gov.sa/en/home/) · [ZATCA E-Invoicing](https://zatca.gov.sa/en/E-Invoicing/Introduction/Pages/What-is-e-invoicing.aspx)
