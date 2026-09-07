# INTEGIN Metadata-Driven Field Form Delivery

**Status:** Proposed architecture only; no app update, schema migration, or runtime-policy change is authorised by this document.  
**Date:** 2026-08-20  
**Primary requirement:** New ordinary fields, bindings, rules, layouts, and templates must reach the Field app as approved signed metadata packages rather than requiring a Flutter release every time.

## Direct decision

> The Field app will contain a **universal approved form renderer**. It downloads and interprets approved, versioned, signed form packages. It does **not** download or execute arbitrary SQL, JavaScript, Dart code, native code, or unreviewed scripts.

This means the owner can create, approve, and assign a new form or add an ordinary field without waiting for an app release, as long as the new form uses a control that the installed Field renderer already understands.

## What is delivered to the inspector app

The inspector receives a compact **Form Package**, not a new app binary. A package is data/configuration, verified before use.

| Package section | Contents | Example |
| --- | --- | --- |
| Manifest | Package ID, tenant/organization, template/version IDs, publication state, effective date, required renderer capabilities, hashes, signature, and expiry/revocation metadata | “Inspection template v7, requires `measurement`, `yes_no_na`, `photo`, `repeater`.” |
| Form layout | Sections, groups, order, labels, help text, responsive layout directives, accessibility information | “Asset details” section followed by “Examination checklist.” |
| Field definitions | Stable field key, type, label, binding key, constraints, units, option-set reference, default/display behaviour | `core.asset.serial_number`, type `text`, one line, read-only or editable policy. |
| Ruleset | Declarative show-if, require-if, applicability, calculated-display, and evidence rules | Show “Installed correctly?” only if “First examination?” is Yes. |
| Repeater definitions | Allowed repeated rows/groups, min/max instances, ordering, parent-child relationships | Multiple shackle components or multiple test measurements. |
| Option sets | Stable option codes and localized labels | `yes`, `no`, `not_applicable`; English/Arabic labels. |
| Binding catalog subset | Only approved keys used by this form, with data type, cardinality, unit, and permission policy | `core.asset.serial_number` is scalar text; `core.inspection.findings[]` is a list. |
| Presentation assets | Approved logos/icons/help illustrations only when required and tenant-scoped | No arbitrary remote URLs. |

The package is versioned, hash-checked, signature-checked, tenant-scoped, cacheable offline, and revocable. It can be delivered as a complete package on first assignment and a compact approved delta later.

## What happens when you add a field, column, or cell

### A. Add an ordinary form field — no app update

If the Field app already supports the control type, the new field is simply added to a draft template and published as a new package version.

| Owner action | Server/package action | Field app action |
| --- | --- | --- |
| Add “Manufacturer” text field | Adds field definition using registered key `core.asset.manufacturer` | Downloads package v8 and renders a built-in text control. |
| Add “Condition” Yes/No/N/A question | Adds a controlled `yes_no_na` field and option set | Renders the built-in exclusive-choice control. |
| Add “If Yes, installed correctly?” | Adds child field plus declarative conditional rule | Shows it only when the parent answer is Yes. |
| Add a photo requirement for rejected results | Adds attachment field plus `require-if outcome = rejected` rule | Renders built-in photo control only when the rule applies. |
| Add a repeated shackle list | Adds approved repeater/table definition | Renders built-in repeat-row component with stable local instance IDs. |

No new Flutter screen, no hard-coded form, and no app-store update is needed in these cases.

### B. Add a new database column — usually no Field update

There are two different cases:

| Change | What must change | Does Field need an update? |
| --- | --- | --- |
| Add a new **dynamic form question** | New template field and typed answer row; no physical database column is necessary | No, if it uses a built-in field type. |
| Add a new **canonical core column**, such as `assets.manufacturer` | Backend schema migration, Go API/binding-registry update, and server deployment | No Field update if the field is text/number/date/etc. already supported. |
| Add a new **technical binding** with existing type, such as `lifting.leg_count` integer | Binding registry and server resolver; optional schema change depending on canonical source model | No Field update if integer/measurement control is supported. |
| Add a completely new interaction type | Renderer capability and possibly server validation support | Yes, a planned Field renderer release is required first. |

The key point is that a **database migration or backend deployment is not the same thing as a Field app release**. Most ordinary new fields will change only the approved package and perhaps the server model; the Field renderer remains unchanged.

## The built-in capability palette

To avoid routine releases, the first Field renderer must deliberately support a broad, controlled set of generic components.

| Capability family | Built-in control types |
| --- | --- |
| Text | Short text, bounded multi-line text, read-only display, identifier/serial input. |
| Numeric and units | Integer, decimal, measurement with canonical/display unit, tolerance, calculated display. |
| Time and location | Date, time, date-time with timezone, controlled site/location selector, optional geo capture. |
| Choices | Yes/No/N/A, radio group, dropdown, multiselect, controlled status/result. |
| Evidence | Photo, document attachment, signature, barcode/QR scan, evidence selector. |
| Structure | Section, group, divider, repeatable group/table, conditional block, checklist. |
| Technical data | Measurements, test rows, finding rows, component/asset rows, standards/reference selection. |
| Presentation | Hint text, warning, calculated read-only value, rule explanation, localized labels. |

If a new form uses only these capabilities, it can be delivered immediately as data. The package declares `required_capabilities`, and the Field app verifies it before accepting the package.

