# INTEGIN LLC Master Onboarding Architecture & Commercial Strategy Record

**Document Type:** Master Strategic Architecture Record  
**Target Identity:** INTEGIN LLC (Integrated Inspection & Assurance Platform)  
**Status:** Approved Architectural Blueprint (Recorded in Repository)  
**Date:** 2026-09-02  

---

## 1. Executive Summary & Purpose

This record establishes the authoritative **Onboarding, Governance, and Trust Architecture** for INTEGIN as an enterprise B2B Software-as-a-Service (SaaS) entity (INTEGIN LLC). 

It resolves the structural debate between:
1. **Commercial Velocity (CRM / SaaS Plane):** Frictionless customer signup, fleet asset migration, billing tiers, and instant-value sandbox previews.
2. **Industrial Safety & Legal Non-Repudiation (Assurance Plane):** Verified human competency gates (LEEA / NDT / API), hardware-bound cryptographic edge trust (Ed25519), and four-eyes review policies.

---

## 2. The 6-Act System Architecture

```
+========================================================================================+
|                              INTEGIN 6-ACT LIFECYCLE                                   |
+========================================================================================+
|  ACT 0: PLATFORM GENESIS (Founder & Master Operator)                                  |
|  • Master Root CA generation (Ed25519)                                                 |
|  • PostgreSQL Row-Level Security (RLS) multi-tenant partitioning baseline              |
|  • Governed inspection standards library (BS 7121, ASME B30.5, ISO 4309, LEEA)         |
|  • OpenBao secrets vault & encrypted object storage (S3/RustFS)                       |
+----------------------------------------------------------------------------------------+
                                           │
                                           ▼
+----------------------------------------------------------------------------------------+
|  ACT 1: TENANT PROVISIONING (Inspection Firm Admin Onboarding)                         |
|  • Step 1: Legal Identity & Tax Registration (CR Number, VAT/ZATCA, Country)           |
|  • Step 2: Accreditation Badges & Branding (LEEA/ISO 17020 logos, Official Stamp)      |
|  • Step 3: Certificate Governance & Numbering Pattern (`APEX-CRN-YYYY-SEQ`)            |
|  • Step 4: Active Disciplines (Mobile Cranes, Lifting Tackle, NDT, Pressure Vessels)   |
|  • Step 5: Verification & Production Activation                                        |
+----------------------------------------------------------------------------------------+
                                           │
                                           ▼
+----------------------------------------------------------------------------------------+
|  ACT 2: CLIENT & ASSET FLEET INTAKE (CRM AI Staging Pipeline)                         |
|  • Client organization & physical operating site mapping                               |
|  • Multi-format ingestion: Spreadsheets (Excel/CSV) & Legacy PDF Certificates (OCR)    |
|  • Isolated Staging Quarantine: Safe Working Load (SWL) and serial collision check     |
+----------------------------------------------------------------------------------------+
                                           │
                                           ▼
+----------------------------------------------------------------------------------------+
|  ACT 3: INSPECTOR COMPETENCY & DEVICE PAIRING (The Edge Handshake)                     |
|  • Competency Stacking: Inspector uploads credentials (LEEA, ASNT NDT Level II, API)   |
|  • Technical Authority Gate: QA Manager reviews and approves scope of authority        |
|  • Scan-to-Pair QR Handshake: Tablet hardware generates Ed25519 keypair in Secure Keystore|
+----------------------------------------------------------------------------------------+
                                           │
                                           ▼
+----------------------------------------------------------------------------------------+
|  ACT 4: ZERO-SIGNAL FIELD EXECUTION (The Disconnected Edge)                            |
|  • 100% offline inspection on rugged industrial tablet                                 |
|  • Local tamper-proof event logging with SQLite Write-Ahead Logging (WAL)              |
|  • Field cryptographic signing of checklist findings, photos, and client rep signature|
+----------------------------------------------------------------------------------------+
                                           │
                                           ▼
+----------------------------------------------------------------------------------------+
|  ACT 5: VERIFICATION & CERTIFICATE ISSUANCE (Cloud Sync & Public QR)                   |
|  • Server validates Device Signature, Package Hash, and Inspector active qualifications|
|  • Mandatory Four-Eyes QA Review & Digital Seal stamping                               |
|  • Real-time dual-language (Arabic/English) PDF certificate release & public QR portal |
+========================================================================================+
```

