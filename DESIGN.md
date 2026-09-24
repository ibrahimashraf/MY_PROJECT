---
name: INTEGIN
description: "Provisional incumbent system for evidence-led field and operations interfaces"
colors:
  authority-teal: "#0f766e"
  field-sage: "#537a68"
  accent-surface: "#f0fdfa"
  accent-border: "#99f6e4"
  rail-surface: "#f5f5f5"
  action-blue: "#2563eb"
  sync-green: "#16a34a"
  status-valid: "#059669"
  status-warning: "#d97706"
  status-critical: "#b91c1c"
  status-info: "#0369a1"
  valid-surface: "#dcfce7"
  critical-surface: "#fee2e2"
  warning-surface: "#fef3c7"
  info-surface: "#e0f2fe"
  paper: "#ffffff"
  canvas-mist: "#f9fafb"
  surface-muted: "#f3f4f6"
  ink: "#111827"
  heading-ink: "#08060d"
  muted-slate: "#6b7280"
  border-slate: "#e5e7eb"
  border-strong: "#cccccc"
  dark-canvas: "#16171d"
  dark-border: "#2e303a"
  dark-muted: "#9ca3af"
  dark-accent: "#c084fc"
typography:
  display:
    fontFamily: "system-ui, 'Segoe UI', Roboto, sans-serif"
    fontSize: "56px"
    fontWeight: 500
    lineHeight: 1
    letterSpacing: "-1.68px"
  headline:
    fontFamily: "system-ui, 'Segoe UI', Roboto, sans-serif"
    fontSize: "24px"
    fontWeight: 500
    lineHeight: 1.18
    letterSpacing: "-0.24px"
  title:
    fontFamily: "system-ui, 'Segoe UI', Roboto, sans-serif"
    fontSize: "14px"
    fontWeight: 500
    lineHeight: 1.4
  body:
    fontFamily: "system-ui, 'Segoe UI', Roboto, sans-serif"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.5
  utility:
    fontFamily: "system-ui, 'Segoe UI', Roboto, sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: 1.4
  label:
    fontFamily: "system-ui, 'Segoe UI', Roboto, sans-serif"
    fontSize: "11px"
    fontWeight: 700
    lineHeight: 1.4
    letterSpacing: "0.04em"
  data:
    fontFamily: "ui-monospace, Consolas, monospace"
    fontSize: "11px"
    fontWeight: 400
    lineHeight: 1.4
rounded:
  micro: "3px"
  control: "4px"
  compact: "6px"
  surface: "8px"
  panel: "10px"
  pill: "999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "20px"
  xxl: "24px"
  section: "32px"
components:
  button-action:
    backgroundColor: "{colors.action-blue}"
    textColor: "{colors.paper}"
    rounded: "{rounded.control}"
    padding: "8px 16px"
  button-sync:
    backgroundColor: "{colors.sync-green}"
    textColor: "{colors.paper}"
    rounded: "{rounded.control}"
    padding: "6px 12px"
  button-secondary:
    backgroundColor: "{colors.canvas-mist}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    padding: "8px 16px"
  button-danger:
    backgroundColor: "{colors.status-critical}"
    textColor: "{colors.paper}"
    rounded: "{rounded.control}"
    padding: "8px 16px"
  input-standard:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    padding: "8px"
  panel-surface:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.surface}"
    padding: "16px"
  status-chip:
    backgroundColor: "{colors.info-surface}"
    textColor: "{colors.status-info}"
    rounded: "{rounded.pill}"
    padding: "2px 6px"
  data-table:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    padding: "8px"
  nav-sidebar:
    backgroundColor: "{colors.rail-surface}"
    textColor: "{colors.ink}"
    width: "220px"
    padding: "16px"
  boundary-banner:
    backgroundColor: "{colors.accent-surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.surface}"
    padding: "12px 14px"
---

# Design System: INTEGIN

## Overview

**Creative North Star: "The Field Ledger / Control Room"**

INTEGIN's incumbent interface behaves like an operational instrument for people who need to know what is trusted, what is pending, what failed, and what remains server-controlled. The field client favors explicit authority and recovery states; the operations console favors compact data, filters, evidence metadata, audit trails, and administrative actions. Both surfaces are functional first, with little decorative language.

The system is calm enough for repeated use, direct enough for field conditions, and technical enough for operators who need to inspect identifiers, timestamps, digests, and authority metadata. It is not a finalized brand identity. The current colors were generated in implementation code and are recorded as provisional observed tokens; the React console and Flutter client have not been normalized into one approved palette.

