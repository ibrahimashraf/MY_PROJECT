<!-- impeccable-schema: 1.0 -->
# PRODUCT.md — INTEGIN Product Context & Design Direction

## Positioning
- **Product Name:** INTEGIN (Integrated Inspection, Asset Assurance & Calibration Management)
- **Category:** Industrial Asset Integrity & Conformity Assessment (TIC) Platform
- **Mission:** High-assurance industrial inspection and lifecycle trust platform replacing manual, paper, and vulnerable digital inspection workflows with mathematically verified, tamper-evident records.
- **Core Value Proposition:** Single-source truth connecting field asset witness statements, ISO 17020 §6.2 tool calibration gating, FIPS 140-3 biometric hardware attestation, and zero-backend-cost public verification.

## Operating Context
- **Primary Operational Environments:**
  1. *Harsh Offshore & Industrial Edge:* Extreme sunlight glare, greasy gloves, ATEX Zone 1 explosive atmospheres, intermittent or disconnected satellite connectivity.
  2. *Office Engineering Trailer & HQ:* Multi-inspector scope oversight, 4-eyes technical review, and commercial billing release.
  3. *Public / Third-Party Audit Point:* Instant QR scan on physical equipment tags without logins or latency.
- **Key Target Personas:**
  - *Certified NDT / Rig Inspector:* Prioritizes touch reliability, sunlight readability, fast draft cloning, offline authority.
  - *Technical Authority (Reviewer):* Prioritizes high-density defect visualization, audit trail continuity, 4-eyes signing.
  - *Operations Coordinator:* Prioritizes multi-inspector progress, timesheet aggregation, billing release.
  - *Asset Owner / Auditor:* Prioritizes zero-knowledge cryptographic verification.

## Evidence on Hand
- **Active Codebase & Frontends:**
  - `field_app`: Offline-first Flutter mobile/desktop app with biometric hardware attestation.
  - `quiet-signal`: React 19 + TypeScript + Vite operations dashboard.
  - `tools/public-verifier`: Stateless HTML5 + WebCrypto browser verifier (`verify.integin.com`).
  - `tools/lifting-simulator`: 2D Canvas & 3D WebGL kinematic lift planners.
- **Governing Architecture & Standards:**
  - ISO/IEC 17020 §6.2 (Calibrated Tool Registry & Proof Isolation)
  - FIPS 140-3 (Hardware Security Module & Biometric Attestation)
  - ASME B30.5 / ISO 4309 (Lifting & Wire Rope Criteria)
  - W3C Decentralized Identifiers (`did:integin:...`)

## Product Principles
1. **Physical Reality Over Digital Speed:** Software must never certify an asset uninspected; every finding is bound to a verified inspector and calibrated tool.
2. **Absolute Proof Isolation:** Evidence (photos, signatures, receipts) belongs to one physical event and must never bleed across assets.
3. **High-Contrast Industrial Safety Aesthetic:** Deep slate chassis (`#0B0F19`, `#1E293B`) with high-visibility safety amber (`#F59E0B`), verified emerald (`#10B981`), and crimson defect warnings (`#EF4444`).
4. **Greasy-Glove Affordance:** Minimum 48×48dp touch targets, clear tactile visual feedback, and zero modal traps.
