---
name: INTEGIN
description: High-assurance industrial inspection, asset integrity & calibration trust platform
colors:
  primary: "#f59e0b"
  primary-hover: "#d97706"
  surface-chassis: "#0b0f19"
  surface-card: "#1e293b"
  surface-border: "#334155"
  text-primary: "#f8fafc"
  text-muted: "#94a3b8"
  status-verified: "#10b981"
  status-defect: "#ef4444"
  status-neutral: "#64748b"
typography:
  display:
    fontFamily: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
    fontSize: "36px"
    fontWeight: 700
    lineHeight: 1.15
    letterSpacing: "-0.02em"
  headline:
    fontFamily: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
    fontSize: "24px"
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: "-0.01em"
  title:
    fontFamily: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
    fontSize: "18px"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "normal"
  body:
    fontFamily: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "normal"
  label:
    fontFamily: "ui-monospace, Menlo, Consolas, monospace"
    fontSize: "13px"
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: "0.04em"
rounded:
  sm: "4px"
  md: "8px"
  lg: "12px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "16px"
  lg: "24px"
  xl: "32px"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.surface-chassis}"
    rounded: "{rounded.sm}"
    padding: "14px 28px"
    height: "48px"
  button-primary-hover:
    backgroundColor: "{colors.primary-hover}"
    textColor: "{colors.surface-chassis}"
    rounded: "{rounded.sm}"
    padding: "14px 28px"
    height: "48px"
  card-chassis:
    backgroundColor: "{colors.surface-card}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.md}"
    padding: "16px 20px"
---

# Design System: INTEGIN

## Overview

**Creative North Star: "Industrial Rig & High-Contrast Safety"**

INTEGIN operates in unforgiving physical and technical environments: harsh offshore rigs under direct blinding sunlight glare, explosive ATEX Zone 1 hazardous atmospheres, oily-gloved field operators, and cryptographic zero-knowledge third-party audit verifiers. The visual design system rejects decorative fluff, low-contrast pastel gradients, and delicate micro-interactions in favor of high-density utility, extreme contrast ratios, and tactile reliability.

Every screen embodies structural rigidity and tamper-evident authenticity. The visual foundation is built on deep structural slate (`#0B0F19`, `#1E293B`) paired with high-visibility safety amber (`#F59E0B`), verified emerald (`#10B981`), and unequivocal defect crimson (`#EF4444`). All primary interactive touchpoints adhere strictly to physical affordances—minimum 48×48dp target bounding boxes, clear mechanical state feedback, and monospaced cryptographic hashes.

**Key Characteristics:**
- **High-Visibility Industrial Contrast:** Designed to remain legible at 10,000+ lux ambient daylight on ruggedized field tablets.
- **Greasy-Glove Affordance:** All interactive controls maintain minimum 48dp tactile hit regions with 4px distinct border separation.
- **Tamper-Evident State Hierarchy:** Cryptographic verifications, calibrated tool validity, and defect severities carry unmistakable color codes.
- **Monospace Cryptographic Clarity:** Envelope signatures, DID hashes, and Merkle proofs render exclusively in tabular monospace.

## Colors

The INTEGIN palette utilizes dark structural chassis backgrounds layered with high-chroma safety and diagnostic accents.

### Primary
- **Safety Amber** (`#F59E0B`): Primary action accent, active inspection focus, high-visibility guidance cues.
- **Amber Deep** (`#D97706`): Pressed and active hover state for primary amber actions.

### Secondary
- **Verified Emerald** (`#10B981`): Cryptographically sealed records, passing tolerances, calibrated tool certification status.

### Tertiary
- **Defect Crimson** (`#EF4444`): Out-of-tolerance measurements, expired tool calibrations, failed cryptographic proofs, CAR triggers.

### Neutral
- **Deep Chassis** (`#0B0F19`): Root viewport and baseline chassis background.
- **Card Surface** (`#1E293B`): Elevated inspection panels, scope grouping cards, and inspector trays.
- **Surface Border** (`#334155`): High-definition 1px structural container and separator borders.
- **Text Primary** (`#F8FAFC`): Primary inspection labels, reading values, high-contrast headings.
- **Text Muted** (`#94A3B8`): Contextual labels, unit indicators, timestamp metadata.

### Named Rules
**The Sunlight Glare Rule.** Interactive elements and state indicators must achieve a minimum contrast ratio of 7:1 against their immediate chassis background.
**The Tri-Color Safety Doctrine.** Emerald (`#10B981`), Amber (`#F59E0B`), and Crimson (`#EF4444`) are reserved strictly for verification state, active warning/guidance, and non-conformity. They may never be used for purely aesthetic decoration.

