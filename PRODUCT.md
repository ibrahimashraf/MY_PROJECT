<!-- impeccable:product-schema 1 -->
# Product

## Platform

adaptive

## Users
- **Certified NDT / Rig Inspector:** Primary field user in ATEX Zone 1 hazardous environments, offshore rigs, and industrial plants. Requires tactile, high-glare readability, safe draft cloning, and offline-first signing.
- **Technical Authority (Reviewer):** Office-based engineering authority conducting 4-eyes review, CAR resolution, and technical validation.
- **Operations Coordinator:** Manages multi-inspector assignments, scope completion, and commercial billing release.
- **Asset Owner / Regulatory Auditor:** Public / third-party verifier conducting instant QR cryptographic verification without backend dependencies.

## Product Purpose
High-assurance industrial inspection, asset integrity, and calibration trust platform replacing manual, paper, and vulnerable digital inspection workflows with mathematically verified, tamper-evident records.

## Positioning
Single-source truth directly binding physical field asset witness statements, ISO 17020 §6.2 tool calibration gating, FIPS 140-3 biometric hardware attestation, and zero-backend-cost public cryptographic verification.

## Operating Context
- **Harsh Offshore & Industrial Edge:** Extreme sunlight glare, oily/greasy gloves, ATEX Zone 1 explosive atmospheres, intermittent or zero satellite connectivity.
- **Office Engineering Trailer & HQ:** High-density multi-inspector scope oversight, dual-control 4-eyes technical review, and invoice release.
- **Public / Third-Party Audit Point:** Instant sovereign offline QR scan on physical equipment tags without logins, networks, or databases.

## Capabilities and Constraints
- **Offline Authority & Biometrics:** Offline cryptographic authority tokens bound to secure device hardware keys.
- **ISO 17020 §6.2 Tool Registry:** Inspection measurements strictly locked if calibrated measuring devices are uncalibrated or expired.
- **Pure Go Core & Multi-Tenant RLS:** Zero-CGO pure Go backend enforcing strict PostgreSQL tenant isolation.
- **Tamper-Evident Merkle Trees:** Cryptographic evidence sealing using SHA-256 Merkle trees and Ed25519 envelopes.

## Brand Commitments
- **Name:** INTEGIN
- **Visual Personality:** Industrial Rig & High-Contrast Safety (deep slate chassis `#0B0F19`/`#1E293B`, safety amber `#F59E0B`, verified emerald `#10B981`, defect crimson `#EF4444`).
- **Tactile Standard:** Minimum 48×48dp hit targets for all field interactions; monospaced display for cryptographic identifiers.

## Evidence on Hand
- `field_app`: Offline-first Flutter mobile/desktop app with biometric hardware attestation (`field_app/`).
- `quiet-signal`: React 19 + TypeScript + Vite operations dashboard (`quiet-signal/`).
- `tools/public-verifier`: Stateless HTML5 + WebCrypto browser verifier (`tools/public-verifier/index.html`).
- `tools/lifting-simulator`: Kinematic lift planners (`tools/lifting-simulator/`).

## Product Principles
1. **Physical Reality Over Digital Speed:** Software must never certify an asset uninspected; every finding is bound to a verified inspector and calibrated tool.
2. **Absolute Proof Isolation:** Evidence (photos, signatures, receipts) belongs to one physical event and must never bleed across assets.
3. **High-Contrast Industrial Safety Aesthetic:** Deep slate chassis with high-visibility safety amber, verified emerald, and crimson defect warnings.
4. **Greasy-Glove Affordance:** Minimum 48×48dp touch targets, clear tactile visual feedback, and zero modal traps.

## Accessibility & Inclusion
- High ambient contrast ratio (>7:1) for extreme outdoor sunlight readability.
- Touch targets strictly $\ge 48\times 48\text{dp}$ with distinctive mechanical borders.
- Full color-blind redundancy: status states pair distinctive icons/tokens with semantic colors.