**Key Characteristics:**
- State and authority are visible before decoration.
- Dense operational information remains readable through hierarchy, labels, and monospace data.
- Layout adapts from mobile field navigation to tablet rails and desktop split panes.
- Semantic green, amber, red, blue, and neutral states are used as operational signals.
- Flat surfaces and borders are the default; elevation is reserved for meaningful state changes.

## Colors

The current palette is functional and provisional: neutral paper and slate surfaces carry most of the interface, teal and green indicate trusted or actionable paths, and blue, amber, and red identify specific field or administrative actions.

### Primary

- **Provisional Authority Teal** (`#0f766e`): React operations-console accent used for confidence, banners, and active system context.
- **Field Sage** (`#537a68`): Flutter Material 3 seed color for the field client's generated color scheme.

### Secondary

- **Action Blue** (`#2563eb`): React field inspection actions such as local save and new inspection.
- **Sync Green** (`#16a34a`): React field synchronization controls and successful sync feedback.

### Tertiary

- **Valid Green** (`#059669`): healthy and integrity-valid states.
- **Warning Amber** (`#d97706`): degraded, pending, or attention-required states.
- **Critical Red** (`#b91c1c`): invalid, failed, revoked, or destructive states.
- **Information Blue** (`#0369a1`): zone and informational labels.

### Neutral

- **Paper** (`#ffffff`): primary surface and input background.
- **Mist Canvas** (`#f9fafb`): secondary controls, trace panels, and quiet backgrounds.
- **Muted Surface** (`#f3f4f6`): code, compact chips, and secondary UI fills.
- **Ink** (`#111827`): primary React body text.
- **Heading Ink** (`#08060d`): base heading text in the global stylesheet.
- **Muted Slate** (`#6b7280`): metadata, labels, helper text, and secondary content.
- **Border Slate** (`#e5e7eb`): standard panel and control borders.
- **Strong Border** (`#cccccc`): data-table and explicit inline borders.

### Named Rules

**The Evidence Before Ornament Rule.** Do not add decorative treatment that competes with authority, sync, validation, or failure state.

**The Provisional Palette Rule.** These values document observed implementation, not approved brand identity; do not extend or normalize them without a human design decision.

## Typography

**Display Font:** System UI (`system-ui, Segoe UI, Roboto, sans-serif`)
**Body Font:** System UI (`system-ui, Segoe UI, Roboto, sans-serif`)
**Label/Mono Font:** UI monospace (`ui-monospace, Consolas, monospace`) for identifiers, digests, fingerprints, and code.

**Character:** The pairing is intentionally plain and operational. Hierarchy comes from weight, size, spacing, and semantic placement rather than decorative type. Flutter currently uses Material 3 defaults and has no custom font family.

### Hierarchy

- **Display** (500, 56px, line-height 1): Base global `h1` treatment; large React headings when the surface does not override them.
- **Headline** (500, 24px, line-height 1.18): Section and page headings in the base React stylesheet.
- **Title** (500, 14px): Panel headings, signal titles, and compact Flutter section headings.
- **Body** (400, 13px, line-height 1.5): Explanations, rationales, labels, and form content.
- **Utility** (400, 12px): Metadata, environment status, helper text, and secondary controls.
- **Label** (700, 11px, 0.04em): Uppercase or compact operational labels.
- **Data** (400, 11px monospace): IDs, object keys, timestamps, digests, fingerprints, and serialized audit details.

### Named Rule

**The Data Gets a Monospace Voice Rule.** Use monospace for values that must be compared, copied, audited, or verified; do not use it for prose.

## Layout

The system uses responsive, task-oriented layouts rather than a single marketing grid. The React operations console uses a centered maximum width of approximately 1120px, a 320px sidebar/secondary column plus flexible primary content, and a one-column collapse below 900px. Administration pages use a 220px navigation rail and padded work areas; data-heavy pages prioritize full-width tables.

The Flutter field client uses three explicit screen classes: compact below 600px with bottom navigation and one column, medium from 600px to 1024px with a navigation rail, and expanded at 1024px and above with an extended rail and optional 5:7 master-detail split. Field content commonly uses 20px to 24px card padding and 8px to 20px internal gaps.

Keep reading order stable when layouts collapse. Do not hide authority, connectivity, or queue state to preserve a cleaner composition.

## Elevation & Depth

The current system is a state-led hybrid. Most React panels and signals are flat, separated by one-pixel borders, surface tints, and spacing. A base shadow token exists for ambient lift, but it is not consistently applied. Flutter Material cards may receive platform elevation, while their important hierarchy still comes from borders, spacing, headings, and status content.

