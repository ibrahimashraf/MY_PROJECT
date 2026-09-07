# 🌍 INTEGIN Master 12-Tier Architecture: Deep Engineering Specification

**Document Version:** 1.0.0 (Enterprise Aerospace & Defense Standard)  
**Reconciled Date:** 2026-09-07  
**Governance Authority:** ADR-2026-09-07-DEEP-ARCH-VS-SPRINT2-EXECUTION  
**Primary References:** [`GLOBAL_ARCHITECTURE_PLAN.md`](./GLOBAL_ARCHITECTURE_PLAN.md), [`TRACKER.md`](../../TRACKER.md) v3.3.0  
**Scope:** Tiers L0 through L11, Binary Wire Protocols, Cryptographic Invariants, Sovereign Schemas, and State Machines.

---

## 📑 Table of Contents

1. [Architectural Invariants & Cross-Tier Authority Matrix](#1-architectural-invariants--cross-tier-authority-matrix)
2. [Tier L0: Global Root PKI Authority & Hierarchical Key Derivation](#2-tier-l0-global-root-pki-authority--hierarchical-key-derivation)
3. [Tier L1: Dynamic AST Engine & Mathematical Engineering Grammar](#3-tier-l1-dynamic-ast-engine--mathematical-engineering-grammar)
4. [Tier L2: Universal 7-Pillar Sovereign Regulatory & Compliance Matrix](#4-tier-l2-universal-7-pillar-sovereign-regulatory--compliance-matrix)
5. [Tier L3: Dynamic Inspection Package Scoping & Multi-Discipline Matrix](#5-tier-l3-dynamic-inspection-package-scoping--multi-discipline-matrix)
6. [Tier L4: Enterprise Hierarchy & High-Scale Async Ingress](#6-tier-l4-enterprise-hierarchy--high-scale-async-ingress)
7. [Tier L5: Certificate Governance, 4-Eyes QA & Cryptographic Sealing](#7-tier-l5-certificate-governance-4-eyes-qa--cryptographic-sealing)
8. [Tier L6: Dynamic Inspector Credentialing & Skill Matrix Verification](#8-tier-l6-dynamic-inspector-credentialing--skill-matrix-verification)
9. [Tier L7: Dynamic Tool Calibration & Metrological Traceability (ISO 17020 § 6.2)](#9-tier-l7-dynamic-tool-calibration--metrological-traceability-iso-17020--62)
10. [Tier L8: Universal FIPS 140-3 Hardware Tablet Attestation](#10-tier-l8-universal-fips-140-3-hardware-tablet-attestation)
11. [Tier L9: W3C Decentralized Asset Passport & Technical Quarantine Lifecycle](#11-tier-l9-w3c-decentralized-asset-passport--technical-quarantine-lifecycle)
12. [Tier L10: Bitemporal Merkle-CRDT Tamper-Proof Audit Ledger (64-Byte Cache-Line)](#12-tier-l10-bitemporal-merkle-crdt-tamper-proof-audit-ledger-64-byte-cache-line)
13. [Tier L11: Edge Zero-Knowledge QR Trust Gateway & Stateless WebCrypto](#13-tier-l11-edge-zero-knowledge-qr-trust-gateway--stateless-webcrypto)

---

## 1. Architectural Invariants & Cross-Tier Authority Matrix

```
┌────────────────────────────────────────────────────────────────────────────────────────────────┐
│                             CROSS-TIER INVARIANT & AUTHORITY MATRIX                            │
├──────┬────────────────────────────┬─────────────────────────────┬──────────────────────────────┤
│ Tier │ Inbound Dependency         │ Authoritative Output        │ Forbidden Action / Invariant │
├──────┼────────────────────────────┼─────────────────────────────┼──────────────────────────────┤
│ L0   │ Root Cryptographic Seed    │ Validated Tenant License    │ Zero mutation of tenant data │
│ L1   │ Standard Body Metadata     │ Evaluated CEL AST (<15µs)   │ Zero raw paywalled PDFs      │
│ L2   │ National Regulatory Law    │ Sovereign Compliance Config │ Zero hardcoded national DDL  │
│ L3   │ Asset Classification       │ Scoped Inspection Package   │ No untyped checklist items   │
│ L4   │ Enterprise Tenant Context  │ Reconciled Work Order State │ No unpartitioned mutations   │
│ L5   │ Field Inspection Evidence  │ Cryptographic PDF/A-3b Cert │ No un-QA'd public issuance   │
│ L6   │ Inspector Identity Claim   │ Validated Competency Ticket │ No expired credential bypass │
│ L7   │ Physical Tool Sensor Data  │ Metrological Traceability   │ No expired calibration use   │
│ L8   │ Mobile Sensor & User Input │ Hardware-Signed Outbox      │ Private keys never touch RAM │
│ L9   │ Physical Asset Serial      │ W3C DID Asset Passport      │ No manual quarantine bypass  │
│ L10  │ Ordered Mutation Stream    │ 64-Byte Merkle-CRDT Node    │ Zero SQL UPDATE or DELETE    │
│ L11  │ Public Browser Request     │ Client WebCrypto Validation │ Zero server DB read/compute  │
└──────┴────────────────────────────┴─────────────────────────────┴──────────────────────────────┘
```

---

## 2. Tier L0: Global Root PKI Authority & Hierarchical Key Derivation

### 2.1 Cryptographic Key Hierarchy
```
           [INTEGIN Master Root CA (Ed25519 / Air-Gapped HSM)]
                                    │
                       ┌────────────┴────────────┐
                       ▼                         ▼
         [Tenant Alpha Sub-CA]         [Tenant Beta Sub-CA]
                 │                               │
       ┌─────────┴─────────┐           ┌─────────┴─────────┐
       ▼                   ▼           ▼                   ▼
[Office Issuer Key] [Tablet Key] [Office Issuer Key] [Tablet Key]
```

### 2.2 License Covenant Token Wire Format (`pkg/licensing/schema.go`)
```go
type LicenseCovenant struct {
    LicenseDID        string    `json:"license_did"`         // did:integin:license:<uuid>
    TenantDID         string    `json:"tenant_did"`          // did:integin:tenant:<uuid>
    IssuedAt          time.Time `json:"issued_at"`
    ExpiresAt         time.Time `json:"expires_at"`          // Hard expiration
    MaxFieldSeats     uint32    `json:"max_field_seats"`     // Concurrent tablets
    MaxStorageBytes   uint64    `json:"max_storage_bytes"`   // S3 allocation
    EnabledTiers      uint32    `json:"enabled_tiers"`       // Bitmask: L0-L11
    JurisdictionCodes []string  `json:"jurisdiction_codes"`  // ["SA", "AE", "US", "GB"]
    SignatureEd25519  []byte    `json:"signature_ed25519"`   // 64-byte signature
}
```

---

## 3. Tier L1: Dynamic AST Engine & Mathematical Engineering Grammar

### 3.1 Non-Turing Expression Sandbox Constraints
1.  **Memory Cap**: Bounded to strictly $\le 4\text{MB}$ heap during evaluation.
2.  **Latency SLA**: Evaluation strictly $< 50\mu\text{s}$ (target $< 15\mu\text{s}$).
3.  **Zero System Calls**: Zero network, disk, file, or goroutine execution.
4.  **Math Extensions**: Custom CEL function overloads for crane boom trigonometry:
    *   `math.Sin(rad float64) float64`
    *   `math.Cos(rad float64) float64`
    *   `math.Sqrt(val float64) float64`
    *   `math.Deg2Rad(deg float64) float64`

### 3.2 Formal Formula Definition Catalog
*   **ASME B30.5 Mobile Crane Proof-Load**:
    $$\text{ProofLoad}(C) = \begin{cases} C \times 1.25 & \text{if } C \le 20\text{t} \\ C \times 1.20 & \text{if } 20\text{t} < C \le 50\text{t} \\ C \times 1.15 & \text{if } C > 50\text{t} \end{cases}$$
*   **ISO 4309 Wire-Rope Discard Criteria**:
    $$\text{Discard} = (n_{\text{broken\_outer\_6d}} \ge 6) \lor (n_{\text{broken\_outer\_30d}} \ge 12) \lor \left(\frac{d_{\text{nominal}} - d_{\text{measured}}}{d_{\text{nominal}}} \ge 0.07\right)$$
*   **API 510 Pressure Vessel Minimum Wall Thickness**:
    $$t_{\text{min}} = \frac{P \times R}{S \times E - 0.6P} + C_{\text{allowance}}$$

---

## 4. Tier L2: Universal 7-Pillar Sovereign Regulatory & Compliance Matrix

### 4.1 The 7 Sovereign Compliance Pillars (195+ Countries)

```
       ┌─────────────────────────────────────────────────────────────┐
       │             7-PILLAR SOVEREIGN REGULATORY ENGINE            │
       └──────────────────────────────┬──────────────────────────────┘
                                      │
        ┌──────────────┬──────────────┼──────────────┬──────────────┐
        ▼              ▼              ▼              ▼              ▼
   1. Corporate   2. Tax & E-     3. Standards   4. Field       5. Env & ESG
      Registry       Invoicing       Conformity     Safety         (NCVC/EPA)
        │              │              │              │              │
        └──────────────┴──────────────┼──────────────┴──────────────┘
                                      │
                       ┌──────────────┴──────────────┐
                       ▼                             ▼
                6. Offshore/Energy            7. Data Sovereignty
                   (Mawani/BSEE)                 (SDAIA/GDPR)
```

### 4.2 Complete Country Profile Schema (`pkg/jurisdictions/schema.go`)
```go
type SovereignJurisdictionProfile struct {
    ISO2              string            `json:"iso2"`               // "SA"
    ISO3              string            `json:"iso3"`               // "SAU"
    CurrencyCode      string            `json:"currency_code"`      // "SAR"
    TaxAuthority      string            `json:"tax_authority"`      // "ZATCA"
    TaxIDLabel        string            `json:"tax_id_label"`       // "VAT / TRN"
    TaxRegexPattern   string            `json:"tax_regex_pattern"`  // "^3[0-9]{13}3$"
    EInvoicingFormat  string            `json:"e_invoicing_format"` // "UBL_2_1_ZATCA_PHASE_2"
    SafetyRegulator   string            `json:"safety_regulator"`   // "HRSD / High Commission"
    StandardsBody     string            `json:"standards_body"`     // "SASO"
    OffshoreRegulator string            `json:"offshore_regulator"` // "Mawani / Ports Authority"
    DataPrivacyLaw    string            `json:"data_privacy_law"`   // "SDAIA PDPL (Cabinet Dec. 98)"
    SubDivisions      []SubDivisionSpec `json:"sub_divisions"`
}
```

---

## 5. Tier L3: Dynamic Inspection Package Scoping & Multi-Discipline Matrix

*   **Discipline Classifications**:
    1.  **LIFTING**: Mobile Cranes (ASME B30.5), Overhead Traveling Cranes (B30.2), Forklifts (B56.1), Slings & Rigging Tackle (ASME B30.9 / LEEA).
    2.  **NDT (Non-Destructive Testing)**: Ultrasonic (UT), Magnetic Particle (MT), Liquid Penetrant (PT), Radiographic (RT), Eddy Current (ET).
    3.  **PRESSURE**: Boilers & Pressure Vessels (ASME BPVC Sec VIII / API 510), Piping (API 570), Aboveground Storage Tanks (API 653).
    4.  **ELECTRICAL**: Explosive Atmospheres (ATEX Directive 2014/34/EU / IECEx 60079).
*   **Statistical AQL Sampling (ISO 2859-1)**:
    *   Dynamically calculates sample size based on lot size and Inspection Level II (Normal / Tightened / Reduced).

---

## 6. Tier L4: Enterprise Hierarchy & High-Scale Async Ingress

### 6.1 Multinational Organizational Tree
```
[Global Holding Corporation]
            │
  ┌─────────┴─────────┐
  ▼                   ▼
[KSA Operating LLC] [UAE FZE Entity]
  │                   │
  ├─ Eastern Region   ├─ Abu Dhabi Offshore Yard
  │    ├─ Dammam Yard └─ Dubai JAFZA Workshop
  │    └─ Jubail Base
  └─ Western Region (Yanbu Rig Support)
```

### 6.2 Stripe-Grade Idempotency (`internal/idempotency`)
*   **Table**: `sync_idempotency_cache`
    *   `key_hash bytea PRIMARY KEY` (SHA-256 of `Idempotency-Key` + `tenant_id`)
    *   `request_hash bytea NOT NULL` (SHA-256 of canonical HTTP request body)
    *   `response_payload jsonb NOT NULL`
    *   `http_status smallint NOT NULL`
    *   `expires_at timestamptz NOT NULL` (24-hour automatic eviction)
*   **Monotonic State Machine**: `APPLIED` $\longrightarrow$ `DUPLICATE` $\longrightarrow$ `HELD` $\longrightarrow$ `CONFLICT` $\longrightarrow$ `SECURITY_FAILURE`.

---

## 7. Tier L5: Certificate Governance, 4-Eyes QA & Cryptographic Sealing

### 7.1 Four-Eyes Approval Directed Acyclic Graph (DAG)
```
[Field Inspector Draft] ──(Hardware Enclave Sign)──▶ [Supervisor Endorsement]
                                                              │
[Publicly Trustable PDF/A-3b] ◀──(Anti-Tamper Seal)── [Technical Authority QA]
```

### 7.2 Cryptographic Certificate Seal
*   **Format**: ISO 19005-3 PDF/A-3b with embedded machine-readable XML payload (`urn:integin:cert:v1`).
*   **Signer**: Ed25519 signature over SHA-256 digest of:
    $$\text{Digest} = \text{SHA256}(\text{TenantDID} \parallel \text{AssetDID} \parallel \text{StandardDID} \parallel \text{FormulaResults} \parallel \text{ToolSerials} \parallel \text{InspectorKey})$$

---

## 8. Tier L6: Dynamic Inspector Credentialing & Skill Matrix Verification

*   **Mandatory Verification Seam**:
    *   `LEEA Foundation / Lifting Inspector Qualification` (validity: 3 years).
    *   `ASNT SNT-TC-1A / ISO 9712 NDT Level II/III` (validity: 5 years).
    *   `Annual Visual Acuity`: Jaeger J1 near vision & Ishihara color perception.
*   **Automated Dispatch Gate**:
    $$\text{CanInspect}(\text{Inspector}, \text{Asset}) \iff (\text{CredentialActive} = \text{true}) \land (\text{AcuityValid} = \text{true}) \land (\text{AssetDiscipline} \in \text{CertDisciplines})$$

---

## 9. Tier L7: Dynamic Tool Calibration & Metrological Traceability (ISO 17020 § 6.2)

### 9.1 Metrological Traceability Chain
$$\text{Field Sensor (Load Cell / Gauge)} \xrightarrow[\text{Cal Cert}]{\text{Traceable}} \text{Accredited Lab (ISO/IEC 17025)} \xrightarrow[\text{NMI Cert}]{\text{Traceable}} \text{National Metrology Standard (NIST / SASO-NMEC)}$$

### 9.2 Automated Hard Lockout Rule
*   If `CurrentTimestamp >= CalibrationExpiryTimestamp`, work order execution is hard-blocked at both tablet UI and database API gateway:
    $$\text{LockoutTrigger} \implies \text{HTTP 422 Unprocessable Entity} \ (\text{ERR\_TOOL\_CALIBRATION\_EXPIRED})$$

---

## 10. Tier L8: Universal FIPS 140-3 Hardware Tablet Attestation

### 10.1 Silicon Enclave Architecture
*   **Apple iOS**: Secure Enclave Processor (SEP) initialized with `kSecAccessControlPrivateKeyUsage` requiring TouchID/FaceID biometric validation per work order.
*   **Android**: Android StrongBox KeyStore Keymaster enforcing hardware-backed master key derivation.
*   **Zero-Exposure Law**: Private signing keys *never* enter application memory, heap, or persistent disk storage.

### 10.2 Cryptographic Handshake & Nonce Ingress
```
Tablet (Offline) ───[Signs Outbox with Enclave Key + Monotonic Nonce]───▶ integin-server
                                                                              │
                                                            [Verifies Nonce > LastNonce]
                                                                              ▼
                                                                  [Reconciles Mutation]
```

---

## 11. Tier L9: W3C Decentralized Asset Passport & Technical Quarantine Lifecycle

### 11.1 Asset DID Document (`did:integin:asset:<uuid>`)
```json
{
  "@context": ["https://www.w3.org/ns/did/v1", "https://schema.integin.com/v1"],
  "id": "did:integin:asset:e3b0c442-98fc-1c14-9afb-4c8996fb9242",
  "controller": "did:integin:tenant:d4a1b2c3",
  "verificationMethod": [{
    "id": "did:integin:asset:e3b0c442#key-1",
    "type": "Ed25519VerificationKey2020",
    "controller": "did:integin:asset:e3b0c442",
    "publicKeyMultibase": "z6MkmvW...9e2a"
  }],
  "equipmentCategory": "MOBILE_CRANE",
  "ratedCapacity": "100_TON",
  "serialNumber": "LIEB-2024-NK1000",
  "quarantineStatus": "NORMAL_OPERATIONAL"
}
```

### 11.2 Autonomous Quarantine State Machine
```
[OPERATIONAL] ──(Proof Load Fails / Wire Discard Exceeded)──▶ [QUARANTINED]
                                                                   │
    [RESTORED] ◀──(Major Repair + 4-Eyes Recertification)──────────┘
```

---

## 12. Tier L10: Bitemporal Merkle-CRDT Tamper-Proof Audit Ledger (64-Byte Cache-Line)

### 12.1 Sub-Nano 64-Byte Memory-Aligned Struct Layout (`pkg/ledger/node.go`)
```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Blake3Hash (32 Bytes)                   |
|                        (Bytes 00 to 31)                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                      ParentHashPrefix (8 Bytes)               |
|                        (Bytes 32 to 39)                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                      TransactionTime (8 Bytes)                |
|                    (Unix Nano, Bytes 40 to 47)                |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        ValidTime (8 Bytes)                    |
|                    (Unix Nano, Bytes 48 to 55)                |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   EventType   | Severity |      Padding / Reserved (6 Bytes)  |
|    (Byte 56)  | (Byte 57)|            (Bytes 58 to 63)        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
TOTAL STRUCT SIZE = EXACTLY 64 BYTES (1 CPU CACHE LINE)
```

```go
type MerkleCRDTNode struct {
    Blake3Hash       [32]byte // Merkle node digest
    ParentHashPrefix uint64   // Fast 8-byte parent pointer
    TransactionTime  int64    // System monotonic Unix nano
    ValidTime        int64    // Field tablet GPS Unix nano
    EventType        uint8    // Mutation category code
    Severity         uint8    // Operational severity code
    Reserved         [6]byte  // Zero padding to enforce exact 64B size
}
```

---

## 13. Tier L11: Edge Zero-Knowledge QR Trust Gateway & Stateless WebCrypto

### 13.1 Stateless Client-Side Verification Fragment Protocol
```
https://verify.integin.com/#sig=BASE64_ED25519_SIG&did=ASSET_DID&cert=CERT_SHA256&root=MERKLE_ROOT
```
*   **The Zero-Cost Guarantee**:
    1.  The browser extracts the URL fragment (`window.location.hash`).
    2.  The browser calls the native browser `crypto.subtle.verify("Ed25519", pubKey, sig, certData)`.
    3.  **Result**: 100% cryptographic verification occurs locally in browser memory. The vendor server incurs **$0 compute cost**, makes **0 database calls**, and is **100% immune to DDoS attacks**.

---

*Official Deep Architecture Specification — Approved for Production Realization.*