## Typography

**Display Font:** System UI / Roboto / Segoe UI (system native for zero-latency offline loading)
**Body Font:** System UI / Roboto / Segoe UI
**Label/Mono Font:** UI Monospace / Consolas / Menlo (for DIDs, Merkle hashes, serial numbers, cryptographic proofs)

**Character:** Utilitarian, crisp, and dense without visual congestion. Monospaced elements are treated as first-class citizens.

### Hierarchy
- **Display** (700, 36px, line-height: 1.15): Top-level inspection headings, asset integrity status banners.
- **Headline** (600, 24px, line-height: 1.2): Section containers, equipment group headers, work order titles.
- **Title** (600, 18px, line-height: 1.3): Inspection prompt labels, tool calibration headers, modal titles.
- **Body** (400, 15px, line-height: 1.5): Descriptive notes, inspector witness text, procedure instructions (max line length 70ch).
- **Label / Code** (500, 13px, line-height: 1.4, monospace): DIDs (`did:integin:...`), Merkle roots, ISO calibration dates, GPS coords.

### Named Rules
**The Cryptographic Monospace Rule.** Any value representing a cryptographic proof, hash, signature, or tool serial number must render in monospaced font with tabular numerals.

## Layout

- **Grid Model:** 8px base grid rhythm (4px micro-increments for compact telemetry).
- **Density:** High-density desktop trailers / office dashboards; chunky 48dp tactile field layouts on tablets.
- **Container Strategy:** Rigid bordered cards (`#334155` 1px border) with clear spatial grouping by asset location and inspector assignment.

## Elevation & Depth

Surfaces are predominantly flat with tonal layering (`#0B0F19` $\rightarrow$ `#1E293B`). Ambient soft drop shadows are avoided to maintain razor-sharp legibility on field screens.

### Shadow Vocabulary
- **Tactile Lift** (`box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.5)`): Reserved exclusively for floating action buttons and active modal dialogs.

### Named Rules
**The Flat-By-Default Rule.** Depth is established through tonal step (`#0B0F19` base $\rightarrow$ `#1E293B` panel $\rightarrow$ `#334155` border), never through decorative multi-layer blur shadows.

## Shapes

- **Corner Language:** Tight, functional radii (4px for buttons and inputs; 8px for containers; 12px for primary dialogs).
- **Borders:** Crisp, 1px solid structural definition (`#334155`). Never borderless floating cards in data-heavy screens.

## Components

### Buttons
- **Shape:** 4px radius (`rounded.sm`). Minimum height 48px for touch reliability.
- **Primary:** Background `#F59E0B`, text `#0B0F19`, font-weight 600.
- **Hover / Focus:** Background `#D97706`, 2px outer outline in `#F59E0B`.
- **Secondary / Outlined:** Background transparent, 1px border `#334155`, text `#F8FAFC`.

### Cards / Containers
- **Corner Style:** 8px radius (`rounded.md`).
- **Background:** `#1E293B`.
- **Border:** 1px solid `#334155`.
- **Internal Padding:** 16px to 20px.

### Inputs / Fields
- **Style:** Background `#0B0F19`, 1px border `#334155`, text `#F8FAFC`, minimum height 48px.
- **Focus:** 1px border `#F59E0B`, outline 1px solid `#F59E0B`.
- **Error:** 1px border `#EF4444`, text `#EF4444`.

### Verification Badge (Signature Component)
- **Style:** Monospaced pill badge with 1px border.
- **Verified:** Background `rgba(16, 185, 129, 0.15)`, text `#10B981`, border `#10B981`.
- **Invalid / Defect:** Background `rgba(239, 68, 68, 0.15)`, text `#EF4444`, border `#EF4444`.

## Do's and Don'ts

### Do:
- **Do** maintain a minimum 48×48dp hit target on all clickable and tappable elements in field workflows.
- **Do** format all cryptographic hashes, DIDs, signatures, and calibration serial numbers in monospaced font.
- **Do** enforce 7:1 minimum contrast on all critical telemetry and status indicators.
- **Do** use tonal surface steps (`#0B0F19` to `#1E293B`) to establish hierarchy.

### Don't:
- **Don't** use soft decorative pastels, low-contrast grey-on-grey text, or subtle multi-layer drop shadows.
- **Don't** use amber, emerald, or red colors for decorative or non-status elements.
- **Don't** allow interactive buttons smaller than 40px in mobile/field contexts.
- **Don't** trap field inspectors in non-dismissible modal loops or multi-page wizards without offline state persistence.