## The strict no-script boundary

The user’s desired “instructions delivered to the app” are correct, but they must be **declarative instructions**, not executable code.

| Do deliver in a package | Do not deliver in a package |
| --- | --- |
| Field type, label, binding key, validation rule, option set, visibility rule, layout, table/repeater definition, fit rule | SQL migration, raw SQL query, arbitrary JavaScript, Dart source, native module, shell script, secret, direct database URL, or unreviewed external code. |
| Limited rule DSL: `show-if`, `require-if`, `equals`, `in`, `greater-than`, `all`, `any`, controlled calculation | General-purpose programming language or arbitrary file execution. |
| Registered function with declared inputs/outputs, implemented in the approved app/server engine | A package-defined function that can access device, database, network, or filesystem freely. |

This is what makes instant form evolution safe offline. SQL changes belong in reviewed backend migrations. New Flutter/Dart behaviour belongs in a reviewed app release. Ordinary form evolution belongs in a signed metadata package.

## Package publication and sync flow

1. An authorised administrator edits a **draft** template in the web designer.
2. The server validates every binding, type, option, rule, repeater, asset, localization string, and required Field capability.
3. An authorised reviewer approves the immutable template/package version.
4. The server creates a canonical package manifest, hashes it, and signs it with the approved package authority.
5. The package is assigned to a tenant/organization, work order, inspector, and validity window.
6. The Field app synchronizes, requests its assignment manifest, verifies package signature/hash/capability compatibility, and saves the package locally.
7. The Field renderer builds the form from the package using only known components and rules.
8. The inspector works offline. Answers and local repeat instances use deterministic IDs and the exact package/ruleset version.
9. On synchronization, Go revalidates the package version, bindings, rules, field types, evidence, and tenant scope. It accepts, rejects, or returns machine-readable conflict/correction instructions.
10. The certificate/issuance process uses the approved inspection/template versions and resolved snapshot; it never relies on an untrusted client rendering decision.

## Local behaviour on update, incompatibility, and revocation

| Situation | Required Field behaviour |
| --- | --- |
| New compatible package available, no draft work in progress | Download, verify, cache, and use it for newly assigned work. |
| New compatible package but local inspection uses old version | Keep the old assigned version for that inspection; do not silently rewrite active answers. |
| Package requires an unsupported capability | Reject package activation and show a clear “Field update required” message. Do not hide unknown fields. |
| Package signature/hash/tenant scope invalid | Reject package; retain known-good cached version only if it remains assigned and valid. |
| Package revoked before work starts | Do not open it; sync asks the server for a replacement or reassignment. |
| Package revoked while offline work is in progress | Preserve locally captured evidence/answers, mark the work as requiring reconciliation, and let the server decide the accepted path after sync. |
| Binding/ruleset/template has changed | New work uses the new version; completed or assigned work stays tied to its original version unless a governed migration/reinspection workflow explicitly exists. |

## Security and assurance controls

| Control | Required behaviour |
| --- | --- |
| Tenant isolation | Package manifest, bindings, assets, and cached records all carry tenant/organization scope; server and database enforce it. |
| Device trust | Field package delivery uses the existing device-authenticated path; package verification does not grant broader workflow authority. |
| Integrity | Canonical package hash and signature verified before activation; package provenance/approval audit retained. |
| Capability safety | Package cannot activate a renderer capability that is not built into the installed Field app. |
| Rule determinism | Versioned ruleset is evaluated locally for usability and server-side for authority; snapshot retains ruleset hash/outcomes. |
| Offline conflict safety | Local operation log and base row version prevent silent overwrites; server reports explicit conflicts. |
| Data minimization | Field receives only the approved package and records assigned/necessary for the work, not unrestricted tenant data. |
| Rollback | New package versions do not mutate older assigned or issued records. Use a new approved version or governed reassignment. |

## The rare cases that genuinely require an app update

An app update is required only when the business requests a new **renderer capability**, for example:

- A new custom drawing/canvas editor.
- A specialized 3D/AR measurement experience.
- A new hardware-device protocol.
- A new complex NDT image-analysis viewer.
- A new kind of cryptographic capture or signature interaction.

Before such a release, the designer does not offer that new component type. Once the renderer update is approved and deployed, future metadata packages can use it without another release for every new form.

## Final owner-facing rule

> You will normally add a field, column binding, question, conditional rule, photo requirement, table row, or certificate cell in the web designer. You approve it, and the inspector app synchronizes the signed package and understands it immediately through its built-in universal renderer. You do **not** release the app each time. You release the app only when you introduce a genuinely new kind of interaction that the universal renderer never learned before.

## Recommended next implementation sequence

1. Define the capability palette and Field package manifest/schema.
2. Implement the binding-key registry, types, option sets, and declarative rules DSL.
3. Implement package validation, canonicalization, signing, assignment, and device-scoped delivery.
4. Implement the generic Flutter renderer, local package cache, and compatible-package checks.
5. Implement offline answer capture, repeaters, file placeholders, operation log, and server reconciliation.
6. Build the web form designer over the same package schema.
7. Add the fixed-layout certificate designer and bind it to the same registry/snapshot system.

No package enforcement change, OIDC change, OpenBao activation, external integration, production rollout, or arbitrary code execution is part of this proposal.