### Shadow Vocabulary

- **Ambient Low** (`rgba(0, 0, 0, 0.1) 0 10px 15px -3px, rgba(0, 0, 0, 0.05) 0 4px 6px -2px`): Reserved base token for a genuinely lifted or focused state; do not apply as a default card treatment.
- **Ambient Dark** (`rgba(0, 0, 0, 0.4) 0 10px 15px -3px, rgba(0, 0, 0, 0.25) 0 4px 6px -2px`): Dark-mode counterpart from the base stylesheet; use only where the surface is intentionally elevated.

### Named Rule

**The Flat-At-Rest Rule.** Keep operational surfaces flat at rest; use elevation only when it communicates focus, selection, or a meaningful state transition.

## Shapes

The form language is lightly rounded and rectilinear. React controls commonly use 4px corners, compact controls use 6px, cards and panels use 8px to 10px, and status labels use pill-shaped 999px corners. One-pixel gray borders are the primary separator. Flutter uses Material 3 shapes and outlined input borders, with explicit 8px containers for structured field controls.

Use consistent radius within a surface. Do not introduce glass, blur, glow, or mixed corner systems merely to make the product feel more futuristic.

## Components

### Buttons

- **Shape:** Compact rectangular controls with a 4px radius; the field voice is direct and work-oriented.
- **Primary:** Action Blue with white text and 8px 16px padding for field actions; Sync Green with white text and 6px 12px padding for synchronization.
- **Hover / Focus:** The current React implementation relies heavily on native button states; add visible focus and pressed feedback when normalizing components, without changing action meaning.
- **Secondary / Ghost / Tertiary:** Mist or paper surfaces with ink text and a quiet border; use for cancellation, inspection, and non-primary transitions.
- **Danger:** Critical Red with white text for destructive or irreversible actions such as revocation; pair with explicit confirmation and reason text.

### Chips

- **Style:** Compact pill labels with a lightly tinted semantic background and darker semantic text.
- **State:** Used for advisory zones, validity, legal hold, and small status classifications. The label text must carry meaning; color alone is insufficient.

### Cards / Containers

- **Corner Style:** 8px to 10px in React; Material 3 card defaults in Flutter.
- **Background:** Paper or a quiet neutral surface.
- **Shadow Strategy:** Flat at rest; border and spacing establish hierarchy.
- **Border:** One-pixel slate or strong gray border where containment matters.
- **Internal Padding:** 12px to 16px for compact panels, 20px to 24px for field work cards.

### Inputs / Fields

- **Style:** Paper background, native or Material outline border, compact 4px to 8px radius, and visible labels.
- **Focus:** Use the platform's visible focus treatment; do not communicate validity by border color alone.
- **Error / Disabled:** Keep errors adjacent to the field, preserve the submitted value, and make disabled or unavailable actions explain why.

### Navigation

- **Style:** React uses a fixed 220px light navigation column; Flutter uses Material navigation bars and rails.
- **Default / Hover / Active States:** Keep the current section obvious through Material selection behavior or a restrained active treatment; do not hide the current location.
- **Mobile Treatment:** Flutter moves to bottom navigation below 600px; React collapses its content grid below 900px.

### Status and Authority Components

- **Status Card:** Surface connectivity, authority validity, expiry, queue count, failures, and tenant context together.
- **Advisory Boundary:** Use a quiet tinted banner to state that advisory output is non-blocking and cannot mutate authoritative workflow state.
- **Evidence Metadata:** Prefer digest, object key, integrity, legal-hold, and capture-time metadata over rendering plaintext evidence.

## Do's and Don'ts

### Do:

- **Do** make online/offline, trusted/blocked, queued, failed, held, and server-authoritative states explicit.
- **Do** use semantic colors with text labels, icons, or shape cues so status is not color-only.
- **Do** preserve adaptive navigation and stable reading order across compact, tablet, and desktop layouts.
- **Do** use monospace for identifiers, digests, fingerprints, timestamps, and other auditable values.
- **Do** keep primary work surfaces flat, bordered, and easy to scan.

### Don't:

- **Don't** treat the provisional AI-generated palette as approved brand identity.
- **Don't** introduce gradients, glass, glow, or decorative motion that obscure operational state.
- **Don't** hide errors, revocations, expiry, conflicts, or non-authoritative boundaries behind a nominal success state.
- **Don't** use a client-side status or color as proof of server authorization.
- **Don't** force the React and Flutter surfaces into one token system without an explicit design decision.
