# Certificate Form — Frontend Handoff Contract

- **Date:** 2026-09-17
- **Status:** Backend implemented and green; awaiting frontend implementation
- **Scope:** DAARSOFT `:5173` certificate wizard (`CreateCertificateForm`) + Review step + sign-off
- **Backend commits:** `a32cb3b` (signature events, evidence binding, migration 0084); hybrid-waiver slice uncommitted at time of writing

## 1. Wizard order and gating

Keep the 7-step order. **Risk Assessment (step 3) gates Inspection (step 5)** — Inspection stays locked until the RA is complete with all fields in §2 filled. Step ticks must mean validated data, not visited pages. Critical components named in the RA must flow into Checklist items and Pictures requirements.

## 2. Risk Assessment step fields

**(a) Equipment condition:** overall condition, age/fatigue/corrosion flags, component notes (ropes, brakes, safety devices), end-of-life indicators, carried defects + monitoring conditions.

**(b) Environment, two blocks:**
- *Equipment environment:* indoor/outdoor, weather exposure, corrosives, explosive atmosphere, transport/erection damage risk.
- *Site readiness (8 items, each pass/fail):* contamination (oil/chemicals/debris/water + control); lighting (adequate, incl. temporary for night/confined); paperwork on-site (examination scheme, prior report, manufacturer manual/limits, lift plan); footwear/PPE; PTW number where required; **electrical isolation** (lockable point, tag, test-before-touch + recorded result); **mechanical isolation** per energy source — hydraulic, pneumatic, spring, gravitational (no blanket "isolated ✓"); **LOTO** (named applier, key custody, removal authority). Non-isolatable sources declared explicitly with controls.

Site/isolation items are captured at step 3 and **re-confirmed at step 5 start** (pre-filled, inspector confirms or amends); any fail blocks inspection.

**(c) Operations and loads:** usage frequency, load range (near max SWL?), lift nature (long/short, shock/overload possible, hot/dynamic loads), stated assumptions (component life, duty cycle) that trigger scheme review on change.

**(d) Examiner:** identity + qualification basis — actual qualifications or "Company Appointed Examiner". Never the phrase "LEEA qualified" (LEEA G25).

**(e) Next-examination-due proposal** (LOLER Schedule 1 report item).

## 3. Review & Submit step

Render 6 read-only sections (Client, Template, Risk, Checklist, Inspection, Pictures), each with an Edit-back link. Every row shows a value or an explicit `—`; blank cards are banned. Failed picture loads render `⚠ unavailable — retry`, never empty bands. Header strip always visible: `job_number`, certificate number (or DRAFT), revision, certificate state, Created/Remaining/Total per scope item, plus the RA rows (§2) and day-of re-confirmation states.

**Submit gating:** template `APPROVED`, zero pending checklist rows, ≥1 loaded picture (or recorded waiver), sign-off event present. Disable on click; send idempotency key; land on certificate number + `PENDING_APPROVAL`.

## 4. Sign-off API calls

Register ink via evidence registration **before** signing; the sign body references it.

**Sign (client/verifier):** `POST /certificates/{id}/sign`
```json
{"signer_id":"<must equal authenticated actor>","signer_name":"…","capacity":"client|verifier","statement_version":"client_ack_v1","image_sha256_hex":"<64hex of PNG>","image_bytes":12345,"image_evidence_id":"<evidence id>","snapshot_sha256_hex":"<review content hash>"}
```
Empty body → 400. Inspector capacity or inspector ID → rejected (they attest instead).

**Attest (inspector, no ink):** `POST /certificates/{id}/attest`
```json
{"attestor_name":"…","qualification_basis":"…","statement_version":"inspector_attest_v1","snapshot_sha256_hex":"…"}
```
Requires `certificate.attest` capability; only the assigned inspector.

**Waive (skip signature with reason):** `POST /certificates/{id}/sign-waiver`
```json
{"granted_by":"<must equal authenticated actor>","reason":"…","capacity":"client|verifier","authorized_by":"<office authorizer — REQUIRED when granter is the on-site inspector, else omit>"}
```

**Pad behavior:** show versioned consent statement at signing (AR + EN, e.g. `client_ack_v1`); server clock stamps; Clear wipes image + stamp together; controls lock on submit.

**Consent text (v1):** AR: `أقرّ بأن الفحص الموصوف أعلاه قد أُنجز وأن النتائج مدوّنة بدقة` / EN: "I confirm the examination described above was completed and the findings accurately recorded." Inspector variant: findings accurately recorded + qualification basis.

## 5. Regulatory basis (cited 2026-09-17)

- HSE L113 (LOLER ACOP, 2nd ed.), paras 9–11, 54–57, 64 — risk assessment duty via Management Regulations 1999, applied before equipment selection/use; no standalone LOLER RA regulation.
- LEEA 032 (Written Schemes of Examination), item 11 — scheme must contain an RA covering (a) equipment condition, (b) environment, (c) number/nature of operations and loads.
- LEEA COPSULE — interim-inspection scope derives from the RA; thorough examination is the safety net.
- HSE INDG422 — between-examination inspections where the RA identifies significant risk.
- LEEA G25 (rev 2, 27/10/2025) — record actual qualifications or "Company Appointed Examiner".

## 6. Production release prerequisites

1. Apply `migrations/0084_certificate_signature_audit_actions.sql` to prod DB.
2. Grant `certificate.attest` capability to inspector roles in membership seeding.
3. Frontend sends real evidence IDs (evidence registration wired to the store).
4. Agree statement-version strings (`client_ack_v1`, `inspector_attest_v1`, `sign_waiver_v1` or local names).
5. Known pre-existing failure (not blocking this contract): TSA token-imprint mismatch — see `TSA_TOKEN_IMPRINT_DEFECT_2026-09-17.md`.