---

## 3. Core Architectural Debates & Resolutions (INTEGIN LLC Playbook)

### 3.1. Instant Self-Serve vs. Strict Verification Gating
* **Resolution:** **The Sandbox Tiered Model**. 
  * Anyone can sign up in 30 seconds and test a simulated crane inspection with watermarked `"DEMO / SAMPLE"` certificates.
  * Generating legally binding, non-watermarked, numbered certificates requires completing the QA Competency Review & Device Enrollment gate.

### 3.2. Device-Bound Cryptographic Pairing vs. Standard Cloud User Login
* **Resolution:** **Hardware Cryptographic Binding + Emergency Field Delegation**.
  * Primary mode binds the mobile app to the physical hardware Secure Enclave via Ed25519 public key registration.
  * If a tablet breaks in the field, an **Emergency Field Handover QR** allows temporary delegation of work orders without compromising the audit trail.

### 3.3. Asset Fleet Ingestion Strategy
* **Resolution:** **Client Self-Upload + AI Staging Quarantine**.
  * Clients upload messy legacy spreadsheets or old PDF certificates.
  * Ingestion occurs in an isolated **Staging Table**. The system auto-detects missing mandatory safety attributes (e.g. SWL, Serial Number) for batch resolution before committing to the live registry.

### 3.4. Commercial Strategy & Business Model
* **Resolution:** **Hybrid Tiered Pricing (SaaS Base + Certificate Overage)**.
  * Base monthly subscription includes core CRM, scheduling, and standard templates.
  * Usage fee ($1.00 - $2.50) per cryptographically issued certificate, aligning software revenue with the customer's billing volume.

---

## 4. Resilience, Power-Cut & Crash Recovery Guarantees

1. **Power Cut on Field Tablet:**
   * SQLite configured with `PRAGMA journal_mode=WAL` and immediate `fsync` on every checklist interaction.
   * On reboot, the tablet auto-recovers the exact unsealed work order with all captured photos and checkboxes intact.
2. **Server-Side Crash / Power Loss:**
   * PostgreSQL ACID transaction boundaries guarantee that partial multi-table onboarding writes roll back cleanly.
   * Recovery Time Objective (RTO) ≤ 4 hours; Recovery Point Objective (RPO) ≤ 1 hour.
3. **Browser Tab Closed / PC Crash:**
   * Dual-layer persistence: immediate client-side `localStorage` mirroring + debounced backend draft auto-saving.
   * Upon re-opening `app.integin.com`, the user resumes exactly at the active wizard step.

---

## 5. Repository Artifacts Landed

* **Architecture Specification:** [`docs/architecture/INTEGIN_ONBOARDING_SYSTEM_SPECIFICATION.md`](file:///c:/MY_PROJECT/docs/architecture/INTEGIN_ONBOARDING_SYSTEM_SPECIFICATION.md)
* **Master Strategy Record (This File):** [`docs/architecture/INTEGIN_LLC_ONBOARDING_STRATEGY_RECORD_2026-09-02.md`](file:///c:/MY_PROJECT/docs/architecture/INTEGIN_LLC_ONBOARDING_STRATEGY_RECORD_2026-09-02.md)
* **Go Backend Contracts:** [`integin-pilot-source/pkg/onboarding/onboarding_contracts.go`](file:///c:/MY_PROJECT/integin-pilot-source/pkg/onboarding/onboarding_contracts.go)
* **Interactive Bilingual Onboarding Wizard UI:** [`tools/onboarding-wizard/`](file:///c:/MY_PROJECT/tools/onboarding-wizard/)
